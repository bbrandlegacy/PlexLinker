package patterns

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

type PlexNamer struct {
	settings *PlexSettings
}

type PlexSettings struct {
	CreateSeasonFolders bool
	IncludeYear         bool
	IncludeQuality      bool
	IncludeCodec        bool
	SanitizeFilenames   bool
}

func NewPlexNamer(settings *PlexSettings) *PlexNamer {
	if settings == nil {
		settings = &PlexSettings{
			CreateSeasonFolders: true,
			IncludeYear:         true,
			IncludeQuality:      false,
			IncludeCodec:        false,
			SanitizeFilenames:   true,
		}
	}
	return &PlexNamer{settings: settings}
}

// GeneratePlexName creates a Plex-compatible filename from TV show info
func (pn *PlexNamer) GeneratePlexName(info *TVShowInfo) string {
	var parts []string

	// Show name with optional year
	showName := info.ShowName
	if pn.settings.IncludeYear && info.Year != "" {
		showName = fmt.Sprintf("%s (%s)", showName, info.Year)
	}

	// Add region if present
	if info.Region != "" {
		showName = fmt.Sprintf("%s (%s)", showName, info.Region)
	}

	parts = append(parts, showName, "-")

	// Season and episode
	if info.IsSpecial {
		if info.EpisodeEnd > 0 {
			parts = append(parts, fmt.Sprintf("S00E%02d-E%02d", info.Episode, info.EpisodeEnd))
		} else {
			parts = append(parts, fmt.Sprintf("S00E%02d", info.Episode))
		}
	} else {
		if info.EpisodeEnd > 0 {
			parts = append(parts, fmt.Sprintf("S%02dE%02d-E%02d", info.Season, info.Episode, info.EpisodeEnd))
		} else {
			parts = append(parts, fmt.Sprintf("S%02dE%02d", info.Season, info.Episode))
		}
	}

	// Episode title
	if info.EpisodeTitle != "" {
		parts = append(parts, "-", info.EpisodeTitle)
	}

	// Optional quality and codec
	if pn.settings.IncludeQuality && info.Quality != "" {
		parts = append(parts, "-", info.Quality)
	}

	if pn.settings.IncludeCodec && info.VideoCodec != "" {
		parts = append(parts, "-", strings.ToUpper(info.VideoCodec))
	}

	filename := strings.Join(parts, " ")

	// Add extension
	filename = fmt.Sprintf("%s.%s", filename, info.Extension)

	// Sanitize filename
	if pn.settings.SanitizeFilenames {
		filename = pn.sanitizeFilename(filename)
	}

	return filename
}

// GenerateSeasonFolder creates the season folder name
func (pn *PlexNamer) GenerateSeasonFolder(info *TVShowInfo) string {
	if info.IsSpecial {
		return "Specials"
	}
	return fmt.Sprintf("Season %02d", info.Season)
}

// GenerateShowFolder creates the show folder name
func (pn *PlexNamer) GenerateShowFolder(info *TVShowInfo) string {
	showName := info.ShowName
	if pn.settings.IncludeYear && info.Year != "" {
		showName = fmt.Sprintf("%s (%s)", showName, info.Year)
	}

	if info.Region != "" {
		showName = fmt.Sprintf("%s (%s)", showName, info.Region)
	}

	if pn.settings.SanitizeFilenames {
		showName = pn.sanitizeFilename(showName)
	}

	return showName
}

// GenerateFullPath creates the complete path structure
func (pn *PlexNamer) GenerateFullPath(basePath string, info *TVShowInfo) string {
	showFolder := pn.GenerateShowFolder(info)
	filename := pn.GeneratePlexName(info)

	if pn.settings.CreateSeasonFolders {
		seasonFolder := pn.GenerateSeasonFolder(info)
		return filepath.Join(basePath, showFolder, seasonFolder, filename)
	}

	return filepath.Join(basePath, showFolder, filename)
}

// sanitizeFilename removes or replaces invalid filename characters
func (pn *PlexNamer) sanitizeFilename(filename string) string {
	// Remove or replace invalid characters for cross-platform compatibility
	invalidChars := regexp.MustCompile(`[<>:"/\\|?*]`)
	filename = invalidChars.ReplaceAllString(filename, "")

	// Replace multiple spaces with single space
	multiSpace := regexp.MustCompile(`\s+`)
	filename = multiSpace.ReplaceAllString(filename, " ")

	// Trim spaces and dots from the end (Windows doesn't like them)
	filename = strings.TrimRight(filename, " .")

	return filename
}

// ValidatePlexStructure checks if the generated structure is valid
func (pn *PlexNamer) ValidatePlexStructure(info *TVShowInfo) error {
	if info.ShowName == "" {
		return fmt.Errorf("show name cannot be empty")
	}

	if info.Season < 0 {
		return fmt.Errorf("season number cannot be negative")
	}

	if info.Episode <= 0 {
		return fmt.Errorf("episode number must be positive")
	}

	if info.EpisodeEnd > 0 && info.EpisodeEnd <= info.Episode {
		return fmt.Errorf("episode end must be greater than episode start")
	}

	if !info.IsVideoFile() {
		return fmt.Errorf("file is not a video file")
	}

	return nil
}

// PreviewRename shows what the rename operation would look like
func (pn *PlexNamer) PreviewRename(currentPath string, info *TVShowInfo) (string, string, error) {
	if err := pn.ValidatePlexStructure(info); err != nil {
		return "", "", err
	}

	basePath := filepath.Dir(currentPath)
	newPath := pn.GenerateFullPath(basePath, info)

	return currentPath, newPath, nil
}

// GetConflicts checks for potential naming conflicts
func (pn *PlexNamer) GetConflicts(infos []*TVShowInfo) []string {
	var conflicts []string
	nameCount := make(map[string]int)

	for _, info := range infos {
		name := pn.GeneratePlexName(info)
		nameCount[name]++
	}

	for name, count := range nameCount {
		if count > 1 {
			conflicts = append(conflicts, fmt.Sprintf("%s (appears %d times)", name, count))
		}
	}

	return conflicts
}