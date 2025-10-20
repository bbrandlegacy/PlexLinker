package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"plexlinker/internal/browser"
	"plexlinker/internal/config"
	"plexlinker/internal/downloader"
)

func TestGenerateSimpleScript(t *testing.T) {
	tests := []struct {
		name         string
		config       *config.Config
		selectedURLs map[string]*browser.DirectoryItem
		wantError    bool
		checkContent []string // Strings that should appear in script
	}{
		{
			name: "Windows batch script",
			config: &config.Config{
				Username:     "testuser",
				Password:     "testpass",
				ScriptFormat: config.ScriptFormatBAT,
			},
			selectedURLs: map[string]*browser.DirectoryItem{
				"https://example.com/file1.mkv": {Name: "file1.mkv", Type: browser.TypeFile},
				"https://example.com/file2.mp4": {Name: "file2.mp4", Type: browser.TypeFile},
			},
			wantError: false,
			checkContent: []string{
				"@echo off",
				"aria2c -x4 --http-user testuser --http-passwd testpass",
				"https://example.com/file1.mkv",
				"https://example.com/file2.mp4",
				"pause",
			},
		},
		{
			name: "Unix shell script",
			config: &config.Config{
				Username:     "testuser",
				Password:     "testpass",
				ScriptFormat: config.ScriptFormatSH,
			},
			selectedURLs: map[string]*browser.DirectoryItem{
				"https://example.com/file1.mkv": {Name: "file1.mkv", Type: browser.TypeFile},
			},
			wantError: false,
			checkContent: []string{
				"#!/bin/bash",
				"aria2c -x4 --http-user testuser --http-passwd testpass",
				"https://example.com/file1.mkv",
				"Press Enter to close",
			},
		},
		{
			name: "Text format",
			config: &config.Config{
				Username:     "testuser",
				Password:     "testpass",
				ScriptFormat: config.ScriptFormatTXT,
			},
			selectedURLs: map[string]*browser.DirectoryItem{
				"https://example.com/file1.mkv": {Name: "file1.mkv", Type: browser.TypeFile},
			},
			wantError: false,
			checkContent: []string{
				"aria2c -x4 --http-user testuser --http-passwd testpass",
				"https://example.com/file1.mkv",
			},
		},
		{
			name: "Empty selection",
			config: &config.Config{
				Username:     "testuser",
				Password:     "testpass",
				ScriptFormat: config.ScriptFormatBAT,
			},
			selectedURLs: map[string]*browser.DirectoryItem{},
			wantError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sg := downloader.NewScriptGenerator(tt.config)
			scriptPath, clipboardContent, err := sg.GenerateSimpleScript(tt.selectedURLs)

			if (err != nil) != tt.wantError {
				t.Errorf("GenerateSimpleScript() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if err == nil {
				// Verify script was created
				if _, statErr := os.Stat(scriptPath); statErr != nil {
					t.Errorf("Script file not created at %s: %v", scriptPath, statErr)
					return
				}

				// Read and verify script content
				content, readErr := os.ReadFile(scriptPath)
				if readErr != nil {
					t.Errorf("Failed to read script: %v", readErr)
					return
				}

				scriptContent := string(content)
				for _, check := range tt.checkContent {
					if !strings.Contains(scriptContent, check) {
						t.Errorf("Script missing expected content: %q\nFull script:\n%s", check, scriptContent)
					}
				}

				// Verify clipboard content contains commands
				if !strings.Contains(clipboardContent, "aria2c") {
					t.Error("Clipboard content missing aria2c command")
				}

				// Cleanup
				os.Remove(scriptPath)
			}
		})
	}
}

