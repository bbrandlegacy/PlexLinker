package ui

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/gdamore/tcell/v2"
)

// SignalHandler manages graceful shutdown on signals
type SignalHandler struct {
	screen   *tcell.Screen
	shutdown chan bool
	done     chan bool
}

func NewSignalHandler(screen *tcell.Screen) *SignalHandler {
	return &SignalHandler{
		screen:   screen,
		shutdown: make(chan bool, 1),
		done:     make(chan bool, 1),
	}
}

// Start begins monitoring for signals that indicate disconnection
func (sh *SignalHandler) Start() {
	go sh.monitorSignals()
}

// Stop shuts down the signal monitoring
func (sh *SignalHandler) Stop() {
	close(sh.shutdown)
	<-sh.done
}

// ShouldExit returns true if a disconnect signal was received
func (sh *SignalHandler) ShouldExit() bool {
	select {
	case <-sh.shutdown:
		return true
	default:
		return false
	}
}

func (sh *SignalHandler) monitorSignals() {
	sigChan := make(chan os.Signal, 1)

	// Monitor signals that indicate session termination
	signal.Notify(sigChan,
		syscall.SIGTERM, // Termination signal
		syscall.SIGINT,  // Interrupt signal (Ctrl+C)
		syscall.SIGHUP,  // Hangup signal (SSH disconnect)
		syscall.SIGPIPE, // Broken pipe (connection lost)
		syscall.SIGQUIT, // Quit signal
	)

	// Also monitor for stdin closure (common in SSH disconnects)
	// Use a separate channel to signal stdin monitor to exit
	stdinDone := make(chan bool, 1)
	go sh.monitorStdin(stdinDone)

	select {
	case sig := <-sigChan:
		// Log the signal for debugging (if needed)
		_ = sig
		sh.cleanup()
		sh.shutdown <- true
	case <-sh.shutdown:
		// Explicit shutdown requested
	}

	// Clean up stdin monitor goroutine
	select {
	case stdinDone <- true:
	default:
	}

	signal.Stop(sigChan)
	sh.done <- true
}

func (sh *SignalHandler) monitorStdin(done chan bool) {
	// Check if stdin is still available
	// This helps detect when SSH connection is lost
	stat, err := os.Stdin.Stat()
	if err != nil {
		select {
		case sh.shutdown <- true:
		case <-done:
			return
		}
		return
	}

	// If stdin is not a character device or pipe, it might be closed
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		// Additional check for broken pipe or closed connection
		if stat.Size() == 0 && (stat.Mode()&os.ModeNamedPipe) == 0 {
			select {
			case sh.shutdown <- true:
			case <-done:
				return
			}
			return
		}
	}

	// Wait for exit signal
	<-done
}

func (sh *SignalHandler) cleanup() {
	if sh.screen != nil {
		(*sh.screen).Fini()
	}
}
