package test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"plexlinker/internal/browser"
)

func TestParseHTMLDirectory(t *testing.T) {
	tests := []struct {
		name          string
		htmlFile      string
		currentPath   string
		wantItemCount int
		wantError     bool
	}{
		{
			name:          "Valid HTML directory listing",
			htmlFile:      "sample.html",
			currentPath:   "/media",
			wantItemCount: 8, // 2 dirs + 5 video files + 1 audio file
			wantError:     false,
		},
		{
			name:          "Empty directory",
			htmlFile:      "empty.html",
			currentPath:   "/empty",
			wantItemCount: 0,
			wantError:     false,
		},
		{
			name:          "Malformed HTML",
			htmlFile:      "malformed.html",
			currentPath:   "/malformed",
			wantItemCount: 0, // Parser requires proper table structure
			wantError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			htmlPath := filepath.Join("fixtures", tt.htmlFile)
			htmlContent, err := os.ReadFile(htmlPath)
			if err != nil {
				t.Fatalf("Failed to read fixture %s: %v", tt.htmlFile, err)
			}

			listing, err := browser.ParseHTMLDirectory(string(htmlContent), tt.currentPath)

			if (err != nil) != tt.wantError {
				t.Errorf("ParseHTMLDirectory() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if err == nil {
				if len(listing.Items) != tt.wantItemCount {
					t.Errorf("ParseHTMLDirectory() got %d items, want %d", len(listing.Items), tt.wantItemCount)
					for i, item := range listing.Items {
						t.Logf("  Item %d: %s (Type: %d)", i, item.Name, item.Type)
					}
				}

				if listing.CurrentPath != tt.currentPath {
					t.Errorf("ParseHTMLDirectory() currentPath = %v, want %v", listing.CurrentPath, tt.currentPath)
				}
			}
		})
	}
}

func TestParseHTMLDirectoryWithSort(t *testing.T) {
	htmlPath := filepath.Join("fixtures", "sample.html")
	htmlContent, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatalf("Failed to read fixture: %v", err)
	}

	t.Run("Sort by date enabled", func(t *testing.T) {
		listing, err := browser.ParseHTMLDirectoryWithSort(string(htmlContent), "/media", true)
		if err != nil {
			t.Fatalf("ParseHTMLDirectoryWithSort() error = %v", err)
		}

		// Verify directories come first
		foundFile := false
		for _, item := range listing.Items {
			if item.Type == browser.TypeFile {
				foundFile = true
			} else if foundFile {
				t.Error("Directory found after file - sort is incorrect")
			}
		}

		// Check that files with dates are sorted newest first (within their category)
		var lastTime time.Time
		for _, item := range listing.Items {
			if item.Type == browser.TypeFile && !item.ModifiedTime.IsZero() {
				if !lastTime.IsZero() && item.ModifiedTime.After(lastTime) {
					t.Errorf("Files not sorted by date (newest first): %s (%v) appears after %v",
						item.Name, item.ModifiedTime, lastTime)
				}
				lastTime = item.ModifiedTime
			}
		}
	})

	t.Run("Sort by date disabled", func(t *testing.T) {
		listing, err := browser.ParseHTMLDirectoryWithSort(string(htmlContent), "/media", false)
		if err != nil {
			t.Fatalf("ParseHTMLDirectoryWithSort() error = %v", err)
		}

		// Items should be in HTML order (not date sorted)
		// Just verify we got items
		if len(listing.Items) == 0 {
			t.Error("Expected items but got none")
		}
	})
}

