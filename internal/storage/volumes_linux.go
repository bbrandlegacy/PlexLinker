//go:build linux

package storage

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

type linuxVolumeProvider struct{}

func newPlatformVolumeProvider() VolumeProvider { return linuxVolumeProvider{} }

func (linuxVolumeProvider) ListVolumes(ctx context.Context) ([]Volume, error) {
	file, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return []Volume{inspectLinuxVolume("/", "")}, nil
	}
	defer func() { _ = file.Close() }()
	seen := map[string]bool{}
	var volumes []Volume
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		fields := strings.Fields(scanner.Text())
		if len(fields) < 10 {
			continue
		}
		separator := -1
		for i, field := range fields {
			if field == "-" {
				separator = i
				break
			}
		}
		if separator < 0 || separator+2 >= len(fields) {
			continue
		}
		mount := strings.ReplaceAll(strings.ReplaceAll(fields[4], `\040`, " "), `\134`, `\`)
		filesystem := fields[separator+1]
		if seen[mount] || skipLinuxFilesystem(filesystem) {
			continue
		}
		seen[mount] = true
		volumes = append(volumes, inspectLinuxVolume(mount, filesystem))
	}
	if len(volumes) == 0 {
		volumes = append(volumes, inspectLinuxVolume("/", ""))
	}
	return volumes, scanner.Err()
}

func inspectLinuxVolume(path, filesystem string) Volume {
	volume := Volume{Path: path, Label: filepath.Base(path), FileSystem: filesystem, Kind: VolumeFixed, Available: true}
	if path == "/" {
		volume.Label = "System"
	}
	if strings.HasPrefix(filesystem, "nfs") || filesystem == "cifs" || filesystem == "smb3" {
		volume.Kind = VolumeNetwork
	}
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		volume.Available, volume.Error = false, err.Error()
		return volume
	}
	volume.TotalBytes = stat.Blocks * uint64(stat.Bsize)
	volume.FreeBytes = stat.Bavail * uint64(stat.Bsize)
	return volume
}

func skipLinuxFilesystem(name string) bool {
	switch name {
	case "proc", "sysfs", "devtmpfs", "devpts", "cgroup", "cgroup2", "securityfs", "pstore", "debugfs", "tracefs", "configfs", "fusectl", "mqueue", "hugetlbfs":
		return true
	default:
		return false
	}
}
