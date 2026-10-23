package client

import (
	"context"
	"errors"
	"testing"

	"plexlinker/internal/config"
)

func TestBuildURLStaysUnderConfiguredRoot(t *testing.T) {
	t.Parallel()
	client := NewClient(&config.Config{BaseURL: "https://media.example.test/private/files/"})
	url, err := client.buildURL("Shows/Season 1/Episode.mkv")
	if err != nil {
		t.Fatal(err)
	}
	if url != "https://media.example.test/private/files/Shows/Season%201/Episode.mkv" {
		t.Fatalf("url = %q", url)
	}
	for _, path := range []string{"../outside", "//evil.example/file", "https://evil.example/file", "/absolute"} {
		if _, err := client.buildURL(path); err == nil {
			t.Errorf("expected %q to be rejected", path)
		}
	}
}

func TestFetchDirectoryHonorsCancelledContext(t *testing.T) {
	t.Parallel()
	client := NewClient(&config.Config{BaseURL: "https://media.example.test/"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := client.FetchDirectoryContext(ctx, "")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
}
