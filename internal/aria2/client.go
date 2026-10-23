// Package aria2 implements the subset of aria2's JSON-RPC API used by PlexLinker.
package aria2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"
)

type Client struct {
	endpoint   string
	secret     string
	httpClient *http.Client
	requestID  atomic.Uint64
}

type Option func(*Client)

func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) { c.httpClient = client }
}

func NewClient(endpoint, secret string, options ...Option) *Client {
	c := &Client{
		endpoint: endpoint,
		secret:   secret,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
	for _, option := range options {
		option(c)
	}
	return c
}

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      string `json:"id"`
	Method  string `json:"method"`
	Params  []any  `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *RPCError       `json:"error,omitempty"`
}

type RPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("aria2 RPC error %d: %s", e.Code, e.Message)
}

func (c *Client) call(ctx context.Context, method string, params []any, result any) error {
	if c.secret != "" {
		params = append([]any{"token:" + c.secret}, params...)
	}
	reqBody := rpcRequest{
		JSONRPC: "2.0",
		ID:      strconv.FormatUint(c.requestID.Add(1), 10),
		Method:  method,
		Params:  params,
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("encode aria2 request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create aria2 request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call aria2: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("aria2 returned HTTP %s", resp.Status)
	}

	var rpcResp rpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return fmt.Errorf("decode aria2 response: %w", err)
	}
	if rpcResp.Error != nil {
		return rpcResp.Error
	}
	if result == nil {
		return nil
	}
	if err := json.Unmarshal(rpcResp.Result, result); err != nil {
		return fmt.Errorf("decode aria2 result: %w", err)
	}
	return nil
}

type AddOptions map[string]string

func (c *Client) AddURI(ctx context.Context, uri string, options AddOptions) (string, error) {
	var gid string
	err := c.call(ctx, "aria2.addUri", []any{[]string{uri}, options}, &gid)
	return gid, err
}

type File struct {
	Index           string   `json:"index"`
	Path            string   `json:"path"`
	Length          string   `json:"length"`
	CompletedLength string   `json:"completedLength"`
	URIs            []URIRef `json:"uris"`
}

type URIRef struct {
	URI    string `json:"uri"`
	Status string `json:"status"`
}

type Job struct {
	GID             string `json:"gid"`
	Status          string `json:"status"`
	TotalLength     string `json:"totalLength"`
	CompletedLength string `json:"completedLength"`
	DownloadSpeed   string `json:"downloadSpeed"`
	ErrorCode       string `json:"errorCode"`
	ErrorMessage    string `json:"errorMessage"`
	Dir             string `json:"dir"`
	Files           []File `json:"files"`
}

var jobKeys = []string{
	"gid", "status", "totalLength", "completedLength", "downloadSpeed",
	"errorCode", "errorMessage", "dir", "files",
}

func (c *Client) TellActive(ctx context.Context) ([]Job, error) {
	var jobs []Job
	err := c.call(ctx, "aria2.tellActive", []any{jobKeys}, &jobs)
	return jobs, err
}

func (c *Client) TellWaiting(ctx context.Context, offset, count int) ([]Job, error) {
	var jobs []Job
	err := c.call(ctx, "aria2.tellWaiting", []any{offset, count, jobKeys}, &jobs)
	return jobs, err
}

func (c *Client) TellStopped(ctx context.Context, offset, count int) ([]Job, error) {
	var jobs []Job
	err := c.call(ctx, "aria2.tellStopped", []any{offset, count, jobKeys}, &jobs)
	return jobs, err
}

func (c *Client) Pause(ctx context.Context, gid string) error {
	return c.call(ctx, "aria2.pause", []any{gid}, nil)
}

func (c *Client) Unpause(ctx context.Context, gid string) error {
	return c.call(ctx, "aria2.unpause", []any{gid}, nil)
}

func (c *Client) Remove(ctx context.Context, gid string, force bool) error {
	method := "aria2.remove"
	if force {
		method = "aria2.forceRemove"
	}
	return c.call(ctx, method, []any{gid}, nil)
}

type GlobalStat struct {
	DownloadSpeed string `json:"downloadSpeed"`
	UploadSpeed   string `json:"uploadSpeed"`
	NumActive     string `json:"numActive"`
	NumWaiting    string `json:"numWaiting"`
	NumStopped    string `json:"numStopped"`
}

func (c *Client) GetGlobalStat(ctx context.Context) (GlobalStat, error) {
	var stat GlobalStat
	err := c.call(ctx, "aria2.getGlobalStat", nil, &stat)
	return stat, err
}

func (c *Client) SaveSession(ctx context.Context) error {
	return c.call(ctx, "aria2.saveSession", nil, nil)
}

func (c *Client) Version(ctx context.Context) (string, error) {
	var version struct {
		Version string `json:"version"`
	}
	err := c.call(ctx, "aria2.getVersion", nil, &version)
	return version.Version, err
}

// Shutdown stops aria2. It is primarily used by managed integration tests and
// explicit service administration; closing the PlexLinker TUI never calls it.
func (c *Client) Shutdown(ctx context.Context, force bool) error {
	method := "aria2.shutdown"
	if force {
		method = "aria2.forceShutdown"
	}
	return c.call(ctx, method, nil, nil)
}
