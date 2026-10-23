package v2tui

import (
	"context"
	"os/exec"

	"plexlinker/internal/browser"
	"plexlinker/internal/client"
	"plexlinker/internal/config"
	"plexlinker/internal/downloads"
	"plexlinker/internal/storage"
)

type RemoteService interface {
	List(context.Context, string) (*browser.DirectoryListing, error)
	FullURL(string) string
}

type remoteClient struct{ client *client.Client }

func NewRemoteService(httpClient *client.Client) RemoteService {
	return &remoteClient{client: httpClient}
}

func (r *remoteClient) List(ctx context.Context, path string) (*browser.DirectoryListing, error) {
	// The legacy HTTP client is timeout-bounded. Context-aware transport is added
	// at this adapter boundary so the TUI never performs network work in Update.
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	html, err := r.client.FetchDirectoryContext(ctx, path)
	if err != nil {
		return nil, err
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	return browser.ParseHTMLDirectoryWithSort(html, path, path == "")
}

func (r *remoteClient) FullURL(path string) string { return r.client.GetFullURL(path) }

type VolumeService interface {
	ListVolumes(context.Context) ([]storage.Volume, error)
}
type DirectoryService interface {
	List(context.Context, string) ([]storage.Entry, error)
	CreateDirectory(string) error
	IsWritable(string) error
}
type DownloadService interface {
	Submit(context.Context, []downloads.Request) ([]downloads.Submitted, error)
	Snapshot(context.Context) ([]downloads.Job, error)
	Pause(context.Context, string) error
	Resume(context.Context, string) error
	Remove(context.Context, string, bool) error
	RemoveResult(context.Context, string) error
}

type Services struct {
	Remote         RemoteService
	Volumes        VolumeService
	Directories    DirectoryService
	Downloads      DownloadService
	Username       string
	Password       string
	StartupMessage string
	Config         *config.Config
	SaveConfig     func(*config.Config) error
	Version        string
	RenamerCommand func() *exec.Cmd
}
