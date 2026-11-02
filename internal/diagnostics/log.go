// Package diagnostics provides the persistent, user-visible PlexLinker event log.
package diagnostics

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Log struct {
	path      string
	mu        sync.Mutex
	lastEntry string
	lastWrite time.Time
}

func New(path string) (*Log, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create diagnostics directory: %w", err)
	}
	return &Log{path: path}, nil
}

func (l *Log) Path() string { return l.path }

func (l *Log) Write(level, message string) {
	if l == nil || strings.TrimSpace(message) == "" {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	entry := strings.ToUpper(level) + " " + strings.TrimSpace(message)
	if entry == l.lastEntry && time.Since(l.lastWrite) < time.Minute {
		return
	}
	l.lastEntry, l.lastWrite = entry, time.Now()
	if info, err := os.Stat(l.path); err == nil && info.Size() >= 2<<20 {
		_ = os.Remove(l.path + ".1")
		_ = os.Rename(l.path, l.path+".1")
	}
	file, err := os.OpenFile(l.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer func() { _ = file.Close() }()
	message = strings.ReplaceAll(strings.TrimSpace(message), "\n", " | ")
	_, _ = fmt.Fprintf(file, "%s %-5s %s\n", time.Now().Format(time.RFC3339), strings.ToUpper(level), message)
}

func (l *Log) Tail(limit int) ([]string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	file, err := os.Open(l.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open diagnostics log: %w", err)
	}
	defer func() { _ = file.Close() }()
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if limit > 0 && len(lines) > limit {
			lines = lines[len(lines)-limit:]
		}
	}
	return lines, scanner.Err()
}

func (l *Log) Clear() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := os.WriteFile(l.path, nil, 0o600); err != nil {
		return fmt.Errorf("clear diagnostics log: %w", err)
	}
	_ = os.Remove(l.path + ".1")
	l.lastEntry, l.lastWrite = "", time.Time{}
	return nil
}
