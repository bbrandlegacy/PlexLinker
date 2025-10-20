package browser

import (
	"regexp"
	"sort"
	"strings"
	"time"
)

type ItemType int

const (
	TypeDirectory ItemType = iota
	TypeFile
)

type DirectoryItem struct {
	Name         string
	Path         string
	Type         ItemType
	Selected     bool
	ModifiedTime time.Time
}

func (di *DirectoryItem) IsVideoFile() bool {
	if di.Type != TypeFile {
		return false
	}

	videoExtensions := []string{
		".3g2", ".3gp", ".amv", ".asf", ".avi", ".divx", ".drc",
		".dv", ".f4f", ".f4p", ".f4v", ".flv", ".gif", ".gifv",
		".m1v", ".m2ts", ".m2v", ".m4v", ".mng", ".mov", ".mp2",
		".mp4", ".mpe", ".mpeg", ".mpg", ".mpv", ".mpv2", ".mts",
		".mxf", ".mkv", ".nsv", ".ogv", ".qt", ".rm", ".rmvb",
		".roq", ".svi", ".ts", ".viv", ".vob", ".webm", ".wmv",
		".xvid", ".yuv",
	}

	lowerName := strings.ToLower(di.Name)
	for _, ext := range videoExtensions {
		if strings.HasSuffix(lowerName, ext) {
			return true
		}
	}
	return false
}

func (di *DirectoryItem) IsAudioFile() bool {
	if di.Type != TypeFile {
		return false
	}

	audioExtensions := []string{
		".3ga", ".8svx", ".aac", ".ac3", ".aiff", ".alac", ".ape",
		".au", ".awb", ".cda", ".dct", ".dss", ".dts", ".dvf",
		".flac", ".gsm", ".iklax", ".ivs", ".m4a", ".m4b", ".m4p",
		".mmf", ".mogg", ".mp3", ".mpc", ".msv", ".nmf", ".oga",
		".ogg", ".opus", ".pcm", ".ra", ".raw", ".rf64", ".sln",
		".tta", ".voc", ".vox", ".wav", ".wma", ".wv",
	}

	lowerName := strings.ToLower(di.Name)
	for _, ext := range audioExtensions {
		if strings.HasSuffix(lowerName, ext) {
			return true
		}
	}
	return false
}

func (di *DirectoryItem) GetFileIcon() string {
	if di.Type == TypeDirectory {
		return "📁"
	}

	lowerName := strings.ToLower(di.Name)

	// Video file icons
	if strings.HasSuffix(lowerName, ".mp4") {
		return "🎬"
	}
	if strings.HasSuffix(lowerName, ".mkv") {
		return "🎥"
	}
	if strings.HasSuffix(lowerName, ".avi") {
		return "📹"
	}
	if strings.HasSuffix(lowerName, ".mov") {
		return "🎦"
	}
	if strings.HasSuffix(lowerName, ".wmv") {
		return "📺"
	}
	if strings.HasSuffix(lowerName, ".webm") {
		return "🌐"
	}
	if strings.HasSuffix(lowerName, ".flv") {
		return "⚡"
	}
	if strings.HasSuffix(lowerName, ".m4v") {
		return "📱"
	}
	if strings.HasSuffix(lowerName, ".3gp") || strings.HasSuffix(lowerName, ".3g2") {
		return "📞"
	}
	if strings.HasSuffix(lowerName, ".mpg") || strings.HasSuffix(lowerName, ".mpeg") {
		return "🎞️"
	}
	if strings.HasSuffix(lowerName, ".ts") || strings.HasSuffix(lowerName, ".m2ts") || strings.HasSuffix(lowerName, ".mts") {
		return "📡"
	}
	if strings.HasSuffix(lowerName, ".vob") {
		return "💿"
	}

	// Audio file icons
	if strings.HasSuffix(lowerName, ".mp3") {
		return "🎵"
	}
	if strings.HasSuffix(lowerName, ".flac") || strings.HasSuffix(lowerName, ".alac") {
		return "🎶"
	}
	if strings.HasSuffix(lowerName, ".wav") || strings.HasSuffix(lowerName, ".aiff") {
		return "🔊"
	}
	if strings.HasSuffix(lowerName, ".aac") || strings.HasSuffix(lowerName, ".m4a") {
		return "🎧"
	}
	if strings.HasSuffix(lowerName, ".ogg") || strings.HasSuffix(lowerName, ".opus") {
		return "🎼"
	}
	if strings.HasSuffix(lowerName, ".wma") {
		return "🎤"
	}

	// Generic video/audio fallbacks
	if di.IsVideoFile() {
		return "🎬"
	}
	if di.IsAudioFile() {
		return "🎵"
	}

	// Default file icon
	return "📄"
}

