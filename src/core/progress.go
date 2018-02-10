package core

import (
	"io"
	"regexp"
	"strconv"
)

// A progressWriter implements progress display for a target; when it's written to
// it attempts to infer progress from the output.
// Right now the heuristic to measure progress is pretty simple but we may expand it later.
type progressWriter struct {
	t  *BuildTarget
	w  io.Writer
	re *regexp.Regexp
}

func newProgressWriter(t *BuildTarget, w io.Writer) io.Writer {
	return &progressWriter{t: t, w: w, re: regexp.MustCompile(`\[ *([0-9]+)%\]`)}
}

// Write implements the io.Writer interface
func (w *progressWriter) Write(b []byte) (int, error) {
	if matches := w.re.FindAllSubmatch(b, -1); matches != nil {
		if f, err := strconv.ParseFloat(string(matches[len(matches)-1][1]), 32); err == nil {
			w.t.Progress = float32(f)
		}
	}
	return w.w.Write(b)
}
