# PlexLinker v2 roadmap

Status: v2.0.0 implementation and native Windows/Linux acceptance are complete.
The original v1 line remains preserved by `baseline/v1-2026-10-23` and the
legacy release artifact.

## Stage 0 — Preserve and stabilize

- Pin the v1 baseline with `baseline/v1-2026-10-23`.
- Align the Go toolchain, CI, release, formatting, linting, and tests.
- Add dependency review, CodeQL, Dependabot, and release concurrency guards.
- Capture the v2 architecture and compatibility boundaries.

## Stage 1 — Durable download engine

- Implement a typed aria2 JSON-RPC client.
- Add queue, active, waiting, stopped, pause, resume, retry, and remove flows.
- Persist aria2 sessions and reconnect after PlexLinker restarts.
- Retain the script backend until Windows acceptance testing passes.

## Stage 2 — Filesystems and destinations

- Introduce a platform-neutral volume and directory API.
- Support Windows drive letters, UNC paths, labels, free space, and favorites.
- Support `/Volumes` on macOS and mount points on Linux.
- Add fake providers for deterministic macOS development and CI.

## Stage 3 — TUI v2

- Move to Bubble Tea v2, Bubbles v2, and Lip Gloss v2.
- Provide remote browser, selection basket, destination picker, and downloads.
- Make network and filesystem work cancellable and event-driven.
- Add headless model tests and a local demo directory server.

Initial implementation uses Bubble Tea 2.1, Bubbles 2.2, and Lip Gloss 2.0,
which require Go 1.26. The legacy tcell executable remains buildable during the
migration and is replaced only after Windows acceptance testing.

## Stage 4 — Service integration

- Windows: Task Scheduler-managed aria2, with optional psmux convenience.
- Linux: systemd user service, with optional tmux convenience.
- macOS: development subprocess and optional launchd/Homebrew service support.

## Stage 5 — Migration and acceptance

- Migrate configuration without destroying the v1 file.
- Cross-build all supported targets and run Windows CI.
- Test queue recovery and SSH disconnect behavior on the target Windows host.
- Prepare release notes and a merge candidate; do not merge without approval.

Automated native-Windows acceptance covers Task Scheduler registration, RPC
reconnection, a real aria2 transfer, storage writes, and authenticated remote
browsing. The target-host test remains intentionally separate because GitHub
Actions cannot reproduce that server's SSH session, VPN, or storage mappings.
