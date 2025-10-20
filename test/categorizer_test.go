package test

import (
	"testing"

	"plexlinker/internal/downloader"
)

func TestCategorizeFiles(t *testing.T) {
	tests := []struct {
		name           string
		files          []string
		wantTVShows    int
		wantMovies     int
		wantOther      int
		specificChecks map[string]downloader.MediaCategory // filename -> expected category
	}{
		{
			name: "Mixed TV shows and movies",
			files: []string{
				"Breaking.Bad.S01E01.mkv",
				"The.Matrix.1999.mkv",
				"Game.of.Thrones.1x01.mp4",
				"Inception.2010.mp4",
				"The.Office.S02E05.avi",
			},
			wantTVShows: 3,
			wantMovies:  2,
			wantOther:   0,
			specificChecks: map[string]downloader.MediaCategory{
				"Breaking.Bad.S01E01.mkv":  downloader.CategoryTVShows,
				"The.Matrix.1999.mkv":      downloader.CategoryMovies,
				"Game.of.Thrones.1x01.mp4": downloader.CategoryTVShows,
			},
		},
		{
			name: "All movies",
			files: []string{
				"The.Matrix.1999.mkv",
				"Inception.2010.mp4",
				"Interstellar.2014.mkv",
				"Pulp.Fiction.1994.avi",
			},
			wantTVShows: 0,
			wantMovies:  4,
			wantOther:   0,
		},
		{
			name: "All TV shows with various patterns",
			files: []string{
				"Breaking.Bad.S01E01.mkv",    // s01e01 pattern
				"Game.of.Thrones.1x01.mp4",   // 1x01 pattern
				"Friends.Season 1.mkv",       // season with space
				"The.Office.ep01.mkv",        // ep01 pattern
				"Seinfeld.episode 5.mp4",     // episode with space
			},
			wantTVShows: 5,
			wantMovies:  0,
			wantOther:   0,
		},
		{
			name:        "Empty list",
			files:       []string{},
			wantTVShows: 0,
			wantMovies:  0,
			wantOther:   0,
		},
		{
			name: "Edge cases - season patterns",
			files: []string{
				"Show.s1.mkv",                // Short season number
				"Show.s99e99.mkv",            // Large episode numbers
				"Show.S01.mkv",               // Season only, no episode
				"Movie.About.Seasons.2020.mkv", // Word "season" in title
			},
			wantTVShows: 3,
			wantMovies:  1,
			wantOther:   0,
			specificChecks: map[string]downloader.MediaCategory{
				"Show.s1.mkv":                downloader.CategoryTVShows,
				"Movie.About.Seasons.2020.mkv": downloader.CategoryMovies,
			},
		},
		{
			name: "Case insensitivity",
			files: []string{
				"BREAKING.BAD.S01E01.MKV",
				"the.office.S02E05.AVI",
				"Friends.1X01.mp4",
				"MOVIE.2020.MKV",
			},
			wantTVShows: 3,
			wantMovies:  1,
			wantOther:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := downloader.CategorizeFiles(tt.files)

			if len(result.TVShows) != tt.wantTVShows {
				t.Errorf("TV shows count = %d, want %d", len(result.TVShows), tt.wantTVShows)
				t.Logf("TV shows: %v", result.TVShows)
			}

			if len(result.Movies) != tt.wantMovies {
				t.Errorf("Movies count = %d, want %d", len(result.Movies), tt.wantMovies)
				t.Logf("Movies: %v", result.Movies)
			}

			if len(result.Other) != tt.wantOther {
				t.Errorf("Other count = %d, want %d", len(result.Other), tt.wantOther)
				t.Logf("Other: %v", result.Other)
			}

			// Check specific file categorizations
			for filename, expectedCategory := range tt.specificChecks {
				found := false
				switch expectedCategory {
				case downloader.CategoryTVShows:
					for _, f := range result.TVShows {
						if f == filename {
							found = true
							break
						}
					}
				case downloader.CategoryMovies:
					for _, f := range result.Movies {
						if f == filename {
							found = true
							break
						}
					}
				case downloader.CategoryOther:
					for _, f := range result.Other {
						if f == filename {
							found = true
							break
						}
					}
				}

				if !found {
					t.Errorf("File %s expected in category %s but not found", filename, expectedCategory)
				}
			}
		})
	}
}

