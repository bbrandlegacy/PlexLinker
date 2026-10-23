package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"plexlinker/internal/aria2"
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
	ariaClient := aria2.NewClient(cfg.Aria2Endpoint, cfg.Aria2Secret)
	model := v2tui.New(v2tui.Services{
		Remote:      v2tui.NewRemoteService(client.NewClient(cfg)),
		Volumes:     storage.NewVolumeProvider(),
		Directories: storage.NewBrowser(),
		Downloads:   downloads.NewService(ariaClient),
		Username:    cfg.Username,
		Password:    cfg.Password,
	})
	program := tea.NewProgram(model)
	_, err = program.Run()
	return err
}
