package aria2manager

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"plexlinker/internal/aria2"
)

func TestManagedAria2DownloadIntegration(t *testing.T) {
	if os.Getenv("PLEXLINKER_INTEGRATION") != "1" {
		t.Skip("set PLEXLINKER_INTEGRATION=1 to run")
	}
	executable, err := exec.LookPath("aria2c")
	if err != nil {
		t.Skip("aria2c not available")
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	secret, err := GenerateSecret()
	if err != nil {
		t.Fatal(err)
	}
	endpoint := fmt.Sprintf("http://127.0.0.1:%d/jsonrpc", port)
	managerConfig := Config{Executable: executable, Endpoint: endpoint, Secret: secret, AutoStart: true, DataDirectory: t.TempDir()}
	manager, err := New(managerConfig)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if _, err := manager.Ensure(ctx); err != nil {
		t.Fatal(err)
	}
	// Reconnect through a new manager instance. This proves ownership of the
	// service is independent from the manager that launched it.
	reconnected, err := New(managerConfig)
	if err != nil {
		t.Fatal(err)
	}
	status, err := reconnected.Ensure(ctx)
	if err != nil {
		t.Fatalf("reconnect to managed aria2: %v", err)
	}
	if !status.Running {
		t.Fatal("reconnected manager reported aria2 as stopped")
	}
	client := aria2.NewClient(endpoint, secret)
	t.Cleanup(func() { _ = client.Shutdown(context.Background(), true) })

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("plexlinker integration")) }))
	defer server.Close()
	destination := t.TempDir()
	gid, err := client.AddURI(ctx, server.URL+"/file.mkv", aria2.AddOptions{"dir": destination, "out": "file.mkv"})
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		jobs, err := client.TellStopped(ctx, 0, 10)
		if err == nil {
			for _, job := range jobs {
				if job.GID == gid && job.Status == "complete" {
					data, readErr := os.ReadFile(filepath.Join(destination, "file.mkv"))
					if readErr != nil {
						t.Fatal(readErr)
					}
					if string(data) != "plexlinker integration" {
						t.Fatalf("data = %q", data)
					}
					return
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("aria2 download did not complete")
}
