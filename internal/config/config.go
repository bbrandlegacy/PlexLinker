package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type ScriptFormat string

const (
	ScriptFormatTXT ScriptFormat = "txt"
	ScriptFormatBAT ScriptFormat = "bat"
	ScriptFormatSH  ScriptFormat = "sh"
)

type Config struct {
	BaseURL      string       `json:"base_url"`
	Username     string       `json:"username"`
	Password     string       `json:"password"`
	DebugMode    bool         `json:"debug_mode"`
	ScriptFormat ScriptFormat `json:"script_format"`
	MoviesPath   string       `json:"movies_path"`
	TVShowsPath  string       `json:"tv_shows_path"`
	OtherPath    string       `json:"other_path"`
	Usernames    []string     `json:"usernames,omitempty"`
	LastUser     string       `json:"last_user,omitempty"`
}

func NewConfig() *Config {
	return &Config{
		BaseURL:      "",
		Username:     "",
		Password:     "",
		DebugMode:    false,
		ScriptFormat: ScriptFormatBAT,
		MoviesPath:   "auto",
		TVShowsPath:  "auto",
		OtherPath:    "auto",
		Usernames:    []string{},
		LastUser:     "",
	}
}

// Normalize performs normalization of config values (e.g., adding trailing slashes)
// Call this before Validate() to ensure config is in canonical form
func (c *Config) Normalize() {
	// Ensure BaseURL has trailing slash
	if c.BaseURL != "" && !strings.HasSuffix(c.BaseURL, "/") {
		c.BaseURL += "/"
	}

	// Normalize script format
	if c.ScriptFormat == "" {
		c.ScriptFormat = ScriptFormatBAT
	}
}

// Validate checks if the config has all required fields and valid values
// Does not mutate the config - call Normalize() first if needed
func (c *Config) Validate() error {
	if c.BaseURL == "" {
		return fmt.Errorf("base URL cannot be empty")
	}

	if _, err := url.Parse(c.BaseURL); err != nil {
		return fmt.Errorf("invalid base URL: %v", err)
	}

	if c.Username == "" {
		return fmt.Errorf("username cannot be empty")
	}

	if c.Password == "" {
		return fmt.Errorf("password cannot be empty")
	}

	if c.MoviesPath == "" {
		return fmt.Errorf("movies path cannot be empty")
	}

	if c.TVShowsPath == "" {
		return fmt.Errorf("tv shows path cannot be empty")
	}

	if c.OtherPath == "" {
		return fmt.Errorf("other path cannot be empty")
	}

	return nil
}

func GetConfigPath() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %v", err)
	}

	configDir := filepath.Dir(execPath)
	return filepath.Join(configDir, "plexlinker.json"), nil
}

func LoadConfig() (*Config, error) {
	// Priority 1: Check environment variables first (for development/debugging)
	if baseURL := os.Getenv("PLEX_BASE_URL"); baseURL != "" {
		config := &Config{
			BaseURL:  baseURL,
			Username: os.Getenv("PLEX_USERNAME"),
			Password: os.Getenv("PLEX_PASSWORD"),
		}
		return config, nil
	}

	// Priority 2: Load from config file
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return NewConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	config := NewConfig()
	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	// Normalize config values after loading
	config.Normalize()

	return config, nil
}

func (c *Config) Save() error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}
