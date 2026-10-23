package v2tui

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"plexlinker/internal/browser"
	"plexlinker/internal/config"
	media "plexlinker/internal/downloader"
	"plexlinker/internal/downloads"
	"plexlinker/internal/storage"
)

type fakeRemote struct{ listing *browser.DirectoryListing }

func (f fakeRemote) List(context.Context, string) (*browser.DirectoryListing, error) {
	return f.listing, nil
}
func (fakeRemote) FullURL(path string) string { return "https://example.test/" + path }

type treeRemote map[string]*browser.DirectoryListing

func (tree treeRemote) List(_ context.Context, path string) (*browser.DirectoryListing, error) {
	return tree[path], nil
}
func (treeRemote) FullURL(path string) string { return "https://example.test/" + path }

type fakeVolumes struct{}

func (fakeVolumes) ListVolumes(context.Context) ([]storage.Volume, error) {
	return []storage.Volume{{Path: `W:\`, Label: "Media"}}, nil
}

type fakeDirectories struct{}

func (fakeDirectories) List(context.Context, string) ([]storage.Entry, error) { return nil, nil }
func (fakeDirectories) CreateDirectory(string) error                          { return nil }
func (fakeDirectories) IsWritable(string) error                               { return nil }

type unwritableDirectories struct{ fakeDirectories }

func (unwritableDirectories) IsWritable(path string) error {
	return errors.New(`destination "` + path + `" is not writable: media is write protected`)
}

type fakeDownloads struct {
	submitted []downloads.Request
	jobs      []downloads.Job
	results   []string
}

func (f *fakeDownloads) Submit(_ context.Context, requests []downloads.Request) ([]downloads.Submitted, error) {
	f.submitted = requests
	return []downloads.Submitted{{GID: "gid"}}, nil
}
func (f *fakeDownloads) Snapshot(context.Context) ([]downloads.Job, error) { return f.jobs, nil }
func (f *fakeDownloads) Pause(context.Context, string) error               { return nil }
func (f *fakeDownloads) Resume(context.Context, string) error              { return nil }
func (f *fakeDownloads) Remove(context.Context, string, bool) error        { return nil }
func (f *fakeDownloads) RemoveResult(_ context.Context, gid string) error {
	f.results = append(f.results, gid)
	return nil
}

func testModel() (*Model, *fakeDownloads) {
	downloadService := &fakeDownloads{}
	cfg := config.NewConfig()
	cfg.MoviesPath = `W:\movies`
	cfg.TVShowsPath = `W:\shows`
	model := New(Services{
		Remote:  fakeRemote{listing: &browser.DirectoryListing{Items: []browser.DirectoryItem{{Name: "Show.S01E01.mkv", Path: "shows/episode.mkv", Type: browser.TypeFile}}}},
		Volumes: fakeVolumes{}, Directories: fakeDirectories{}, Downloads: downloadService,
		Username: "media", Password: "secret",
		Config: cfg,
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
	if model.tab != TabDownloads {
		t.Fatalf("tab = %v, want downloads", model.tab)
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

func TestCreateDirectoryPromptStartsWithChildPathPrefix(t *testing.T) {
	t.Parallel()
	model, _ := testModel()
	model.pickerTarget = pickerSelection
	model.destinationPath = filepath.Join("root", "media")

	key := tea.KeyPressMsg(tea.Key{Text: "n", Code: 'n'})
	if _, command := model.handleDestinationPickerKey(key); command != nil {
		t.Fatal("opening the directory prompt should not start asynchronous work")
	}
	want := model.destinationPath + string(filepath.Separator)
	if got := model.destinationInput.Value(); got != want {
		t.Fatalf("directory prompt value = %q, want %q", got, want)
	}
}

func TestGoToPathPromptStartsEmpty(t *testing.T) {
	t.Parallel()
	model, _ := testModel()
	model.pickerTarget = pickerSelection
	model.destinationPath = filepath.Join("root", "media")

	key := tea.KeyPressMsg(tea.Key{Text: "g", Code: 'g'})
	if _, command := model.handleDestinationPickerKey(key); command != nil {
		t.Fatal("opening the path prompt should not start asynchronous work")
	}
	if got := model.destinationInput.Value(); got != "" {
		t.Fatalf("go-to-path prompt value = %q, want an empty field", got)
	}
}

func TestChildPathPrefixPreservesExistingSeparators(t *testing.T) {
	t.Parallel()
	for _, path := range []string{"/", "/tmp/", `W:\`, `\\server\share\`} {
		if got := childPathPrefix(path); got != path {
			t.Errorf("childPathPrefix(%q) = %q", path, got)
		}
	}
}

func TestDownloadRequiresPerItemDestination(t *testing.T) {
	t.Parallel()
	model, _ := testModel()
	model.selected["https://example.test/file"] = selectedFile{Name: "file"}
	if command := model.submitSelected(); command != nil {
		t.Fatal("unexpected command")
	}
	if !strings.Contains(model.status, "Choose a destination") {
		t.Fatalf("status = %q", model.status)
	}
}

func TestSettingsSave(t *testing.T) {
	t.Parallel()
	model, _ := testModel()
	cfg := config.NewConfig()
	model.services.Config = cfg
	saved := false
	model.services.SaveConfig = func(updated *config.Config) error { saved = true; return nil }
	model.initializeSettings()
	model.settingsInputs[0].SetValue("https://media.example.test/files/")
	model.settingsInputs[1].SetValue("user")
	model.settingsInputs[2].SetValue("pass")
	model.settingsInputs[settingMoviesPath].SetValue(`W:\films`)
	model.settingsInputs[settingTVPath].SetValue(`N:\shows`)
	model.settingsInputs[settingAriaEndpoint].SetValue("http://127.0.0.1:6800/jsonrpc")
	model.settingsInputs[settingConcurrency].SetValue("5")
	message := model.saveSettings()()
	model.Update(message)
	if !saved || cfg.MaxConcurrentDownloads != 5 || cfg.BaseURL != "https://media.example.test/files/" || cfg.MoviesPath != `W:\films` || cfg.TVShowsPath != `N:\shows` {
		t.Fatalf("saved=%v config=%#v", saved, cfg)
	}
}

func TestSelectionInfersMovieAndTVDestinations(t *testing.T) {
	t.Parallel()
	model, _ := testModel()
	tv := model.makeSelectedFile("tv", "Show.S01E01.mkv")
	movie := model.makeSelectedFile("movie", "Film.2026.mkv")
	if tv.Category != media.CategoryTVShows || tv.Destination != `W:\shows` {
		t.Fatalf("TV selection = %#v", tv)
	}
	if movie.Category != media.CategoryMovies || movie.Destination != `W:\movies` {
		t.Fatalf("movie selection = %#v", movie)
	}
}

func TestSelectAllTogglesOnlyCurrentDirectory(t *testing.T) {
	t.Parallel()
	model, _ := testModel()
	model.tab = TabRemote
	model.remoteItems = []browser.DirectoryItem{
		{Name: "One.mkv", Path: "current/One.mkv", Type: browser.TypeFile},
		{Name: "Two.mp4", Path: "current/Two.mp4", Type: browser.TypeFile},
		{Name: "Folder", Path: "current/Folder", Type: browser.TypeDirectory},
	}
	model.selected["https://example.test/elsewhere.mkv"] = model.makeSelectedFile("https://example.test/elsewhere.mkv", "Elsewhere.mkv")
	model.selectAll()
	if len(model.selected) != 3 {
		t.Fatalf("selected after first toggle = %#v", model.selected)
	}
	model.selectAll()
	if len(model.selected) != 1 || model.selected["https://example.test/elsewhere.mkv"].Name != "Elsewhere.mkv" {
		t.Fatalf("selected after second toggle = %#v", model.selected)
	}
}

func TestForceDownloadBypassesWritePreflight(t *testing.T) {
	t.Parallel()
	model, downloadService := testModel()
	model.services.Directories = unwritableDirectories{}
	file := model.makeSelectedFile("https://example.test/Movie.mkv", "Movie.mkv")
	model.selected[file.URL] = file

	model.Update(model.submitSelected()())
	if len(downloadService.submitted) != 0 || !strings.Contains(model.status, "Shift+D") {
		t.Fatalf("preflight status=%q submitted=%#v", model.status, downloadService.submitted)
	}
	model.Update(model.submitSelectedWithoutPreflight()())
	if len(downloadService.submitted) != 1 {
		t.Fatalf("forced submission = %#v", downloadService.submitted)
	}
}

func TestSelectionDestinationOverrideResetAndRemove(t *testing.T) {
	t.Parallel()
	model, _ := testModel()
	file := model.makeSelectedFile("movie", "Film.mkv")
	file.Destination = `N:\special`
	model.selected[file.URL] = file
	model.resetSelectedDestination()
	if got := model.selected[file.URL].Destination; got != `W:\movies` {
		t.Fatalf("reset destination = %q", got)
	}
	model.selectionColumn = 0
	model.activateSelection()
	if len(model.selected) != 0 {
		t.Fatalf("selection was not removed: %#v", model.selected)
	}
}

func TestSelectionPickerCommitsCurrentDirectory(t *testing.T) {
	t.Parallel()
	model, _ := testModel()
	file := model.makeSelectedFile("movie", "Film.mkv")
	model.selected[file.URL] = file
	model.selectionColumn = 1
	if command := model.activateSelection(); command == nil {
		t.Fatal("expected initial directory load")
	}
	model.destinationPath = `N:\one-off`
	model.finishDestinationPicker()
	if got := model.selected[file.URL].Destination; got != `N:\one-off` {
		t.Fatalf("destination = %q", got)
	}
}

func TestDownloadUsesPerItemDestinations(t *testing.T) {
	t.Parallel()
	model, downloadService := testModel()
	first := model.makeSelectedFile("movie", "Film.mkv")
	second := model.makeSelectedFile("tv", "Show.S02E03.mkv")
	first.Destination = `N:\special-film`
	model.selected[first.URL] = first
	model.selected[second.URL] = second
	message := model.submitSelected()()
	model.Update(message)
	got := make(map[string]string)
	for _, request := range downloadService.submitted {
		got[request.URL] = request.Destination
	}
	if got["movie"] != `N:\special-film` || got["tv"] != `W:\shows` {
		t.Fatalf("submitted destinations = %#v", got)
	}
}

func TestSettingsPathEnterOpensPickerAndCommitUpdatesField(t *testing.T) {
	t.Parallel()
	model, _ := testModel()
	model.tab = TabSettings
	model.focusSetting(settingMoviesPath)
	if _, command := model.handleSettingsKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})); command == nil {
		t.Fatal("expected directory load")
	}
	if model.pickerTarget != pickerMoviesSetting {
		t.Fatalf("picker target = %v", model.pickerTarget)
	}
	model.destinationPath = `N:\movies`
	model.finishDestinationPicker()
	if got := model.settingsInputs[settingMoviesPath].Value(); got != `N:\movies` {
		t.Fatalf("movie path = %q", got)
	}
}

func TestSettingsEscapeReturnsToPreviousView(t *testing.T) {
	t.Parallel()
	model, _ := testModel()
	model.tab = TabDownloads

	model.handleKey(tea.KeyPressMsg(tea.Key{Text: "5", Code: '5'}))
	if model.tab != TabSettings {
		t.Fatalf("tab = %v, want settings", model.tab)
	}
	model.handleSettingsKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc}))
	if model.tab != TabDownloads {
		t.Fatalf("tab = %v, want previous downloads view", model.tab)
	}
}

func TestBrowserArrowNavigation(t *testing.T) {
	t.Parallel()
	model, _ := testModel()
	model.remotePath = "Shows/Season 1"
	model.remoteItems = []browser.DirectoryItem{{Name: "Folder", Path: "Shows/Season 1/Folder", Type: browser.TypeDirectory}}
	if _, command := model.handleKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyRight})); command == nil {
		t.Fatal("right arrow did not open directory")
	}
	if _, command := model.handleKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyLeft})); command == nil {
		t.Fatal("left arrow did not navigate to parent")
	}
}

func TestHeaderUsesV2NavigationOrder(t *testing.T) {
	t.Parallel()
	model, _ := testModel()
	view := model.View().Content
	for _, label := range []string{"1 Browser", "2 Selection", "3 Downloads", "4 TV Renamer", "5 Settings"} {
		if !strings.Contains(view, label) {
			t.Errorf("view missing %q", label)
		}
	}
	if strings.Contains(view, "Destination]") {
		t.Fatalf("standalone destination tab remains: %s", view)
	}
}

func TestRetryFailedDownloadReusesRetainedRequest(t *testing.T) {
	t.Parallel()
	model, downloadService := testModel()
	model.tab = TabDownloads
	model.jobs = []downloads.Job{{
		GID: "failed-gid", Name: "Movie.mkv", SourceURL: "https://example.test/Movie.mkv",
		Destination: `N:\movies`, State: downloads.StateError,
	}}
	command := model.retryJob()
	if command == nil {
		t.Fatal("missing retry command")
	}
	model.Update(command())
	if len(downloadService.results) != 1 || downloadService.results[0] != "failed-gid" {
		t.Fatalf("removed results = %#v", downloadService.results)
	}
	if len(downloadService.submitted) != 1 {
		t.Fatalf("submitted = %#v", downloadService.submitted)
	}
	request := downloadService.submitted[0]
	if request.URL != "https://example.test/Movie.mkv" || request.Destination != `N:\movies` || request.Filename != "Movie.mkv" {
		t.Fatalf("retry request = %#v", request)
	}
}

func TestRemoveStoppedDownloadUsesResultRemoval(t *testing.T) {
	t.Parallel()
	model, downloadService := testModel()
	model.tab = TabDownloads
	model.jobs = []downloads.Job{{GID: "complete-gid", Name: "Movie.mkv", State: downloads.StateComplete}}
	command := model.removeJob()
	if command == nil {
		t.Fatal("missing remove command")
	}
	model.Update(command())
	if len(downloadService.results) != 1 || downloadService.results[0] != "complete-gid" {
		t.Fatalf("removed results = %#v", downloadService.results)
	}
}

func TestRecursiveSelection(t *testing.T) {
	t.Parallel()
	model, _ := testModel()
	model.services.Remote = treeRemote{
		"":      {CurrentPath: "", Items: []browser.DirectoryItem{{Name: "Shows", Path: "Shows", Type: browser.TypeDirectory}, {Name: "movie.mkv", Path: "movie.mkv", Type: browser.TypeFile}}},
		"Shows": {CurrentPath: "Shows", Items: []browser.DirectoryItem{{Name: "episode.mkv", Path: "Shows/episode.mkv", Type: browser.TypeFile}}},
	}
	message := model.selectRecursive("")()
	model.Update(message)
	if len(model.selected) != 2 {
		t.Fatalf("selected = %#v", model.selected)
	}
}

func TestVisibleRangeFollowsCursor(t *testing.T) {
	t.Parallel()
	model, _ := testModel()
	model.height = 13 // five content rows
	start, end := model.visibleRange(8, 10)
	if start != 5 || end != 10 {
		t.Fatalf("range = %d:%d", start, end)
	}
}
