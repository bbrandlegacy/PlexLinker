package browser

import (
	"fmt"
	"net/url"
	pathpkg "path"
	"regexp"
	"sort"
	"strings"
	"time"

	xhtml "golang.org/x/net/html"
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

var videoExtensions = map[string]struct{}{
	".3g2": {}, ".3gp": {}, ".amv": {}, ".asf": {}, ".avi": {}, ".av1": {},
	".divx": {}, ".drc": {}, ".dv": {}, ".f4f": {}, ".f4p": {}, ".f4v": {},
	".flv": {}, ".gif": {}, ".gifv": {}, ".h264": {}, ".h265": {}, ".hevc": {},
	".m1v": {}, ".m2p": {}, ".m2t": {}, ".m2ts": {}, ".m2v": {}, ".m4v": {},
	".mkv": {}, ".mk3d": {}, ".mng": {}, ".mov": {}, ".mp4": {}, ".mpe": {},
	".mpeg": {}, ".mpg": {}, ".mpv": {}, ".mpv2": {}, ".mts": {}, ".mxf": {},
	".nsv": {}, ".ogm": {}, ".ogv": {}, ".qt": {}, ".rm": {}, ".rmvb": {},
	".roq": {}, ".svi": {}, ".ts": {}, ".v210": {}, ".viv": {}, ".vob": {},
	".webm": {}, ".wmv": {}, ".xvid": {}, ".y4m": {}, ".yuv": {},
}

var audioExtensions = map[string]struct{}{
	".3ga": {}, ".8svx": {}, ".aac": {}, ".ac3": {}, ".adts": {}, ".aif": {},
	".aifc": {}, ".aiff": {}, ".alac": {}, ".amr": {}, ".ape": {}, ".au": {},
	".awb": {}, ".caf": {}, ".cda": {}, ".dct": {}, ".dff": {}, ".dsf": {},
	".dss": {}, ".dts": {}, ".dtshd": {}, ".dvf": {}, ".eac3": {}, ".ec3": {},
	".flac": {}, ".gsm": {}, ".iklax": {}, ".ivs": {}, ".m4a": {}, ".m4b": {},
	".m4p": {}, ".mid": {}, ".midi": {}, ".mka": {}, ".mlp": {}, ".mmf": {},
	".mogg": {}, ".mp1": {}, ".mp2": {}, ".mp3": {}, ".mpa": {}, ".mpc": {},
	".msv": {}, ".nmf": {}, ".oga": {}, ".ogg": {}, ".opus": {}, ".pcm": {},
	".ra": {}, ".raw": {}, ".rf64": {}, ".snd": {}, ".sln": {}, ".tak": {},
	".thd": {}, ".truehd": {}, ".tta": {}, ".voc": {}, ".vox": {}, ".wav": {},
	".w64": {}, ".wma": {}, ".wv": {},
}

func (di *DirectoryItem) hasExtension(extensions map[string]struct{}) bool {
	for _, candidate := range []string{di.Name, di.Path} {
		if hasExtension(candidate, extensions) {
			return true
		}
	}
	return false
}

func hasExtension(candidate string, extensions map[string]struct{}) bool {
	extension := strings.ToLower(pathpkg.Ext(strings.TrimSpace(candidate)))
	_, ok := extensions[extension]
	return ok
}

func hasMediaExtension(candidate string) bool {
	return hasExtension(candidate, videoExtensions) || hasExtension(candidate, audioExtensions)
}

func (di *DirectoryItem) IsVideoFile() bool {
	if di.Type != TypeFile {
		return false
	}
	return di.hasExtension(videoExtensions)
}

func (di *DirectoryItem) IsAudioFile() bool {
	if di.Type != TypeFile {
		return false
	}

	return di.hasExtension(audioExtensions)
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

	document, err := htmlpkgParse(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("parse directory HTML: %w", err)
	}
	dateRegex := regexp.MustCompile(`(\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}(?::\d{2})?|\d{2}-\w{3}-\d{4}\s+\d{2}:\d{2}|\d{1,2}/\d{1,2}/\d{4}\s+\d{1,2}:\d{2}\s*(?:AM|PM)?)`)
	seen := make(map[string]bool)
	var walk func(*xhtml.Node)
	walk = func(node *xhtml.Node) {
		if node.Type == xhtml.ElementNode && node.Data == "a" {
			href := attribute(node, "href")
			name := strings.TrimSpace(nodeText(node))
			if item, ok := directoryItemFromLink(href, name, currentPath); ok && !seen[item.Path] {
				containerText := nodeText(nearestContainer(node))
				if date := dateRegex.FindString(containerText); date != "" {
					item.ModifiedTime = parseDateTime(date)
				}
				if item.Type == TypeDirectory || item.IsVideoFile() || item.IsAudioFile() {
					seen[item.Path] = true
					listing.Items = append(listing.Items, item)
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(document)

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

var htmlpkgParse = xhtml.Parse

func directoryItemFromLink(href, name, currentPath string) (DirectoryItem, bool) {
	href, name = strings.TrimSpace(href), strings.TrimSpace(name)
	if href == "" || name == "" || href == "../" || name == ".." || name == "." || strings.HasPrefix(href, "#") {
		return DirectoryItem{}, false
	}
	parsed, err := url.Parse(href)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || strings.HasPrefix(href, "//") {
		return DirectoryItem{}, false
	}
	linkPath := parsed.EscapedPath()
	decodedPath, err := url.PathUnescape(linkPath)
	if err != nil {
		return DirectoryItem{}, false
	}
	if strings.HasPrefix(decodedPath, "/") || decodedPath == ".." || strings.HasPrefix(decodedPath, "../") {
		return DirectoryItem{}, false
	}
	for _, segment := range strings.Split(decodedPath, "/") {
		if segment == ".." {
			return DirectoryItem{}, false
		}
	}
	directory := strings.HasSuffix(decodedPath, "/") || strings.HasSuffix(name, "/")
	decodedPath = strings.TrimSuffix(decodedPath, "/")
	if decodedPath == "" {
		return DirectoryItem{}, false
	}
	itemType := TypeFile
	if directory {
		itemType = TypeDirectory
		name = strings.TrimSuffix(name, "/")
	} else if !hasMediaExtension(name) && hasMediaExtension(decodedPath) {
		// Some indexes use generic or truncated anchor text even though href
		// contains the complete filename. Preserve that filename for display,
		// classification, and aria2's output option.
		name = pathpkg.Base(decodedPath)
	}
	return DirectoryItem{Name: name, Path: combineURLPath(currentPath, decodedPath), Type: itemType}, true
}

func attribute(node *xhtml.Node, name string) string {
	for _, attribute := range node.Attr {
		if strings.EqualFold(attribute.Key, name) {
			return attribute.Val
		}
	}
	return ""
}

func nodeText(node *xhtml.Node) string {
	if node == nil {
		return ""
	}
	var builder strings.Builder
	var walk func(*xhtml.Node)
	walk = func(current *xhtml.Node) {
		if current.Type == xhtml.TextNode {
			builder.WriteString(current.Data)
			builder.WriteByte(' ')
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return strings.Join(strings.Fields(builder.String()), " ")
}

func nearestContainer(node *xhtml.Node) *xhtml.Node {
	for current := node; current != nil; current = current.Parent {
		if current.Type == xhtml.ElementNode && (current.Data == "tr" || current.Data == "pre" || current.Data == "li") {
			return current
		}
	}
	return node.Parent
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
	return strings.TrimPrefix(pathpkg.Join("/", basePath, relativePath), "/")
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