type DirectoryListing struct {
	Items       []DirectoryItem
	CurrentPath string
}

func ParseHTMLDirectory(html, currentPath string) (*DirectoryListing, error) {
	return ParseHTMLDirectoryWithSort(html, currentPath, true)
}

func ParseHTMLDirectoryWithSort(html, currentPath string, sortByDate bool) (*DirectoryListing, error) {
	listing := &DirectoryListing{
		Items:       make([]DirectoryItem, 0),
		CurrentPath: currentPath,
	}

	// Parse table rows to extract links and dates
	rowRegex := regexp.MustCompile(`(?s)<tr>.*?</tr>`)
	linkRegex := regexp.MustCompile(`<a\s+[^>]*href=["']([^"']+)["'][^>]*>([^<]+)</a>`)
	// Common date formats in directory listings: "2024-01-15 14:30", "15-Jan-2024 14:30", etc.
	dateRegex := regexp.MustCompile(`(\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}|\d{2}-\w{3}-\d{4}\s+\d{2}:\d{2}|\d{1,2}/\d{1,2}/\d{4}\s+\d{1,2}:\d{2}\s*(?:AM|PM)?)`)

	rows := rowRegex.FindAllString(html, -1)

	for _, row := range rows {
		linkMatches := linkRegex.FindStringSubmatch(row)
		if len(linkMatches) < 3 {
			continue
		}

		href := strings.TrimSpace(linkMatches[1])
		name := strings.TrimSpace(linkMatches[2])

		if href == "" || name == "" || href == "../" || name == ".." || name == "." {
			continue
		}

		if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") ||
			strings.HasPrefix(href, "mailto:") || strings.HasPrefix(href, "#") {
			continue
		}

		var itemType ItemType
		var itemPath string

		if strings.HasSuffix(href, "/") {
			itemType = TypeDirectory
			itemPath = combineURLPath(currentPath, strings.TrimSuffix(href, "/"))
		} else {
			itemType = TypeFile
			itemPath = combineURLPath(currentPath, href)
		}

		// Parse modification time from row
		modTime := time.Time{}
		dateMatches := dateRegex.FindStringSubmatch(row)
		if len(dateMatches) > 0 {
			modTime = parseDateTime(dateMatches[0])
		}

		item := DirectoryItem{
			Name:         name,
			Path:         itemPath,
			Type:         itemType,
			Selected:     false,
			ModifiedTime: modTime,
		}

		if itemType == TypeFile && !item.IsVideoFile() && !item.IsAudioFile() {
			continue
		}

		listing.Items = append(listing.Items, item)
	}

	// Only sort by date if requested (typically only for root directory)
	if sortByDate {
		// Sort by modification time (newest first), then by type (directories first), then by name
		sort.Slice(listing.Items, func(i, j int) bool {
			// Directories come before files
			if listing.Items[i].Type != listing.Items[j].Type {
				return listing.Items[i].Type == TypeDirectory
			}

			// If both have valid times, sort by time (newest first)
			if !listing.Items[i].ModifiedTime.IsZero() && !listing.Items[j].ModifiedTime.IsZero() {
				return listing.Items[i].ModifiedTime.After(listing.Items[j].ModifiedTime)
			}

			// If one has time and other doesn't, prioritize the one with time
			if !listing.Items[i].ModifiedTime.IsZero() {
				return true
			}
			if !listing.Items[j].ModifiedTime.IsZero() {
				return false
			}

			// Fall back to alphabetical
			return listing.Items[i].Name < listing.Items[j].Name
		})
	}
	// Otherwise, items remain in HTML order

	return listing, nil
}

func parseDateTime(dateStr string) time.Time {
	dateStr = strings.TrimSpace(dateStr)

	// Try common date formats
	formats := []string{
		"2006-01-02 15:04",
		"2006-01-02 15:04:05",
		"02-Jan-2006 15:04",
		"01/02/2006 3:04 PM",
		"1/2/2006 3:04 PM",
		"01/02/2006 15:04",
		"1/2/2006 15:04",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t
		}
	}

	return time.Time{}
}

func combineURLPath(basePath, relativePath string) string {
	if basePath == "" || basePath == "/" {
		return relativePath
	}

	if strings.HasSuffix(basePath, "/") {
		return basePath + relativePath
	}
	return basePath + "/" + relativePath
}

func GetParentPath(path string) string {
	if path == "" || path == "/" {
		return ""
	}

	path = strings.TrimSuffix(path, "/")

	lastSlash := strings.LastIndex(path, "/")
	if lastSlash == -1 {
		return ""
	}

	if lastSlash == 0 {
		return ""
	}

	return path[:lastSlash]
}

func NormalizePath(path string) string {
	if path == "" {
		return ""
	}

	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")

	if path == "" {
		return ""
	}

	return path
}
