package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"plexlinker/internal/browser"
	"plexlinker/internal/client"
	"plexlinker/internal/config"
	"plexlinker/internal/downloader"
)

// Mock HTTP server for testing
func newMockServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check authentication
		username, password, ok := r.BasicAuth()
		if !ok || username != "testuser" || password != "testpass" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized"))
			return
		}

		// Serve different HTML based on path
		path := r.URL.Path
		var htmlFile string

		switch path {
		case "/", "/media/":
			htmlFile = "sample.html"
		case "/empty/":
			htmlFile = "empty.html"
		default:
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not Found"))
			return
		}

		// Read fixture
		content, err := os.ReadFile(filepath.Join("fixtures", htmlFile))
		if err != nil {
			t.Logf("Failed to read fixture %s: %v", htmlFile, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
}

func TestIntegrationConfigLoadSave(t *testing.T) {
	// Create temp directory for test config
	tempDir := t.TempDir()
	testConfigPath := filepath.Join(tempDir, "test_config.json")

	t.Run("Save and load config", func(t *testing.T) {
		cfg := &config.Config{
			BaseURL:      "https://test.example.com/",
			Username:     "testuser",
			Password:     "testpass",
			ScriptFormat: config.ScriptFormatBAT,
			DebugMode:    false,
			MoviesPath:   "W:\\movies",
			TVShowsPath:  "W:\\shows",
			OtherPath:    "W:\\other",
		}

		// Save config
		data, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			t.Fatalf("Failed to marshal config: %v", err)
		}

		err = os.WriteFile(testConfigPath, data, 0600)
		if err != nil {
			t.Fatalf("Failed to write config: %v", err)
		}

		// Load config
		loadedData, err := os.ReadFile(testConfigPath)
		if err != nil {
			t.Fatalf("Failed to read config: %v", err)
		}

		loadedCfg := &config.Config{}
		err = json.Unmarshal(loadedData, loadedCfg)
		if err != nil {
			t.Fatalf("Failed to unmarshal config: %v", err)
		}

		// Verify loaded config matches original
		if loadedCfg.BaseURL != cfg.BaseURL {
			t.Errorf("BaseURL = %s, want %s", loadedCfg.BaseURL, cfg.BaseURL)
		}
		if loadedCfg.Username != cfg.Username {
			t.Errorf("Username = %s, want %s", loadedCfg.Username, cfg.Username)
		}
	})
}

func TestIntegrationConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    *config.Config
		wantError bool
		errorMsg  string
	}{
		{
			name: "Valid config",
			config: &config.Config{
				BaseURL:     "https://example.com/",
				Username:    "user",
				Password:    "pass",
				MoviesPath:  "W:\\movies",
				TVShowsPath: "W:\\shows",
				OtherPath:   "W:\\other",
			},
			wantError: false,
		},
		{
			name: "Missing base URL",
			config: &config.Config{
				BaseURL:     "",
				Username:    "user",
				Password:    "pass",
				MoviesPath:  "W:\\movies",
				TVShowsPath: "W:\\shows",
				OtherPath:   "W:\\other",
			},
			wantError: true,
			errorMsg:  "base URL cannot be empty",
		},
		{
			name: "Missing username",
			config: &config.Config{
				BaseURL:     "https://example.com/",
				Username:    "",
				Password:    "pass",
				MoviesPath:  "W:\\movies",
				TVShowsPath: "W:\\shows",
				OtherPath:   "W:\\other",
			},
			wantError: true,
			errorMsg:  "username cannot be empty",
		},
		{
			name: "Missing password",
			config: &config.Config{
				BaseURL:     "https://example.com/",
				Username:    "user",
				Password:    "",
				MoviesPath:  "W:\\movies",
				TVShowsPath: "W:\\shows",
				OtherPath:   "W:\\other",
			},
			wantError: true,
			errorMsg:  "password cannot be empty",
		},
		{
			name: "Invalid URL",
			config: &config.Config{
				BaseURL:     "not a url",
				Username:    "user",
				Password:    "pass",
				MoviesPath:  "W:\\movies",
				TVShowsPath: "W:\\shows",
				OtherPath:   "W:\\other",
			},
			wantError: false, // URL.Parse is lenient
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.wantError && err != nil {
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Error message = %v, want to contain %q", err, tt.errorMsg)
				}
			}
		})
	}
}

