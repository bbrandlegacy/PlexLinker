// Package aria2manager installs and starts a persistent local aria2 RPC service.
package aria2manager

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"plexlinker/internal/aria2"
)

type Config struct {
	Executable    string
	Endpoint      string
	Secret        string
	AutoStart     bool
	MaxConcurrent int
	DataDirectory string
}

type Manager struct {
	config Config
	client *aria2.Client
}

type Status struct {
	Running                      bool
	Version, ConfigPath, Message string
}

func New(config Config) (*Manager, error) {
	if config.Endpoint == "" {
		config.Endpoint = "http://127.0.0.1:6800/jsonrpc"
	}
	if config.Executable == "" {
		config.Executable = "aria2c"
	}
	if config.MaxConcurrent <= 0 {
		config.MaxConcurrent = 3
	}
	if config.DataDirectory == "" {
		root, err := os.UserConfigDir()
		if err != nil {
			return nil, fmt.Errorf("find user config directory: %w", err)
		}
		config.DataDirectory = filepath.Join(root, "PlexLinker", "aria2")
	}
	return &Manager{config: config, client: aria2.NewClient(config.Endpoint, config.Secret)}, nil
}

func GenerateSecret() (string, error) {
	buffer := make([]byte, 24)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate aria2 secret: %w", err)
	}
	return hex.EncodeToString(buffer), nil
}

func (m *Manager) Ensure(ctx context.Context) (Status, error) {
	if version, err := m.client.Version(ctx); err == nil {
		return Status{Running: true, Version: version, Message: "Connected to aria2 " + version}, nil
	}
	if !m.config.AutoStart {
		return Status{}, fmt.Errorf("aria2 is unavailable at %s and automatic startup is disabled", m.config.Endpoint)
	}
	if err := validateLocalEndpoint(m.config.Endpoint); err != nil {
		return Status{}, err
	}
	configPath, err := m.writeConfig()
	if err != nil {
		return Status{}, err
	}
	if err := startPlatform(ctx, m.config.Executable, configPath); err != nil {
		return Status{ConfigPath: configPath}, err
	}

	deadline := time.Now().Add(8 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		version, err := m.client.Version(ctx)
		if err == nil {
			return Status{Running: true, Version: version, ConfigPath: configPath, Message: "Started persistent aria2 " + version}, nil
		}
		lastErr = err
		select {
		case <-ctx.Done():
			return Status{ConfigPath: configPath}, ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
	return Status{ConfigPath: configPath}, fmt.Errorf("aria2 did not become ready: %w", lastErr)
}

func (m *Manager) writeConfig() (string, error) {
	if strings.ContainsAny(m.config.Secret, "\r\n") {
		return "", fmt.Errorf("aria2 secret contains a newline")
	}
	if err := os.MkdirAll(m.config.DataDirectory, 0o700); err != nil {
		return "", fmt.Errorf("create aria2 directory: %w", err)
	}
	sessionPath := filepath.Join(m.config.DataDirectory, "aria2.session")
	if file, err := os.OpenFile(sessionPath, os.O_CREATE, 0o600); err != nil {
		return "", fmt.Errorf("create aria2 session: %w", err)
	} else {
		if err := file.Close(); err != nil {
			return "", fmt.Errorf("close aria2 session: %w", err)
		}
	}
	parsed, _ := url.Parse(m.config.Endpoint)
	_, portText, _ := net.SplitHostPort(parsed.Host)
	if portText == "" {
		portText = "6800"
	}
	lines := []string{
		"enable-rpc=true", "rpc-listen-all=false", "rpc-listen-port=" + portText,
		"rpc-secret=" + m.config.Secret, "input-file=" + sessionPath, "save-session=" + sessionPath,
		"save-session-interval=30", fmt.Sprintf("max-concurrent-downloads=%d", m.config.MaxConcurrent),
		"continue=true", "auto-file-renaming=false", "allow-overwrite=false", "file-allocation=none",
		"console-log-level=warn", "log=" + filepath.Join(m.config.DataDirectory, "aria2.log"),
	}
	configPath := filepath.Join(m.config.DataDirectory, "aria2.conf")
	if err := os.WriteFile(configPath, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		return "", fmt.Errorf("write aria2 config: %w", err)
	}
	return configPath, nil
}

func validateLocalEndpoint(endpoint string) error {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme != "http" {
		return fmt.Errorf("automatic aria2 startup requires a local HTTP endpoint")
	}
	host := parsed.Hostname()
	if host != "127.0.0.1" && host != "localhost" && host != "::1" {
		return fmt.Errorf("refusing to auto-start aria2 for non-local endpoint %q", endpoint)
	}
	return nil
}
