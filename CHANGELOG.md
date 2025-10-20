# Changelog

All notable changes to PlexLinker will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial public release
- Interactive HTML directory browser with keyboard navigation
- Smart TV show vs movie auto-categorization
- Sequential TV show episode downloads
- Windows Task Scheduler integration for SSH-persistent downloads
- Multi-user support with username selection
- Cross-platform support (Windows, macOS, Linux)
- Debug mode with detailed logging
- Script generation preview with execution options
- "Generate Only" mode (key 6) - create scripts without execution
- Master launcher batch file for cleaner script management
- Centralized logger with namespace support
- Platform-aware default paths

### Features
- **Directory Browser**
  - Navigate with arrow keys or hjkl
  - Cursor position memory when navigating back
  - Smart directory sorting by modification date
  - Real-time selected file counter
  - Non-blocking recursive selection with progress

- **File Selection**
  - Individual file selection
  - Select all in current directory (key 1)
  - Recursive select all (key 2)
  - Deselect all (key 3)
  - Selected files viewer (key 4)

- **Script Generation**
  - Auto-categorization (TV shows, movies, other)
  - Sequential TV show episode ordering
  - Multi-script generation (separate scripts per category)
  - TV show folder browser with inline creation
  - Preview before generation
  - Key 5: Generate & Run (via Task Scheduler)
  - Key 6: Generate Only (no execution)

- **Task Scheduler Integration (Windows)**
  - SSH-persistent downloads
  - Automatic task cleanup after completion
  - Visible PowerShell windows with progress
  - Fallback to direct execution if Task Scheduler fails
  - Debug logging (only when debug mode enabled)

### Technical
- Thread-safe centralized logger
- Platform-specific build tags (`//go:build`)
- Safe goroutine management with cancellation
- HTTP basic authentication support
- JSON configuration with validation
- Shell-escaping for security

### Documentation
- Comprehensive README with use cases
- Contributing guidelines
- MIT License
- CLAUDE.md for development context

## [1.0.0] - 2025-01-XX (Planned First Release)

Initial stable release

### Planned Improvements
- GitHub Actions CI/CD pipeline
- Pre-built binaries for all platforms
- Docker support
- Configuration wizard for first-time setup
- Script template customization
- Download progress monitoring
- Retry logic for failed downloads

---

## Version History Format

### [Version] - YYYY-MM-DD

#### Added
- New features

#### Changed
- Changes to existing functionality

#### Deprecated
- Features that will be removed in upcoming releases

#### Removed
- Features removed in this release

#### Fixed
- Bug fixes

#### Security
- Security-related changes

---

**Note:** This project is currently in active development. Version 1.0.0 will be the first stable release.