func TestDirectoryItemIsVideoFile(t *testing.T) {
	tests := []struct {
		name     string
		itemName string
		itemType browser.ItemType
		want     bool
	}{
		{"MP4 file", "movie.mp4", browser.TypeFile, true},
		{"MKV file", "episode.mkv", browser.TypeFile, true},
		{"AVI file", "video.avi", browser.TypeFile, true},
		{"MOV file", "clip.mov", browser.TypeFile, true},
		{"WEBM file", "video.webm", browser.TypeFile, true},
		{"MP3 file (not video)", "song.mp3", browser.TypeFile, false},
		{"TXT file", "readme.txt", browser.TypeFile, false},
		{"Directory with video extension", "movies.mp4", browser.TypeDirectory, false},
		{"Case insensitive", "VIDEO.MP4", browser.TypeFile, true},
		{"No extension", "video", browser.TypeFile, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &browser.DirectoryItem{
				Name: tt.itemName,
				Type: tt.itemType,
			}
			if got := item.IsVideoFile(); got != tt.want {
				t.Errorf("IsVideoFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDirectoryItemIsAudioFile(t *testing.T) {
	tests := []struct {
		name     string
		itemName string
		itemType browser.ItemType
		want     bool
	}{
		{"MP3 file", "song.mp3", browser.TypeFile, true},
		{"FLAC file", "track.flac", browser.TypeFile, true},
		{"WAV file", "audio.wav", browser.TypeFile, true},
		{"AAC file", "music.aac", browser.TypeFile, true},
		{"OGG file", "sound.ogg", browser.TypeFile, true},
		{"MP4 file (not audio)", "video.mp4", browser.TypeFile, false},
		{"TXT file", "readme.txt", browser.TypeFile, false},
		{"Directory with audio extension", "music.mp3", browser.TypeDirectory, false},
		{"Case insensitive", "AUDIO.MP3", browser.TypeFile, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &browser.DirectoryItem{
				Name: tt.itemName,
				Type: tt.itemType,
			}
			if got := item.IsAudioFile(); got != tt.want {
				t.Errorf("IsAudioFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetFileIcon(t *testing.T) {
	tests := []struct {
		name     string
		itemName string
		itemType browser.ItemType
		wantIcon string
	}{
		{"Directory", "folder", browser.TypeDirectory, "📁"},
		{"MP4 video", "movie.mp4", browser.TypeFile, "🎬"},
		{"MKV video", "episode.mkv", browser.TypeFile, "🎥"},
		{"MP3 audio", "song.mp3", browser.TypeFile, "🎵"},
		{"FLAC audio", "track.flac", browser.TypeFile, "🎶"},
		{"Generic video", "video.webm", browser.TypeFile, "🌐"},
		{"Generic file", "document.txt", browser.TypeFile, "📄"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &browser.DirectoryItem{
				Name: tt.itemName,
				Type: tt.itemType,
			}
			if got := item.GetFileIcon(); got != tt.wantIcon {
				t.Errorf("GetFileIcon() = %v, want %v", got, tt.wantIcon)
			}
		})
	}
}

func TestPathUtilities(t *testing.T) {
	t.Run("GetParentPath", func(t *testing.T) {
		tests := []struct {
			input string
			want  string
		}{
			{"/media/movies/2024", "/media/movies"},
			{"/media/movies", "/media"},
			{"/media", ""},
			{"", ""},
			{"/", ""},
			{"/media/", ""}, // GetParentPath trims trailing slash, so /media/ -> /media -> ""
		}

		for _, tt := range tests {
			t.Run(tt.input, func(t *testing.T) {
				got := browser.GetParentPath(tt.input)
				if got != tt.want {
					t.Errorf("GetParentPath(%q) = %q, want %q", tt.input, got, tt.want)
				}
			})
		}
	})

	t.Run("NormalizePath", func(t *testing.T) {
		tests := []struct {
			input string
			want  string
		}{
			{"/media/movies/", "media/movies"},
			{"/media/movies", "media/movies"},
			{"media/movies/", "media/movies"},
			{"media/movies", "media/movies"},
			{"/", ""},
			{"", ""},
		}

		for _, tt := range tests {
			t.Run(tt.input, func(t *testing.T) {
				got := browser.NormalizePath(tt.input)
				if got != tt.want {
					t.Errorf("NormalizePath(%q) = %q, want %q", tt.input, got, tt.want)
				}
			})
		}
	})
}

func TestParseDateTime(t *testing.T) {
	htmlPath := filepath.Join("fixtures", "sample.html")
	htmlContent, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatalf("Failed to read fixture: %v", err)
	}

	listing, err := browser.ParseHTMLDirectory(string(htmlContent), "/media")
	if err != nil {
		t.Fatalf("ParseHTMLDirectory() error = %v", err)
	}

	// Check that at least some items have parsed dates
	foundDate := false
	for _, item := range listing.Items {
		if !item.ModifiedTime.IsZero() {
			foundDate = true
			// Verify the date is reasonable (not in the future, not before 2020)
			if item.ModifiedTime.After(time.Now()) {
				t.Errorf("Item %s has future date: %v", item.Name, item.ModifiedTime)
			}
			if item.ModifiedTime.Year() < 2020 {
				t.Errorf("Item %s has suspiciously old date: %v", item.Name, item.ModifiedTime)
			}
		}
	}

	if !foundDate {
		t.Error("No dates were parsed from HTML - date parsing may be broken")
	}
}

func TestFiltering(t *testing.T) {
	htmlPath := filepath.Join("fixtures", "sample.html")
	htmlContent, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatalf("Failed to read fixture: %v", err)
	}

	listing, err := browser.ParseHTMLDirectory(string(htmlContent), "/media")
	if err != nil {
		t.Fatalf("ParseHTMLDirectory() error = %v", err)
	}

	// Verify that only media files (video/audio) are included (plus directories)
	for _, item := range listing.Items {
		if item.Type == browser.TypeFile {
			if !item.IsVideoFile() && !item.IsAudioFile() {
				t.Errorf("Non-media file found in listing: %s", item.Name)
			}
		}
	}
}
