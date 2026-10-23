package patterns

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type TVShowInfo struct {
	ShowName     string
	Season       int
	Episode      int
	EpisodeEnd   int // For multi-episode files (S01E01-E02)
	EpisodeTitle string
	Year         string
	Quality      string
	AudioInfo    string
	VideoCodec   string
	ReleaseGroup string
	Region       string
	Extension    string
	OriginalName string
	IsSpecial    bool
}

type PatternMatcher struct {
	patterns []PatternRule
}

type PatternRule struct {
	Name    string
	Pattern *regexp.Regexp
	Extract func(matches []string) (*TVShowInfo, error)
}

func NewPatternMatcher() *PatternMatcher {
	pm := &PatternMatcher{}
	pm.initializePatterns()
	return pm
}

func (pm *PatternMatcher) initializePatterns() {
	pm.patterns = []PatternRule{
		// Standard format: Show.Name.S01E01.Episode.Title.1080p.x264-GROUP.mkv
		{
			Name:    "Standard SxxExx",
			Pattern: regexp.MustCompile(`(?i)^(.+?)[.\s_-]+[Ss](\d{1,2})[Ee](\d{1,2})(?:[Ee](\d{1,2}))?(?:[Ee]\d{1,2})*(?:[.\s_-]+(.+?))?(?:[.\s_-]+(\d{3,4}p))?(?:[.\s_-]+(x264|x265|h264|h265|hevc))?(?:[.\s_-]+(.+?))?\.(\w+)$`),
			Extract: pm.extractStandardSE,
		},
		// Alternative format: Show Name 1x01 Episode Title
		{
			Name:    "Alternative xEE",
			Pattern: regexp.MustCompile(`(?i)^(.+?)[.\s_-]+(\d{1,2})x(\d{1,2})(?:[.\s_-]+(.+?))?(?:[.\s_-]+(\d{3,4}p))?(?:[.\s_-]+(x264|x265|h264|h265|hevc))?(?:[.\s_-]+(.+?))?\.(\w+)$`),
			Extract: pm.extractAlternativeX,
		},
		// Season Episode format: Show Name Season 1 Episode 01
		{
			Name:    "Season Episode",
			Pattern: regexp.MustCompile(`(?i)^(.+?)[.\s_-]+season[.\s_-]+(\d{1,2})[.\s_-]+episode[.\s_-]+(\d{1,2})(?:[.\s_-]+(.+?))?(?:[.\s_-]+(\d{3,4}p))?(?:[.\s_-]+(x264|x265|h264|h265|hevc))?(?:[.\s_-]+(.+?))?\.(\w+)$`),
			Extract: pm.extractSeasonEpisode,
		},
		// Simple format: Show Name 101 (season 1, episode 01)
		{
			Name:    "Three Digit",
			Pattern: regexp.MustCompile(`(?i)^(.+?)[.\s_-]+(\d)(\d{2})(?:[.\s_-]+(.+?))?(?:[.\s_-]+(\d{3,4}p))?(?:[.\s_-]+(x264|x265|h264|h265|hevc))?(?:[.\s_-]+(.+?))?\.(\w+)$`),
			Extract: pm.extractThreeDigit,
		},
		// Year-based shows: Show Name (2019) S01E01
		{
			Name:    "Year Show",
			Pattern: regexp.MustCompile(`(?i)^(.+?)[.\s_-]+\((\d{4})\)[.\s_-]+[Ss](\d{1,2})[Ee](\d{1,2})(?:[Ee](\d{1,2}))?(?:[.\s_-]+(.+?))?(?:[.\s_-]+(\d{3,4}p))?(?:[.\s_-]+(x264|x265|h264|h265|hevc))?(?:[.\s_-]+(.+?))?\.(\w+)$`),
			Extract: pm.extractYearShow,
		},
		// Specials: Show Name S00E01
		{
			Name:    "Specials",
			Pattern: regexp.MustCompile(`(?i)^(.+?)[.\s_-]+[Ss]00[Ee](\d{1,2})(?:[.\s_-]+(.+?))?(?:[.\s_-]+(\d{3,4}p))?(?:[.\s_-]+(x264|x265|h264|h265|hevc))?(?:[.\s_-]+(.+?))?\.(\w+)$`),
			Extract: pm.extractSpecials,
		},
		// Regional: Show Name US S01E01
		{
			Name:    "Regional",
			Pattern: regexp.MustCompile(`(?i)^(.+?)[.\s_-]+(US|UK|AU|CA)[.\s_-]+[Ss](\d{1,2})[Ee](\d{1,2})(?:[Ee](\d{1,2}))?(?:[.\s_-]+(.+?))?(?:[.\s_-]+(\d{3,4}p))?(?:[.\s_-]+(x264|x265|h264|h265|hevc))?(?:[.\s_-]+(.+?))?\.(\w+)$`),
			Extract: pm.extractRegional,
		},
	}
}

