package client

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"plexlinker/internal/config"
	"plexlinker/internal/logger"
)

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
	fullURL := c.buildURL(path)
	logger.Get().Client().Printf("FetchDirectory: Starting request to %s", fullURL)

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		logger.Get().Client().Printf("FetchDirectory: Failed to create request: %v", err)
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.SetBasicAuth(c.config.Username, c.config.Password)
	req.Header.Set("User-Agent", "PlexLinker/1.0")
	logger.Get().Client().Printf("FetchDirectory: Sending HTTP request to %s", fullURL)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		logger.Get().Client().Printf("FetchDirectory: HTTP request failed: %v", err)
		return "", fmt.Errorf("failed to fetch directory: %v", err)
	}
	defer resp.Body.Close()

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
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Get().Client().Printf("FetchDirectory: Failed to read response body: %v", err)
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	logger.Get().Client().Printf("FetchDirectory: Successfully fetched %d bytes", len(body))
	return string(body), nil
}

func (c *Client) buildURL(path string) string {
	path = strings.TrimPrefix(path, "/")

	baseURL, err := url.Parse(c.config.BaseURL)
	if err != nil {
		return c.config.BaseURL + path
	}

	resolvedURL, err := baseURL.Parse(path)
	if err != nil {
		return c.config.BaseURL + path
	}

	return resolvedURL.String()
}

func (c *Client) ValidateConnection() error {
	_, err := c.FetchDirectory("")
	if err != nil {
		return fmt.Errorf("connection validation failed: %v", err)
	}
	return nil
}

func (c *Client) GetFullURL(path string) string {
	return c.buildURL(path)
}
