//go:build !windows

package downloader

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

func (sg *ScriptGenerator) executeScriptPlatform(scriptPath string, fileCount int, username string) (*ScriptExecution, error) {
	// Convert to absolute path
	absPath, err := filepath.Abs(scriptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %v", err)
	}

	// On non-Windows, execute the script in a new terminal window
	cmd := exec.Command("sh", absPath)

	// Start the process
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start script: %v", err)
	}

	execution := &ScriptExecution{
		ScriptPath:      absPath,
		PID:             cmd.Process.Pid,
		ExecutionMethod: "direct",
		TaskName:        "",
		Warning:         false,
	}

	return execution, nil
}

// cleanupOrphanedTasksPlatform is a no-op on non-Windows platforms
func cleanupOrphanedTasksPlatform(username string) (int, error) {
	return 0, nil
}
