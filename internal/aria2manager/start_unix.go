//go:build !windows && !linux

package aria2manager

import (
	"context"
	"fmt"
	"os/exec"
)

func startPlatform(ctx context.Context, executable, configPath string) error {
	path, err := exec.LookPath(executable)
	if err != nil {
		return fmt.Errorf("aria2c is not installed or not on PATH: %w", err)
	}
	command := exec.CommandContext(ctx, path, "--conf-path="+configPath, "--daemon=true")
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("start aria2 daemon: %w (%s)", err, output)
	}
	return nil
}
