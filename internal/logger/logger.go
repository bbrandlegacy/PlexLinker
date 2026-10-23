package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
)

var (
	instance *Logger
	once     sync.Once
)

// Logger provides centralized debug logging with multiple namespaces
type Logger struct {
	enabled bool
	file    *os.File
	loggers map[string]*log.Logger
	mu      sync.RWMutex
}

// Config holds logger configuration
type Config struct {
	Enabled  bool
	FilePath string
	ToStderr bool
}

// Initialize sets up the global logger instance (call once at startup)
func Initialize(cfg Config) error {
	var initErr error
	once.Do(func() {
		instance = &Logger{
			enabled: cfg.Enabled,
			loggers: make(map[string]*log.Logger),
		}

		if !cfg.Enabled {
			return
		}

		// Open log file
		file, err := os.OpenFile(cfg.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			initErr = fmt.Errorf("failed to open log file: %w", err)
			return
		}
		instance.file = file

		// Determine output destination
		var output io.Writer
		if cfg.ToStderr {
			output = io.MultiWriter(file, os.Stderr)
		} else {
			output = file
		}

		// Pre-create common loggers
		instance.createLogger("MAIN", output)
		instance.createLogger("UI", output)
		instance.createLogger("CLIENT", output)
		instance.createLogger("BROWSER", output)
		instance.createLogger("DOWNLOADER", output)
		instance.createLogger("CONFIG", output)
		instance.createLogger("TVRENAMER", output)
	})

	return initErr
}

// Get returns the global logger instance
func Get() *Logger {
	if instance == nil {
		// Return disabled logger if not initialized
		return &Logger{enabled: false, loggers: make(map[string]*log.Logger)}
	}
	return instance
}

// Close closes the log file (call on shutdown)
func Close() error {
	if instance != nil && instance.file != nil {
		return instance.file.Close()
	}
	return nil
}

// createLogger creates a namespaced logger
func (l *Logger) createLogger(namespace string, output io.Writer) {
	prefix := fmt.Sprintf("[%s] ", namespace)
	l.loggers[namespace] = log.New(output, prefix, log.Ldate|log.Ltime|log.Lshortfile)
}

// getLogger returns a logger for the given namespace
func (l *Logger) getLogger(namespace string) *log.Logger {
	if !l.enabled {
		return log.New(io.Discard, "", 0)
	}

	l.mu.RLock()
	logger, exists := l.loggers[namespace]
	l.mu.RUnlock()

	if exists {
		return logger
	}

	// Create on-demand if not exists
	l.mu.Lock()
	defer l.mu.Unlock()

	// Double-check after acquiring write lock
	if logger, exists := l.loggers[namespace]; exists {
		return logger
	}

	var output io.Writer
	if l.file != nil {
		output = l.file
	} else {
		output = os.Stderr
	}
	l.createLogger(namespace, output)
	return l.loggers[namespace]
}

// Namespace loggers
func (l *Logger) Main() *log.Logger       { return l.getLogger("MAIN") }
func (l *Logger) UI() *log.Logger         { return l.getLogger("UI") }
func (l *Logger) Client() *log.Logger     { return l.getLogger("CLIENT") }
func (l *Logger) Browser() *log.Logger    { return l.getLogger("BROWSER") }
func (l *Logger) Downloader() *log.Logger { return l.getLogger("DOWNLOADER") }
func (l *Logger) Config() *log.Logger     { return l.getLogger("CONFIG") }
func (l *Logger) TVRenamer() *log.Logger  { return l.getLogger("TVRENAMER") }

// Helper methods for common operations
func (l *Logger) Printf(namespace, format string, v ...interface{}) {
	l.getLogger(namespace).Printf(format, v...)
}

func (l *Logger) Println(namespace string, v ...interface{}) {
	l.getLogger(namespace).Println(v...)
}

// Enabled returns whether logging is enabled
func (l *Logger) Enabled() bool {
	return l.enabled
}
