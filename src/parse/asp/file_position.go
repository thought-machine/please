package asp

import (
	"fmt"
	"os"

	"golang.org/x/exp/slices"
)

// A FilePosition is the more user-friendly equivalent to the Position type.
// Line and Column are both 1-indexed.
type FilePosition struct {
	Filename string
	Offset   int
	Line     int
	Column   int
}

// String implements the fmt.Stringer interface.
func (pos FilePosition) String() string {
	return fmt.Sprintf("%s:%d:%d", pos.Filename, pos.Line, pos.Column)
}

// A File represents a file being parsed and is useful for converting raw Positions to FilePositions.
type File struct {
	Name        string
	lineOffsets []int
}

// NewFile creates a File from the given path.
// N.B. This involves reading the entire file, and is typically only expected to be done on
// an error path. Hence the return value does not error itself, and can return an
func NewFile(path string) *File {
	b, _ := os.ReadFile(path)
	return newFile(path, b)
}

// newFile creates a File from a buffer.
func newFile(name string, buf []byte) *File {
	// N.B. The line offsets are the index of the preceding newline character.
	// This happens to be convenient when we binary search it (to avoid falling off the front
	// of the array, etc).
	f := &File{Name: name, lineOffsets: []int{-1}}
	for i, x := range buf {
		if x == '\n' {
			f.lineOffsets = append(f.lineOffsets, i)
		}
	}
	return f
}

// Pos converts a Position to a FilePosition based on this file.
func (f *File) Pos(pos Position) FilePosition {
	i := int(pos)
	line, _ := slices.BinarySearch(f.lineOffsets, i)
	lineOffset := f.lineOffsets[line-1]
	return FilePosition{
		Filename: f.Name,
		Offset:   i,
		Line:     line,
		Column:   i - lineOffset,
	}
}
