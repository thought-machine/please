package cli

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"
)

// For calculating the size of the console window; this is pretty important when we're writing
// arbitrary-length log messages around the interactive display.
type winsize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

// WindowSize finds and returns the size of the console window as (rows, columns)
func WindowSize() (int, int, error) {
	ws := winsize{}
	if ret, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(syscall.Stderr),
		uintptr(tiocgwinsz()),
		uintptr(unsafe.Pointer(&ws)),
	); int(ret) == -1 {
		return 25, 80, fmt.Errorf("error %d getting window size", int(errno))
	}
	return int(ws.Row), int(ws.Col), nil
}

// tiocgwinsz returns the ioctl number corresponding to TIOCGWINSZ.
// We could determine this using cgo which would be more robust, but I'd really
// rather not invoke cgo for something as static as this.
func tiocgwinsz() int {
	if runtime.GOOS == "linux" {
		return 0x5413
	}
	return 1074295912 // OSX and FreeBSD.
}
