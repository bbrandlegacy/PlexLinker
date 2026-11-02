//go:build windows

package updater

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func installStaged(staged, rollback string) error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	if err := verifyWindowsReplaceAccess(executable); err != nil {
		return err
	}
	if err := copyWindowsFile(executable, rollback); err != nil {
		return fmt.Errorf("create rollback binary: %w", err)
	}
	return scheduleWindowsReplacement(staged, executable)
}

func installRollback(rollback string) error {
	if _, err := os.Stat(rollback); err != nil {
		return fmt.Errorf("rollback binary is unavailable: %w", err)
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	if err := verifyWindowsReplaceAccess(executable); err != nil {
		return err
	}
	return scheduleWindowsRollback(rollback, executable)
}

func verifyWindowsReplaceAccess(executable string) error {
	file, err := os.CreateTemp(filepath.Dir(executable), ".plexlinker-update-test-*")
	if err != nil {
		return fmt.Errorf("PlexLinker cannot update %s: %w; rerun the latest setup package once to enable in-app updates", executable, err)
	}
	name := file.Name()
	_ = file.Close()
	return os.Remove(name)
}

func scheduleWindowsReplacement(source, destination string) error {
	parent := os.Getpid()
	if value := os.Getenv("PLEXLINKER_UPDATE_PARENT_PID"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			parent = parsed
		}
	}
	script := fmt.Sprintf(`$ErrorActionPreference='Stop'; Wait-Process -Id %d -ErrorAction SilentlyContinue; Copy-Item -LiteralPath '%s' -Destination '%s' -Force; Start-Process -FilePath '%s'`, parent, escapePowerShell(source), escapePowerShell(destination), escapePowerShell(destination))
	command := exec.Command("powershell.exe", "-NoProfile", "-WindowStyle", "Hidden", "-Command", script)
	if err := command.Start(); err != nil {
		return fmt.Errorf("schedule Windows replacement: %w", err)
	}
	return command.Process.Release()
}

func scheduleWindowsRollback(rollback, destination string) error {
	parent := os.Getpid()
	if value := os.Getenv("PLEXLINKER_UPDATE_PARENT_PID"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			parent = parsed
		}
	}
	swap := rollback + ".swap"
	script := fmt.Sprintf(`$ErrorActionPreference='Stop'; Wait-Process -Id %d -ErrorAction SilentlyContinue; Copy-Item -LiteralPath '%s' -Destination '%s' -Force; Copy-Item -LiteralPath '%s' -Destination '%s' -Force; Move-Item -LiteralPath '%s' -Destination '%s' -Force; Start-Process -FilePath '%s'`, parent, escapePowerShell(destination), escapePowerShell(swap), escapePowerShell(rollback), escapePowerShell(destination), escapePowerShell(swap), escapePowerShell(rollback), escapePowerShell(destination))
	command := exec.Command("powershell.exe", "-NoProfile", "-WindowStyle", "Hidden", "-Command", script)
	if err := command.Start(); err != nil {
		return fmt.Errorf("schedule Windows rollback: %w", err)
	}
	return command.Process.Release()
}

func copyWindowsFile(source, destination string) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		return err
	}
	return os.WriteFile(destination, data, 0o700)
}

func escapePowerShell(value string) string {
	return strings.ReplaceAll(filepath.Clean(value), "'", "''")
}
