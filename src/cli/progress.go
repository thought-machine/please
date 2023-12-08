package cli

import (
	"io"
	"strings"

	"github.com/dustin/go-humanize"
)

// progressReader implements an io.ReadCloser that shows a progress bar in the terminal which
// updates as more is read.
// You should take care to close this in order to clean up the terminal afterwards.
// Note that it is fairly hardcoded to our use case right now (i.e. downloading Please tarballs),
// it probably doesn't generalise perfectly to things of arbitrary sizes.
type progressReader struct {
	reader                    io.ReadCloser
	message                   string
	current, last, max, width int
	interactive               bool
}

// This is the minimum column width we support. Below this it becomes hard to print sensible
// content (and later we can run into issues with repeats, see #967)
const minCols = 40

// NewProgressReader returns a new progress bar reader.
// total describes the total size of it, in bytes. It can be zero.
func NewProgressReader(reader io.ReadCloser, total int, message string) io.ReadCloser {
	r := &progressReader{
		message:     message,
		max:         total,
		reader:      reader,
		width:       80,
		interactive: StdErrIsATerminal,
	}
	if StdErrIsATerminal {
		_, cols, err := WindowSize()
		if err != nil {
			log.Warning("Error getting terminal size: %s", err)
			r.interactive = false
		}
		r.width = cols
		if cols < minCols { // Too small to print much of use at this point, and save a crash (see #967)
			r.interactive = false
		}
	}
	return r
}

// Read implements the io.Reader interface
func (pr *progressReader) Read(b []byte) (int, error) {
	n, err := pr.reader.Read(b)
	pr.current += n
	pr.update()
	pr.last = pr.current
	return n, err
}

// Close implements the io.Closer interface
// It closes the internal reader as well as cleaning up itself.
func (pr *progressReader) Close() error {
	if pr.interactive {
		// Clear out the line.
		Printf("${RESETLN}")
	} else {
		// Can't clear out the line, just move down to the next one.
		Printf("\n")
	}
	return pr.reader.Close()
}

// update refreshes the display.
func (pr *progressReader) update() {
	if !pr.interactive {
		// Can't do interactive things, just print dots.
		if pr.current > pr.last {
			Printf(strings.Repeat(".", (pr.current-pr.last)/100000))
		}
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
	totalCols := pr.width - minCols // Pretty arbitrary amount of overhead to make sure we have space.
	currentPos := int(proportion * float64(totalCols))
	if currentPos > totalCols {
		currentPos = totalCols
	}
	before := strings.Repeat("=", currentPos)
	after := strings.Repeat(" ", totalCols-currentPos)
	Printf("${RESETLN}${BOLD_WHITE}%s: %s / %s ${GREY}[%s>%s] ${BOLD_WHITE}%0.1f%%${RESET}", pr.message, currentBytes, maxBytes, before, after, percentage)
}
