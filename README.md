# PlexLinker

PlexLinker is a Windows-first terminal application for browsing an authenticated
HTTPS directory and handing selected files to a persistent aria2 download
service. It is designed to be operated through SSH: downloads continue after
the SSH connection and TUI close.

PlexLinker v2.0.0 is the current production release. The original application
remains preserved by the `baseline/v1-2026-10-23` tag and legacy release binary.

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

## Download

Choose your platform and download the latest release directly in your browser:

| Platform | Recommended download |
|---|---|
| Windows 10/11 or Server (64-bit) | [Download Windows setup package](https://github.com/bbrandlegacy/PlexLinker/releases/latest/download/plexlinker-windows-amd64-setup.zip) |
| Linux on Intel/AMD (x86-64) | [Download Linux AMD64 setup package](https://github.com/bbrandlegacy/PlexLinker/releases/latest/download/plexlinker-linux-amd64-setup.tar.gz) |
| Linux on ARM64/AArch64 | [Download Linux ARM64 setup package](https://github.com/bbrandlegacy/PlexLinker/releases/latest/download/plexlinker-linux-arm64-setup.tar.gz) |
| macOS on Intel | [Download macOS Intel binary](https://github.com/bbrandlegacy/PlexLinker/releases/latest/download/plexlinker-darwin-amd64) |
| macOS on Apple Silicon | [Download macOS Apple Silicon binary](https://github.com/bbrandlegacy/PlexLinker/releases/latest/download/plexlinker-darwin-arm64) |

Most Windows computers use the Windows package. Most Linux PCs and servers use
AMD64; Raspberry Pi 4/5 and ARM servers generally use ARM64. Extract the setup
archive and run the included installer. All files and checksums are also
available on the [latest release page](https://github.com/bbrandlegacy/PlexLinker/releases/latest).

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

### One-shot setup

Download and run the setup package for your host.

Linux AMD64:

```bash
curl -fLO https://github.com/bbrandlegacy/PlexLinker/releases/latest/download/plexlinker-linux-amd64-setup.tar.gz
mkdir -p plexlinker-setup
tar -xzf plexlinker-linux-amd64-setup.tar.gz -C plexlinker-setup
cd plexlinker-setup
./setup-linux.sh
```

Linux ARM64 uses the same commands with `linux-amd64` replaced by
`linux-arm64`.

Windows PowerShell:

```powershell
curl.exe -fLO https://github.com/bbrandlegacy/PlexLinker/releases/latest/download/plexlinker-windows-amd64-setup.zip
Expand-Archive .\plexlinker-windows-amd64-setup.zip -DestinationPath .\plexlinker-setup -Force
.\plexlinker-setup\setup-windows.cmd
```

Keep the platform binary and setup files in the same extracted directory.

- Windows: double-click `setup-windows.cmd`. It explains the changes, asks for
  confirmation, and uses the normal UAC administrator prompt.
- Linux: run `./setup-linux.sh`. It explains the changes, asks for confirmation,
  validates sudo, installs the matching binary globally, enables user lingering,
  and invalidates sudo's temporary credential timestamp when finished.

PlexLinker also performs this preflight on first interactive launch and offers
to run the colocated installer. Neither installer reads or stores an
administrator password.

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
- `/` opens a live, case-insensitive filename filter. Press `Enter` to keep the
  filtered list or `Esc` to clear it.

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
- `r` refreshes mounted drives and free-space information.

External and network volumes are ranked ahead of the system disk. With a
default destination set to `auto`, PlexLinker looks for conventional Movies or
Shows folders on writable media volumes and creates the selected conventional
folder only when a download is submitted.

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

Settings accept the local terminal's normal paste shortcut over SSH. Saving
with `Ctrl+S` validates and applies changed HTTP credentials, paths, aria2 RPC
settings, and concurrency without restarting PlexLinker.

The final Settings row opens **Developer & maintenance**, where you can inspect
or clear the persistent message/error log, manually check for a release,
install a checksum-verified update, or restore the rollback binary. Update
checks are never automatic. The first successful update saves the currently
installed executable as the rollback version.

The diagnostics log is stored beside `config.json` as `plexlinker.log`.

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
artifacts, and performs native Windows and Linux installer, aria2 persistence,
and real-download acceptance tests.

## Documentation

- [v2 operator guide](docs/V2_GUIDE.md)
- [v2 architecture](docs/V2_ARCHITECTURE.md)
- [v2 roadmap and acceptance state](docs/V2_ROADMAP.md)
- [changelog](CHANGELOG.md)

## License

MIT. See [LICENSE](LICENSE).
