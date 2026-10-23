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
	BaseURL                string       `json:"base_url"`
	Username               string       `json:"username"`
	Password               string       `json:"password"`
	DebugMode              bool         `json:"debug_mode"`
	ScriptFormat           ScriptFormat `json:"script_format"`
	MoviesPath             string       `json:"movies_path"`
	TVShowsPath            string       `json:"tv_shows_path"`
	OtherPath              string       `json:"other_path"`
	Usernames              []string     `json:"usernames,omitempty"`
	LastUser               string       `json:"last_user,omitempty"`
	Aria2Endpoint          string       `json:"aria2_endpoint,omitempty"`
	Aria2Secret            string       `json:"aria2_secret,omitempty"`
	Aria2AutoStart         bool         `json:"aria2_auto_start"`
	MaxConcurrentDownloads int          `json:"max_concurrent_downloads,omitempty"`
	FavoritePaths          []string     `json:"favorite_paths,omitempty"`
}

func NewConfig() *Config {
	return &Config{
		BaseURL:                "",
		Username:               "",
		Password:               "",
		DebugMode:              false,
		ScriptFormat:           ScriptFormatBAT,
		MoviesPath:             "auto",
		TVShowsPath:            "auto",
		OtherPath:              "auto",
		Usernames:              []string{},
		LastUser:               "",
		Aria2Endpoint:          "http://127.0.0.1:6800/jsonrpc",
		Aria2AutoStart:         true,
		MaxConcurrentDownloads: 3,
		FavoritePaths:          []string{},
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
	if c.Aria2Endpoint == "" {
		c.Aria2Endpoint = "http://127.0.0.1:6800/jsonrpc"
	}
	if c.MaxConcurrentDownloads <= 0 {
		c.MaxConcurrentDownloads = 3
	}
}

// ValidateV2 validates the settings required by the RPC-based v2 application.
func (c *Config) ValidateV2() error {
	c.Normalize()
	if err := c.Validate(); err != nil {
		return err
	}
	endpoint, err := url.Parse(c.Aria2Endpoint)
	if err != nil || endpoint.Scheme == "" || endpoint.Host == "" {
		return fmt.Errorf("invalid aria2 endpoint %q", c.Aria2Endpoint)
	}
	return nil
}

// Validate checks if the config has all required fields and valid values
// Does not mutate the config - call Normalize() first if needed
func (c *Config) Validate() error {
	if c.BaseURL == "" {
		return fmt.Errorf("base URL cannot be empty")
	}

	baseURL, err := url.Parse(c.BaseURL)
	if err != nil || (baseURL.Scheme != "http" && baseURL.Scheme != "https") || baseURL.Host == "" {
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

// GetV2ConfigPath returns the per-user v2 config location. PLEXLINKER_CONFIG
// can override it for portable installs, development, and automated tests.
func GetV2ConfigPath() (string, error) {
	if configured := os.Getenv("PLEXLINKER_CONFIG"); configured != "" {
		return configured, nil
	}
	root, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find user config directory: %w", err)
	}
	return filepath.Join(root, "PlexLinker", "config.json"), nil
}

// LoadV2Config loads the per-user config, migrating an existing adjacent v1
// config on first use. Environment variables remain the highest priority.
func LoadV2Config() (*Config, error) {
	if os.Getenv("PLEX_BASE_URL") != "" {
		return LoadConfig()
	}
	path, err := GetV2ConfigPath()
	if err != nil {
		return nil, err
	}
	if data, readErr := os.ReadFile(path); readErr == nil {
		cfg := NewConfig()
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse v2 config: %w", err)
		}
		cfg.Normalize()
		return cfg, nil
	} else if !os.IsNotExist(readErr) {
		return nil, fmt.Errorf("read v2 config: %w", readErr)
	}

	legacyPath, legacyErr := GetConfigPath()
	if legacyErr == nil {
		if _, statErr := os.Stat(legacyPath); statErr == nil {
			cfg, err := LoadConfig()
			if err != nil {
				return nil, err
			}
			if err := cfg.SaveV2(); err != nil {
				return nil, fmt.Errorf("migrate v1 config: %w", err)
			}
			return cfg, nil
		}
	}
	return NewConfig(), nil
}

// SaveV2 atomically writes the per-user v2 configuration with user-only permissions.
func (c *Config) SaveV2() error {
	path, err := GetV2ConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("encode v2 config: %w", err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary config: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write temporary config: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("sync temporary config: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary config: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace v2 config: %w", err)
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
		config := NewConfig()
		config.BaseURL = baseURL
		config.Username = os.Getenv("PLEX_USERNAME")
		config.Password = os.Getenv("PLEX_PASSWORD")
		if endpoint := os.Getenv("PLEXLINKER_ARIA2_ENDPOINT"); endpoint != "" {
			config.Aria2Endpoint = endpoint
		}
		if secret := os.Getenv("PLEXLINKER_ARIA2_SECRET"); secret != "" {
			config.Aria2Secret = secret
		}
		config.Normalize()
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