func TestGetCategory(t *testing.T) {
	tests := []struct {
		filename string
		want     downloader.MediaCategory
	}{
		// TV Show patterns
		{"Breaking.Bad.S01E01.mkv", downloader.CategoryTVShows},
		{"Game.of.Thrones.1x01.mp4", downloader.CategoryTVShows},
		{"Friends.Season 1.mkv", downloader.CategoryTVShows},
		{"The.Office.ep01.mkv", downloader.CategoryTVShows},
		{"Seinfeld.episode 5.mp4", downloader.CategoryTVShows},
		{"Show.s1.mkv", downloader.CategoryTVShows},
		{"Show.s99e99.mkv", downloader.CategoryTVShows},
		{"Show.S01.mkv", downloader.CategoryTVShows},

		// Movie patterns (no season indicators)
		{"The.Matrix.1999.mkv", downloader.CategoryMovies},
		{"Inception.2010.mp4", downloader.CategoryMovies},
		{"Interstellar.2014.mkv", downloader.CategoryMovies},
		{"Pulp.Fiction.1994.avi", downloader.CategoryMovies},
		{"Movie.About.Seasons.2020.mkv", downloader.CategoryMovies},
		{"Random.Movie.mkv", downloader.CategoryMovies},
		{"NoExtension", downloader.CategoryMovies},

		// Edge cases
		{"", downloader.CategoryMovies}, // Empty string -> movie
		{"s01", downloader.CategoryTVShows}, // Just season indicator
		{"1x01", downloader.CategoryTVShows}, // Just episode pattern
		{"UPPERCASE.S01E01.MKV", downloader.CategoryTVShows}, // Case insensitive
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := downloader.GetCategory(tt.filename)
			if got != tt.want {
				t.Errorf("GetCategory(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestCategorizedFilesMethods(t *testing.T) {
	files := []string{
		"Breaking.Bad.S01E01.mkv",
		"The.Matrix.1999.mkv",
		"Game.of.Thrones.1x01.mp4",
		"Inception.2010.mp4",
	}

	categorized := downloader.CategorizeFiles(files)

	t.Run("HasTVShows", func(t *testing.T) {
		if !categorized.HasTVShows() {
			t.Error("HasTVShows() = false, want true")
		}
	})

	t.Run("HasMovies", func(t *testing.T) {
		if !categorized.HasMovies() {
			t.Error("HasMovies() = false, want true")
		}
	})

	t.Run("HasOther", func(t *testing.T) {
		if categorized.HasOther() {
			t.Error("HasOther() = true, want false")
		}
	})

	t.Run("TotalFiles", func(t *testing.T) {
		total := categorized.TotalFiles()
		if total != 4 {
			t.Errorf("TotalFiles() = %d, want 4", total)
		}
	})
}

func TestTVShowPatternRegression(t *testing.T) {
	// Regression tests for specific patterns that might break
	tests := []struct {
		filename    string
		shouldBe    downloader.MediaCategory
		description string
	}{
		{
			filename:    "The.Big.Bang.Theory.S01E01.720p.HDTV.x264-DIMENSION.mkv",
			shouldBe:    downloader.CategoryTVShows,
			description: "Complex TV show filename with quality tags",
		},
		{
			filename:    "Movie.2020.1080p.BluRay.x264-SPARKS.mkv",
			shouldBe:    downloader.CategoryMovies,
			description: "Movie with quality tags (no season pattern)",
		},
		{
			filename:    "Documentary.About.Season.Changes.2021.mkv",
			shouldBe:    downloader.CategoryMovies,
			description: "Movie with 'season' in title but no pattern",
		},
		{
			filename:    "Show.S00E00.Special.mkv",
			shouldBe:    downloader.CategoryTVShows,
			description: "TV show special with S00 pattern",
		},
		{
			filename:    "Series.Name.101.mkv",
			shouldBe:    downloader.CategoryMovies,
			description: "Number that looks like episode but isn't marked as such",
		},
		{
			filename:    "Series.Name.episode 101.mkv",
			shouldBe:    downloader.CategoryTVShows,
			description: "Episode 101 with 'episode' keyword and space",
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			got := downloader.GetCategory(tt.filename)
			if got != tt.shouldBe {
				t.Errorf("GetCategory(%q) = %v, want %v - %s",
					tt.filename, got, tt.shouldBe, tt.description)
			}
		})
	}
}

func TestSeasonPatternVariations(t *testing.T) {
	// Test various season/episode pattern formats
	tvShowPatterns := []string{
		// Standard patterns
		"Show.S01E01.mkv",
		"Show.s01e01.mkv",
		"Show.S1E1.mkv",
		"Show.s1e1.mkv",

		// Episode range
		"Show.S01E01-E02.mkv",
		"Show.S01E01E02.mkv",

		// Season only
		"Show.S01.mkv",
		"Show.s1.mkv",

		// Alternative formats
		"Show.1x01.mkv",
		"Show.1X01.mkv",
		"Show.01x01.mkv",

		// Word-based (requires space after "season"/"episode")
		"Show.Season 1.mkv",
		"Show.season 1.mkv",
		"Show.Season 01.mkv",

		// Episode patterns
		"Show.ep01.mkv",
		"Show.EP01.mkv",
		"Show.episode 1.mkv",
		"Show.Episode 1.mkv",
	}

	moviePatterns := []string{
		"Movie.2020.mkv",
		"Movie.Name.1999.mkv",
		"Random.Movie.mkv",
		"Movie.Part.1.mkv", // Part 1, not S01 or 1x01
		"Movie.Vol.1.mkv",
	}

	t.Run("TV show patterns", func(t *testing.T) {
		for _, filename := range tvShowPatterns {
			got := downloader.GetCategory(filename)
			if got != downloader.CategoryTVShows {
				t.Errorf("Expected %q to be categorized as TV show, got %v", filename, got)
			}
		}
	})

	t.Run("Movie patterns", func(t *testing.T) {
		for _, filename := range moviePatterns {
			got := downloader.GetCategory(filename)
			if got != downloader.CategoryMovies {
				t.Errorf("Expected %q to be categorized as movie, got %v", filename, got)
			}
		}
	})
}
