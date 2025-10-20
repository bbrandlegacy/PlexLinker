# PlexLinker - Interactive HTML Directory Browser & aria2c Script Generator

A Go CLI application that provides an interactive HTML directory browser and generates aria2c download scripts for selected files with tmux integration.

## Development Environment

### Build Commands
```bash
# Build for current platform
go build -o plexlinker ./cmd/plexlinker

# Build for Windows (primary target)
GOOS=windows GOARCH=amd64 go build -o plexlinker.exe ./cmd/plexlinker
```

### Run Commands
```bash
go run ./cmd/plexlinker
```

### Test Commands
```bash
go test ./...
```

### Lint Commands
```bash
go vet ./...
go fmt ./...
```

## Project Structure
```
plexlinker/
├── cmd/plexlinker/          # Main application entry point
│   └── main.go              # Application state machine, signal handling
├── internal/
│   ├── config/              # Configuration management
│   │   └── config.go        # JSON config load/save, validation, normalization
│   ├── logger/              # Centralized debug logging (NEW)
│   │   └── logger.go        # Thread-safe logger with namespaces
│   ├── paths/               # Platform-aware path defaults (NEW)
│   │   └── defaults.go      # Windows/Unix default paths
│   ├── browser/             # Directory browsing & parsing logic
│   │   └── parser.go        # HTML parsing, sorting, media filtering
│   ├── ui/                  # Terminal UI components
│   │   ├── browser.go       # Main directory browser UI (non-blocking recursive scan)
│   │   ├── menu.go          # Main menu UI
│   │   ├── settings.go      # Settings configuration UI
│   │   ├── selected_viewer.go   # Selected files viewer
│   │   ├── script_viewer.go     # Script viewer with syntax highlighting
│   │   ├── script_generator.go  # Script generation window with preview
│   │   ├── confirm_modal.go     # Confirmation modal dialog
│   │   ├── clipboard.go     # Clipboard operations (cross-platform)
│   │   ├── signal.go        # Signal handling for SSH disconnects
│   │   ├── window_windows.go    # Windows-specific window management
│   │   └── window_default.go    # Default (non-Windows) stub
│   ├── downloader/          # aria2c script generation
│   │   ├── generator.go     # Multi-script generation with categorization
│   │   ├── categorizer.go   # Auto-categorization (TV shows vs movies)
│   │   ├── exec_windows.go  # Windows script execution
│   │   └── exec_default.go  # Unix script execution
│   ├── client/              # HTTP client for web server
│   │   └── client.go        # HTTP basic auth, request handling
│   └── tvrenamer/           # TV show renaming functionality
│       ├── config/          # TV renamer config
│       ├── filesystem/      # Local filesystem browser
│       ├── patterns/        # TV show pattern matching
│       │   ├── parser.go    # Multi-pattern filename parsing
│       │   └── plex.go      # Plex naming format generation
│       └── ui/              # TV renamer UI
│           ├── tv_browser.go    # Filesystem browser UI
│           ├── preview.go       # Rename preview UI
│           └── execute.go       # Rename execution UI
├── test/                    # Test files and archives
│   └── archive/             # Archived test files
├── .claude/                 # Claude Code configuration
│   └── agents/              # Specialized agent definitions
├── go.mod
├── go.sum
├── CLAUDE.md                # This file
└── AUDIT_REPORT.md          # Comprehensive audit results
```

## Features

### HTML Directory Browser
- Connect to authenticated web servers
- Interactive directory navigation with keyboard controls
- **Smart directory sorting by modification date (newest first)**
- **Cursor position memory - returns to exact position when navigating back**
- File and directory selection for aria2c script generation
- Video and audio file filtering
- Visual indicators for cursor position and selected items
- **Selected files viewer (press '4') - dedicated screen for reviewing selections**

### File Selection
- Individual file selection (Enter on file)
- Select all in current directory (press '3')
- **Recursive select all (press '2') - iterates through all subdirectories**
- Deselect all (press '5')
- Real-time selected file counter in header

