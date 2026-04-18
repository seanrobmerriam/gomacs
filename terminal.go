package main

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

// enableRawMode puts the terminal into raw mode and returns the original settings
func enableRawMode(fd int) (*syscall.Termios, error) {
	var orig syscall.Termios
	if _, _, err := syscall.Syscall6(
		syscall.SYS_IOCTL, uintptr(fd),
		ioctlReadTermios,
		uintptr(unsafe.Pointer(&orig)),
		0, 0, 0,
	); err != 0 {
		return nil, fmt.Errorf("tcgetattr: %v", err)
	}

	raw := orig
	raw.Iflag &^= syscall.BRKINT | syscall.ICRNL | syscall.INPCK | syscall.ISTRIP | syscall.IXON
	raw.Oflag &^= syscall.OPOST
	raw.Cflag |= syscall.CS8
	raw.Lflag &^= syscall.ECHO | syscall.ICANON | syscall.IEXTEN | syscall.ISIG
	raw.Cc[syscall.VMIN] = 1
	raw.Cc[syscall.VTIME] = 0

	if _, _, err := syscall.Syscall6(
		syscall.SYS_IOCTL, uintptr(fd),
		ioctlWriteTermios,
		uintptr(unsafe.Pointer(&raw)),
		0, 0, 0,
	); err != 0 {
		return nil, fmt.Errorf("tcsetattr: %v", err)
	}

	return &orig, nil
}

// disableRawMode restores the original terminal settings
func disableRawMode(fd int, orig *syscall.Termios) {
	syscall.Syscall6(
		syscall.SYS_IOCTL, uintptr(fd),
		ioctlWriteTermios,
		uintptr(unsafe.Pointer(orig)),
		0, 0, 0,
	)
}

type winsize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

// getTerminalSize returns the terminal width and height in characters
func getTerminalSize(fd int) (int, int, error) {
	var ws winsize
	if _, _, err := syscall.Syscall(
		syscall.SYS_IOCTL, uintptr(fd),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(&ws)),
	); err != 0 {
		return 0, 0, fmt.Errorf("TIOCGWINSZ: %v", err)
	}
	return int(ws.Col), int(ws.Row), nil
}

// enterAltScreen switches to the alternate screen buffer
func enterAltScreen(out *os.File) {
	fmt.Fprint(out, "\x1b[?1049h")
}

// exitAltScreen restores the main screen buffer
func exitAltScreen(out *os.File) {
	fmt.Fprint(out, "\x1b[?1049l")
}

// enableMouseReporting enables X10 mouse click and scroll reporting
func enableMouseReporting(out *os.File) {
	fmt.Fprint(out, "\x1b[?1000h")
}

// disableMouseReporting disables mouse reporting
func disableMouseReporting(out *os.File) {
	fmt.Fprint(out, "\x1b[?1000l")
}
