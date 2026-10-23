# PlexLinker

PlexLinker is a Windows-first terminal application for browsing an authenticated
HTTPS directory and handing selected files to a persistent aria2 download
service. It is designed to be operated through SSH: downloads continue after
the SSH connection and TUI close.

> v2 is under review on `v2-development`. `main` and the
> `baseline/v1-2026-10-23` tag preserve the original application until the
> owner completes target-server acceptance and explicitly approves the merge.

## What v2 provides

- Bubble Tea v2 TUI with keyboard-first Browser, Selection, Downloads, TV
  Renamer, and Settings views.
- Persistent aria2 JSON-RPC downloads with session recovery, live progress,
  pause/resume, and removal.
- Windows Task Scheduler management so aria2 survives SSH disconnects.
- Native Windows drive-letter and UNC browsing, including `W:`, `N:`, and
  saved destination favorites.
- Linux systemd-user persistence with daemon fallback, plus macOS development
  support.
- Automatic filename-based Movie/TV classification and configurable default
  library paths.
- One-time destination overrides for individual selected files.
- A built-in Windows acceptance doctor and a local authenticated demo server.

## Requirements

- aria2 1.37 or newer (`aria2c` or `aria2c.exe` on `PATH`)
- Windows 10/11 or Windows Server for the primary deployment
- Go 1.26.10 when building from source

Linux and macOS are supported. Windows is the primary production target.

## Build

Build v2 for the current machine:

```bash
go build -o plexlinker ./cmd/plexlinker-v2
```

Build all release targets:

```bash
make build-all
```

Artifacts are written to `dist/`. The release workflow builds Windows AMD64,
Linux AMD64/ARM64, and macOS AMD64/ARM64 binaries.

The preserved v1 application can still be built explicitly:

```bash
go build -o plexlinker-v1 ./cmd/plexlinker
```

## First run

1. Put the PlexLinker v2 binary and aria2 on `PATH`.
2. Run PlexLinker and press `5` for Settings.
3. Configure the remote HTTPS URL and Basic Auth credentials.
4. Configure the default Movies and TV destinations. Press `Enter` on either
   path to browse available drives, mounts, and directories.
5. Press `Ctrl+S`, exit, and restart PlexLinker so connection changes take
   effect.

PlexLinker starts or reconnects to aria2 before opening the TUI. On Windows,
it creates a limited-privilege `PlexLinker_aria2` scheduled task. RPC listens
on loopback and uses a generated secret stored in the per-user config.

## Keyboard workflow

| Key | View/action |
|---|---|
| `1` | Browser |
| `2` | Selection |
| `3` | Downloads |
| `4` | TV Renamer |
| `5` | Settings |
| `Tab` / `Shift+Tab` | Cycle stateful views |
| `q` / `Ctrl+C` | Quit |

### Browser

- `Up`/`Down` selects a row.
- `Right` or `Enter` opens a folder; `Left` returns to its parent.
- `Enter` or `Space` toggles a file selection.
- `a` toggles all files in the current directory: select all, then deselect all.
- `s` recursively selects files below the current directory.

### Selection

Each row is presented as `filename → destination`.

- `Up`/`Down` selects a file.
- `Left`/`Right` selects the filename or destination column.
- `Enter` on the filename removes that selection.
- `Enter` on the destination opens the directory picker for a one-time change.
- `r` restores the highlighted file's inferred default destination.
- `c` clears all selections.
- `d` validates every destination and starts the downloads.
- `Shift+D` bypasses the write preflight and lets aria2 attempt the download.

Names containing markers such as `S01`, `S01E01`, `Season 1`, `1x01`, or
`Episode 1` default to TV. Other filenames default to Movies.

### Directory picker

- `Up`/`Down` selects a drive or folder.
- `Right` or `Enter` opens it; `Left` returns to its parent.
- `Esc` accepts the current folder and returns to Selection or Settings.
- `g` enters a path directly, including `\\server\share\folder`.
- `n` creates and enters a directory.
- `f` saves the current path as a favorite.

### Downloads

- `p` or `Enter` pauses/resumes the selected job.
- `t` retries a failed job using its retained source URL and destination.
- `x` removes the selected job and its aria2 result.
- `r` refreshes the list.

## Configuration

The v2 config is stored per user:

- Windows: `%AppData%\PlexLinker\config.json`
- macOS: `~/Library/Application Support/PlexLinker/config.json`
- Linux: `${XDG_CONFIG_HOME:-~/.config}/PlexLinker/config.json`

Set `PLEXLINKER_CONFIG` to select a custom config file. An existing v1 config
beside the executable is copied on first use and is never modified in place.

Supported environment overrides are `PLEX_BASE_URL`, `PLEX_USERNAME`,
`PLEX_PASSWORD`, `PLEXLINKER_ARIA2_ENDPOINT`, and
`PLEXLINKER_ARIA2_SECRET`.

## Windows and SSH acceptance

Run the doctor as the same Windows account used over SSH:

```powershell
plexlinker.exe doctor --destination 'W:\Movies'
```

If mapped drives are not visible to OpenSSH or Task Scheduler, use a UNC path:

```powershell
plexlinker.exe doctor --destination '\\server\share\Movies'
```

Then start a small download, disconnect SSH, reconnect later, and confirm the
same job in Downloads. That final test validates the real VPN account, storage
mapping, and server policy that CI cannot reproduce.

aria2 persistence does not require a terminal multiplexer. Use psmux on Windows
or tmux on Linux only when the exact TUI screen should remain attached.

## Local development without production links

Start the authenticated fixture server:

```bash
go run ./cmd/plexlinker-demo-server
```

In another terminal:

```bash
PLEX_BASE_URL=http://127.0.0.1:8787/ \
PLEX_USERNAME=plex \
PLEX_PASSWORD=demo \
PLEXLINKER_ARIA2_SECRET=demo-rpc-secret \
go run ./cmd/plexlinker-v2
```

The fixture exposes nested Movies and TV folders with tiny deterministic files.

## Verification

```bash
go test ./...
go test -race ./...
go vet ./...
```

The CI matrix tests current and pinned Go on Windows, Linux, and macOS, runs
golangci-lint, CodeQL, dependency review, govulncheck, cross-builds release
artifacts, and performs a native Windows aria2 persistence acceptance test.

## Documentation

- [v2 operator guide](docs/V2_GUIDE.md)
- [v2 architecture](docs/V2_ARCHITECTURE.md)
- [v2 roadmap and acceptance state](docs/V2_ROADMAP.md)
- [changelog](CHANGELOG.md)

## License

MIT. See [LICENSE](LICENSE).
