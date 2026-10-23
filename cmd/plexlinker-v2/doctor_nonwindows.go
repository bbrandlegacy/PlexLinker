//go:build !windows

package main

import (
	"context"
	"runtime"
)

func persistenceProbe(context.Context) (string, error) {
	if runtime.GOOS == "linux" {
		return "aria2 is managed by systemd-user when available, otherwise daemon mode", nil
	}
	return "aria2 is running in daemon mode", nil
}
