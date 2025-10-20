# PlexLinker Debug Guide

## Environment Variables for Development

### Credentials (Secure - Never Written to Disk)

Set these in GoLand's Run Configuration to keep your credentials out of any config files:

- `PLEX_BASE_URL` - Your server base URL (e.g., `https://example.com/media/`)
- `PLEX_USERNAME` - Your username
- `PLEX_PASSWORD` - Your password
- `PLEX_OUTPUT_DIR` - Output directory for scripts (optional, defaults to `./downloads`)

### Debug Logging

- `PLEX_DEBUG=1` - Enable comprehensive debug logging

When debug mode is enabled:
- Logs are written to `plexlinker_debug.log` in the current directory
- Logs are also printed to stderr (visible in GoLand console)
- Logs include timestamps and file:line information

## Setting Up in GoLand

1. **Run → Edit Configurations**
2. **Environment variables** section, add:
   ```
   PLEX_BASE_URL=https://your-server.com/media/
   PLEX_USERNAME=your_username
   PLEX_PASSWORD=your_password
   PLEX_OUTPUT_DIR=/path/to/downloads
   PLEX_DEBUG=1
   ```

## Debug Log Output

The debug logs will show you:
- Application startup and initialization
- Configuration loading (with sanitized output - no passwords)
- State transitions (Menu → Browser → Settings, etc.)
- HTTP requests (URLs, status codes, response sizes)
- Directory loading and parsing
- Event handling (key presses, timeouts)
- Error conditions with full context

### Example Log Output:
```
[DEBUG] 2025/09/29 12:34:56 main.go:58: === PlexLinker Debug Mode Started ===
[DEBUG] 2025/09/29 12:34:56 main.go:73: Starting run()
[DEBUG] 2025/09/29 12:34:56 main.go:75: Loading configuration...
[DEBUG] 2025/09/29 12:34:56 main.go:80: Config loaded: BaseURL=https://example.com/, Username=user, OutputDir=./downloads
[CLIENT] 2025/09/29 12:34:57 client.go:42: FetchDirectory: Starting request to https://example.com/
[CLIENT] 2025/09/29 12:34:57 client.go:52: FetchDirectory: Sending HTTP request to https://example.com/
[CLIENT] 2025/09/29 12:34:58 client.go:61: FetchDirectory: Received response with status 200
```

## Common Hang Scenarios

With debug logging enabled, you can identify where the app hangs:

1. **Hangs during HTTP request**: Look for "Sending HTTP request" without "Received response"
   - Possible causes: Network timeout, firewall, DNS issues

2. **Hangs in event loop**: Look for repeated "Polling for event" messages
   - Possible causes: Screen not updating, terminal issues

3. **Hangs during directory parsing**: Look for "Parsing HTML" without "Successfully loaded"
   - Possible causes: Invalid HTML structure, parser bug

## Priority Order

The app loads configuration in this order:

1. **Environment variables** (highest priority)
2. **Config file** (`plexlinker.json`)

This means you can test different servers/credentials without touching the config file.