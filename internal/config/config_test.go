package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateV2RejectsRelativeURLs(t *testing.T) {
	cfg := NewConfig()
	cfg.BaseURL, cfg.Username, cfg.Password = "media/files", "user", "pass"
	if err := cfg.ValidateV2(); err == nil {
		t.Fatal("expected invalid base URL")
	}
	cfg.BaseURL = "https://media.example.test/files"
	cfg.Aria2Endpoint = "not-an-endpoint"
	if err := cfg.ValidateV2(); err == nil {
		t.Fatal("expected invalid aria2 endpoint")
	}
}

func TestSaveAndLoadV2Config(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.json")
	t.Setenv("PLEXLINKER_CONFIG", path)
	cfg := NewConfig()
	cfg.BaseURL, cfg.Username, cfg.Password = "https://media.example.test/", "user", "pass"
	cfg.FavoritePaths = []string{`W:\movies`, `\\nas\shows`}
	if err := cfg.SaveV2(); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadV2Config()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.BaseURL != cfg.BaseURL || len(loaded.FavoritePaths) != 2 {
		t.Fatalf("loaded = %#v", loaded)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("config mode = %o", info.Mode().Perm())
	}
}
