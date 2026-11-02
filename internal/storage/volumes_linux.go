//go:build linux

package storage

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
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
		source := fields[separator+2]
		if seen[mount] || skipLinuxFilesystem(filesystem) {
			continue
		}
		seen[mount] = true
		volumes = append(volumes, inspectLinuxVolumeSource(mount, filesystem, source))
	}
	if len(volumes) == 0 {
		volumes = append(volumes, inspectLinuxVolume("/", ""))
	}
	RankVolumes(volumes)
	return volumes, scanner.Err()
}

func inspectLinuxVolume(path, filesystem string) Volume {
	return inspectLinuxVolumeSource(path, filesystem, "")
}

func inspectLinuxVolumeSource(path, filesystem, source string) Volume {
	volume := Volume{Path: path, Label: filepath.Base(path), FileSystem: filesystem, Kind: VolumeFixed, Available: true, Writable: unix.Access(path, unix.W_OK) == nil, System: path == "/"}
	if path == "/" {
		volume.Label = "System"
	}
	if strings.HasPrefix(filesystem, "nfs") || filesystem == "cifs" || filesystem == "smb3" {
		volume.Kind = VolumeNetwork
	}
	if strings.HasPrefix(source, "/dev/") && linuxDeviceExternal(filepath.Base(source)) {
		volume.External = true
		volume.Kind = VolumeRemovable
	}
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		volume.Available, volume.Error = false, err.Error()
		return volume
	}
	volume.TotalBytes = stat.Blocks * uint64(stat.Bsize)
	volume.FreeBytes = stat.Bavail * uint64(stat.Bsize)
	DiscoverMediaPaths(&volume)
	return volume
}

func linuxDeviceExternal(name string) bool {
	path := filepath.Join("/sys/class/block", name)
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil && strings.Contains(strings.ToLower(resolved), "/usb") {
		return true
	}
	if err == nil {
		path = resolved
	}
	for current := path; current != "/" && current != "."; current = filepath.Dir(current) {
		data, err := os.ReadFile(filepath.Join(current, "removable"))
		if err == nil && strings.TrimSpace(string(data)) == "1" {
			return true
		}
	}
	return false
}

func skipLinuxFilesystem(name string) bool {
	switch name {
	case "proc", "sysfs", "devtmpfs", "devpts", "tmpfs", "ramfs", "autofs", "nsfs", "binfmt_misc", "cgroup", "cgroup2", "securityfs", "pstore", "debugfs", "tracefs", "configfs", "fusectl", "mqueue", "hugetlbfs":
		return true
	default:
		return false
	}
}