func TestGenerateCategorizedScripts(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.Config
		categorized *downloader.CategorizedFiles
		urlMapping  map[string]string
		moviesPath  string
		tvShowsPath string
		otherPath   string
		wantScripts int
		wantError   bool
	}{
		{
			name: "Mixed movies and TV shows",
			config: &config.Config{
				Username:     "testuser",
				Password:     "testpass",
				ScriptFormat: config.ScriptFormatBAT,
			},
			categorized: &downloader.CategorizedFiles{
				Movies:  []string{"movie1.mkv", "movie2.mp4"},
				TVShows: []string{"show.s01e01.mkv", "show.s01e02.mkv"},
				Other:   []string{},
			},
			urlMapping: map[string]string{
				"movie1.mkv":      "https://example.com/movie1.mkv",
				"movie2.mp4":      "https://example.com/movie2.mp4",
				"show.s01e01.mkv": "https://example.com/show.s01e01.mkv",
				"show.s01e02.mkv": "https://example.com/show.s01e02.mkv",
			},
			moviesPath:  "W:\\movies",
			tvShowsPath: "W:\\shows",
			otherPath:   "W:\\other",
			wantScripts: 2, // movies + tvshows
			wantError:   false,
		},
		{
			name: "Only movies",
			config: &config.Config{
				Username:     "testuser",
				Password:     "testpass",
				ScriptFormat: config.ScriptFormatBAT,
			},
			categorized: &downloader.CategorizedFiles{
				Movies:  []string{"movie1.mkv"},
				TVShows: []string{},
				Other:   []string{},
			},
			urlMapping: map[string]string{
				"movie1.mkv": "https://example.com/movie1.mkv",
			},
			moviesPath:  "W:\\movies",
			tvShowsPath: "W:\\shows",
			otherPath:   "W:\\other",
			wantScripts: 1, // movies only
			wantError:   false,
		},
		{
			name: "All three categories",
			config: &config.Config{
				Username:     "testuser",
				Password:     "testpass",
				ScriptFormat: config.ScriptFormatSH,
			},
			categorized: &downloader.CategorizedFiles{
				Movies:  []string{"movie.mkv"},
				TVShows: []string{"show.s01e01.mkv"},
				Other:   []string{"other.mp3"},
			},
			urlMapping: map[string]string{
				"movie.mkv":       "https://example.com/movie.mkv",
				"show.s01e01.mkv": "https://example.com/show.s01e01.mkv",
				"other.mp3":       "https://example.com/other.mp3",
			},
			moviesPath:  "/movies",
			tvShowsPath: "/shows",
			otherPath:   "/other",
			wantScripts: 3, // all categories
			wantError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sg := downloader.NewScriptGenerator(tt.config)
			scriptPaths, err := sg.GenerateCategorizedScripts(
				tt.categorized,
				tt.urlMapping,
				tt.moviesPath,
				tt.tvShowsPath,
				tt.otherPath,
			)

			if (err != nil) != tt.wantError {
				t.Errorf("GenerateCategorizedScripts() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if err == nil {
				if len(scriptPaths) != tt.wantScripts {
					t.Errorf("Got %d scripts, want %d", len(scriptPaths), tt.wantScripts)
				}

				// Verify each script exists and has correct content
				for _, scriptPath := range scriptPaths {
					if _, statErr := os.Stat(scriptPath); statErr != nil {
						t.Errorf("Script not found: %s", scriptPath)
						continue
					}

					content, readErr := os.ReadFile(scriptPath)
					if readErr != nil {
						t.Errorf("Failed to read script %s: %v", scriptPath, readErr)
						continue
					}

					scriptContent := string(content)

					// Check for aria2c commands
					if !strings.Contains(scriptContent, "aria2c") {
						t.Errorf("Script %s missing aria2c command", scriptPath)
					}

					// Check for credentials
					if !strings.Contains(scriptContent, "testuser") {
						t.Errorf("Script %s missing username", scriptPath)
					}

					// Check category-specific content
					if strings.Contains(scriptPath, "movies") {
						if !strings.Contains(scriptContent, tt.moviesPath) {
							t.Errorf("Movies script missing destination path: %s", tt.moviesPath)
						}
					}
					if strings.Contains(scriptPath, "tvshows") {
						if !strings.Contains(scriptContent, tt.tvShowsPath) {
							t.Errorf("TV shows script missing destination path: %s", tt.tvShowsPath)
						}
					}

					// Cleanup
					os.Remove(scriptPath)
				}
			}
		})
	}
}

