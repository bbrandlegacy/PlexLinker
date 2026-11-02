//go:build windows

package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
)

type windowsVolumeProvider struct{}

func newPlatformVolumeProvider() VolumeProvider { return windowsVolumeProvider{} }

func (windowsVolumeProvider) ListVolumes(ctx context.Context) ([]Volume, error) {
	mask, err := windows.GetLogicalDrives()
	if err != nil {
		return nil, fmt.Errorf("enumerate Windows drives: %w", err)
	}
	volumes := make([]Volume, 0, 8)
	for index := 0; index < 26; index++ {
		if mask&(1<<index) == 0 {
			continue
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		root := fmt.Sprintf("%c:\\", 'A'+index)
		volumes = append(volumes, inspectWindowsVolume(root))
	}
	RankVolumes(volumes)
	return volumes, nil
}

func inspectWindowsVolume(root string) Volume {
	systemRoot := strings.ToUpper(filepath.VolumeName(os.Getenv("SystemRoot")) + `\`)
	volume := Volume{Path: root, Kind: VolumeUnknown, Available: true, Writable: true, System: strings.EqualFold(root, systemRoot)}
	rootPtr, err := windows.UTF16PtrFromString(root)
	if err != nil {
		volume.Available, volume.Error = false, err.Error()
		return volume
	}
	volume.Kind = windowsDriveKind(windows.GetDriveType(rootPtr))
	volume.External = volume.Kind == VolumeRemovable
	label := make([]uint16, windows.MAX_PATH+1)
	filesystem := make([]uint16, windows.MAX_PATH+1)
	var flags uint32
	if err := windows.GetVolumeInformation(rootPtr, &label[0], uint32(len(label)), nil, nil, &flags, &filesystem[0], uint32(len(filesystem))); err == nil {
		volume.Label = windows.UTF16ToString(label)
		volume.FileSystem = windows.UTF16ToString(filesystem)
		volume.Writable = flags&windows.FILE_READ_ONLY_VOLUME == 0
	} else if volume.Kind != VolumeOptical {
		volume.Available = false
		volume.Error = err.Error()
	}
	var available, total, free uint64
	if err := windows.GetDiskFreeSpaceEx(rootPtr, &available, &total, &free); err == nil {
		volume.TotalBytes = total
		volume.FreeBytes = available
	} else if volume.Error == "" {
		volume.Error = err.Error()
	}
	volume.Label = strings.TrimSpace(volume.Label)
	DiscoverMediaPaths(&volume)
	return volume
}

func windowsDriveKind(kind uint32) VolumeKind {
	switch kind {
	case windows.DRIVE_FIXED:
		return VolumeFixed
	case windows.DRIVE_REMOTE:
		return VolumeNetwork
	case windows.DRIVE_REMOVABLE:
		return VolumeRemovable
	case windows.DRIVE_CDROM:
		return VolumeOptical
	case windows.DRIVE_RAMDISK:
		return VolumeVirtual
	default:
		return VolumeUnknown
	}
}
