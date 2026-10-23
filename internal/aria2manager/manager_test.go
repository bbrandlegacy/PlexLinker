package aria2manager

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateSecret(t *testing.T) {
	t.Parallel()
	first, err := GenerateSecret()
	if err != nil {
		t.Fatal(err)
	}
	second, err := GenerateSecret()
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 48 || first == second {
		t.Fatalf("secrets are not independently generated: %q %q", first, second)
	}
}

func TestWriteConfigPersistsSession(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	manager, err := New(Config{Endpoint: "http://127.0.0.1:6801/jsonrpc", Secret: "secret", MaxConcurrent: 7, DataDirectory: root})
	if err != nil {
		t.Fatal(err)
	}
	path, err := manager.writeConfig()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	for _, expected := range []string{"rpc-listen-all=false", "rpc-listen-port=6801", "rpc-secret=secret", "max-concurrent-downloads=7", "save-session-interval=30"} {
		if !strings.Contains(content, expected) {
			t.Fatalf("config missing %q:\n%s", expected, content)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "aria2.session")); err != nil {
		t.Fatal(err)
	}
}

func TestValidateLocalEndpoint(t *testing.T) {
	t.Parallel()
	if err := validateLocalEndpoint("http://localhost:6800/jsonrpc"); err != nil {
		t.Fatal(err)
	}
	if err := validateLocalEndpoint("https://remote.example.test/jsonrpc"); err == nil {
		t.Fatal("expected remote endpoint rejection")
	}
}
