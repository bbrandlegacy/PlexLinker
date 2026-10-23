package downloads

import (
	"context"
	"errors"
	"testing"

	"plexlinker/internal/aria2"
)

type fakeBackend struct {
	added                    []aria2.AddOptions
	saveCount                int
	active, waiting, stopped []aria2.Job
	addErr                   error
}

func (f *fakeBackend) AddURI(_ context.Context, _ string, options aria2.AddOptions) (string, error) {
	f.added = append(f.added, options)
	return "gid-1", f.addErr
}
func (f *fakeBackend) TellActive(context.Context) ([]aria2.Job, error) { return f.active, nil }
func (f *fakeBackend) TellWaiting(context.Context, int, int) ([]aria2.Job, error) {
	return f.waiting, nil
}
func (f *fakeBackend) TellStopped(context.Context, int, int) ([]aria2.Job, error) {
	return f.stopped, nil
}
func (f *fakeBackend) Pause(context.Context, string) error        { return nil }
func (f *fakeBackend) Unpause(context.Context, string) error      { return nil }
func (f *fakeBackend) Remove(context.Context, string, bool) error { return nil }
func (f *fakeBackend) SaveSession(context.Context) error          { f.saveCount++; return nil }

func TestSubmitMapsCredentialsWithoutScripts(t *testing.T) {
	t.Parallel()
	backend := &fakeBackend{}
	results, err := NewService(backend).Submit(context.Background(), []Request{{URL: "https://example.test/file.mkv", Destination: `W:\movies`, Username: "user", Password: "pass"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].GID != "gid-1" {
		t.Fatalf("results = %#v", results)
	}
	if backend.added[0]["dir"] != `W:\movies` || backend.added[0]["http-passwd"] != "pass" || backend.added[0]["continue"] != "true" {
		t.Fatalf("options = %#v", backend.added[0])
	}
	if backend.saveCount != 1 {
		t.Fatalf("save count = %d", backend.saveCount)
	}
}

func TestSubmitReportsPartialFailure(t *testing.T) {
	t.Parallel()
	backend := &fakeBackend{addErr: errors.New("offline")}
	results, err := NewService(backend).Submit(context.Background(), []Request{{URL: "bad", Destination: "/tmp"}, {URL: "https://example.test/file", Destination: "/tmp"}})
	if err == nil || len(results) != 2 || results[0].Error == nil || results[1].Error == nil {
		t.Fatalf("results=%#v err=%v", results, err)
	}
	if backend.saveCount != 0 {
		t.Fatal("session should not be saved after submission errors")
	}
}

func TestSnapshotConvertsAndSorts(t *testing.T) {
	t.Parallel()
	backend := &fakeBackend{active: []aria2.Job{{GID: "a", Status: "active", TotalLength: "100", CompletedLength: "25", DownloadSpeed: "10", Files: []aria2.File{{Path: "/media/a.mkv"}}}}, stopped: []aria2.Job{{GID: "b", Status: "complete", Files: []aria2.File{{Path: "/media/b.mkv"}}}}}
	jobs, err := NewService(backend).Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 2 || jobs[0].GID != "a" || jobs[0].Progress() != .25 || jobs[1].State != StateComplete {
		t.Fatalf("jobs = %#v", jobs)
	}
}