### Settings Management
- JSON configuration file for credentials and settings
- **Configurable options:**
  - Base URL, username, password
  - Destination paths for movies, TV shows, and other files
  - Script format (bat/sh/txt)
  - Debug mode toggle
- Input validation and error handling
- Settings auto-save to `plexlinker.json`

### Navigation Controls
- **Arrow keys or hjkl** for navigation
- **Enter** to navigate into directories or select files
- **Backspace/h** to go up directories
- **Esc** to exit to main menu
- **Tab** to switch to running script window (Windows only)
- **Number key shortcuts:**
  - 1: Select all in current directory
  - 2: Select all recursively
  - 3: Deselect all
  - 4: Show selected files viewer
  - 5: Generate script (with media type selection)
  - 0: Exit to main menu

### Script Generation (Press '5')
- **Unified Script Generation Window** - Full-screen interactive preview before generation
- **Auto-Categorization:** Automatically detects TV shows vs movies based on filename patterns
  - Season indicators: s01, s1e1, season 1, 1x01, episode 1
  - Files with season patterns → TV Shows
  - All other files → Movies
- **Multi-Script Generation:** Creates separate scripts for each category
  - `plexhelper_movies.bat` - Movies script with configurable destination
  - `plexhelper_tvshows.bat` - TV shows script with folder selection
  - `plexhelper_other.bat` - Other files script (if applicable)
- **Smart Script Output:**
  - Scripts are saved to the same directory as the plexlinker executable
  - Scripts automatically `cd` to the configured output directory before running aria2c
  - Downloads are saved to the configured destination (e.g., `W:\movies`)
  - "auto" paths are resolved to platform defaults (`W:\movies` on Windows, `/w/movies` on Unix)
  - Scripts persist and can be re-run manually anytime
- **TV Show Folder Management:**
  - Browse existing folders in configured TV shows path
  - Create new folders inline with current directory context
  - Keyboard-driven folder browser (↑↓/jk to navigate)
- **Preview Summary:** Natural language description of what will happen
  - File counts per category (📺 TV Shows, 🎬 Movies, 📁 Other)
  - Destination paths for each category
  - Total script count
  - Script output locations
- **Multi-format support:**
  - `.bat` - Windows batch file with headers and auto-cd to destination
  - `.sh` - Unix shell script with headers and auto-cd to destination
  - `.txt` - Plain text command string
- **Automatic script execution:**
  - Each script launches immediately in a new window after generation
  - Windows: New CMD window per script (runs from destination directory)
  - Unix: New terminal per script (runs from destination directory)
  - Status message shows script count and output location

### Advanced Features
- **Debug mode** - Enable via settings for detailed logging to `plexlinker_debug.log`
- **Event polling optimization** - 100ms timeout for responsive key handling
- **Confirmation modals** - Interactive dialogs with keyboard navigation
- **Script viewer** - Review generated scripts before use
- **TV Renamer** (separate module) - Bulk rename TV shows to Plex format

## Dependencies
- github.com/gdamore/tcell/v2 - Terminal UI library
- Standard Go libraries for HTTP, JSON, file I/O, exec

## Configuration File Format
```json
{
  "base_url": "https://example.com/media/",
  "username": "your_username",
  "password": "your_password",
  "script_format": "bat",
  "debug_mode": false,
  "movies_path": "W:\\movies",
  "tv_shows_path": "W:\\shows",
  "other_path": "W:\\other"
}
```

### Configuration Fields
- `base_url`: Web server base URL for directory browsing
- `username`: HTTP basic auth username
- `password`: HTTP basic auth password
- `script_format`: Output format - "bat", "sh", or "txt" (default: "bat")
- `debug_mode`: Enable debug logging (default: false)
- `movies_path`: Destination path for movie files (required, use "auto" for default)
- `tv_shows_path`: Root path for TV show folders (required, use "auto" for default)
- `other_path`: Destination path for uncategorized files (required, use "auto" for default)

