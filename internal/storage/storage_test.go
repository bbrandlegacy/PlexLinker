package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestBrowserListsDirectoriesFirst(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a-file.mkv"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "z-directory"), 0o755); err != nil {
		t.Fatal(err)
	}
	entries, err := NewBrowser().List(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || !entries[0].IsDir || entries[0].Name != "z-directory" {
		t.Fatalf("entries = %#v", entries)
	}
}

func TestBrowserCreateAndWritable(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "shows", "Example")
	browser := NewBrowser()
	if err := browser.CreateDirectory(path); err != nil {
		t.Fatal(err)
	}
	if err := browser.IsWritable(path); err != nil {
		t.Fatal(err)
	}
	items, err := os.ReadDir(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("write test leaked files: %#v", items)
	}
}

func TestBrowserHonorsCancelledContext(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NewBrowser().List(ctx, t.TempDir()); err != context.Canceled {
		t.Fatalf("error = %v", err)
	}
}

func TestRankVolumesPrefersWritableExternalSpace(t *testing.T) {
	t.Parallel()
	volumes := []Volume{
		{Path: "/", System: true, Writable: true, FreeBytes: 10 << 40},
		{Path: "/mnt/small", External: true, Writable: true, FreeBytes: 1 << 40},
		{Path: "/mnt/large", External: true, Writable: true, FreeBytes: 2 << 40},
		{Path: "/mnt/locked", External: true, Writable: false, FreeBytes: 3 << 40},
	}
	RankVolumes(volumes)
	if volumes[0].Path != "/mnt/large" || volumes[1].Path != "/mnt/small" || volumes[len(volumes)-1].Path != "/mnt/locked" {
		t.Fatalf("ranked volumes = %#v", volumes)
	}
}

func TestDiscoverMediaPathsFindsConventionalFolders(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "Plex", "Movies"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "TV Shows"), 0o755); err != nil {
		t.Fatal(err)
	}
	volume := Volume{Path: root, Available: true}
	DiscoverMediaPaths(&volume)
	if volume.MoviesPath != filepath.Join(root, "Plex", "Movies") || volume.TVPath != filepath.Join(root, "TV Shows") {
		t.Fatalf("discovered paths = %#v", volume)
	}
}
