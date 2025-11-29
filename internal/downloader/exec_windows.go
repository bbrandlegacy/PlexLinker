//go:build windows

package downloader

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os/exec"
	"path/filepath"
	"plexlinker/internal/logger"
	"strings"
	"time"
)

func (sg *ScriptGenerator) executeScriptPlatform(scriptPath string, fileCount int, username string) (*ScriptExecution, error) {
	// Convert to absolute path
	absPath, err := filepath.Abs(scriptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %v", err)
	}

	// Try Task Scheduler execution first (SSH-safe)
	execution, err := sg.executeViaTaskScheduler(absPath, username)
	if err == nil {
		return execution, nil
	}

	// Task Scheduler failed - log error and fall back to direct execution with warning
	logger.Get().Main().Printf("WARNING: Task Scheduler execution failed: %v", err)
	logger.Get().Main().Printf("Falling back to direct PowerShell execution (SSH-unsafe)")

	return sg.executeDirect(absPath)
}

// executeViaTaskScheduler creates and runs a scheduled task for SSH persistence
func (sg *ScriptGenerator) executeViaTaskScheduler(scriptPath string, username string) (*ScriptExecution, error) {
	// Extract script name from path (without extension)
	scriptName := filepath.Base(scriptPath)
	scriptName = strings.TrimSuffix(scriptName, filepath.Ext(scriptName))

	// Generate unique task name: PlexLinker_{username}_{scriptname}_{timestamp_with_microseconds}_{random4}
	timestamp := time.Now().Format("20060102_150405.000000") // Include microseconds for uniqueness
	random := generateRandomString(4)
	taskName := fmt.Sprintf("PlexLinker_%s_%s_%s_%s", username, scriptName, timestamp, random)

	// Build task action: PowerShell directly (no batch wrapper)
	// Use powershell.exe with -WindowStyle Normal to ensure visible window
	// Pass task name via environment variable for self-cleanup
	taskAction := fmt.Sprintf(`powershell.exe -NoExit -WindowStyle Normal -ExecutionPolicy Bypass -Command "$env:PLEXLINKER_TASK_NAME='%s'; & '%s'"`, taskName, scriptPath)

	// Calculate start time (5 seconds from now) for immediate execution
	// Using a very short delay to ensure task is registered before attempting to run
	startTime := time.Now().Add(5 * time.Second)
	startDate := startTime.Format("01/02/2006")
	startClock := startTime.Format("15:04")

	// Build schtasks /create command
	// Use /sc ONCE with near-future time for reliable immediate execution
	createCmd := exec.Command("schtasks",
		"/create",
		"/tn", taskName,
		"/tr", taskAction,
		"/sc", "ONCE",
		"/st", startClock,
		"/sd", startDate,
		"/rl", "LIMITED",
		"/f",
	)

	// Log the task creation command for debugging (only if debug mode enabled)
	logger.Get().Main().Printf("[TASK SCHEDULER] Creating task: %s", taskName)
	logger.Get().Main().Printf("[TASK SCHEDULER] Command: schtasks /create /tn %s /tr \"%s\" /sc ONCE /st %s /sd %s /rl LIMITED /f",
		taskName, taskAction, startClock, startDate)

	// Execute create command
	output, err := createCmd.CombinedOutput()
	if err != nil {
		logger.Get().Main().Printf("[TASK SCHEDULER] Creation FAILED: %v", err)
		logger.Get().Main().Printf("[TASK SCHEDULER] Output: %s", string(output))
		return nil, fmt.Errorf("task creation failed: %v (output: %s)", err, string(output))
	}
	logger.Get().Main().Printf("[TASK SCHEDULER] Task created successfully")
	logger.Get().Main().Printf("[TASK SCHEDULER] Output: %s", string(output))

	// Task is scheduled to run in 5 seconds - no need to manually trigger with /run
	// The /run command was causing duplicate executions because the scheduled time would still trigger
	logger.Get().Main().Printf("[TASK SCHEDULER] Task scheduled to run at %s", startTime.Format("15:04:05"))

	execution := &ScriptExecution{
		ScriptPath:      scriptPath,
		PID:             0, // PID not available for scheduled tasks
		ExecutionMethod: "task_scheduler",
		TaskName:        taskName,
		Warning:         false,
	}

	return execution, nil
}

// executeDirect launches script directly (fallback, SSH-unsafe)
func (sg *ScriptGenerator) executeDirect(scriptPath string) (*ScriptExecution, error) {
	// Launch PowerShell window to execute the .ps1 script
	// Use Start-Process to open a new PowerShell window with ExecutionPolicy bypass
	psCommand := fmt.Sprintf("Start-Process powershell -ArgumentList '-NoExit', '-ExecutionPolicy', 'Bypass', '-File', '\"%s\"'", scriptPath)
	cmd := exec.Command("powershell", "-Command", psCommand)

	// Start the process
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start script: %v", err)
	}

	// Wait for the start command to complete (doesn't wait for the script)
	go cmd.Wait()

	execution := &ScriptExecution{
		ScriptPath:      scriptPath,
		PID:             cmd.Process.Pid,
		ExecutionMethod: "direct",
		TaskName:        "",
		Warning:         true, // Fallback mode - warn user
	}

	return execution, nil
}

// generateRandomString generates a random alphanumeric string of specified length
func generateRandomString(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		result[i] = chars[n.Int64()]
	}
	return string(result)
}

// cleanupOrphanedTasksPlatform removes old PlexLinker tasks for the specified user
func cleanupOrphanedTasksPlatform(username string) (int, error) {
	// Query all tasks matching PlexLinker_{username}_*
	taskPrefix := fmt.Sprintf("PlexLinker_%s_", username)

	// List all tasks using schtasks /query
	cmd := exec.Command("schtasks", "/query", "/fo", "LIST", "/v")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// schtasks returns error if no tasks found - this is ok
		return 0, nil
	}

	// Parse output to find matching task names
	lines := strings.Split(string(output), "\n")
	tasksToDelete := make([]string, 0)
	var currentTaskName string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Look for task name line
		if strings.HasPrefix(line, "TaskName:") {
			taskName := strings.TrimSpace(strings.TrimPrefix(line, "TaskName:"))
			// Extract just the task name (remove path prefix like \)
			parts := strings.Split(taskName, "\\")
			if len(parts) > 0 {
				taskName = parts[len(parts)-1]
			}
			currentTaskName = taskName
		}

		// Look for "Next Run Time" line to check if task is old
		if strings.HasPrefix(line, "Next Run Time:") && currentTaskName != "" {
			// If task name matches our prefix and next run time indicates it's finished
			if strings.HasPrefix(currentTaskName, taskPrefix) {
				// Check if it says "N/A" or is from a past date
				if strings.Contains(line, "N/A") || strings.Contains(line, "Disabled") {
					tasksToDelete = append(tasksToDelete, currentTaskName)
				}
			}
			currentTaskName = ""
		}
	}

	// Delete old tasks
	deletedCount := 0
	for _, taskName := range tasksToDelete {
		deleteCmd := exec.Command("schtasks", "/delete", "/tn", taskName, "/f")
		if err := deleteCmd.Run(); err == nil {
			deletedCount++
		}
	}

	return deletedCount, nil
}
