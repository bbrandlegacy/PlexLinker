// Package downloads contains the platform-neutral download queue used by the v2 TUI.
package downloads

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"plexlinker/internal/aria2"
)

type State string

const (
	StateActive   State = "active"
	StateWaiting  State = "waiting"
	StatePaused   State = "paused"
	StateComplete State = "complete"
	StateError    State = "error"
	StateRemoved  State = "removed"
)

type Request struct {
	URL         string
	Destination string
	Filename    string
	Username    string
	Password    string
	Headers     []string
}

type Submitted struct {
	Request Request
	GID     string
	Error   error
}

type Job struct {
	GID            string
	Name           string
	Destination    string
	State          State
	TotalBytes     uint64
	CompletedBytes uint64
	BytesPerSecond uint64
	ErrorCode      string
	ErrorMessage   string
}

func (j Job) Progress() float64 {
	if j.TotalBytes == 0 {
		return 0
	}
	return float64(j.CompletedBytes) / float64(j.TotalBytes)
}

type Backend interface {
	AddURI(context.Context, string, aria2.AddOptions) (string, error)
	TellActive(context.Context) ([]aria2.Job, error)
	TellWaiting(context.Context, int, int) ([]aria2.Job, error)
	TellStopped(context.Context, int, int) ([]aria2.Job, error)
	Pause(context.Context, string) error
	Unpause(context.Context, string) error
	Remove(context.Context, string, bool) error
	SaveSession(context.Context) error
}

type Service struct{ backend Backend }

func NewService(backend Backend) *Service { return &Service{backend: backend} }

func (s *Service) Submit(ctx context.Context, requests []Request) ([]Submitted, error) {
	results := make([]Submitted, 0, len(requests))
	var failures []error
	for _, request := range requests {
		if err := validateRequest(request); err != nil {
			results = append(results, Submitted{Request: request, Error: err})
			failures = append(failures, err)
			continue
		}
		options := aria2.AddOptions{
			"dir":                request.Destination,
			"continue":           "true",
			"auto-file-renaming": "false",
		}
		if request.Filename != "" {
			options["out"] = request.Filename
		}
		if request.Username != "" {
			options["http-user"] = request.Username
		}
		if request.Password != "" {
			options["http-passwd"] = request.Password
		}
		if len(request.Headers) > 0 {
			options["header"] = strings.Join(request.Headers, "\n")
		}
		gid, err := s.backend.AddURI(ctx, request.URL, options)
		results = append(results, Submitted{Request: request, GID: gid, Error: err})
		if err != nil {
			failures = append(failures, fmt.Errorf("submit %q: %w", request.URL, err))
		}
		if ctx.Err() != nil {
			break
		}
	}
	if len(failures) == 0 {
		if err := s.backend.SaveSession(ctx); err != nil {
			return results, fmt.Errorf("save aria2 session: %w", err)
		}
		return results, nil
	}
	return results, errors.Join(failures...)
}

func validateRequest(request Request) error {
	parsed, err := url.ParseRequestURI(request.URL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("invalid download URL %q", request.URL)
	}
	if request.Destination == "" {
		return fmt.Errorf("destination is required for %q", request.URL)
	}
	if request.Filename != "" && filepath.Base(request.Filename) != request.Filename {
		return fmt.Errorf("filename %q must not contain a path", request.Filename)
	}
	return nil
}

func (s *Service) Snapshot(ctx context.Context) ([]Job, error) {
	active, activeErr := s.backend.TellActive(ctx)
	waiting, waitingErr := s.backend.TellWaiting(ctx, 0, 1000)
	stopped, stoppedErr := s.backend.TellStopped(ctx, 0, 1000)
	if err := errors.Join(activeErr, waitingErr, stoppedErr); err != nil {
		return nil, err
	}
	jobs := make([]Job, 0, len(active)+len(waiting)+len(stopped))
	for _, raw := range append(append(active, waiting...), stopped...) {
		jobs = append(jobs, convertJob(raw))
	}
	sort.SliceStable(jobs, func(i, j int) bool {
		if stateOrder(jobs[i].State) != stateOrder(jobs[j].State) {
			return stateOrder(jobs[i].State) < stateOrder(jobs[j].State)
		}
		return jobs[i].Name < jobs[j].Name
	})
	return jobs, nil
}

func (s *Service) Pause(ctx context.Context, gid string) error  { return s.backend.Pause(ctx, gid) }
func (s *Service) Resume(ctx context.Context, gid string) error { return s.backend.Unpause(ctx, gid) }
func (s *Service) Remove(ctx context.Context, gid string, force bool) error {
	return s.backend.Remove(ctx, gid, force)
}

func convertJob(raw aria2.Job) Job {
	job := Job{GID: raw.GID, Destination: raw.Dir, State: State(raw.Status), ErrorCode: raw.ErrorCode, ErrorMessage: raw.ErrorMessage}
	job.TotalBytes, _ = strconv.ParseUint(raw.TotalLength, 10, 64)
	job.CompletedBytes, _ = strconv.ParseUint(raw.CompletedLength, 10, 64)
	job.BytesPerSecond, _ = strconv.ParseUint(raw.DownloadSpeed, 10, 64)
	if len(raw.Files) > 0 {
		job.Name = filepath.Base(raw.Files[0].Path)
	}
	return job
}

func stateOrder(state State) int {
	switch state {
	case StateActive:
		return 0
	case StateWaiting, StatePaused:
		return 1
	case StateError:
		return 2
	case StateComplete:
		return 3
	default:
		return 4
	}
}
