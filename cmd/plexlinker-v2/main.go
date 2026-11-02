package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/gdamore/tcell/v2"
	"plexlinker/internal/aria2"
	"plexlinker/internal/aria2manager"
	"plexlinker/internal/bootstrap"
	"plexlinker/internal/buildinfo"
	"plexlinker/internal/client"
	"plexlinker/internal/config"
	"plexlinker/internal/diagnostics"
	"plexlinker/internal/downloads"
	"plexlinker/internal/storage"
	tvui "plexlinker/internal/tvrenamer/ui"
	"plexlinker/internal/updater"
	"plexlinker/internal/v2tui"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version", "--version", "-version":
			fmt.Printf("PlexLinker %s\n", buildinfo.Version)
			return
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
		case "update":
			if err := runMaintenance("update"); err != nil {
				fmt.Fprintln(os.Stderr, "PlexLinker update:", err)
				os.Exit(1)
			}
			return
		case "rollback":
			if err := runMaintenance("rollback"); err != nil {
				fmt.Fprintln(os.Stderr, "PlexLinker rollback:", err)
				os.Exit(1)
			}
			return
		}
	}
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "PlexLinker:", err)
		os.Exit(1)
	}
}

func run() error {
	if err := bootstrap.Ensure(); err != nil {
		return err
	}
	cfg, err := config.LoadV2Config()
	if err != nil {
		return err
	}
	cfg.Normalize()
	configPath, err := config.GetV2ConfigPath()
	if err != nil {
		return err
	}
	dataDirectory := filepath.Dir(configPath)
	eventLog, err := diagnostics.New(filepath.Join(dataDirectory, "plexlinker.log"))
	if err != nil {
		return err
	}
	eventLog.Write("INFO", "PlexLinker "+buildinfo.Version+" starting")
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
		eventLog.Write("ERROR", startupMessage)
	} else {
		startupMessage = status.Message
		eventLog.Write("INFO", startupMessage)
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
		ApplyConfig: func(updated *config.Config) (v2tui.AppliedServices, error) {
			updated.Normalize()
			if err := updated.ValidateV2(); err != nil {
				return v2tui.AppliedServices{}, err
			}
			newAriaClient := aria2.NewClient(updated.Aria2Endpoint, updated.Aria2Secret)
			newManager, err := aria2manager.New(aria2manager.Config{Endpoint: updated.Aria2Endpoint, Secret: updated.Aria2Secret, AutoStart: updated.Aria2AutoStart, MaxConcurrent: updated.MaxConcurrentDownloads})
			if err != nil {
				return v2tui.AppliedServices{}, err
			}
			ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
			defer cancel()
			ariaStatus, err := newManager.Ensure(ctx)
			if err != nil {
				return v2tui.AppliedServices{}, fmt.Errorf("apply aria2 settings: %w", err)
			}
			if err := newAriaClient.ChangeGlobalOption(ctx, map[string]string{"max-concurrent-downloads": strconv.Itoa(updated.MaxConcurrentDownloads)}); err != nil {
				return v2tui.AppliedServices{}, fmt.Errorf("apply aria2 concurrency: %w", err)
			}
			if err := updated.SaveV2(); err != nil {
				return v2tui.AppliedServices{}, err
			}
			return v2tui.AppliedServices{
				Remote: v2tui.NewRemoteService(client.NewClient(updated)), Downloads: downloads.NewService(newAriaClient),
				Username: updated.Username, Password: updated.Password, Config: updated,
				Message: "Settings saved and applied • " + ariaStatus.Message,
			}, nil
		},
		Version:        buildinfo.Version,
		RenamerCommand: func() *exec.Cmd { executable, _ := os.Executable(); return exec.Command(executable, "rename") },
		Diagnostics:    eventLog,
		Maintenance:    updater.New(buildinfo.Version, dataDirectory),
		MaintenanceCommand: func(action string) *exec.Cmd {
			executable, _ := os.Executable()
			command := exec.Command(executable, action)
			command.Env = append(os.Environ(), "PLEXLINKER_UPDATE_PARENT_PID="+strconv.Itoa(os.Getpid()))
			return command
		},
	})
	program := tea.NewProgram(model)
	_, err = program.Run()
	return err
}

func runMaintenance(action string) error {
	configPath, err := config.GetV2ConfigPath()
	if err != nil {
		return err
	}
	manager := updater.New(buildinfo.Version, filepath.Dir(configPath))
	switch action {
	case "update":
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		latest, err := manager.Check(ctx)
		if err != nil {
			return err
		}
		if !latest.Newer {
			return fmt.Errorf("%s is already the latest release", buildinfo.Version)
		}
		fmt.Printf("Update %s to %s from %s? [y/N]: ", buildinfo.Version, latest.Version, latest.URL)
		answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		if answer = strings.TrimSpace(strings.ToLower(answer)); answer != "y" && answer != "yes" {
			return fmt.Errorf("update cancelled")
		}
		fmt.Printf("Downloading and verifying %s…\n", latest.Asset)
		release, err := manager.InstallLatest(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("Installed PlexLinker %s. PlexLinker will close; reopen it to use the update.\n", release.Version)
		return nil
	case "rollback":
		fmt.Print("Replace PlexLinker with the saved rollback binary? [y/N]: ")
		answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		if answer = strings.TrimSpace(strings.ToLower(answer)); answer != "y" && answer != "yes" {
			return fmt.Errorf("rollback cancelled")
		}
		if err := manager.Rollback(); err != nil {
			return err
		}
		fmt.Println("Rollback installed. PlexLinker will close; reopen it to use the previous version.")
		return nil
	default:
		return fmt.Errorf("unknown maintenance action %q", action)
	}
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
