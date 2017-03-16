package cli

import (
	"io"
	"strconv"
	"strings"

	"github.com/dustin/go-humanize"
)

// progressReader implements an io.ReadCloser that shows a progress bar in the terminal which
// updates as more is read.
// You should take care to close this in order to clean up the terminal afterwards.
// Note that it is fairly hardcoded to our use case right now (i.e. downloading Please tarballs),
// it probably doesn't generalise perfectly to things of arbitrary sizes.
type progressReader struct {
	current, last, max, width int
	reader                    io.Reader
}

// NewProgressReader returns a new progress bar reader.
func NewProgressReader(reader io.Reader, total string) io.ReadCloser {
	i, _ := strconv.Atoi(total)
	_, cols := WindowSize()
	return &progressReader{
		max:    i, // If we failed above this is zero, that's handled later.
		reader: reader,
		width:  cols,
	}
}

// Read implements the io.Reader interface
func (pr *progressReader) Read(b []byte) (int, error) {
	n, err := pr.reader.Read(b)
	pr.current = pr.current + n
	pr.update()
	pr.last = pr.current
	return n, err
}

// Close implements the io.Closer interface
// This implementation never returns an error.
func (pr *progressReader) Close() error {
	if StdErrIsATerminal {
		// Clear out the line.
		Printf("${RESETLN}")
	} else {
		// Can't clear out the line, just move down to the next one.
		Printf("\n")
	}
	return nil
}

// update refreshes the display.
func (pr *progressReader) update() {
	if !StdErrIsATerminal {
		// Can't do interactive things, just print dots.
		Printf(strings.Repeat(".", (pr.current-pr.last)/1000))
		return
	}
	currentBytes := humanize.Bytes(uint64(pr.current))
	if pr.max == 0 {
		// we don't know how big the download is going to be, just show the downloaded size.
		// this shouldn't happen normally, our server does return the content size.
		Printf("${RESETLN}%s...", currentBytes)
		return
	}
	maxBytes := humanize.Bytes(uint64(pr.max))
	proportion := float64(pr.current) / float64(pr.max)
	percentage := 100.0 * proportion
	totalCols := pr.width - 30 // Pretty arbitrary amount of overhead to make sure we have space.
	currentPos := int(proportion * float64(totalCols))
	if currentPos > totalCols {
		currentPos = totalCols
	}
	before := strings.Repeat("=", currentPos)
	after := strings.Repeat(" ", totalCols-currentPos)
	Printf("${RESETLN}${BOLD_WHITE}%s / %s ${GREY}[%s>%s] ${BOLD_WHITE}%0.1f%%${RESET}", currentBytes, maxBytes, before, after, percentage)
}
