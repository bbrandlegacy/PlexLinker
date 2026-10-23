# PlexLinker v2 architecture

The TUI is deliberately disposable. Durable state belongs to aria2.

```text
Bubble Tea TUI
  ├── remote HTTPS browser ── Basic Auth ──> directory server
  ├── destination browser ────────────────> local drives/mounts/UNC
  └── download service ── JSON-RPC ───────> persistent aria2
                                                │
                                                ├── session file
                                                └── destination storage
```

## Boundaries

- `internal/client`: context-aware, root-bound HTTPS directory transport.
- `internal/browser`: DOM-based directory listing parser and media filtering.
- `internal/storage`: platform-neutral browsing with native volume providers.
- `internal/aria2`: typed JSON-RPC transport.
- `internal/aria2manager`: config/session generation and platform persistence.
- `internal/downloads`: queue operations and domain-state conversion.
- `internal/v2tui`: event-driven presentation and user interaction.
- `cmd/plexlinker-v2`: dependency composition only.

Platform code is restricted to volume enumeration and aria2 startup. The
remote browser, queue, configuration, and TUI execute unchanged on Windows,
Linux, and macOS.

## Persistence

- Windows registers and runs a limited-privilege Task Scheduler task.
- Linux prefers a systemd user service and falls back to aria2 daemon mode.
- macOS uses aria2 daemon mode for development and supported local use.
- psmux/tmux are optional and only preserve the TUI session.

aria2 writes active work to `aria2.session` every 30 seconds and loads that
file when restarted. PlexLinker calls `aria2.saveSession` after successful job
submission as an additional durability boundary.

## Compatibility

The v1 tcell application remains under `cmd/plexlinker` and is built as a
legacy rollback artifact. The v2 release workflow builds `cmd/plexlinker-v2`
under the canonical PlexLinker filenames. No merge to `main` is performed
without explicit owner approval.
