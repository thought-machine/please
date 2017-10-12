// For writing out JSON trace files which Chrome can interpret nicely for us.
// See https://docs.google.com/document/d/1CvAClvFfyA5R-PhYUmn5OOQtYMH4h6I0nSsKchNAySU

package output

import "encoding/json"
import "fmt"
import "os"

import "core"

var traces = make([]traceEntry, 0, 1000)

func addTrace(result *core.BuildResult, previous core.BuildLabel, active bool) {
	// It's a bit fiddly to keep all the phases in line here.
	if result.Label != previous {
		traces = append(traces, translateEvent(result, "B"))
	} else if !active {
		traces = append(traces, translateEvent(result, "E"))
	} else {
		traces = append(traces, translateEvent(result, "E"))
		traces = append(traces, translateEvent(result, "B"))
	}
}

func writeTrace(traceFile string) {
	file, err := os.Create(traceFile)
	if err != nil {
		log.Errorf("Couldn't create trace file: %s", err)
		return
	}
	defer file.Close()
	file.Write(formatTrace())
}

func formatTrace() []byte {
	var out traceObjectFormat
	out.OtherData.Version = "Please v" + core.PleaseVersion.String()
	out.TraceEvents = traces
	data, err := json.Marshal(out)
	if err != nil {
		log.Errorf("Error serialising JSON trace data: %s", err)
	}
	return data
}

func translateEvent(result *core.BuildResult, phase string) traceEntry {
	entry := traceEntry{
		Name: result.Label.String(),
		Cat:  result.Status.Category(),
		Ph:   phase,
		Pid:  0, // This isn't really important, there's only one process.
		Ts:   result.Time.UnixNano() / 1000,
	}
	entry.Tid = fmt.Sprintf("Builder %d", result.ThreadID)
	entry.Args.Description = result.Description
	if result.Err != nil {
		entry.Args.Err = fmt.Sprintf("%s", result.Err)
	}
	return entry
}

type traceObjectFormat struct {
	TraceEvents []traceEntry `json:"traceEvents"`
	OtherData   struct {
		Version string `json:"version"`
	} `json:"otherData"`
	// Ignoring other properties for now.
}

type traceEntry struct {
	Name string `json:"name"`
	Cat  string `json:"cat"`
	Ph   string `json:"ph"`
	Pid  int32  `json:"pid"`
	Tid  string `json:"tid"`
	Ts   int64  `json:"ts"`
	Args struct {
		Description string `json:"description"`
		Err         string `json:"err"`
	} `json:"args"`
}
