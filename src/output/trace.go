// For writing out JSON trace files which Chrome can interpret nicely for us.
// See https://docs.google.com/document/d/1CvAClvFfyA5R-PhYUmn5OOQtYMH4h6I0nSsKchNAySU

package output

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/thought-machine/please/src/cli/logging"
	"github.com/thought-machine/please/src/core"
)

var log = logging.Log

// A traceWriter is responsible for writing the JSON trace info.
type traceWriter struct {
	b      *bufio.Writer
	f      *os.File
	active map[core.BuildLabel]struct{}
	first  bool // have we written the first record
}

// newTraceWriter returns a new traceWriter writing to the given file.
// The filename may be empty in which case it will silently discard all information given.
func newTraceWriter(filename string) *traceWriter {
	f, err := os.Create(filename)
	if err != nil {
		log.Errorf("Couldn't create trace file: %s", err)
		return &traceWriter{}
	}
	b := bufio.NewWriter(f)
	// To be well-formed the file has to start with a [ in JSON array format.
	// This is more robust than the object format and we don't write anything of use into that anyway.
	b.Write([]byte("[\n"))
	return &traceWriter{
		b:      b,
		f:      f,
		active: map[core.BuildLabel]struct{}{},
	}
}

// Close closes this write and any associated files.
func (tw *traceWriter) Close() error {
	if _, err := tw.b.Write([]byte{'\n', ']', '\n'}); err != nil {
		return err
	} else if err := tw.b.Flush(); err != nil {
		return err
	}
	return tw.f.Close()
}

// AddTrace adds a single trace to this writer.
func (tw *traceWriter) AddTrace(threadID int, result *core.BuildResult, active bool) {
	// It's a bit fiddly to keep all the phases in line here.
	if !active {
		tw.writeEvent(threadID, result, "E") // end the span for this target
		delete(tw.active, result.Label)
	} else if _, present := tw.active[result.Label]; !present {
		tw.writeEvent(threadID, result, "B") // begin a new span for this target
		tw.active[result.Label] = struct{}{}
	} else {
		tw.writeEvent(threadID, result, "i")
	}
}

func (tw *traceWriter) writeEvent(threadID int, result *core.BuildResult, phase string) {
	if !tw.first {
		tw.first = true
	} else {
		tw.b.Write([]byte{',', '\n'})
	}
	entry := traceEntry{
		Name:  result.Label.String(),
		Cat:   result.Status.Category(),
		Ph:    phase,
		Pid:   0, // This isn't really important, there's only one process.
		Ts:    result.Time.UnixNano() / 1000,
		Cname: "thread_state_runnable", // Colours have to fit available names, this is blueish.
	}
	entry.Tid = fmt.Sprintf("Builder %d", threadID)
	entry.Args.Description = result.Description
	if result.Err != nil {
		entry.Args.Err = fmt.Sprintf("%s", result.Err)
		entry.Cname = "terrible"
	} else if entry.Cat == "Test" {
		entry.Cname = "good"
	}
	b, _ := json.Marshal(entry)
	tw.b.Write(b)
}

type traceEntry struct {
	Name  string `json:"name"`
	Cat   string `json:"cat"`
	Ph    string `json:"ph"`
	Pid   int32  `json:"pid"`
	Tid   string `json:"tid"`
	Ts    int64  `json:"ts"`
	Cname string `json:"cname,omitempty"`
	Args  struct {
		Description string `json:"description"`
		Err         string `json:"err,omitempty"`
	} `json:"args"`
}
