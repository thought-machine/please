package cli

import (
	"os"

	"golang.org/x/crypto/ssh/terminal"
)

// WindowSize finds and returns the size of the console window as (rows, columns)
func WindowSize() (int, int, error) {
	cols, rows, err := terminal.GetSize(int(os.Stderr.Fd()))
	if err != nil {
		return 25, 80, err
	}
	return rows, cols, err
}
