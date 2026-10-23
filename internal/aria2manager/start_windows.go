//go:build windows

package aria2manager

import (
	"context"
	"fmt"
	"os/exec"
)

const aria2TaskName = "PlexLinker_aria2"

func startPlatform(ctx context.Context, executable, configPath string) error {
	path, err := exec.LookPath(executable)
	if err != nil {
		return fmt.Errorf("aria2c.exe is not installed or not on PATH: %w", err)
	}
	action := fmt.Sprintf(`"%s" --conf-path="%s"`, path, configPath)
	create := exec.CommandContext(ctx, "schtasks", "/create", "/tn", aria2TaskName, "/tr", action, "/sc", "ONLOGON", "/rl", "LIMITED", "/f")
	if output, err := create.CombinedOutput(); err != nil {
		return fmt.Errorf("register persistent aria2 task: %w (%s)", err, output)
	}
	run := exec.CommandContext(ctx, "schtasks", "/run", "/tn", aria2TaskName)
	if output, err := run.CombinedOutput(); err != nil {
		return fmt.Errorf("start persistent aria2 task: %w (%s)", err, output)
	}
	return nil
}
