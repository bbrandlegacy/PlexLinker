package paths

import (
	"runtime"
	"strings"
)

// Default destination paths for different media types
const (
	// Windows paths (primary target platform)
	WindowsMoviesPath  = "W:\\movies"
	WindowsTVShowsPath = "W:\\shows"
	WindowsOtherPath   = "W:\\other"

	// Unix paths (Git Bash / WSL mounts)
	UnixMoviesPath  = "/w/movies"
	UnixTVShowsPath = "/w/shows"
	UnixOtherPath   = "/w/other"
)

// GetDefaultMoviesPath returns the platform-appropriate default movies path
func GetDefaultMoviesPath() string {
	if runtime.GOOS == "windows" {
		return WindowsMoviesPath
	}
	return UnixMoviesPath
}

// GetDefaultTVShowsPath returns the platform-appropriate default TV shows path
func GetDefaultTVShowsPath() string {
	if runtime.GOOS == "windows" {
		return WindowsTVShowsPath
	}
	return UnixTVShowsPath
}

// GetDefaultOtherPath returns the platform-appropriate default path for other files
func GetDefaultOtherPath() string {
	if runtime.GOOS == "windows" {
		return WindowsOtherPath
	}
	return UnixOtherPath
}

// ResolveAutoPath converts "auto" to the platform-appropriate default path
func ResolveAutoPath(path, mediaType string) string {
	if path == "auto" || path == "" {
		switch mediaType {
		case "movies":
			return GetDefaultMoviesPath()
		case "shows", "tvshows":
			return GetDefaultTVShowsPath()
		case "other":
			return GetDefaultOtherPath()
		default:
			return GetDefaultOtherPath()
		}
	}
	return NormalizeDestinationPath(path)
}

// NormalizeDestinationPath canonicalizes user-entered Windows paths without
// applying host-OS filepath rules. This matters when editing a Windows config
// on macOS/Linux and when migrating values that contain doubled separators.
// UNC paths retain exactly two leading backslashes.
func NormalizeDestinationPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" || path == "auto" {
		return path
	}
	if len(path) >= 2 && isASCIILetter(path[0]) && path[1] == ':' {
		return path[:2] + collapseWindowsSeparators(path[2:])
	}
	if strings.HasPrefix(path, `\\`) {
		return `\\` + strings.TrimLeft(collapseWindowsSeparators(strings.TrimLeft(path, `\`)), `\`)
	}
	return path
}

func collapseWindowsSeparators(value string) string {
	value = strings.ReplaceAll(value, "/", `\`)
	for strings.Contains(value, `\\`) {
		value = strings.ReplaceAll(value, `\\`, `\`)
	}
	return value
}

func isASCIILetter(value byte) bool {
	return value >= 'A' && value <= 'Z' || value >= 'a' && value <= 'z'
}
