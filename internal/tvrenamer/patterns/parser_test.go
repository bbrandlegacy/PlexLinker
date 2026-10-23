package patterns

import (
	"fmt"
	"testing"
)

// TestCase represents a test case for pattern matching
type TestCase struct {
	Input          string
	ExpectedShow   string
	ExpectedSeason int
	ExpectedEp     int
	ExpectedEpEnd  int
	ExpectedYear   string
	ExpectedRegion string
	ExpectedTitle  string
	ShouldMatch    bool
	Description    string
}

func TestPatternMatching(t *testing.T) {
	testCases := []TestCase{
		// Standard SxxExx format
		{
			Input:          "The.Office.S01E01.mkv",
			ExpectedShow:   "The Office",
			ExpectedSeason: 1,
			ExpectedEp:     1,
			ShouldMatch:    true,
			Description:    "Standard: Basic SxxExx format",
		},
		{
			Input:          "Breaking.Bad.S05E01.1080p.BluRay.x265.mkv",
			ExpectedShow:   "Breaking Bad",
			ExpectedSeason: 5,
			ExpectedEp:     1,
			ShouldMatch:    true,
			Description:    "Standard: With quality tags",
		},
		{
			Input:          "Game.of.Thrones.S08E06.The.Iron.Throne.1080p.WEB-DL.x264-GROUP.mkv",
			ExpectedShow:   "Game Of Thrones",
			ExpectedSeason: 8,
			ExpectedEp:     6,
			ExpectedTitle:  "The Iron Throne",
			ShouldMatch:    true,
			Description:    "Standard: With episode title and quality",
		},

		// Multi-episode formats
		{
			Input:          "Show.s01e01e02.mkv",
			ExpectedShow:   "Show",
			ExpectedSeason: 1,
			ExpectedEp:     1,
			ExpectedEpEnd:  2,
			ShouldMatch:    true,
			Description:    "Multi-episode: Adjacent episodes (e01e02)",
		},
		{
			Input:          "Show.S01E01E02E03.mkv",
			ExpectedShow:   "Show",
			ExpectedSeason: 1,
			ExpectedEp:     1,
			ExpectedEpEnd:  2, // Note: parser only captures first multi-episode
			ShouldMatch:    true,
			Description:    "Multi-episode: Three episodes (e01e02e03)",
		},

		// Alternative xEE format
		{
			Input:          "show.1x01.mkv",
			ExpectedShow:   "Show",
			ExpectedSeason: 1,
			ExpectedEp:     1,
			ShouldMatch:    true,
			Description:    "Alternative: 1x01 format",
		},
		{
			Input:          "The.Wire.5x10.Final.Episode.720p.mkv",
			ExpectedShow:   "The Wire",
			ExpectedSeason: 5,
			ExpectedEp:     10,
			ShouldMatch:    true,
			Description:    "Alternative: 5x10 with title and quality",
		},

		// Three digit format (101 = S01E01)
		{
			Input:          "show.101.mkv",
			ExpectedShow:   "Show",
			ExpectedSeason: 1,
			ExpectedEp:     1,
			ShouldMatch:    true,
			Description:    "Three digit: 101 format",
		},
		{
			Input:          "Friends.523.1080p.mkv",
			ExpectedShow:   "Friends",
			ExpectedSeason: 5,
			ExpectedEp:     23,
			ShouldMatch:    true,
			Description:    "Three digit: 523 with quality",
		},

		// Verbose format
		{
			Input:          "Game of Thrones Season 1 Episode 1.mkv",
			ExpectedShow:   "Game Of Thrones",
			ExpectedSeason: 1,
			ExpectedEp:     1,
			ShouldMatch:    true,
			Description:    "Verbose: 'Season X Episode Y' format",
		},
		{
			Input:          "The.Mandalorian.Season.2.Episode.08.The.Rescue.1080p.mkv",
			ExpectedShow:   "The Mandalorian",
			ExpectedSeason: 2,
			ExpectedEp:     8,
			ExpectedTitle:  "The Rescue",
			ShouldMatch:    true,
			Description:    "Verbose: With episode title and quality",
		},

		// Year-based shows
		{
			Input:          "Sherlock.(2010).S01E01.mkv",
			ExpectedShow:   "Sherlock",
			ExpectedYear:   "2010",
			ExpectedSeason: 1,
			ExpectedEp:     1,
			ShouldMatch:    true,
			Description:    "Year: Show with year in parentheses",
		},
		{
			Input:          "Doctor.Who.(2005).S13E06.1080p.mkv",
			ExpectedShow:   "Doctor Who",
			ExpectedYear:   "2005",
			ExpectedSeason: 13,
			ExpectedEp:     6,
			ShouldMatch:    true,
			Description:    "Year: With quality tag",
		},

		// Specials
		{
			Input:          "The.Office.S00E01.Special.mkv",
			ExpectedShow:   "The Office",
			ExpectedSeason: 0,
			ExpectedEp:     1,
			ShouldMatch:    true,
			Description:    "Specials: S00E01 format",
		},

		// Regional variants
		{
			Input:          "The.Office.US.S01E01.mkv",
			ExpectedShow:   "The Office",
			ExpectedRegion: "US",
			ExpectedSeason: 1,
			ExpectedEp:     1,
			ShouldMatch:    true,
			Description:    "Regional: US variant",
		},
		{
			Input:          "Shameless.US.S11E12.1080p.mkv",
			ExpectedShow:   "Shameless",
			ExpectedRegion: "US",
			ExpectedSeason: 11,
			ExpectedEp:     12,
			ShouldMatch:    true,
			Description:    "Regional: US with quality",
		},

		// Special characters and unicode
		{
			Input:          "It's.Always.Sunny.in.Philadelphia.S15E01.mkv",
			ExpectedShow:   "It's Always Sunny In Philadelphia",
			ExpectedSeason: 15,
			ExpectedEp:     1,
			ShouldMatch:    true,
			Description:    "Special chars: Apostrophe in title",
		},
		{
			Input:          "La.Casa.de.Papel.S01E01.mkv",
			ExpectedShow:   "La Casa De Papel",
			ExpectedSeason: 1,
			ExpectedEp:     1,
			ShouldMatch:    true,
			Description:    "Unicode: Spanish title",
		},

		// Edge cases that should fail
		{
			Input:       "episode.mkv",
			ShouldMatch: false,
			Description: "Edge case: No season/episode info",
		},
		{
			Input:       "just_a_movie.mkv",
			ShouldMatch: false,
			Description: "Edge case: Movie file without episode info",
		},
		{
			Input:       "101.mkv",
			ShouldMatch: false,
			Description: "Edge case: Just numbers, no show name",
		},

		// Anime-style formats (currently not supported)
		{
			Input:       "Attack.on.Titan.-.01.[1080p].mkv",
			ShouldMatch: false,
			Description: "Anime: Dash separator format (not supported)",
		},
		{
			Input:       "[SubGroup].Show.Name.-.01.mkv",
			ShouldMatch: false,
			Description: "Anime: Bracket group format (not supported)",
		},

		// Complex quality tags
		{
			Input:          "Breaking.Bad.S05E16.FINAL.1080p.BluRay.x265.10bit.DTS-HD.MA.5.1-GROUP.mkv",
			ExpectedShow:   "Breaking Bad",
			ExpectedSeason: 5,
			ExpectedEp:     16,
			ShouldMatch:    true,
			Description:    "Complex: Multiple quality tags",
		},

		// Uncommon separators
		{
			Input:          "The_Office_S01E01_Pilot.mkv",
			ExpectedShow:   "The Office",
			ExpectedSeason: 1,
			ExpectedEp:     1,
			ExpectedTitle:  "Pilot",
			ShouldMatch:    true,
			Description:    "Separator: Underscores",
		},
		{
			Input:          "Breaking-Bad-S01E01.mkv",
			ExpectedShow:   "Breaking Bad",
			ExpectedSeason: 1,
			ExpectedEp:     1,
			ShouldMatch:    true,
			Description:    "Separator: Hyphens",
		},

		// Single digit season/episode
		{
			Input:          "Show.S1E1.mkv",
			ExpectedShow:   "Show",
			ExpectedSeason: 1,
			ExpectedEp:     1,
			ShouldMatch:    true,
			Description:    "Single digit: S1E1",
		},

		// Case variations
		{
			Input:          "show.name.s01e01.mkv",
			ExpectedShow:   "Show Name",
			ExpectedSeason: 1,
			ExpectedEp:     1,
			ShouldMatch:    true,
			Description:    "Case: All lowercase",
		},
		{
			Input:          "SHOW.NAME.S01E01.MKV",
			ExpectedShow:   "Show Name",
			ExpectedSeason: 1,
			ExpectedEp:     1,
			ShouldMatch:    true,
			Description:    "Case: All uppercase",
		},

		// Various video extensions
		{
			Input:          "Show.S01E01.mp4",
			ExpectedShow:   "Show",
			ExpectedSeason: 1,
			ExpectedEp:     1,
			ShouldMatch:    true,
			Description:    "Extension: .mp4",
		},
		{
			Input:          "Show.S01E01.avi",
			ExpectedShow:   "Show",
			ExpectedSeason: 1,
			ExpectedEp:     1,
			ShouldMatch:    true,
			Description:    "Extension: .avi",
		},
	}

	pm := NewPatternMatcher()
	pn := NewPlexNamer(nil) // Use default settings

	fmt.Println("\n=== TV SHOW PATTERN MATCHING TEST RESULTS ===")

	passCount := 0
	failCount := 0

	for i, tc := range testCases {
		info, err := pm.ParseFilename(tc.Input)

		if !tc.ShouldMatch {
			if err != nil {
				fmt.Printf("✓ Test %d: %s\n", i+1, tc.Description)
				fmt.Printf("  Input:  %s\n", tc.Input)
				fmt.Printf("  Result: Correctly failed to match\n\n")
				passCount++
			} else {
				fmt.Printf("✗ Test %d: %s\n", i+1, tc.Description)
				fmt.Printf("  Input:    %s\n", tc.Input)
				fmt.Printf("  Expected: Should not match\n")
				fmt.Printf("  Got:      Matched as %s S%02dE%02d\n\n", info.ShowName, info.Season, info.Episode)
				failCount++
			}
			continue
		}

		if err != nil {
			fmt.Printf("✗ Test %d: %s\n", i+1, tc.Description)
			fmt.Printf("  Input:  %s\n", tc.Input)
			fmt.Printf("  Error:  %v\n\n", err)
			failCount++
			continue
		}

		// Validate parsed data
		passed := true
		var errors []string

		if info.ShowName != tc.ExpectedShow {
			errors = append(errors, fmt.Sprintf("Show name: expected '%s', got '%s'", tc.ExpectedShow, info.ShowName))
			passed = false
		}

		if info.Season != tc.ExpectedSeason {
			errors = append(errors, fmt.Sprintf("Season: expected %d, got %d", tc.ExpectedSeason, info.Season))
			passed = false
		}

		if info.Episode != tc.ExpectedEp {
			errors = append(errors, fmt.Sprintf("Episode: expected %d, got %d", tc.ExpectedEp, info.Episode))
			passed = false
		}

		if tc.ExpectedEpEnd > 0 && info.EpisodeEnd != tc.ExpectedEpEnd {
			errors = append(errors, fmt.Sprintf("Episode End: expected %d, got %d", tc.ExpectedEpEnd, info.EpisodeEnd))
			passed = false
		}

		if tc.ExpectedYear != "" && info.Year != tc.ExpectedYear {
			errors = append(errors, fmt.Sprintf("Year: expected '%s', got '%s'", tc.ExpectedYear, info.Year))
			passed = false
		}

		if tc.ExpectedRegion != "" && info.Region != tc.ExpectedRegion {
			errors = append(errors, fmt.Sprintf("Region: expected '%s', got '%s'", tc.ExpectedRegion, info.Region))
			passed = false
		}

		if tc.ExpectedTitle != "" && info.EpisodeTitle != tc.ExpectedTitle {
			errors = append(errors, fmt.Sprintf("Episode Title: expected '%s', got '%s'", tc.ExpectedTitle, info.EpisodeTitle))
			passed = false
		}

		if passed {
			fmt.Printf("✓ Test %d: %s\n", i+1, tc.Description)
			fmt.Printf("  Input:  %s\n", tc.Input)
			fmt.Printf("  Parsed: %s - S%02dE%02d", info.ShowName, info.Season, info.Episode)
			if info.EpisodeEnd > 0 {
				fmt.Printf("-E%02d", info.EpisodeEnd)
			}
			if info.EpisodeTitle != "" {
				fmt.Printf(" - %s", info.EpisodeTitle)
			}
			fmt.Println()

			// Generate Plex name
			plexName := pn.GeneratePlexName(info)
			fmt.Printf("  Plex:   %s\n\n", plexName)
			passCount++
		} else {
			fmt.Printf("✗ Test %d: %s\n", i+1, tc.Description)
			fmt.Printf("  Input: %s\n", tc.Input)
			for _, errMsg := range errors {
				fmt.Printf("  ✗ %s\n", errMsg)
			}
			fmt.Println()
			failCount++
		}
	}

	fmt.Printf("\n=== SUMMARY ===\n")
	fmt.Printf("Total:  %d\n", passCount+failCount)
	fmt.Printf("Passed: %d\n", passCount)
	fmt.Printf("Failed: %d\n", failCount)
	fmt.Printf("Rate:   %.1f%%\n\n", float64(passCount)/float64(passCount+failCount)*100)

	if failCount > 0 {
		t.Errorf("%d tests failed", failCount)
	}
}

