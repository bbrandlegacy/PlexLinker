package main

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"plexlinker/internal/aria2"
	"plexlinker/internal/aria2manager"
	"plexlinker/internal/client"
	"plexlinker/internal/config"
	"plexlinker/internal/downloads"
	"plexlinker/internal/storage"
	"plexlinker/internal/v2tui"
)

func main() {
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
	})
	program := tea.NewProgram(model)
	_, err = program.Run()
	return err
}
