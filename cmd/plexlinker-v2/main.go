package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/gdamore/tcell/v2"
	"plexlinker/internal/aria2"
	"plexlinker/internal/aria2manager"
	"plexlinker/internal/buildinfo"
	"plexlinker/internal/client"
	"plexlinker/internal/config"
	"plexlinker/internal/downloads"
	"plexlinker/internal/storage"
	tvui "plexlinker/internal/tvrenamer/ui"
	"plexlinker/internal/v2tui"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "rename":
			if err := runTVRenamer(); err != nil {
				fmt.Fprintln(os.Stderr, "PlexLinker TV renamer:", err)
				os.Exit(1)
			}
			return
		case "doctor":
			if err := runDoctor(os.Args[2:]); err != nil {
				fmt.Fprintln(os.Stderr, "PlexLinker doctor:", err)
				os.Exit(1)
			}
			return
		}
	}
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "PlexLinker v2:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.LoadV2Config()
	if err != nil {
		return err
	}
	cfg.Normalize()
	if cfg.Aria2Secret == "" {
		cfg.Aria2Secret, err = aria2manager.GenerateSecret()
		if err != nil {
			return err
		}
		if err := cfg.SaveV2(); err != nil {
			return err
		}
	}
	ariaClient := aria2.NewClient(cfg.Aria2Endpoint, cfg.Aria2Secret)
	manager, err := aria2manager.New(aria2manager.Config{Endpoint: cfg.Aria2Endpoint, Secret: cfg.Aria2Secret, AutoStart: cfg.Aria2AutoStart, MaxConcurrent: cfg.MaxConcurrentDownloads})
	if err != nil {
		return err
	}
	startupMessage := ""
	startupContext, cancelStartup := context.WithTimeout(context.Background(), 12*time.Second)
	status, startupErr := manager.Ensure(startupContext)
	cancelStartup()
	if startupErr != nil {
		startupMessage = "aria2 startup: " + startupErr.Error()
	} else {
		startupMessage = status.Message
	}
	model := v2tui.New(v2tui.Services{
		Remote:         v2tui.NewRemoteService(client.NewClient(cfg)),
		Volumes:        storage.NewVolumeProvider(),
		Directories:    storage.NewBrowser(),
		Downloads:      downloads.NewService(ariaClient),
		Username:       cfg.Username,
		Password:       cfg.Password,
		StartupMessage: startupMessage,
		Config:         cfg,
		SaveConfig:     func(updated *config.Config) error { return updated.SaveV2() },
		Version:        buildinfo.Version,
		RenamerCommand: func() *exec.Cmd { executable, _ := os.Executable(); return exec.Command(executable, "rename") },
	})
	program := tea.NewProgram(model)
	_, err = program.Run()
	return err
}

func runTVRenamer() error {
	screen, err := tcell.NewScreen()
	if err != nil {
		return fmt.Errorf("create terminal screen: %w", err)
	}
	if err := screen.Init(); err != nil {
		return fmt.Errorf("initialize terminal screen: %w", err)
	}
	defer screen.Fini()
	ui := tvui.NewTVBrowserUI(&screen)
	if err := ui.LoadCurrentDirectory(); err != nil {
		return err
	}
	for {
		ui.Draw()
		switch event := screen.PollEvent().(type) {
		case *tcell.EventKey:
			if ui.HandleKeyEvent(event) {
				return nil
			}
		case *tcell.EventResize:
			screen.Sync()
		case *tcell.EventInterrupt:
			return nil
		}
	}
}