## Development Notes
- **Primary target platform: Windows**
- Cross-platform compatible (Windows, macOS, Linux)
- Graceful error handling with meaningful messages
- Session state persistence between runs
- Cursor position memory for natural navigation flow
- Clean, well-documented Go code following best practices
- Non-blocking UI operations with background goroutines
- Safe goroutine management for async operations with cancellation support
- Centralized logging with namespace prefixes
- Platform-aware default paths via `internal/paths` package
- Designed for SSH into remote Windows machine → run script → disconnect while download continues

## Recent Improvements (2025-10-09)

### ✅ HIGH PRIORITY FIXES - COMPLETED

#### 1. **Non-Blocking Recursive Selection** ✅ FIXED
- **Issue:** Recursive directory selection froze UI for minutes on large trees
- **Fix:** Moved HTTP requests to background goroutine with progress updates
- **Location:** `internal/ui/browser.go:281-422`
- **Features Added:**
  - Real-time progress counter
  - ESC key cancellation support
  - Non-blocking UI during scan
  - Progress channel communication

#### 2. **Centralized Debug Logger** ✅ FIXED
- **Issue:** Debug logger initialized 3 times with duplicate code
- **Fix:** Created `internal/logger/logger.go` package
- **Locations Updated:**
  - `cmd/plexlinker/main.go` - Uses `logger.Get().Main()`
  - `internal/client/client.go` - Uses `logger.Get().Client()`
  - `internal/ui/browser.go` - Uses `logger.Get().UI()`
- **Benefits:**
  - Single initialization point
  - Thread-safe with sync.RWMutex
  - Namespace prefixes ([MAIN], [UI], [CLIENT])
  - Consistent behavior across packages

#### 3. **Path Constants Package** ✅ FIXED
- **Issue:** Hardcoded paths `W:\movies`, `/w/movies` in multiple locations
- **Fix:** Created `internal/paths/defaults.go` package
- **Functions:**
  - `GetDefaultMoviesPath()` - Platform-aware movies path
  - `GetDefaultTVShowsPath()` - Platform-aware TV shows path
  - `GetDefaultOtherPath()` - Platform-aware other files path
  - `ResolveAutoPath()` - Converts "auto" to platform defaults
- **Updated:** `internal/downloader/generator.go` uses path constants

#### 4. **Config Normalize Separation** ✅ FIXED
- **Issue:** `Validate()` mutated config (code smell)
- **Fix:** Separated concerns into two methods
- **Location:** `internal/config/config.go:44-90`
- **Methods:**
  - `Normalize()` - Mutates config (adds trailing slash, etc.)
  - `Validate()` - Pure validation, no mutation
- **Usage:** `LoadConfig()` automatically calls `Normalize()` after loading

#### 5. **Code Cleanup** ✅ COMPLETED
- Removed `internal/ui/timeout.go` (unused, 26 lines)
- Removed `internal/ui/utils.go` (empty file, 2 lines)
- Moved root test files to `test/archive/` directory
- Cleaned up duplicate imports

### ✅ Critical Issues - RESOLVED (Previously Fixed)
1. **Missing UI components** ✅ FIXED
   - All components already exist: `media_type_modal.go`, `show_selector.go`
   - `ParseShowNameFromURL()` implemented in show_selector.go:206-227
   - Script generation (key '5') now compiles successfully

2. **Missing stub file** ✅ FIXED
   - `window_default.go` exists with proper stub implementation
   - Updated build tag from `// +build !windows` to `//go:build !windows`
   - Cross-platform compilation works

### ✅ Medium Priority Issues - RESOLVED
3. **Build tag syntax** ✅ FIXED
   - Updated all files to use `//go:build` directive only
   - Removed deprecated `// +build` lines from:
     - `internal/ui/window_default.go`
     - `internal/ui/window_windows.go`
     - `internal/downloader/exec_windows.go`

4. **File Selection inconsistency** ✅ FIXED
   - Updated status bar text to match actual key bindings:
     - Key '1': SelectCurrent (select all in current directory)
     - Key '2': SelectAll(R) (recursive)
     - Key '3': Deselect (deselect all)
   - browser.go:700 now correctly shows "1:SelectCurrent"