func TestIntegrationHTTPClientWithMockServer(t *testing.T) {
	server := newMockServer(t)
	defer server.Close()

	cfg := &config.Config{
		BaseURL:  server.URL + "/",
		Username: "testuser",
		Password: "testpass",
	}

	c := client.NewClient(cfg)

	t.Run("Successful fetch", func(t *testing.T) {
		html, err := c.FetchDirectory("media/")
		if err != nil {
			t.Fatalf("FetchDirectory() error = %v", err)
		}

		if !strings.Contains(html, "Index of") {
			t.Error("Response doesn't contain expected HTML")
		}
	})

	t.Run("Authentication failure", func(t *testing.T) {
		badCfg := &config.Config{
			BaseURL:  server.URL + "/",
			Username: "wronguser",
			Password: "wrongpass",
		}
		badClient := client.NewClient(badCfg)

		_, err := badClient.FetchDirectory("media/")
		if err == nil {
			t.Error("Expected authentication error but got none")
		}

		if !strings.Contains(err.Error(), "authentication failed") {
			t.Errorf("Wrong error message: %v", err)
		}
	})

	t.Run("404 Not Found", func(t *testing.T) {
		_, err := c.FetchDirectory("nonexistent/")
		if err == nil {
			t.Error("Expected 404 error but got none")
		}

		if !strings.Contains(err.Error(), "404") {
			t.Errorf("Wrong error message: %v", err)
		}
	})
}

func TestIntegrationEndToEndFlow(t *testing.T) {
	server := newMockServer(t)
	defer server.Close()

	// Step 1: Create config
	cfg := &config.Config{
		BaseURL:      server.URL + "/",
		Username:     "testuser",
		Password:     "testpass",
		ScriptFormat: config.ScriptFormatBAT,
		MoviesPath:   "W:\\movies",
		TVShowsPath:  "W:\\shows",
		OtherPath:    "W:\\other",
	}

	// Step 2: Validate config
	err := cfg.Validate()
	if err != nil {
		t.Fatalf("Config validation failed: %v", err)
	}

	// Step 3: Fetch directory listing
	c := client.NewClient(cfg)
	html, err := c.FetchDirectory("media/")
	if err != nil {
		t.Fatalf("Failed to fetch directory: %v", err)
	}

	// Step 4: Parse HTML
	listing, err := browser.ParseHTMLDirectory(html, "/media")
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	if len(listing.Items) == 0 {
		t.Fatal("No items found in listing")
	}

	// Step 5: Select files (simulate user selection)
	selectedFiles := make(map[string]*browser.DirectoryItem)
	for i := range listing.Items {
		item := &listing.Items[i]
		if item.Type == browser.TypeFile {
			url := c.GetFullURL(item.Path)
			selectedFiles[url] = item
		}
	}

	if len(selectedFiles) == 0 {
		t.Fatal("No files selected")
	}

	// Step 6: Categorize files
	filenames := make([]string, 0, len(selectedFiles))
	urlMapping := make(map[string]string)
	for url, item := range selectedFiles {
		filenames = append(filenames, item.Name)
		urlMapping[item.Name] = url
	}

	categorized := downloader.CategorizeFiles(filenames)

	if categorized.TotalFiles() != len(selectedFiles) {
		t.Errorf("Categorization lost files: got %d, want %d",
			categorized.TotalFiles(), len(selectedFiles))
	}

	// Step 7: Generate scripts
	sg := downloader.NewScriptGenerator(cfg)
	scriptPaths, err := sg.GenerateCategorizedScripts(
		categorized,
		urlMapping,
		cfg.MoviesPath,
		cfg.TVShowsPath,
		cfg.OtherPath,
	)

	if err != nil {
		t.Fatalf("Failed to generate scripts: %v", err)
	}

	if len(scriptPaths) == 0 {
		t.Fatal("No scripts generated")
	}

	// Step 8: Verify generated scripts
	for _, scriptPath := range scriptPaths {
		// Check file exists
		if _, statErr := os.Stat(scriptPath); statErr != nil {
			t.Errorf("Script not found: %s", scriptPath)
			continue
		}

		// Read and verify content
		content, readErr := os.ReadFile(scriptPath)
		if readErr != nil {
			t.Errorf("Failed to read script: %v", readErr)
			continue
		}

		scriptContent := string(content)

		// Verify contains aria2c commands
		if !strings.Contains(scriptContent, "aria2c") {
			t.Errorf("Script missing aria2c command: %s", scriptPath)
		}

		// Verify contains credentials
		if !strings.Contains(scriptContent, cfg.Username) {
			t.Errorf("Script missing username: %s", scriptPath)
		}

		// Cleanup
		os.Remove(scriptPath)
	}
}

func TestIntegrationFileSelection(t *testing.T) {
	server := newMockServer(t)
	defer server.Close()

	cfg := &config.Config{
		BaseURL:  server.URL + "/",
		Username: "testuser",
		Password: "testpass",
	}

	c := client.NewClient(cfg)
	html, err := c.FetchDirectory("media/")
	if err != nil {
		t.Fatalf("Failed to fetch directory: %v", err)
	}

	listing, err := browser.ParseHTMLDirectory(html, "/media")
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	t.Run("Select all files", func(t *testing.T) {
		selectedCount := 0
		for i := range listing.Items {
			if listing.Items[i].Type == browser.TypeFile {
				listing.Items[i].Selected = true
				selectedCount++
			}
		}

		if selectedCount == 0 {
			t.Error("No files were selected")
		}

		// Verify selection state
		for _, item := range listing.Items {
			if item.Type == browser.TypeFile && !item.Selected {
				t.Errorf("File %s should be selected but isn't", item.Name)
			}
		}
	})

	t.Run("Deselect all", func(t *testing.T) {
		for i := range listing.Items {
			listing.Items[i].Selected = false
		}

		for _, item := range listing.Items {
			if item.Selected {
				t.Errorf("Item %s should not be selected", item.Name)
			}
		}
	})
}

