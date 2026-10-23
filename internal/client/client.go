package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"plexlinker/internal/config"
	"plexlinker/internal/logger"
)

const maxDirectoryResponseBytes = 16 << 20

type Client struct {
	httpClient *http.Client
	config     *config.Config
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		config: cfg,
	}
}

func (c *Client) FetchDirectory(path string) (string, error) {
	return c.FetchDirectoryContext(context.Background(), path)
}

func (c *Client) FetchDirectoryContext(ctx context.Context, path string) (string, error) {
	fullURL, err := c.buildURL(path)
	if err != nil {
		return "", err
	}
	logger.Get().Client().Printf("FetchDirectory: Starting request to %s", fullURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		logger.Get().Client().Printf("FetchDirectory: Failed to create request: %v", err)
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.config.Username, c.config.Password)
	req.Header.Set("User-Agent", "PlexLinker/1.0")
	logger.Get().Client().Printf("FetchDirectory: Sending HTTP request to %s", fullURL)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		logger.Get().Client().Printf("FetchDirectory: HTTP request failed: %v", err)
		return "", fmt.Errorf("failed to fetch directory: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	logger.Get().Client().Printf("FetchDirectory: Received response with status %d", resp.StatusCode)

	if resp.StatusCode == http.StatusUnauthorized {
		logger.Get().Client().Printf("FetchDirectory: Authentication failed")
		return "", fmt.Errorf("authentication failed: invalid username or password")
	}

	if resp.StatusCode != http.StatusOK {
		logger.Get().Client().Printf("FetchDirectory: Server returned non-OK status: %d %s", resp.StatusCode, resp.Status)
		return "", fmt.Errorf("server returned status %d: %s", resp.StatusCode, resp.Status)
	}

	logger.Get().Client().Printf("FetchDirectory: Reading response body")
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxDirectoryResponseBytes+1))
	if err != nil {
		logger.Get().Client().Printf("FetchDirectory: Failed to read response body: %v", err)
		return "", fmt.Errorf("failed to read response body: %w", err)
	}
	if len(body) > maxDirectoryResponseBytes {
		return "", fmt.Errorf("directory response exceeds %d MiB", maxDirectoryResponseBytes>>20)
	}

	logger.Get().Client().Printf("FetchDirectory: Successfully fetched %d bytes", len(body))
	return string(body), nil
}

func (c *Client) buildURL(path string) (string, error) {
	baseURL, err := url.Parse(c.config.BaseURL)
	if err != nil || baseURL.Scheme == "" || baseURL.Host == "" {
		return "", fmt.Errorf("invalid base URL %q", c.config.BaseURL)
	}
	reference, err := url.Parse(path)
	if err != nil {
		return "", fmt.Errorf("invalid directory path %q: %w", path, err)
	}
	if reference.IsAbs() || reference.Host != "" || strings.HasPrefix(path, "/") {
		return "", fmt.Errorf("directory path must remain on the configured server")
	}
	for _, segment := range strings.Split(reference.Path, "/") {
		if segment == ".." {
			return "", fmt.Errorf("directory path must not traverse above the configured root")
		}
	}
	resolvedURL := baseURL.ResolveReference(reference)
	if !strings.EqualFold(resolvedURL.Scheme, baseURL.Scheme) || !strings.EqualFold(resolvedURL.Host, baseURL.Host) {
		return "", fmt.Errorf("directory path escaped the configured server")
	}
	if !strings.HasPrefix(resolvedURL.EscapedPath(), baseURL.EscapedPath()) {
		return "", fmt.Errorf("directory path escaped the configured root")
	}
	return resolvedURL.String(), nil
}

func (c *Client) ValidateConnection() error {
	_, err := c.FetchDirectory("")
	if err != nil {
		return fmt.Errorf("connection validation failed: %v", err)
	}
	return nil
}

func (c *Client) GetFullURL(path string) string {
	resolved, err := c.buildURL(path)
	if err != nil {
		return ""
	}
	return resolved
}
