// Package bootstrap implements the explicit first-run installer handoff.
package bootstrap

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const skipEnvironment = "PLEXLINKER_SKIP_SETUP"

// Ensure checks whether the one-time host setup has completed and, when it has
// not, asks before invoking the platform installer shipped beside the binary.
func Ensure() error {
	_, ariaErr := exec.LookPath("aria2c")
	if os.Getenv(skipEnvironment) == "1" || (setupComplete() && ariaErr == nil) {
		return nil
	}
	// Preserve existing Windows installations that already have their only
	// external runtime dependency. The Windows installer remains available for
	// portable-to-machine installation and fresh hosts.
	if runtime.GOOS == "windows" && ariaErr == nil {
		return nil
	}
	script, command, changes, err := installer()
	if err != nil {
		return err
	}
	if script == "" {
		return nil
	}
	fmt.Fprintln(os.Stdout, "PlexLinker first-time setup is required.")
	fmt.Fprintln(os.Stdout, "The installer will:")
	for _, change := range changes {
		fmt.Fprintln(os.Stdout, "  - "+change)
	}
	fmt.Fprintln(os.Stdout, "No administrator password or reusable credential will be stored.")
	fmt.Fprint(os.Stdout, "Proceed with setup? [y/N]: ")
	answer, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return fmt.Errorf("read setup confirmation: %w", err)
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	if answer != "y" && answer != "yes" {
		return fmt.Errorf("setup was declined; run %s manually when ready", script)
	}
	cmd := exec.Command(command[0], append(command[1:], script)...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Env = append(os.Environ(), "PLEXLINKER_SOURCE_EXE="+currentExecutable(), "PLEXLINKER_SETUP_CONFIRMED=1")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("first-time setup failed: %w", err)
	}
	if !setupComplete() {
		return fmt.Errorf("installer finished without writing the setup completion marker")
	}
	if _, err := exec.LookPath("aria2c"); err != nil {
		return fmt.Errorf("setup completed; open a new terminal and run PlexLinker again so the updated PATH is loaded")
	}
	return nil
}

func installer() (string, []string, []string, error) {
	directory := filepath.Dir(currentExecutable())
	if runtime.GOOS == "windows" {
		script := filepath.Join(directory, "setup-windows.ps1")
		if _, err := os.Stat(script); err != nil {
			return script, nil, nil, fmt.Errorf("Windows setup script is missing beside PlexLinker: %s", script)
		}
		return script, []string{"powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File"}, []string{
			"request Windows administrator approval through UAC",
			"install aria2 with WinGet if aria2c is missing",
			`install PlexLinker under Program Files and add it to the machine PATH`,
			"validate both commands and record setup completion for this user",
		}, nil
	}
	if runtime.GOOS == "linux" {
		script := filepath.Join(directory, "setup-linux.sh")
		if _, err := os.Stat(script); err != nil {
			return script, nil, nil, fmt.Errorf("Linux setup script is missing beside PlexLinker: %s", script)
		}
		return script, []string{"bash"}, []string{
			"validate sudo access using the operating system password prompt",
			"install aria2 through the detected package manager if it is missing",
			"install PlexLinker as /usr/local/bin/plexlinker",
			"enable systemd user lingering so downloads survive SSH logout and reboot",
			"validate the installation and discard sudo's temporary credential cache",
		}, nil
	}
	return "", nil, nil, nil
}

func currentExecutable() string {
	path, err := os.Executable()
	if err != nil {
		return os.Args[0]
	}
	path, err = filepath.EvalSymlinks(path)
	if err == nil {
		return path
	}
	return os.Args[0]
}

func setupComplete() bool {
	root, err := os.UserConfigDir()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(root, "PlexLinker", "setup-complete"))
	return err == nil
}
