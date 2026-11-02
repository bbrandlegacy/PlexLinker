// Package updater checks and installs signed-by-checksum PlexLinker GitHub releases.
package updater

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const defaultRepository = "bbrandlegacy/PlexLinker"

type Release struct {
	Version string
	URL     string
	Notes   string
	Asset   string
	Newer   bool
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Body    string `json:"body"`
	Assets  []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

type Manager struct {
	repository string
	version    string
	client     *http.Client
	dataDir    string
}

func New(version, dataDir string) *Manager {
	return &Manager{repository: defaultRepository, version: version, dataDir: dataDir, client: &http.Client{Timeout: 30 * time.Second}}
}

func (m *Manager) Check(ctx context.Context) (Release, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/"+m.repository+"/releases/latest", nil)
	if err != nil {
		return Release{}, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "PlexLinker/"+m.version)
	response, err := m.client.Do(request)
	if err != nil {
		return Release{}, fmt.Errorf("check GitHub release: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("check GitHub release: HTTP %s", response.Status)
	}
	var latest githubRelease
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&latest); err != nil {
		return Release{}, fmt.Errorf("decode GitHub release: %w", err)
	}
	asset, err := platformAssetName()
	if err != nil {
		return Release{}, err
	}
	return Release{Version: latest.TagName, URL: latest.HTMLURL, Notes: latest.Body, Asset: asset, Newer: compareVersions(latest.TagName, m.version) > 0}, nil
}

func (m *Manager) InstallLatest(ctx context.Context) (Release, error) {
	release, err := m.Check(ctx)
	if err != nil {
		return release, err
	}
	if !release.Newer {
		return release, fmt.Errorf("%s is already the latest release", m.version)
	}
	metadata, err := m.fetchRelease(ctx)
	if err != nil {
		return release, err
	}
	assetURL, checksumURL := "", ""
	for _, asset := range metadata.Assets {
		switch asset.Name {
		case release.Asset:
			assetURL = asset.URL
		case "checksums.txt":
			checksumURL = asset.URL
		}
	}
	if assetURL == "" || checksumURL == "" {
		return release, fmt.Errorf("release %s is missing %s or checksums.txt", release.Version, release.Asset)
	}
	checksums, err := m.download(ctx, checksumURL, 2<<20)
	if err != nil {
		return release, err
	}
	expected, err := checksumFor(checksums, release.Asset)
	if err != nil {
		return release, err
	}
	binary, err := m.download(ctx, assetURL, 256<<20)
	if err != nil {
		return release, err
	}
	actual := sha256.Sum256(binary)
	if hex.EncodeToString(actual[:]) != expected {
		return release, fmt.Errorf("checksum mismatch for %s", release.Asset)
	}
	if err := os.MkdirAll(filepath.Join(m.dataDir, "updates"), 0o700); err != nil {
		return release, err
	}
	staged := filepath.Join(m.dataDir, "updates", "plexlinker-update"+executableSuffix())
	if err := os.WriteFile(staged, binary, 0o700); err != nil {
		return release, fmt.Errorf("stage update: %w", err)
	}
	if err := installStaged(staged, m.RollbackPath()); err != nil {
		return release, err
	}
	return release, nil
}

func (m *Manager) RollbackPath() string {
	return filepath.Join(m.dataDir, "updates", "plexlinker-rollback"+executableSuffix())
}

func (m *Manager) RollbackAvailable() bool {
	info, err := os.Stat(m.RollbackPath())
	return err == nil && info.Mode().IsRegular()
}

func (m *Manager) Rollback() error { return installRollback(m.RollbackPath()) }

func (m *Manager) fetchRelease(ctx context.Context) (githubRelease, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/"+m.repository+"/releases/latest", nil)
	if err != nil {
		return githubRelease{}, err
	}
	request.Header.Set("User-Agent", "PlexLinker/"+m.version)
	response, err := m.client.Do(request)
	if err != nil {
		return githubRelease{}, err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return githubRelease{}, fmt.Errorf("load release assets: HTTP %s", response.Status)
	}
	var release githubRelease
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&release); err != nil {
		return githubRelease{}, fmt.Errorf("decode release assets: %w", err)
	}
	return release, nil
}

func (m *Manager) download(ctx context.Context, url string, limit int64) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "PlexLinker/"+m.version)
	response, err := m.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("download update: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download update: HTTP %s", response.Status)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("download exceeded %d bytes", limit)
	}
	return data, nil
}

func platformAssetName() (string, error) {
	if runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		return "", fmt.Errorf("updates are unavailable for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	switch runtime.GOOS {
	case "windows":
		if runtime.GOARCH != "amd64" {
			return "", fmt.Errorf("updates are unavailable for windows/%s", runtime.GOARCH)
		}
		return "plexlinker-windows-amd64.exe", nil
	case "linux", "darwin":
		return "plexlinker-" + runtime.GOOS + "-" + runtime.GOARCH, nil
	default:
		return "", fmt.Errorf("updates are unavailable for %s", runtime.GOOS)
	}
}

func checksumFor(data []byte, name string) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && strings.TrimPrefix(fields[len(fields)-1], "*") == name {
			if _, err := hex.DecodeString(fields[0]); err == nil && len(fields[0]) == 64 {
				return strings.ToLower(fields[0]), nil
			}
		}
	}
	return "", fmt.Errorf("checksums.txt has no valid checksum for %s", name)
}

func compareVersions(a, b string) int {
	parse := func(value string) [3]int {
		value = strings.TrimPrefix(strings.TrimSpace(value), "v")
		value = strings.SplitN(value, "-", 2)[0]
		parts := strings.Split(value, ".")
		var result [3]int
		for i := 0; i < len(parts) && i < 3; i++ {
			result[i], _ = strconv.Atoi(parts[i])
		}
		return result
	}
	left, right := parse(a), parse(b)
	for i := range left {
		if left[i] < right[i] {
			return -1
		}
		if left[i] > right[i] {
			return 1
		}
	}
	return 0
}

func executableSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}