func TestScriptMediaTypeHandling(t *testing.T) {
	tests := []struct {
		name         string
		mediaType    string
		showName     string
		scriptFormat config.ScriptFormat
		wantPath     string // Expected path in script
	}{
		{
			name:         "Movies with batch",
			mediaType:    "movies",
			showName:     "",
			scriptFormat: config.ScriptFormatBAT,
			wantPath:     "cd /d W:\\movies",
		},
		{
			name:         "Shows with batch",
			mediaType:    "shows",
			showName:     "Breaking Bad",
			scriptFormat: config.ScriptFormatBAT,
			wantPath:     "cd /d \"W:\\shows\\Breaking Bad\"",
		},
		{
			name:         "Shows without name with batch",
			mediaType:    "shows",
			showName:     "",
			scriptFormat: config.ScriptFormatBAT,
			wantPath:     "cd /d W:\\shows",
		},
		{
			name:         "Movies with shell",
			mediaType:    "movies",
			showName:     "",
			scriptFormat: config.ScriptFormatSH,
			wantPath:     "cd /w/movies",
		},
		{
			name:         "Shows with shell",
			mediaType:    "shows",
			showName:     "Breaking Bad",
			scriptFormat: config.ScriptFormatSH,
			wantPath:     "cd '/w/shows/Breaking Bad'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Username:     "testuser",
				Password:     "testpass",
				ScriptFormat: tt.scriptFormat,
			}

			sg := downloader.NewScriptGenerator(cfg)
			sg.SetMediaType(tt.mediaType, tt.showName)

			selectedURLs := map[string]*browser.DirectoryItem{
				"https://example.com/test.mkv": {Name: "test.mkv", Type: browser.TypeFile},
			}

			scriptPath, _, err := sg.GenerateSimpleScript(selectedURLs)
			if err != nil {
				t.Fatalf("GenerateSimpleScript() error = %v", err)
			}

			content, err := os.ReadFile(scriptPath)
			if err != nil {
				t.Fatalf("Failed to read script: %v", err)
			}

			scriptContent := string(content)
			if !strings.Contains(scriptContent, tt.wantPath) {
				t.Errorf("Script missing expected path %q\nFull script:\n%s", tt.wantPath, scriptContent)
			}

			os.Remove(scriptPath)
		})
	}
}

func TestScriptFileNaming(t *testing.T) {
	tests := []struct {
		format       config.ScriptFormat
		wantFilename string
	}{
		{config.ScriptFormatBAT, "plexhelper.bat"},
		{config.ScriptFormatSH, "plexhelper.sh"},
		{config.ScriptFormatTXT, "plexhelper.txt"},
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			cfg := &config.Config{
				Username:     "testuser",
				Password:     "testpass",
				ScriptFormat: tt.format,
			}

			sg := downloader.NewScriptGenerator(cfg)
			selectedURLs := map[string]*browser.DirectoryItem{
				"https://example.com/test.mkv": {Name: "test.mkv", Type: browser.TypeFile},
			}

			scriptPath, _, err := sg.GenerateSimpleScript(selectedURLs)
			if err != nil {
				t.Fatalf("GenerateSimpleScript() error = %v", err)
			}

			filename := filepath.Base(scriptPath)
			if filename != tt.wantFilename {
				t.Errorf("Script filename = %s, want %s", filename, tt.wantFilename)
			}

			os.Remove(scriptPath)
		})
	}
}

