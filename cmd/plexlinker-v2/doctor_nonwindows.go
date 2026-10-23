//go:build !windows && !linux

package main

import "context"

func persistenceProbe(context.Context) (string, error) {
	return "aria2 is running in daemon mode", nil
}
