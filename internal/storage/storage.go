// Package storage provides platform-neutral destination volume and directory browsing.
package storage

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type VolumeKind string

const (
	VolumeFixed     VolumeKind = "fixed"
	VolumeNetwork   VolumeKind = "network"
	VolumeRemovable VolumeKind = "removable"
	VolumeOptical   VolumeKind = "optical"
	VolumeVirtual   VolumeKind = "virtual"
	VolumeUnknown   VolumeKind = "unknown"
)

type Volume struct {
	Path       string
	Label      string
	FileSystem string
	Kind       VolumeKind
	TotalBytes uint64
	FreeBytes  uint64
	Available  bool
	Writable   bool
	External   bool
	System     bool
	MoviesPath string
	TVPath     string
	Error      string
}

type VolumeProvider interface {
	ListVolumes(context.Context) ([]Volume, error)
}

func NewVolumeProvider() VolumeProvider { return newPlatformVolumeProvider() }

// RankVolumes puts useful media destinations first while retaining every mount.
func RankVolumes(volumes []Volume) {
	sort.SliceStable(volumes, func(i, j int) bool {
		rank := func(v Volume) int {
			switch {
			case v.External && v.Writable:
				return 0
			case v.Kind == VolumeNetwork && v.Writable:
				return 1
			case !v.System && v.Writable:
				return 2
			case v.Writable:
				return 3
			default:
				return 4
			}
		}
		left, right := rank(volumes[i]), rank(volumes[j])
		if left != right {
			return left < right
		}
		if volumes[i].FreeBytes != volumes[j].FreeBytes {
			return volumes[i].FreeBytes > volumes[j].FreeBytes
		}
		return strings.ToLower(volumes[i].Path) < strings.ToLower(volumes[j].Path)
	})
}

// DiscoverMediaPaths looks only at conventional root-level media locations.
// It intentionally avoids an expensive recursive walk of an entire disk.
func DiscoverMediaPaths(volume *Volume) {
	if volume == nil || !volume.Available {
		return
	}
	containers := []string{volume.Path}
	if entries, err := os.ReadDir(volume.Path); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			for _, name := range []string{"Plex", "Media", "Library"} {
				if strings.EqualFold(entry.Name(), name) {
					containers = append(containers, filepath.Join(volume.Path, entry.Name()))
				}
			}
		}
	}
	find := func(names ...string) string {
		for _, container := range containers {
			entries, err := os.ReadDir(container)
			if err != nil {
				continue
			}
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				for _, name := range names {
					if strings.EqualFold(entry.Name(), name) {
						return filepath.Join(container, entry.Name())
					}
				}
			}
		}
		return ""
	}
	volume.MoviesPath = find("Movies", "Movie", "Films")
	volume.TVPath = find("Shows", "TV", "TV Shows", "Series")
}

type Entry struct {
	Name       string
	Path       string
	IsDir      bool
	IsSymlink  bool
	Size       int64
	Mode       fs.FileMode
	Accessible bool
	Error      string
}

type Browser struct{}

func NewBrowser() *Browser { return &Browser{} }

func (b *Browser) List(ctx context.Context, path string) ([]Entry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if path == "" {
		return nil, fmt.Errorf("destination path is empty")
	}
	items, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("read destination %q: %w", path, err)
	}
	entries := make([]Entry, 0, len(items))
	for _, item := range items {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		entry := Entry{
			Name:       item.Name(),
			Path:       filepath.Join(path, item.Name()),
			IsDir:      item.IsDir(),
			IsSymlink:  item.Type()&fs.ModeSymlink != 0,
			Mode:       item.Type(),
			Accessible: true,
		}
		info, infoErr := item.Info()
		if infoErr != nil {
			entry.Accessible = false
			entry.Error = infoErr.Error()
		} else {
			entry.IsDir = info.IsDir()
			entry.Size = info.Size()
			entry.Mode = info.Mode()
		}
		entries = append(entries, entry)
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return entries[i].Name < entries[j].Name
	})
	return entries, nil
}

func (b *Browser) CreateDirectory(path string) error {
	if path == "" {
		return fmt.Errorf("destination path is empty")
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("create destination %q: %w", path, err)
	}
	return nil
}

func (b *Browser) IsWritable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("inspect destination %q: %w", path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("destination %q is not a directory", path)
	}
	file, err := os.CreateTemp(path, ".plexlinker-write-test-*")
	if err != nil {
		return fmt.Errorf("destination %q is not writable: %w", path, err)
	}
	name := file.Name()
	if closeErr := file.Close(); closeErr != nil {
		_ = os.Remove(name)
		return fmt.Errorf("close destination write test: %w", closeErr)
	}
	if err := os.Remove(name); err != nil {
		return fmt.Errorf("remove destination write test: %w", err)
	}
	return nil
}
