# Changelog

All notable changes follow [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added

- One-shot Windows and Linux installers with explicit first-run consent,
  dependency installation, native privilege validation, global command setup,
  and Linux SSH/reboot persistence configuration.
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
- Native Windows acceptance automation that installs aria2, exercises Task
  Scheduler persistence, reconnects through RPC, and verifies a real download.
- Exact release-version reporting through `plexlinker --version`.

### Changed

- Switch directly to Downloads after at least one selected item is accepted by
  aria2.
- Return from Settings to the view that opened it instead of always returning
  to Browser.
- Made `a` toggle all files in the current browser directory and added an
  explicit `Shift+D` override for destination write-preflight failures.
- Reworked v2 navigation around Browser, Selection, Downloads, TV Renamer, and
  Settings, with arrow-key folder traversal and view-specific controls.
- Added filename-based Movie/TV inference, configurable default library paths,
  and one-time per-file destination browsing/reset directly in Selection.
- Replaced regex-only HTML listing parsing with a root-bound DOM parser.
- Moved v2 configuration to the per-user operating-system config directory with atomic writes.
- Upgraded the project to Go 1.26.10 and current Bubble Tea/Bubbles/Lip Gloss releases.
- Updated the retained tcell compatibility layer to 2.13.10.
- Restricted releases to exact semantic-version tags whose commits are already
  contained in `main`, with tests and metadata verification before publishing.

### Fixed

- Restart an unresponsive Linux systemd-user aria2 service and add native Linux
  persistence/download acceptance coverage.
- Bound and serialize aria2 status polling, and restart an unresponsive
  Task Scheduler-managed aria2 process during Windows startup recovery.
- Canonicalized doubled Windows drive and UNC separators from migrated or
  manually entered destinations before displaying or submitting downloads.
- Expanded case-insensitive video and audio discovery and recover full media
  filenames from link targets when directory indexes use shortened labels.
- Added failed-job retry and correct aria2 result removal for completed or
  failed downloads.
- Made direct destination entry start with a clean field so absolute paths,
  Windows drive letters, and UNC paths do not append to the current directory.
- Made directory creation prefill a platform-correct child path.
- Wait for aria2 to release Windows file handles during managed shutdown.

### Preserved

- The original tcell application remains available as the v1 legacy rollback artifact.

[Unreleased]: https://github.com/bbrandlegacy/PlexLinker/compare/baseline/v1-2026-10-23...HEAD
