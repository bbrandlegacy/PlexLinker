package downloader

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// MediaCategory represents the type of media content
type MediaCategory string

const (
	CategoryMovies  MediaCategory = "movies"
	CategoryTVShows MediaCategory = "tv_shows"
	CategoryOther   MediaCategory = "other"
)

// CategorizedFiles groups files by their detected media category
type CategorizedFiles struct {
	Movies  []string
	TVShows []string
	Other   []string
}

// Season indicator patterns to detect TV shows
var tvShowPatterns = []*regexp.Regexp{
	// s01, s1, s01e01, s1e1
	regexp.MustCompile(`(?i)s\d{1,2}(e\d{1,2})?`),
	// season 1, season 01
	regexp.MustCompile(`(?i)season\s*\d{1,2}`),
	// 1x01, 1x1
	regexp.MustCompile(`(?i)\d{1,2}x\d{1,2}`),
	// episode patterns (ep01, ep1, episode 1)
	regexp.MustCompile(`(?i)(ep|episode)\s*\d{1,2}`),
}

// CategorizeFiles analyzes filenames and categorizes them as Movies or TV Shows
// Files without season indicators are categorized as Movies
// TV Shows are automatically sorted by season and episode
func CategorizeFiles(files []string) *CategorizedFiles {
	result := &CategorizedFiles{
		Movies:  make([]string, 0),
		TVShows: make([]string, 0),
		Other:   make([]string, 0),
	}

	for _, file := range files {
		if isTVShow(file) {
			result.TVShows = append(result.TVShows, file)
		} else {
			result.Movies = append(result.Movies, file)
		}
	}

	// Sort TV shows by season and episode
	if len(result.TVShows) > 0 {
		sort.Slice(result.TVShows, func(i, j int) bool {
			return compareTVEpisodes(result.TVShows[i], result.TVShows[j])
		})
	}

	return result
}

// isTVShow checks if a filename contains TV show season indicators
func isTVShow(filename string) bool {
	// Convert to lowercase for case-insensitive matching
	lower := strings.ToLower(filename)

	// Check each pattern
	for _, pattern := range tvShowPatterns {
		if pattern.MatchString(lower) {
			return true
		}
	}

	return false
}

// GetCategory returns the category of a single file
func GetCategory(filename string) MediaCategory {
	if isTVShow(filename) {
		return CategoryTVShows
	}
	return CategoryMovies
}

// Stats returns human-readable statistics about the categorization
func (cf *CategorizedFiles) Stats() string {
	stats := ""
	if len(cf.TVShows) > 0 {
		stats += "TV Shows: " + string(rune(len(cf.TVShows))) + " files\n"
	}
	if len(cf.Movies) > 0 {
		stats += "Movies: " + string(rune(len(cf.Movies))) + " files\n"
	}
	if len(cf.Other) > 0 {
		stats += "Other: " + string(rune(len(cf.Other))) + " files\n"
	}
	return stats
}

// HasTVShows returns true if any TV show files were detected
func (cf *CategorizedFiles) HasTVShows() bool {
	return len(cf.TVShows) > 0
}

// HasMovies returns true if any movie files were detected
func (cf *CategorizedFiles) HasMovies() bool {
	return len(cf.Movies) > 0
}

// HasOther returns true if any other files were detected
func (cf *CategorizedFiles) HasOther() bool {
	return len(cf.Other) > 0
}

// TotalFiles returns the total count of all files
func (cf *CategorizedFiles) TotalFiles() int {
	return len(cf.TVShows) + len(cf.Movies) + len(cf.Other)
}

// extractSeasonEpisode extracts season and episode numbers from a TV show filename
// Returns (season, episode) as integers. Returns (0, 0) if not found.
func extractSeasonEpisode(filename string) (int, int) {
	lower := strings.ToLower(filename)

	// Try s01e01, s1e1 format
	re := regexp.MustCompile(`s(\d{1,2})e(\d{1,2})`)
	if matches := re.FindStringSubmatch(lower); len(matches) >= 3 {
		season, _ := strconv.Atoi(matches[1])
		episode, _ := strconv.Atoi(matches[2])
		return season, episode
	}

	// Try s01, s1 format (no episode)
	re = regexp.MustCompile(`s(\d{1,2})`)
	if matches := re.FindStringSubmatch(lower); len(matches) >= 2 {
		season, _ := strconv.Atoi(matches[1])
		return season, 0
	}

	// Try 1x01 format
	re = regexp.MustCompile(`(\d{1,2})x(\d{1,2})`)
	if matches := re.FindStringSubmatch(lower); len(matches) >= 3 {
		season, _ := strconv.Atoi(matches[1])
		episode, _ := strconv.Atoi(matches[2])
		return season, episode
	}

	// Try "season 1 episode 2" format
	re = regexp.MustCompile(`season\s*(\d{1,2}).*?episode\s*(\d{1,2})`)
	if matches := re.FindStringSubmatch(lower); len(matches) >= 3 {
		season, _ := strconv.Atoi(matches[1])
		episode, _ := strconv.Atoi(matches[2])
		return season, episode
	}

	// Try "season 1" format (no episode)
	re = regexp.MustCompile(`season\s*(\d{1,2})`)
	if matches := re.FindStringSubmatch(lower); len(matches) >= 2 {
		season, _ := strconv.Atoi(matches[1])
		return season, 0
	}

	return 0, 0
}

// compareTVEpisodes compares two TV show filenames for sorting
// Returns true if file1 should come before file2
// Sorts by: Season (ascending) -> Episode (ascending) -> Filename (alphabetical)
func compareTVEpisodes(file1, file2 string) bool {
	season1, episode1 := extractSeasonEpisode(file1)
	season2, episode2 := extractSeasonEpisode(file2)

	// Compare by season first
	if season1 != season2 {
		return season1 < season2
	}

	// Same season, compare by episode
	if episode1 != episode2 {
		return episode1 < episode2
	}

	// Same season and episode (or both 0), sort alphabetically
	return strings.ToLower(file1) < strings.ToLower(file2)
}
