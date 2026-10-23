# Changelog

All notable changes follow [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added

- Bubble Tea v2 interface for remote browsing, selection, destinations, settings, and live downloads.
- Persistent aria2 JSON-RPC queue with session recovery and lifecycle controls.
- Native Windows drive discovery, UNC paths, destination favorites, and directory creation.
- Windows Task Scheduler and Linux systemd user-service integration.
- macOS/Linux volume discovery and aria2 daemon support.
- Authenticated local demo directory server and real aria2 integration test.
- TV renamer handoff from the v2 interface, preserving the existing preview,
  execute, and undo workflow.
- Cross-platform v2 builds, CodeQL, dependency review, Dependabot, and vulnerability scanning.
- Built-in `doctor` command for Windows SSH, Task Scheduler, drive/UNC,
  destination write, aria2 RPC, and remote directory acceptance checks.

### Changed

- Replaced regex-only HTML listing parsing with a root-bound DOM parser.
- Moved v2 configuration to the per-user operating-system config directory with atomic writes.
- Upgraded the project to Go 1.26.10 and current Bubble Tea/Bubbles/Lip Gloss releases.

### Preserved

- The original tcell application remains available as the v1 legacy rollback artifact.

[Unreleased]: https://github.com/bbrandlegacy/PlexLinker/compare/baseline/v1-2026-10-23...HEAD
