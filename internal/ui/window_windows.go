//go:build windows

package ui

import (
	"syscall"
	"unsafe"
)

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	procEnumWindows         = user32.NewProc("EnumWindows")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procIsWindowVisible     = user32.NewProc("IsWindowVisible")
)

// BringProcessWindowToFront brings the window of the specified process to the foreground
func BringProcessWindowToFront(pid int) error {
	hwnd := findWindowByPID(uint32(pid))
	if hwnd == 0 {
		return nil // Window not found (maybe closed)
	}

	// Bring window to foreground
	procSetForegroundWindow.Call(uintptr(hwnd))
	return nil
}

func findWindowByPID(targetPID uint32) uintptr {
	var foundHwnd uintptr

	callback := syscall.NewCallback(func(hwnd, lparam uintptr) uintptr {
		var pid uint32
		procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))

		if pid == targetPID {
			// Check if window is visible
			ret, _, _ := procIsWindowVisible.Call(hwnd)
			if ret != 0 {
				foundHwnd = hwnd
				return 0 // Stop enumeration
			}
		}
		return 1 // Continue enumeration
	})

	procEnumWindows.Call(callback, 0)
	return foundHwnd
}
