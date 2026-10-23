//go:build !windows && !darwin && !linux

package storage

import "context"

type fallbackVolumeProvider struct{}

func newPlatformVolumeProvider() VolumeProvider { return fallbackVolumeProvider{} }
func (fallbackVolumeProvider) ListVolumes(context.Context) ([]Volume, error) {
	return []Volume{{Path: "/", Label: "System", Kind: VolumeFixed, Available: true}}, nil
}
