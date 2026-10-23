//go:build linux

package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func persistenceProbe(ctx context.Context) (string, error) {
	systemctl, err := exec.LookPath("systemctl")
	if err != nil {
		return "aria2 is running in native daemon fallback mode", nil
	}
	output, err := exec.CommandContext(ctx, systemctl, "--user", "is-active", "plexlinker-aria2.service").CombinedOutput()
	if err == nil && strings.TrimSpace(string(output)) == "active" {
		return "systemd user service plexlinker-aria2.service is active", nil
	}
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	// Manager.Ensure already proved RPC connectivity. A missing user bus causes
	// Linux startup to intentionally fall back to aria2's detached daemon mode.
	return fmt.Sprintf("aria2 is running in native daemon fallback mode (systemd-user unavailable: %s)", strings.TrimSpace(string(output))), nil
}
