package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"plexlinker/internal/tvrenamer/patterns"
)

type TVRenamerConfig struct {
	StartingDirectory   string                 `json:"starting_directory"`
	PlexSettings        *patterns.PlexSettings `json:"plex_settings"`
	CreateSeasonFolders bool                   `json:"create_season_folders"`
	AutoBackup          bool                   `json:"auto_backup"`
	LogRenames          bool                   `json:"log_renames"`
	CustomPatterns      []string               `json:"custom_patterns"`
}

func NewTVRenamerConfig() *TVRenamerConfig {
	wd, _ := os.Getwd()
	return &TVRenamerConfig{
		StartingDirectory: wd,
		PlexSettings: &patterns.PlexSettings{
			CreateSeasonFolders: true,
			IncludeYear:         true,
			IncludeQuality:      false,
			IncludeCodec:        false,
			SanitizeFilenames:   true,
		},
		CreateSeasonFolders: true,
		AutoBackup:          false,
		LogRenames:          true,
		CustomPatterns:      []string{},
	}
}

func (c *TVRenamerConfig) Validate() error {
	if c.StartingDirectory == "" {
		return fmt.Errorf("starting directory cannot be empty")
	}

	if _, err := os.Stat(c.StartingDirectory); os.IsNotExist(err) {
		return fmt.Errorf("starting directory does not exist: %s", c.StartingDirectory)
	}

	return nil
}

func GetTVRenamerConfigPath() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %v", err)
	}

	configDir := filepath.Dir(execPath)
	return filepath.Join(configDir, "tvrenamer.json"), nil
}

func LoadTVRenamerConfig() (*TVRenamerConfig, error) {
	configPath, err := GetTVRenamerConfigPath()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return NewTVRenamerConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read TV renamer config file: %v", err)
	}

	config := NewTVRenamerConfig()
	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse TV renamer config file: %v", err)
	}

	return config, nil
}

func (c *TVRenamerConfig) Save() error {
	configPath, err := GetTVRenamerConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal TV renamer config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write TV renamer config file: %v", err)
	}

	return nil
}