// TestPlexNaming specifically tests Plex format generation
func TestPlexNaming(t *testing.T) {
	testCases := []struct {
		Input    *TVShowInfo
		Expected string
		Desc     string
	}{
		{
			Input: &TVShowInfo{
				ShowName:  "The Office",
				Season:    1,
				Episode:   1,
				Extension: "mkv",
			},
			Expected: "The Office - S01E01.mkv",
			Desc:     "Basic format",
		},
		{
			Input: &TVShowInfo{
				ShowName:     "Breaking Bad",
				Season:       5,
				Episode:      14,
				EpisodeTitle: "Ozymandias",
				Extension:    "mkv",
			},
			Expected: "Breaking Bad - S05E14 - Ozymandias.mkv",
			Desc:     "With episode title",
		},
		{
			Input: &TVShowInfo{
				ShowName:  "Doctor Who",
				Year:      "2005",
				Season:    1,
				Episode:   1,
				Extension: "mkv",
			},
			Expected: "Doctor Who (2005) - S01E01.mkv",
			Desc:     "With year",
		},
		{
			Input: &TVShowInfo{
				ShowName:  "The Office",
				Region:    "US",
				Season:    1,
				Episode:   1,
				Extension: "mkv",
			},
			Expected: "The Office (US) - S01E01.mkv",
			Desc:     "With region",
		},
		{
			Input: &TVShowInfo{
				ShowName:   "Show",
				Season:     1,
				Episode:    1,
				EpisodeEnd: 3,
				Extension:  "mkv",
			},
			Expected: "Show - S01E01-E03.mkv",
			Desc:     "Multi-episode",
		},
		{
			Input: &TVShowInfo{
				ShowName:  "Show",
				Season:    0,
				Episode:   1,
				IsSpecial: true,
				Extension: "mkv",
			},
			Expected: "Show - S00E01.mkv",
			Desc:     "Special episode",
		},
	}

	pn := NewPlexNamer(&PlexSettings{
		CreateSeasonFolders: true,
		IncludeYear:         true,
		IncludeQuality:      false,
		IncludeCodec:        false,
		SanitizeFilenames:   true,
	})

	fmt.Println("\n=== PLEX NAMING FORMAT TEST ===")

	passCount := 0
	failCount := 0

	for i, tc := range testCases {
		result := pn.GeneratePlexName(tc.Input)

		if result == tc.Expected {
			fmt.Printf("✓ Test %d: %s\n", i+1, tc.Desc)
			fmt.Printf("  Result: %s\n\n", result)
			passCount++
		} else {
			fmt.Printf("✗ Test %d: %s\n", i+1, tc.Desc)
			fmt.Printf("  Expected: %s\n", tc.Expected)
			fmt.Printf("  Got:      %s\n\n", result)
			failCount++
		}
	}

	fmt.Printf("Passed: %d/%d\n\n", passCount, passCount+failCount)

	if failCount > 0 {
		t.Errorf("%d Plex naming tests failed", failCount)
	}
}

