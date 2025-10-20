//go:build !windows

package ui

// BringProcessWindowToFront is a no-op on non-Windows platforms
func BringProcessWindowToFront(pid int) error {
	return nil // Not supported on this platform
}
