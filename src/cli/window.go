package cli

import (
	"golang.org/x/crypto/ssh/terminal"
	"runtime"
)

// WindowSize finds and returns the size of the console window as (rows, columns)
func WindowSize() (int, int, error) {
	cols, rows, err := terminal.GetSize(tiocgwinsz())
	if err != nil {
		return 25, 80, err
	}
	return rows, cols, err
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
