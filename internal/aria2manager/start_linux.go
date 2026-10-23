//go:build linux

package aria2manager

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func startPlatform(ctx context.Context, executable, configPath string) error {
	path, err := exec.LookPath(executable)
	if err != nil {
		return fmt.Errorf("aria2c is not installed or not on PATH: %w", err)
	}
	if systemctl, lookupErr := exec.LookPath("systemctl"); lookupErr == nil {
		if err := startSystemdUserService(ctx, systemctl, path, configPath); err == nil {
			return nil
		}
	}
	command := exec.CommandContext(ctx, path, "--conf-path="+configPath, "--daemon=true")
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("start aria2 daemon: %w (%s)", err, output)
	}
	return nil
}

func startSystemdUserService(ctx context.Context, systemctl, executable, configPath string) error {
	configRoot, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	unitDirectory := filepath.Join(configRoot, "systemd", "user")
	if err := os.MkdirAll(unitDirectory, 0o700); err != nil {
		return err
	}
	unit := strings.Join([]string{
		"[Unit]", "Description=PlexLinker aria2 download service", "After=network-online.target", "Wants=network-online.target", "",
		"[Service]", "Type=simple", fmt.Sprintf("ExecStart=%s --conf-path=%s", systemdQuote(executable), systemdQuote(configPath)), "Restart=on-failure", "RestartSec=3", "",
		"[Install]", "WantedBy=default.target", "",
	}, "\n")
	unitPath := filepath.Join(unitDirectory, "plexlinker-aria2.service")
	if err := os.WriteFile(unitPath, []byte(unit), 0o600); err != nil {
		return err
	}
	if output, err := exec.CommandContext(ctx, systemctl, "--user", "daemon-reload").CombinedOutput(); err != nil {
		return fmt.Errorf("systemd daemon-reload: %w (%s)", err, output)
	}
	if output, err := exec.CommandContext(ctx, systemctl, "--user", "enable", "--now", "plexlinker-aria2.service").CombinedOutput(); err != nil {
		return fmt.Errorf("enable aria2 user service: %w (%s)", err, output)
	}
	return nil
}

func systemdQuote(value string) string {
	value = strings.ReplaceAll(value, "%", "%%")
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return `"` + value + `"`
}