func (pm *PatternMatcher) ParseFilename(filename string) (*TVShowInfo, error) {
	for _, rule := range pm.patterns {
		matches := rule.Pattern.FindStringSubmatch(filename)
		if matches != nil {
			info, err := rule.Extract(matches)
			if err != nil {
				continue // Try next pattern
			}
			info.OriginalName = filename
			info.Extension = strings.ToLower(filepath.Ext(filename)[1:])
			pm.enrichMetadata(info, filename)
			return info, nil
		}
	}
	return nil, fmt.Errorf("no matching pattern found for: %s", filename)
}

var (
	yearSuffixPattern   = regexp.MustCompile(`\s*\((19\d{2}|20\d{2})\)$`)
	regionSuffixPattern = regexp.MustCompile(`(?i)\s+(US|UK|AU|CA)$`)
	episodeMarker       = regexp.MustCompile(`(?i)(s\d{1,2}e\d{1,2}(?:e\d{1,2})*|\d{1,2}x\d{1,2}|season[.\s_-]+\d{1,2}[.\s_-]+episode[.\s_-]+\d{1,2})`)
	releaseToken        = regexp.MustCompile(`(?i)^(480p|720p|1080p|2160p|4k|bluray|brrip|webrip|web-dl|hdtv|dvdrip|remux|x264|x265|h264|h265|hevc|aac|ac3|dts|10bit|8bit)$`)
)

func (pm *PatternMatcher) enrichMetadata(info *TVShowInfo, filename string) {
	if matches := yearSuffixPattern.FindStringSubmatch(info.ShowName); len(matches) == 2 {
		info.Year = matches[1]
		info.ShowName = strings.TrimSpace(yearSuffixPattern.ReplaceAllString(info.ShowName, ""))
	}
	if matches := regionSuffixPattern.FindStringSubmatch(info.ShowName); len(matches) == 2 {
		info.Region = strings.ToUpper(matches[1])
		info.ShowName = strings.TrimSpace(regionSuffixPattern.ReplaceAllString(info.ShowName, ""))
	}

	base := strings.TrimSuffix(filename, filepath.Ext(filename))
	marker := episodeMarker.FindStringIndex(base)
	if marker == nil {
		return
	}
	remainder := strings.Trim(base[marker[1]:], ". _-")
	if remainder == "" {
		info.EpisodeTitle = ""
		return
	}
	parts := regexp.MustCompile(`[.\s_-]+`).Split(remainder, -1)
	title := make([]string, 0, len(parts))
	for _, part := range parts {
		if releaseToken.MatchString(part) {
			break
		}
		title = append(title, part)
	}
	info.EpisodeTitle = cleanEpisodeTitle(strings.Join(title, " "))
}

func (pm *PatternMatcher) extractStandardSE(matches []string) (*TVShowInfo, error) {
	if len(matches) < 4 {
		return nil, fmt.Errorf("insufficient matches")
	}

	season, err := strconv.Atoi(matches[2])
	if err != nil {
		return nil, err
	}

	episode, err := strconv.Atoi(matches[3])
	if err != nil {
		return nil, err
	}

	info := &TVShowInfo{
		ShowName: cleanShowName(matches[1]),
		Season:   season,
		Episode:  episode,
	}

	// Multi-episode
	if len(matches) > 4 && matches[4] != "" {
		if episodeEnd, err := strconv.Atoi(matches[4]); err == nil {
			info.EpisodeEnd = episodeEnd
		}
	}

	// Episode title, quality, codec, etc.
	if len(matches) > 5 && matches[5] != "" {
		info.EpisodeTitle = cleanEpisodeTitle(matches[5])
	}
	if len(matches) > 6 && matches[6] != "" {
		info.Quality = matches[6]
	}
	if len(matches) > 7 && matches[7] != "" {
		info.VideoCodec = matches[7]
	}
	if len(matches) > 8 && matches[8] != "" {
		info.ReleaseGroup = matches[8]
	}

	return info, nil
}

func (pm *PatternMatcher) extractAlternativeX(matches []string) (*TVShowInfo, error) {
	if len(matches) < 4 {
		return nil, fmt.Errorf("insufficient matches")
	}

	season, err := strconv.Atoi(matches[2])
	if err != nil {
		return nil, err
	}

	episode, err := strconv.Atoi(matches[3])
	if err != nil {
		return nil, err
	}

	info := &TVShowInfo{
		ShowName: cleanShowName(matches[1]),
		Season:   season,
		Episode:  episode,
	}

	if len(matches) > 4 && matches[4] != "" {
		info.EpisodeTitle = cleanEpisodeTitle(matches[4])
	}
	if len(matches) > 5 && matches[5] != "" {
		info.Quality = matches[5]
	}

	return info, nil
}

