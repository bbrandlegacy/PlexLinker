# PlexLinker

> **v2 development is active on `v2-development`.** The current `main` branch
> remains the preserved v1 rollback line until Windows acceptance and explicit
> merge approval. See [the v2 guide](docs/V2_GUIDE.md) and
> [architecture](docs/V2_ARCHITECTURE.md).

> **Interactive HTML Directory Browser & Batch Download Script Generator**

PlexLinker is a powerful Go CLI tool that browses authenticated web directories, intelligently categorizes media files, and generates aria2c download scripts with automatic Windows Task Scheduler integration for SSH-persistent downloads.

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20macOS%20%7C%20Linux-lightgrey.svg)](https://github.com/bbrandlegacy/PlexLinker)

---

## ✨ Features

### 🎯 Core Functionality
- **Interactive Directory Browser** - Navigate authenticated web servers with keyboard controls
- **Smart Media Categorization** - Automatically detects TV shows vs movies based on filename patterns
- **Sequential TV Show Downloads** - Downloads episodes in order (S01E01, S01E02, etc.)
- **Multi-Script Generation** - Creates separate scripts for movies, TV shows, and other files
- **SSH-Persistent Execution** - Windows Task Scheduler integration keeps downloads running after disconnect

### 🎨 User Interface
- **Terminal-based UI** - Clean, responsive interface with tcell
- **Keyboard Navigation** - Arrow keys (hjkl support) for efficient browsing
- **Visual Indicators** - Color-coded file types and selection status
- **Cursor Memory** - Returns to exact position when navigating back
- **Real-time Progress** - Selected file counter and status updates

### ⚙️ Advanced Features
- **HTTP Basic Auth** - Secure authentication for web servers
- **Configuration Management** - JSON-based settings with validation
- **Debug Mode** - Detailed logging for troubleshooting
- **Cross-Platform** - Windows (primary), macOS, and Linux support
- **Script Viewer** - Review generated scripts before execution
- **TV Show Folder Browser** - Create and select destination folders interactively

---

## 📦 Installation

### Prerequisites
- **Go 1.21+** (for building from source)
- **aria2c** - Required for downloads ([Installation Guide](https://aria2.github.io/))

### Download Pre-built Binary

**Windows:**
```powershell
# Download from releases page
curl -L https://github.com/bbrandlegacy/PlexLinker/releases/latest/download/plexlinker.exe -o plexlinker.exe
```

**macOS/Linux:**
```bash
# Download from releases page
curl -L https://github.com/bbrandlegacy/PlexLinker/releases/latest/download/plexlinker -o plexlinker
chmod +x plexlinker
```

### Build from Source

```bash
# Clone repository
git clone https://github.com/bbrandlegacy/PlexLinker.git
cd PlexLinker

# Build for current platform
go build -o plexlinker ./cmd/plexlinker

# Build for Windows (cross-compile)
GOOS=windows GOARCH=amd64 go build -o plexlinker.exe ./cmd/plexlinker
```

---

## 🚀 Quick Start

1. **First Run** - Configure settings:
```bash
./plexlinker
# Select "Settings" → Configure base URL, username, password, and paths
```

2. **Browse & Select Files**:
   - Navigate directories with arrow keys or `hjkl`
   - Press `Enter` to select files or enter directories
   - Press `2` for recursive select all
   - Press `4` to view selected files

3. **Generate & Run Scripts**:
   - Press `5` to open script generation preview
   - Review auto-categorization (TV shows vs movies)
   - Press `b` to browse TV show destination folders
   - Press `5` to generate and execute (via Task Scheduler)
   - Press `6` to generate only (no execution)

---

## 📖 Usage Guide

### Configuration

PlexLinker stores settings in `plexlinker.json`:

```json
{
  "base_url": "https://example.com/media/",
  "username": "your_username",
  "password": "your_password",
  "script_format": "bat",
  "debug_mode": false,
  "movies_path": "W:\\movies",
  "tv_shows_path": "W:\\shows",
  "other_path": "W:\\other",
  "usernames": ["john", "jane"],
  "last_user": "john"
}
```

**Path Options:**
- Use `"auto"` for platform defaults (`W:\movies` on Windows, `/w/movies` on Unix)
- Specify custom absolute paths for each category

### Keyboard Controls

#### Directory Browser
| Key | Action |
|-----|--------|
| `↑↓` or `jk` | Navigate up/down |
| `Enter` | Select file / Enter directory |
| `Backspace` or `h` | Go up one directory |
| `Esc` or `0` | Exit to main menu |
| `1` | Select all in current directory |
| `2` | Recursive select all (scans subdirectories) |
| `3` | Deselect all |
| `4` | Show selected files viewer |
| `5` | Open script generation preview |
| `Tab` | Switch to running script window (Windows only) |

#### Script Generation Preview
| Key | Action |
|-----|--------|
| `5` or `Enter` | Generate & Run (Task Scheduler) |
| `6` | Generate Only (save scripts, no execution) |
| `b` | Browse TV show destination folders |
| `Esc` | Cancel and return |

### Auto-Categorization

PlexLinker automatically detects TV shows using these patterns:
- `s01`, `s1e1`, `s01e01` - Season/episode indicators
- `1x01`, `2x05` - Alternative format
- `season 1`, `episode 5` - Verbose format

**Examples:**
```
Breaking.Bad.S01E01.mkv          → TV Shows
The.Office.2x03.720p.mkv         → TV Shows
Inception.2010.1080p.mkv         → Movies
Documentary.Special.mkv          → Movies
```

### Script Execution Modes

#### Task Scheduler Mode (Windows, Recommended)
- ✅ **SSH-safe** - Downloads continue after disconnect
- ✅ PowerShell window opens and stays visible
- ✅ Automatic task cleanup after completion
- ✅ Multiple concurrent downloads

#### Direct Execution Mode (Fallback)
- ⚠️ **SSH-unsafe** - Downloads stop if SSH disconnects
- Used automatically if Task Scheduler fails
- Warning displayed in status

#### Generate Only Mode (Key 6)
- 📁 Scripts saved to `scripts/` folder
- No automatic execution
- Manual script execution anytime

---

## 🎬 Use Cases

### Primary: Personal Media Server Management
**Scenario:** Downloading media to a Plex/Jellyfin server via SSH
- SSH into remote Windows server
- Browse and select files
- Generate scripts with Task Scheduler
- Disconnect SSH - downloads continue
- Organizes by media type automatically

### Other Potential Use Cases

#### 1. **Web Scraping & Archival**
- Archive entire authenticated web directories
- Preserve directory structure
- Sequential downloads for rate-limited sites
- Resume support via aria2c

#### 2. **Scientific Data Collection**
- Download datasets from authenticated repositories
- Organize by data type automatically
- Batch processing of large file sets
- SSH-persistent for long-running downloads

#### 3. **Corporate File Distribution**
- Browse internal file servers
- Generate download scripts for teams
- Category-based organization
- Audit trail via debug logs

#### 4. **Backup & Sync Operations**
- Selective backup of web-hosted files
- Mirror directory structures locally
- Incremental downloads (aria2c conditional get)
- Automated via generated scripts

#### 5. **Educational Content Management**
- Download course materials from authenticated portals
- Organize by subject/semester automatically
- Share download scripts with students
- Batch operations for multiple courses

#### 6. **Research Data Management**
- Access authenticated research databases
- Categorize publications, datasets, etc.
- Sequential download for quota compliance
- Cross-platform support for lab environments

---

## 🏗️ Architecture

```
plexlinker/
├── cmd/plexlinker/          # Application entry point
│   └── main.go              # State machine, signal handling
├── internal/
│   ├── browser/             # HTML parsing, media filtering
│   ├── client/              # HTTP client with auth
│   ├── config/              # JSON config management
│   ├── downloader/          # Script generation & execution
│   │   ├── categorizer.go   # TV show detection & sorting
│   │   ├── generator.go     # Multi-script generation
│   │   ├── exec_windows.go  # Task Scheduler integration
│   │   └── exec_default.go  # Unix execution
│   ├── logger/              # Centralized debug logging
│   ├── paths/               # Platform-aware defaults
│   └── ui/                  # Terminal UI components
│       ├── browser.go       # Directory browser UI
│       ├── menu.go          # Main menu
│       ├── settings.go      # Settings editor
│       ├── script_generator.go  # Script preview UI
│       └── ...
└── test/
    └── fixtures/            # Test data
```

---

## 🛠️ Development

### Build Commands
```bash
# Build for current platform
go build -o plexlinker ./cmd/plexlinker

# Build for Windows
GOOS=windows GOARCH=amd64 go build -o plexlinker.exe ./cmd/plexlinker

# Run tests
go test ./...

# Run linters
go vet ./...
go fmt ./...
```

### Debug Mode
Enable detailed logging in Settings → Debug Mode:
```bash
# Logs written to plexlinker_debug.log
tail -f plexlinker_debug.log
```

Logs include:
- HTTP requests/responses
- Task Scheduler commands
- UI state transitions
- File selection operations

---

## 🤝 Contributing

Contributions welcome! Please:
1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Code Style
- Follow Go conventions (`gofmt`, `go vet`)
- Add tests for new features
- Update documentation

---

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## 🙏 Acknowledgments

- **[tcell](https://github.com/gdamore/tcell)** - Terminal UI library
- **[aria2](https://aria2.github.io/)** - Download utility
- Inspired by the need for SSH-persistent downloads on remote Windows servers

---

## 🐛 Troubleshooting

### Task Scheduler Not Working
- Enable debug mode to see Task Scheduler logs
- Check Task Scheduler service is running: `sc query schedule`
- Verify `plexlinker_launcher.bat` exists in executable directory

### Downloads Not Sequential
- TV shows are sorted automatically by season/episode
- Verify filenames contain season indicators (`s01e01`, `1x01`, etc.)
- Check categorization in script generation preview

### PowerShell Window Not Opening
- Verify Task Scheduler created task successfully
- Check Windows execution policy: `Get-ExecutionPolicy`
- Review debug logs for Task Scheduler errors

### SSH Disconnect Stops Downloads
- This means fallback mode was used (check status message)
- Task Scheduler service may be disabled
- Enable debug mode to diagnose Task Scheduler failures

---

## 📮 Support

- **Issues:** [GitHub Issues](https://github.com/bbrandlegacy/PlexLinker/issues)
- **Discussions:** [GitHub Discussions](https://github.com/bbrandlegacy/PlexLinker/discussions)
- **Documentation:** [Wiki](https://github.com/bbrandlegacy/PlexLinker/wiki)

---

**Made with ❤️ for the media server community**
