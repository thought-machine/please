package process

import (
	"io"
	"regexp"
	"strconv"
)

// A progressWriter implements progress display for a target; when it's written to
// it attempts to infer progress from the output.
// Right now the heuristic to measure progress is pretty simple but we may expand it later.
type progressWriter struct {
	t Target
	w io.Writer
	p *float32
}

var progressRegex = regexp.MustCompile(`\[ *([0-9]+)%\]`)

func newProgressWriter(t Target, p *float32, w io.Writer) io.Writer {
	return &progressWriter{t: t, p: p, w: w}
}

// Write implements the io.Writer interface
func (w *progressWriter) Write(b []byte) (int, error) {
	if matches := progressRegex.FindAllSubmatch(b, -1); matches != nil {
		if f, err := strconv.ParseFloat(string(matches[len(matches)-1][1]), 32); err == nil {
			*w.p = float32(f)
			w.t.SetProgress(*w.p)
		}
	}
	return w.w.Write(b)
}