func (pm *PatternMatcher) extractSeasonEpisode(matches []string) (*TVShowInfo, error) {
	if len(matches) < 4 {
		return nil, fmt.Errorf("insufficient matches")
	}

	season, err := strconv.Atoi(matches[2])
	if err != nil {
		return nil, err
	}

	episode, err := strconv.Atoi(matches[3])
	if err != nil {
		return nil, err
	}

	return &TVShowInfo{
		ShowName:     cleanShowName(matches[1]),
		Season:       season,
		Episode:      episode,
		EpisodeTitle: cleanEpisodeTitle(matches[4]),
	}, nil
}

func (pm *PatternMatcher) extractThreeDigit(matches []string) (*TVShowInfo, error) {
	if len(matches) < 4 {
		return nil, fmt.Errorf("insufficient matches")
	}

	season, err := strconv.Atoi(matches[2])
	if err != nil {
		return nil, err
	}

	episode, err := strconv.Atoi(matches[3])
	if err != nil {
		return nil, err
	}

	return &TVShowInfo{
		ShowName: cleanShowName(matches[1]),
		Season:   season,
		Episode:  episode,
	}, nil
}

func (pm *PatternMatcher) extractYearShow(matches []string) (*TVShowInfo, error) {
	if len(matches) < 5 {
		return nil, fmt.Errorf("insufficient matches")
	}

	season, err := strconv.Atoi(matches[3])
	if err != nil {
		return nil, err
	}

	episode, err := strconv.Atoi(matches[4])
	if err != nil {
		return nil, err
	}

	return &TVShowInfo{
		ShowName: cleanShowName(matches[1]),
		Year:     matches[2],
		Season:   season,
		Episode:  episode,
	}, nil
}

func (pm *PatternMatcher) extractSpecials(matches []string) (*TVShowInfo, error) {
	if len(matches) < 3 {
		return nil, fmt.Errorf("insufficient matches")
	}

	episode, err := strconv.Atoi(matches[2])
	if err != nil {
		return nil, err
	}

	return &TVShowInfo{
		ShowName:  cleanShowName(matches[1]),
		Season:    0,
		Episode:   episode,
		IsSpecial: true,
	}, nil
}

func (pm *PatternMatcher) extractRegional(matches []string) (*TVShowInfo, error) {
	if len(matches) < 5 {
		return nil, fmt.Errorf("insufficient matches")
	}

	season, err := strconv.Atoi(matches[3])
	if err != nil {
		return nil, err
	}

	episode, err := strconv.Atoi(matches[4])
	if err != nil {
		return nil, err
	}

	return &TVShowInfo{
		ShowName: cleanShowName(matches[1]),
		Region:   matches[2],
		Season:   season,
		Episode:  episode,
	}, nil
}

func cleanShowName(name string) string {
	// Replace dots, underscores with spaces
	name = strings.ReplaceAll(name, ".", " ")
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")

	// Remove extra spaces
	name = regexp.MustCompile(`\s+`).ReplaceAllString(name, " ")
	name = strings.TrimSpace(name)

	// Title case
	words := strings.Fields(name)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}

	return strings.Join(words, " ")
}

func cleanEpisodeTitle(title string) string {
	if title == "" {
		return ""
	}

	// Remove quality indicators and technical info
	qualityPattern := regexp.MustCompile(`(?i)\b(720p|1080p|2160p|4k|x264|x265|h264|h265|hevc|aac|ac3|dts|bluray|brrip|webrip|web-dl|hdtv|dvdrip)\b`)
	title = qualityPattern.ReplaceAllString(title, "")

	// Replace dots, underscores with spaces
	title = strings.ReplaceAll(title, ".", " ")
	title = strings.ReplaceAll(title, "_", " ")
	title = strings.ReplaceAll(title, "-", " ")

	// Remove extra spaces
	title = regexp.MustCompile(`\s+`).ReplaceAllString(title, " ")
	title = strings.TrimSpace(title)

	// Title case
	words := strings.Fields(title)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}

	return strings.Join(words, " ")
}

func (info *TVShowInfo) IsVideoFile() bool {
	videoExts := []string{
		"mp4", "mkv", "avi", "mov", "wmv", "flv", "webm",
		"m4v", "3gp", "mpg", "mpeg", "ts", "m2ts", "mts",
		"vob", "ogv", "asf", "rm", "rmvb", "divx", "xvid",
	}

	for _, ext := range videoExts {
		if strings.EqualFold(info.Extension, ext) {
			return true
		}
	}
	return false
}
