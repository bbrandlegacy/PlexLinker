package v2tui

import (
	"context"
	"testing"

	"plexlinker/internal/browser"
	"plexlinker/internal/downloads"
	"plexlinker/internal/storage"
)

type fakeRemote struct{ listing *browser.DirectoryListing }

func (f fakeRemote) List(context.Context, string) (*browser.DirectoryListing, error) {
	return f.listing, nil
}
func (fakeRemote) FullURL(path string) string { return "https://example.test/" + path }

type fakeVolumes struct{}

func (fakeVolumes) ListVolumes(context.Context) ([]storage.Volume, error) {
	return []storage.Volume{{Path: `W:\`, Label: "Media"}}, nil
}

type fakeDirectories struct{}

func (fakeDirectories) List(context.Context, string) ([]storage.Entry, error) { return nil, nil }
func (fakeDirectories) CreateDirectory(string) error                          { return nil }
func (fakeDirectories) IsWritable(string) error                               { return nil }

type fakeDownloads struct {
	submitted []downloads.Request
	jobs      []downloads.Job
}

func (f *fakeDownloads) Submit(_ context.Context, requests []downloads.Request) ([]downloads.Submitted, error) {
	f.submitted = requests
	return []downloads.Submitted{{GID: "gid"}}, nil
}
func (f *fakeDownloads) Snapshot(context.Context) ([]downloads.Job, error) { return f.jobs, nil }
func (f *fakeDownloads) Pause(context.Context, string) error               { return nil }
func (f *fakeDownloads) Resume(context.Context, string) error              { return nil }
func (f *fakeDownloads) Remove(context.Context, string, bool) error        { return nil }

func testModel() (*Model, *fakeDownloads) {
	downloadService := &fakeDownloads{}
	model := New(Services{
		Remote:  fakeRemote{listing: &browser.DirectoryListing{Items: []browser.DirectoryItem{{Name: "Show.S01E01.mkv", Path: "shows/episode.mkv", Type: browser.TypeFile}}}},
		Volumes: fakeVolumes{}, Directories: fakeDirectories{}, Downloads: downloadService,
		Username: "media", Password: "secret",
	})
	return model, downloadService
}

func TestSelectDestinationAndSubmit(t *testing.T) {
	t.Parallel()
	model, downloadService := testModel()
	model.Update(remoteLoadedMsg{listing: &browser.DirectoryListing{Items: []browser.DirectoryItem{{Name: "Show.S01E01.mkv", Path: "shows/episode.mkv", Type: browser.TypeFile}}}})
	model.activate()
	if len(model.selected) != 1 {
		t.Fatalf("selected = %#v", model.selected)
	}
	model.destinationPath = `W:\shows`
	command := model.submitSelected()
	if command == nil {
		t.Fatal("missing submit command")
	}
	message := command()
	model.Update(message)
	if len(downloadService.submitted) != 1 {
		t.Fatalf("submitted = %#v", downloadService.submitted)
	}
	request := downloadService.submitted[0]
	if request.Destination != `W:\shows` || request.Username != "media" || request.Password != "secret" {
		t.Fatalf("request = %#v", request)
	}
	if len(model.selected) != 0 {
		t.Fatalf("selection was not cleared: %#v", model.selected)
	}
}

func TestDestinationViewFiltersFiles(t *testing.T) {
	t.Parallel()
	model, _ := testModel()
	model.Update(directoryLoadedMsg{path: `W:\`, entries: []storage.Entry{{Name: "Movies", IsDir: true}, {Name: "readme.txt", IsDir: false}}})
	if len(model.destinationEntries) != 1 || model.destinationEntries[0].Name != "Movies" {
		t.Fatalf("entries = %#v", model.destinationEntries)
	}
}

func TestQueueRequiresDestination(t *testing.T) {
	t.Parallel()
	model, _ := testModel()
	model.selected["https://example.test/file"] = selectedFile{Name: "file"}
	if command := model.submitSelected(); command != nil {
		t.Fatal("unexpected command")
	}
	if model.tab != TabDestination {
		t.Fatalf("tab = %v", model.tab)
	}
}
