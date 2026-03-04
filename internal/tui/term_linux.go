//go:build linux

package tui

import (
	"syscall"
	"unsafe"
)

type savedState struct {
	termios syscall.Termios
}

func enableRawMode() (savedState, error) {
	var state savedState
	// Get current terminal state (fd 0 = stdin)
	if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, uintptr(syscall.Stdin), uintptr(syscall.TCGETS), uintptr(unsafe.Pointer(&state.termios))); err != 0 {
		return state, err
	}

	raw := state.termios
	// Clear ECHO and ICANON flags in Local flags
	raw.Lflag &^= syscall.ECHO | syscall.ICANON
	// Set VMIN and VTIME in control characters array
	raw.Cc[syscall.VMIN] = 1
	raw.Cc[syscall.VTIME] = 0

	// Set terminal state
	if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, uintptr(syscall.Stdin), uintptr(syscall.TCSETS), uintptr(unsafe.Pointer(&raw))); err != 0 {
		return state, err
	}

	return state, nil
}

func disableRawMode(state savedState) error {
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, uintptr(syscall.Stdin), uintptr(syscall.TCSETS), uintptr(unsafe.Pointer(&state.termios)))
	if err != 0 {
		return err
	}
	return nil
}
