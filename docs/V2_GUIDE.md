# PlexLinker v2 guide

PlexLinker v2 is a Windows-first, cross-platform terminal application. The TUI
is a controller; a persistent aria2 process owns downloads, so leaving the TUI
or losing SSH does not stop queued work.

## Requirements

- aria2 1.37 or newer available as `aria2c`/`aria2c.exe`
- Windows 10/11 or Windows Server for the primary deployment
- Linux and macOS are supported for development and normal use

The release binary is self-contained apart from aria2. Go 1.26 is only needed
when building from source.

## First run

1. Put `aria2c.exe` and `plexlinker-windows-amd64.exe` on `PATH`.
2. Run PlexLinker and press `5` for Settings.
3. Enter the HTTPS directory URL, HTTP username, and HTTP password.
4. Keep the local aria2 endpoint unless an existing RPC service is preferred.
5. Press `Ctrl+S`, exit, and start PlexLinker again.

On Windows, PlexLinker registers `PlexLinker_aria2` in Task Scheduler and starts
it immediately. Downloads run in that task and survive the SSH client. The RPC
listener binds only to loopback. Its secret and session are generated under the
current user's config directory.

## Normal workflow

- `1`: browse the remote HTTPS directory.
- `Enter`/`Space`: open a directory or toggle a file.
- `a`: select all media files in the current directory.
- `s`: recursively scan and select media below the current directory.
- `3`: choose a destination drive, mount, favorite, or UNC path.
- `g`: type a destination directly, including `\\server\share\folder`.
- `n`: create and enter a destination directory.
- `f`: save the current destination as a favorite.
- `d`: validate the destination and submit selected files to aria2.
- `4`: monitor jobs; `p` pauses/resumes and `x` removes the highlighted job.
- `6`: temporarily hand the terminal to the TV renamer and return when done.

Drive letters mapped in an interactive Windows login are not always visible to
OpenSSH or Task Scheduler. If `W:` or `N:` is absent, use its UNC path and save
that path as a favorite.

## Optional persistent TUI

aria2 persistence does not require a terminal multiplexer. A multiplexer is
useful only when the exact TUI screen should remain attached.

Windows with psmux:

```powershell
winget install marlocarlo.psmux
psmux new-session -s plexlinker plexlinker-windows-amd64.exe
psmux attach -t plexlinker
```

Linux with tmux:

```bash
tmux new-session -s plexlinker ./plexlinker-linux-amd64
tmux attach -t plexlinker
```

## Local macOS development without production links

Terminal one:

```bash
go run ./cmd/plexlinker-demo-server
```

Terminal two:

```bash
PLEX_BASE_URL=http://127.0.0.1:8787/ \
PLEX_USERNAME=plex \
PLEX_PASSWORD=demo \
PLEXLINKER_ARIA2_SECRET=demo-rpc-secret \
go run ./cmd/plexlinker-v2
```

The demo exposes nested shows and movies protected by Basic Auth. Downloaded
files are tiny deterministic fixtures.

## Configuration

The v2 config is stored in the operating system's per-user config directory:

- Windows: `%AppData%\PlexLinker\config.json`
- macOS: `~/Library/Application Support/PlexLinker/config.json`
- Linux: `${XDG_CONFIG_HOME:-~/.config}/PlexLinker/config.json`

Set `PLEXLINKER_CONFIG` to use a portable custom path. On first run, an existing
v1 config beside the executable is copied into the v2 location; the original is
not modified.

Environment overrides for unattended use are `PLEX_BASE_URL`, `PLEX_USERNAME`,
`PLEX_PASSWORD`, `PLEXLINKER_ARIA2_ENDPOINT`, and
`PLEXLINKER_ARIA2_SECRET`.

## Linux persistence

When a systemd user session is available, PlexLinker installs and enables
`plexlinker-aria2.service`. Otherwise, it starts aria2's native daemon mode.
For service survival without any logged-in user, enable lingering once:

```bash
loginctl enable-linger "$USER"
```

## Verification

```bash
go test ./...
go test -race ./...
go vet ./...
PLEXLINKER_INTEGRATION=1 go test -v ./internal/aria2manager \
  -run TestManagedAria2DownloadIntegration -count=1
```
