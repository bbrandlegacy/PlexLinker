//go:build darwin

package storage

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
)

type darwinVolumeProvider struct{}

func newPlatformVolumeProvider() VolumeProvider { return darwinVolumeProvider{} }

func (darwinVolumeProvider) ListVolumes(ctx context.Context) ([]Volume, error) {
	paths := []string{"/"}
	items, err := os.ReadDir("/Volumes")
	if err == nil {
		for _, item := range items {
			if item.IsDir() || item.Type()&os.ModeSymlink != 0 {
				paths = append(paths, filepath.Join("/Volumes", item.Name()))
			}
		}
	}
	volumes := make([]Volume, 0, len(paths))
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		volumes = append(volumes, inspectUnixVolume(path))
	}
	return volumes, nil
}

func inspectUnixVolume(path string) Volume {
	volume := Volume{Path: path, Label: filepath.Base(path), Kind: VolumeFixed, Available: true}
	if path == "/" {
		volume.Label = "System"
	}
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		volume.Available, volume.Error = false, err.Error()
		return volume
	}
	volume.TotalBytes = uint64(stat.Blocks) * uint64(stat.Bsize)
	volume.FreeBytes = uint64(stat.Bavail) * uint64(stat.Bsize)
	return volume
}