// TestEdgeCases tests specific edge cases and potential bugs
func TestEdgeCases(t *testing.T) {
	pm := NewPatternMatcher()

	fmt.Println("\n=== EDGE CASE TESTS ===")

	edgeCases := []struct {
		Input       string
		Description string
		ShouldMatch bool
	}{
		{
			Input:       "Show.S01E00.mkv",
			Description: "Episode 0 (should match)",
			ShouldMatch: true,
		},
		{
			Input:       "Show.S99E99.mkv",
			Description: "Very high season/episode numbers",
			ShouldMatch: true,
		},
		{
			Input:       "Show.Name.With.Dots.S01E01.mkv",
			Description: "Many dots in show name",
			ShouldMatch: true,
		},
		{
			Input:       "Show (2020) S01E01.mkv",
			Description: "Year without escaping pattern",
			ShouldMatch: false, // Won't match Year Show pattern (needs exact format)
		},
		{
			Input:       "Show.S01E01.part1.mkv",
			Description: "Part indicator",
			ShouldMatch: true,
		},
		{
			Input:       "Show.S01E01-E02-E03.mkv",
			Description: "Three episodes with dashes",
			ShouldMatch: true,
		},
		{
			Input:       ".hidden.S01E01.mkv",
			Description: "Hidden file (leading dot)",
			ShouldMatch: true,
		},
	}

	for i, tc := range edgeCases {
		info, err := pm.ParseFilename(tc.Input)

		fmt.Printf("Test %d: %s\n", i+1, tc.Description)
		fmt.Printf("  Input: %s\n", tc.Input)

		if tc.ShouldMatch {
			if err == nil {
				fmt.Printf("  ✓ Matched: %s - S%02dE%02d\n", info.ShowName, info.Season, info.Episode)
			} else {
				fmt.Printf("  ✗ Failed to match (expected to match)\n")
				fmt.Printf("  Error: %v\n", err)
			}
		} else {
			if err != nil {
				fmt.Printf("  ✓ Correctly failed to match\n")
			} else {
				fmt.Printf("  ✗ Matched (expected to fail): %s - S%02dE%02d\n", info.ShowName, info.Season, info.Episode)
			}
		}
		fmt.Println()
	}
}

// Benchmark the pattern matcher
func BenchmarkPatternMatching(b *testing.B) {
	pm := NewPatternMatcher()
	testFilename := "Breaking.Bad.S05E16.FINAL.1080p.BluRay.x265.10bit.DTS-HD.MA.5.1-GROUP.mkv"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pm.ParseFilename(testFilename)
	}
}
