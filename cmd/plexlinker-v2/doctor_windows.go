//go:build windows

package main

import (
	"context"
	"fmt"
	"os/exec"
)

func persistenceProbe(ctx context.Context) (string, error) {
	command := exec.CommandContext(ctx, "schtasks", "/query", "/tn", `PlexLinker_aria2`, "/fo", "LIST")
	if output, err := command.CombinedOutput(); err != nil {
		return "", fmt.Errorf("Task Scheduler entry PlexLinker_aria2 is unavailable: %w (%s)", err, output)
	}
	return "Task Scheduler entry PlexLinker_aria2 is registered", nil
}
