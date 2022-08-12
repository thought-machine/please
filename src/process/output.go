package process

import (
	"fmt"
	"os"
)

// An OutputMode defines how we emit output from subprocesses.
type OutputMode string

const (
	// Default emits everything to stdout/stderr, interleaved with no buffering.
	Default OutputMode = "default"
	// Quiet suppresses all output apart from for failed subprocesses.
	Quiet OutputMode = "quiet"
	// GroupImmediate displays output from each process as it completes.
	GroupImmediate OutputMode = "group_immediate"
)

// RunWithOutput runs a subprocess with the given output mechanism.
// The actual running is done via a callback which should return the output and any error;
// it will need to arrange to attach stdout/err appropriately for Default mode.
func RunWithOutput(mode OutputMode, label string, f func() ([]byte, error)) error {
	switch mode {
	case GroupImmediate:
		out, err := f()
		fmt.Println(label)
		if err == nil {
			os.Stdout.Write(out)
		}
		return err
	case Quiet, Default:
		fallthrough
	default:
		_, err := f()
		return err
	}

}
