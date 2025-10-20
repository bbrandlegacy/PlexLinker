package paths

import "runtime"

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
	return path
}
