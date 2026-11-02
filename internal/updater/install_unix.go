//go:build !windows

package updater

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

func installStaged(staged, rollback string) error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	executable, _ = filepath.EvalSymlinks(executable)
	if err := copyFile(executable, rollback, 0o700); err != nil {
		return fmt.Errorf("create rollback binary: %w", err)
	}
	if err := copyFile(staged, executable, 0o755); err == nil {
		return nil
	}
	command := exec.Command("sudo", "install", "-m", "0755", staged, executable)
	command.Stdin, command.Stdout, command.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("install update with sudo: %w", err)
	}
	return nil
}

func installRollback(rollback string) error {
	if _, err := os.Stat(rollback); err != nil {
		return fmt.Errorf("rollback binary is unavailable: %w", err)
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	executable, _ = filepath.EvalSymlinks(executable)
	temporary := rollback + ".current"
	if err := copyFile(executable, temporary, 0o700); err != nil {
		return err
	}
	if err := copyFile(rollback, executable, 0o755); err != nil {
		command := exec.Command("sudo", "install", "-m", "0755", rollback, executable)
		command.Stdin, command.Stdout, command.Stderr = os.Stdin, os.Stdout, os.Stderr
		if runErr := command.Run(); runErr != nil {
			return fmt.Errorf("restore rollback with sudo: %w", runErr)
		}
	}
	return os.Rename(temporary, rollback)
}

func copyFile(source, destination string, mode os.FileMode) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() { _ = input.Close() }()
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(destination), ".plexlinker-*")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer func() { _ = os.Remove(temporaryName) }()
	if err := temporary.Chmod(mode); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := io.Copy(temporary, input); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryName, destination)
}