func TestScriptContentFormat(t *testing.T) {
	t.Run("Batch script format", func(t *testing.T) {
		cfg := &config.Config{
			Username:     "user123",
			Password:     "pass456",
			ScriptFormat: config.ScriptFormatBAT,
		}

		sg := downloader.NewScriptGenerator(cfg)
		selectedURLs := map[string]*browser.DirectoryItem{
			"https://example.com/file1.mkv": {Name: "file1.mkv", Type: browser.TypeFile},
			"https://example.com/file2.mp4": {Name: "file2.mp4", Type: browser.TypeFile},
		}

		scriptPath, _, err := sg.GenerateSimpleScript(selectedURLs)
		if err != nil {
			t.Fatalf("GenerateSimpleScript() error = %v", err)
		}

		content, err := os.ReadFile(scriptPath)
		if err != nil {
			t.Fatalf("Failed to read script: %v", err)
		}

		scriptContent := string(content)

		// Check Windows line endings
		if !strings.Contains(scriptContent, "\r\n") {
			t.Error("Batch script should use Windows line endings (\\r\\n)")
		}

		// Check for header
		if !strings.Contains(scriptContent, "@echo off") {
			t.Error("Batch script missing @echo off")
		}

		// Check for REM comments
		if !strings.Contains(scriptContent, "REM") {
			t.Error("Batch script missing REM comments")
		}

		// Check for pause at end
		if !strings.HasSuffix(strings.TrimSpace(scriptContent), "pause") {
			t.Error("Batch script should end with pause")
		}

		os.Remove(scriptPath)
	})

	t.Run("Shell script format", func(t *testing.T) {
		cfg := &config.Config{
			Username:     "user123",
			Password:     "pass456",
			ScriptFormat: config.ScriptFormatSH,
		}

		sg := downloader.NewScriptGenerator(cfg)
		selectedURLs := map[string]*browser.DirectoryItem{
			"https://example.com/file1.mkv": {Name: "file1.mkv", Type: browser.TypeFile},
		}

		scriptPath, _, err := sg.GenerateSimpleScript(selectedURLs)
		if err != nil {
			t.Fatalf("GenerateSimpleScript() error = %v", err)
		}

		content, err := os.ReadFile(scriptPath)
		if err != nil {
			t.Fatalf("Failed to read script: %v", err)
		}

		scriptContent := string(content)

		// Check Unix line endings
		if strings.Contains(scriptContent, "\r\n") {
			t.Error("Shell script should use Unix line endings (\\n only)")
		}

		// Check for shebang
		if !strings.HasPrefix(scriptContent, "#!/bin/bash") {
			t.Error("Shell script missing shebang")
		}

		// Check for # comments
		if !strings.Contains(scriptContent, "# aria2c") {
			t.Error("Shell script missing # comments")
		}

		os.Remove(scriptPath)
	})
}

func TestURLQuoting(t *testing.T) {
	cfg := &config.Config{
		Username:     "testuser",
		Password:     "testpass",
		ScriptFormat: config.ScriptFormatBAT,
	}

	sg := downloader.NewScriptGenerator(cfg)

	tests := []struct {
		url          string
		shouldQuoted bool
	}{
		{"https://example.com/file with spaces.mkv", true},
		{"https://example.com/file;test.mkv", true},
		{"https://example.com/file&test.mkv", true},
		{"https://example.com/file.mkv", true},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			selectedURLs := map[string]*browser.DirectoryItem{
				tt.url: {Name: "test.mkv", Type: browser.TypeFile},
			}

			scriptPath, _, err := sg.GenerateSimpleScript(selectedURLs)
			if err != nil {
				t.Fatalf("GenerateSimpleScript() error = %v", err)
			}

			content, err := os.ReadFile(scriptPath)
			if err != nil {
				t.Fatalf("Failed to read script: %v", err)
			}

			scriptContent := string(content)

			// Check URL is quoted
			quotedURL := "\"" + tt.url + "\""
			if !strings.Contains(scriptContent, quotedURL) {
				t.Errorf("URL not properly quoted in script. Expected: %s\nFull script:\n%s",
					quotedURL, scriptContent)
			}

			os.Remove(scriptPath)
		})
	}
}
