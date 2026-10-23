package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"plexlinker/internal/tvrenamer/patterns"
)

type FileItem struct {
	Name        string
	Path        string
	IsDirectory bool
	IsVideo     bool
	Size        int64
	Selected    bool
	TVInfo      *patterns.TVShowInfo
	PlexName    string
	HasError    bool
	Error       string
}

type DirectoryListing struct {
	Items       []FileItem
	CurrentPath string
	Parent      string
}

type FileSystemBrowser struct {
	currentPath    string
	patternMatcher *patterns.PatternMatcher
	plexNamer      *patterns.PlexNamer
}

func NewFileSystemBrowser() *FileSystemBrowser {
	wd, _ := os.Getwd()
	return &FileSystemBrowser{
		currentPath:    wd,
		patternMatcher: patterns.NewPatternMatcher(),
		plexNamer:      patterns.NewPlexNamer(nil),
	}
}

func (fsb *FileSystemBrowser) GetCurrentPath() string {
	return fsb.currentPath
}

func (fsb *FileSystemBrowser) SetCurrentPath(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("path does not exist: %s", absPath)
	}

	fsb.currentPath = absPath
	return nil
}

func (fsb *FileSystemBrowser) NavigateUp() error {
	parent := filepath.Dir(fsb.currentPath)
	if parent == fsb.currentPath {
		return fmt.Errorf("already at root directory")
	}
	return fsb.SetCurrentPath(parent)
}

func (fsb *FileSystemBrowser) NavigateInto(dirName string) error {
	newPath := filepath.Join(fsb.currentPath, dirName)
	return fsb.SetCurrentPath(newPath)
}

func (fsb *FileSystemBrowser) ListDirectory() (*DirectoryListing, error) {
	entries, err := os.ReadDir(fsb.currentPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %v", err)
	}

	listing := &DirectoryListing{
		CurrentPath: fsb.currentPath,
		Parent:      filepath.Dir(fsb.currentPath),
		Items:       make([]FileItem, 0),
	}

	for _, entry := range entries {
		item := FileItem{
			Name:        entry.Name(),
			Path:        filepath.Join(fsb.currentPath, entry.Name()),
			IsDirectory: entry.IsDir(),
		}

		if !entry.IsDir() {
			if info, err := entry.Info(); err == nil {
				item.Size = info.Size()
			}

			// Check if it's a video file and try to parse TV show info
			if fsb.isVideoFile(entry.Name()) {
				item.IsVideo = true
				if tvInfo, err := fsb.patternMatcher.ParseFilename(entry.Name()); err == nil {
					item.TVInfo = tvInfo
					item.PlexName = fsb.plexNamer.GeneratePlexName(tvInfo)
				} else {
					item.HasError = true
					item.Error = err.Error()
				}
			}
		}

		listing.Items = append(listing.Items, item)
	}

	// Sort: directories first, then files
	sort.Slice(listing.Items, func(i, j int) bool {
		if listing.Items[i].IsDirectory != listing.Items[j].IsDirectory {
			return listing.Items[i].IsDirectory
		}
		return strings.ToLower(listing.Items[i].Name) < strings.ToLower(listing.Items[j].Name)
	})

	return listing, nil
}

func (fsb *FileSystemBrowser) GetVideoFiles() ([]*FileItem, error) {
	listing, err := fsb.ListDirectory()
	if err != nil {
		return nil, err
	}

	var videoFiles []*FileItem
	for i := range listing.Items {
		item := &listing.Items[i]
		if item.IsVideo {
			videoFiles = append(videoFiles, item)
		}
	}

	return videoFiles, nil
}

func (fsb *FileSystemBrowser) GetVideoFilesRecursive() ([]*FileItem, error) {
	var allVideoFiles []*FileItem

	err := filepath.Walk(fsb.currentPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		if info.IsDir() {
			return nil
		}

		if fsb.isVideoFile(info.Name()) {
			item := &FileItem{
				Name:        info.Name(),
				Path:        path,
				IsDirectory: false,
				IsVideo:     true,
				Size:        info.Size(),
			}

			if tvInfo, err := fsb.patternMatcher.ParseFilename(info.Name()); err == nil {
				item.TVInfo = tvInfo
				item.PlexName = fsb.plexNamer.GeneratePlexName(tvInfo)
			} else {
				item.HasError = true
				item.Error = err.Error()
			}

			allVideoFiles = append(allVideoFiles, item)
		}

		return nil
	})

	return allVideoFiles, err
}

func (fsb *FileSystemBrowser) isVideoFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	videoExts := []string{
		".mp4", ".mkv", ".avi", ".mov", ".wmv", ".flv", ".webm",
		".m4v", ".3gp", ".mpg", ".mpeg", ".ts", ".m2ts", ".mts",
		".vob", ".ogv", ".asf", ".rm", ".rmvb", ".divx", ".xvid",
	}

	for _, videoExt := range videoExts {
		if ext == videoExt {
			return true
		}
	}
	return false
}

func (fsb *FileSystemBrowser) FilterByPattern(pattern string) ([]*FileItem, error) {
	listing, err := fsb.ListDirectory()
	if err != nil {
		return nil, err
	}

	var filtered []*FileItem
	pattern = strings.ToLower(pattern)

	for i := range listing.Items {
		item := &listing.Items[i]
		if strings.Contains(strings.ToLower(item.Name), pattern) {
			filtered = append(filtered, item)
		}
	}

	return filtered, nil
}

func (fsb *FileSystemBrowser) GetFileIcon(item *FileItem) string {
	if item.IsDirectory {
		return "📁"
	}

	if !item.IsVideo {
		return "📄"
	}

	if item.HasError {
		return "❌"
	}

	if item.TVInfo != nil {
		return "🎬"
	}

	return "📹"
}

func (fsb *FileSystemBrowser) GetFileInfo(item *FileItem) string {
	if item.IsDirectory {
		return "Directory"
	}

	if !item.IsVideo {
		return "File"
	}

	if item.HasError {
		return fmt.Sprintf("Parse Error: %s", item.Error)
	}

	if item.TVInfo != nil {
		info := item.TVInfo
		if info.IsSpecial {
			return fmt.Sprintf("%s - S00E%02d", info.ShowName, info.Episode)
		}
		return fmt.Sprintf("%s - S%02dE%02d", info.ShowName, info.Season, info.Episode)
	}

	return "Video file (no pattern match)"
}

func (fsb *FileSystemBrowser) FormatFileSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	if size < KB {
		return fmt.Sprintf("%d B", size)
	} else if size < MB {
		return fmt.Sprintf("%.1f KB", float64(size)/KB)
	} else if size < GB {
		return fmt.Sprintf("%.1f MB", float64(size)/MB)
	} else {
		return fmt.Sprintf("%.1f GB", float64(size)/GB)
	}
}
