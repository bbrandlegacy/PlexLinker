package diagnostics

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLogTailAndClear(t *testing.T) {
	t.Parallel()
	log, err := New(filepath.Join(t.TempDir(), "plexlinker.log"))
	if err != nil {
		t.Fatal(err)
	}
	log.Write("info", "first")
	log.Write("error", "second\nline")
	log.Write("error", "second\nline")
	lines, err := log.Tail(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 || !strings.Contains(lines[0], "ERROR") || !strings.Contains(lines[0], "second | line") {
		t.Fatalf("lines = %#v", lines)
	}
	if err := log.Clear(); err != nil {
		t.Fatal(err)
	}
	lines, err = log.Tail(10)
	if err != nil || len(lines) != 0 {
		t.Fatalf("lines=%#v err=%v", lines, err)
	}
}