6. **Signal handler goroutine leak** ✅ FIXED
   - Added `stdinDone` channel for clean goroutine shutdown
   - `monitorStdin()` now accepts done channel and exits cleanly
   - Prevents goroutine leak on exit

8. **Error handling in clipboard** ✅ FIXED
   - Now returns proper error when xclip/xsel not found on Linux
   - Error message: "clipboard utilities not found: please install xclip or xsel"
   - Added fmt import to clipboard.go

### ✅ Low Priority / Code Quality - RESOLVED
9-10. **Duplicate file extensions** ✅ FIXED
   - Deduplicated and alphabetically sorted video extensions (43 unique)
   - Deduplicated and alphabetically sorted audio extensions (41 unique)
   - Removed duplicates like .mp4, .mkv, .avi, .mov, .wmv, .m4p, .amr

17. **Command injection risk** ✅ FIXED
   - Added shell-escaping for `showName` in batch scripts (quotes doubled)
   - Added shell-escaping for `showName` in bash scripts (single quote escape)
   - generator.go:157 (batch) and generator.go:182 (bash)

## Remaining Known Issues

### Medium Priority (Not Critical)
5. **Selected files viewer naming** (Minor)
   - Status bar says "4:ShowSelected"
   - Documentation says "Show selected files viewer"
   - Not a bug, just naming inconsistency

7. **Recursive selection UI blocking** ✅ **FIXED (2025-10-09)**
   - ~~`selectRecursiveHelper()` performs blocking HTTP requests~~
   - ~~Could freeze UI for large directory trees~~
   - **Fixed:** Now uses background goroutine with progress updates
   - **Features:** Real-time progress, ESC cancellation, non-blocking UI

### Low Priority (Code Quality)
11. **Unused legacy function** (Non-breaking)
    - `GenerateScript()` exists but `GenerateSimpleScript()` is used
    - Both work, GenerateScript is legacy code path
    - Not causing issues, can be cleaned up later

12. **Debug logging initialization** ✅ **FIXED (2025-10-09)**
    - ~~Three separate debug log initializations~~
    - **Fixed:** Created centralized `internal/logger` package
    - All packages now use `logger.Get().Namespace()`

13. **Config validation mutation** ✅ **FIXED (2025-10-09)**
    - ~~`Validate()` adds trailing slash to BaseURL~~
    - **Fixed:** Separated into `Normalize()` and `Validate()` methods
    - `Normalize()` handles mutations, `Validate()` is pure

14. **Hardcoded paths** ✅ **FIXED (2025-10-09)**
    - ~~W:\movies, W:\shows hardcoded for Windows target~~
    - **Fixed:** Created `internal/paths` package with constants
    - Platform-aware path selection via `GetDefaultXPath()` functions

15. **Missing zero-file message** (Enhancement)
    - Shows "Selected 0 files" when no video/audio found
    - Works but could be clearer
    - **Future improvement:** "No video/audio files in this directory"

16. **Unused files cleanup** ✅ **FIXED (2025-10-09)**
    - **Removed:** `internal/ui/timeout.go` (unused, goroutine leak warning)
    - **Removed:** `internal/ui/utils.go` (empty file)
    - **Moved:** Root test files to `test/archive/` directory

### Security Considerations (Informational)
16. **Credentials in scripts** (By Design)
    - Username/password embedded in generated scripts
    - Visible in process list during aria2c execution
    - **Mitigation:** User should set file permissions (chmod 600)
    - **Alternative:** aria2c supports .netrc file for credentials

## Build Verification
- ✅ macOS build successful: `go build -o plexlinker ./cmd/plexlinker`
- ✅ Windows cross-compile successful: `GOOS=windows GOARCH=amd64 go build -o plexlinker-windows-amd64.exe ./cmd/plexlinker`
- ✅ All critical compilation errors resolved
- ✅ Ready for testing on target Windows environment
- always export this project windows file to plexlinker.exe, it keeps the file size small and presentable