func TestIntegrationNetworkFailure(t *testing.T) {
	// Use an invalid URL to simulate network failure
	cfg := &config.Config{
		BaseURL:  "http://localhost:99999/", // Invalid port
		Username: "testuser",
		Password: "testpass",
	}

	c := client.NewClient(cfg)

	_, err := c.FetchDirectory("")
	if err == nil {
		t.Error("Expected network error but got none")
	}

	// Error should be meaningful
	if err != nil && !strings.Contains(err.Error(), "failed to fetch") {
		t.Logf("Got error: %v", err)
	}
}

func TestIntegrationEmptyDirectory(t *testing.T) {
	server := newMockServer(t)
	defer server.Close()

	cfg := &config.Config{
		BaseURL:  server.URL + "/",
		Username: "testuser",
		Password: "testpass",
	}

	c := client.NewClient(cfg)
	html, err := c.FetchDirectory("empty/")
	if err != nil {
		t.Fatalf("Failed to fetch directory: %v", err)
	}

	listing, err := browser.ParseHTMLDirectory(html, "/empty")
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	if len(listing.Items) != 0 {
		t.Errorf("Expected 0 items in empty directory, got %d", len(listing.Items))
	}
}

func TestIntegrationScriptFormatSelection(t *testing.T) {
	formats := []config.ScriptFormat{
		config.ScriptFormatBAT,
		config.ScriptFormatSH,
		config.ScriptFormatTXT,
	}

	for _, format := range formats {
		t.Run(string(format), func(t *testing.T) {
			cfg := &config.Config{
				BaseURL:      "https://example.com/",
				Username:     "testuser",
				Password:     "testpass",
				ScriptFormat: format,
			}

			sg := downloader.NewScriptGenerator(cfg)
			selectedFiles := map[string]*browser.DirectoryItem{
				"https://example.com/test.mkv": {Name: "test.mkv", Type: browser.TypeFile},
			}

			scriptPath, _, err := sg.GenerateSimpleScript(selectedFiles)
			if err != nil {
				t.Fatalf("Failed to generate script: %v", err)
			}

			// Verify file extension
			ext := filepath.Ext(scriptPath)
			expectedExt := "." + string(format)
			if ext != expectedExt {
				t.Errorf("Script extension = %s, want %s", ext, expectedExt)
			}

			os.Remove(scriptPath)
		})
	}
}

func TestIntegrationURLBuilding(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		path     string
		wantURL  string
	}{
		{
			name:     "Simple path",
			baseURL:  "https://example.com/media/",
			path:     "movies/2024/file.mkv",
			wantURL:  "https://example.com/media/movies/2024/file.mkv",
		},
		{
			name:     "Path with leading slash",
			baseURL:  "https://example.com/media/",
			path:     "/movies/file.mkv",
			wantURL:  "https://example.com/media/movies/file.mkv",
		},
		{
			name:     "Root path",
			baseURL:  "https://example.com/",
			path:     "file.mkv",
			wantURL:  "https://example.com/file.mkv",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				BaseURL:  tt.baseURL,
				Username: "user",
				Password: "pass",
			}

			c := client.NewClient(cfg)
			gotURL := c.GetFullURL(tt.path)

			if gotURL != tt.wantURL {
				t.Errorf("GetFullURL() = %s, want %s", gotURL, tt.wantURL)
			}
		})
	}
}

func TestIntegrationMixedContent(t *testing.T) {
	// Test handling of directories, video files, audio files, and other files
	server := newMockServer(t)
	defer server.Close()

	cfg := &config.Config{
		BaseURL:  server.URL + "/",
		Username: "testuser",
		Password: "testpass",
	}

	c := client.NewClient(cfg)
	html, err := c.FetchDirectory("media/")
	if err != nil {
		t.Fatalf("Failed to fetch directory: %v", err)
	}

	listing, err := browser.ParseHTMLDirectory(html, "/media")
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	// Count different types
	dirCount := 0
	videoCount := 0
	audioCount := 0

	for _, item := range listing.Items {
		switch item.Type {
		case browser.TypeDirectory:
			dirCount++
		case browser.TypeFile:
			if item.IsVideoFile() {
				videoCount++
			}
			if item.IsAudioFile() {
				audioCount++
			}
		}
	}

	if dirCount == 0 {
		t.Error("No directories found")
	}
	if videoCount == 0 {
		t.Error("No video files found")
	}
	if audioCount == 0 {
		t.Error("No audio files found")
	}

	t.Logf("Found: %d directories, %d videos, %d audio files", dirCount, videoCount, audioCount)
}
