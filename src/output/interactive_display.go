package output

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/dustin/go-humanize"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
)

// We only set the terminal title for terminals that at least claim to be xterm
// (note that most terminals do for compatibility; some report as xterm-color, hence HasPrefix)
var terminalClaimsToBeXterm = strings.HasPrefix(os.Getenv("TERM"), "xterm")

type displayer struct {
	state                                               *core.BuildState
	targets                                             []buildingTarget
	numWorkers, maxWorkers, numRemote, maxRows, maxCols int
	stats                                               bool
	lines, lastLines                                    int // mutable - records how many rows we've printed this time
}

func display(ctx context.Context, state *core.BuildState, buildingTargets []buildingTarget) {
	backend := cli.NewLogBackend(state.Config.Display.MaxWorkers)
	go func() {
		sig := make(chan os.Signal, 10)
		signal.Notify(sig, syscall.SIGWINCH)
		for {
			<-sig
			recalcWindowSize(backend)
		}
	}()
	recalcWindowSize(backend)
	backend.SetActive()

	d := &displayer{
		state:      state,
		targets:    buildingTargets,
		numWorkers: state.Config.Please.NumThreads,
		maxWorkers: state.Config.Display.MaxWorkers,
		numRemote:  state.Config.NumRemoteExecutors(),
		maxRows:    backend.MaxInteractiveRows,
		maxCols:    backend.Cols,
		stats:      state.Config.Display.SystemStats,
	}

	d.printLines()
	d.run(ctx, backend)
	setWindowTitle(state, false)
	// Clear it all out.
	d.moveToFirstLine()
	printf("${CLEAR_END}")
	backend.Deactivate()
}

func (d *displayer) run(ctx context.Context, backend *cli.LogBackend) {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	done := ctx.Done()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			d.moveToFirstLine()
			d.printLines()
			for _, line := range backend.Output {
				printf("${ERASE_AFTER}%s\n", line)
				d.lines++
			}
			// Clean out any lines that were visible last time but are not now.
			if d.lines < d.lastLines {
				for i := d.lines; i < d.lastLines; i++ {
					printf("${ERASE_AFTER}\n")
				}
				printf("\x1b[%dA", d.lastLines-d.lines) // Move back up again
			}
			setWindowTitle(d.state, true)
		}
	}
}

// moveToFirstLine resets back to the first line.
func (d *displayer) moveToFirstLine() {
	printf("\x1b[%dA", d.lines)
	d.lastLines = d.lines
	d.lines = 0
}

func (d *displayer) printLines() {
	now := time.Now()
	printf("Building [%d/%d, %3.1fs]:\n", d.state.NumDone(), d.state.NumActive(), time.Since(d.state.StartTime).Seconds())
	d.lines++
	if d.stats {
		printStat("CPU use", d.state.Stats.CPU.Used, d.state.Stats.CPU.Count)
		printStat("I/O", d.state.Stats.CPU.IOWait, d.state.Stats.CPU.Count)
		printStat("Mem use", d.state.Stats.Memory.UsedPercent, 1)
		if d.state.Stats.NumWorkerProcesses > 0 {
			printf("  ${BOLD_WHITE}Worker processes: %d${RESET}", d.state.Stats.NumWorkerProcesses)
		}
		if d.state.RemoteClient != nil {
			in, out, _, _ := d.state.RemoteClient.DataRate()
			printf("  ${BOLD_WHITE}RPC data in: %6s/s out: %6s/s${RESET}", humanize.Bytes(uint64(in)), humanize.Bytes(uint64(out)))
		}
		printf("${ERASE_AFTER}\n")
		d.lines++
	}
	workers := 0
	anyRemote := d.numRemote > 0
	for i := 0; i < d.numWorkers && i < d.maxRows && workers < d.maxWorkers; i++ {
		workers += d.printRow(i, now, anyRemote)
		d.lines++
	}
	if anyRemote {
		active := d.numRemoteActive()
		printf("Remote processes [%3d/%3d active]:   ${ERASE_AFTER}\n", active, d.numRemote)
		d.lines++
		for i := 0; i < d.numRemote && d.lines < d.maxRows && workers < d.maxWorkers; i++ {
			workers += d.printRow(d.numWorkers+i, now, true)
			d.lines++
		}
		if workers < active {
			printf("${RESET}   [%2d more...]${ERASE_AFTER}\n", active-workers)
			d.lines++
		}
	}
	printf("${RESET}")
}

func (d *displayer) numRemoteActive() int {
	count := 0
	for i := 0; i < d.numRemote; i++ {
		if d.targets[d.numWorkers+i].Active {
			count++
		}
	}
	return count
}

func (d *displayer) printRow(i int, now time.Time, remote bool) int {
	d.targets[i].Lock()
	// Take a local copy of the structure, which isn't *that* big, so we don't need to retain the lock
	// while we do potentially blocking things like printing.
	target := d.targets[i].buildingTargetData
	d.targets[i].Unlock()
	label := target.Label.Parent()
	duration := now.Sub(target.Started).Seconds()
	if target.Active && target.Target != nil && target.Target.ShowProgress && target.Target.Progress > 0.0 {
		if target.Target.Progress > 1.0 && target.Target.Progress < 100.0 && target.Target.Progress != target.LastProgress {
			proportionDone := target.Target.Progress / 100.0
			perPercent := float32(duration) / proportionDone
			d.targets[i].Eta = time.Duration(perPercent * (1.0 - proportionDone) * float32(time.Second)).Truncate(time.Second)
			d.targets[i].LastProgress = target.Target.Progress
		}
		if target.Eta > 0 {
			d.printf("${BOLD_WHITE}=> [%4.1fs] ${RESET}%s%s ${BOLD_WHITE}%s${RESET} (%.1f%%, est %s remaining)${ERASE_AFTER}\n",
				duration, target.Colour, label, target.Description, target.Target.Progress, target.Eta)
		} else {
			d.printf("${BOLD_WHITE}=> [%4.1fs] ${RESET}%s%s ${BOLD_WHITE}%s${RESET} (%.1f%% complete)${ERASE_AFTER}\n",
				duration, target.Colour, label, target.Description, target.Target.Progress)
		}
	} else if target.Active {
		d.printf("${BOLD_WHITE}=> [%4.1fs] ${RESET}%s%s ${BOLD_WHITE}%s${ERASE_AFTER}\n",
			duration, target.Colour, label, target.Description)
	} else if time.Since(target.Finished).Seconds() < 0.5 {
		// Only display finished targets for half a second after they're done.
		duration := target.Finished.Sub(target.Started).Seconds()
		if target.Failed {
			d.printf("${BOLD_RED}=> [%4.1fs] ${RESET}%s%s ${BOLD_RED}Failed${ERASE_AFTER}\n",
				duration, target.Colour, label)
		} else if target.Cached {
			d.printf("${BOLD_WHITE}=> [%4.1fs] ${RESET}%s%s ${BOLD_GREY}%s${ERASE_AFTER}\n",
				duration, target.Colour, label, target.Description)
		} else {
			d.printf("${BOLD_WHITE}=> [%4.1fs] ${RESET}%s%s ${WHITE}%s${ERASE_AFTER}\n",
				duration, target.Colour, label, target.Description)
		}
	} else if !remote {
		printf("${BOLD_GREY}=|${ERASE_AFTER}\n")
	} else {
		d.lines-- // Didn't print it
		return 0
	}
	return 1
}

// printStat prints a single statistic with appropriate colours.
func printStat(caption string, stat float64, multiplier int) {
	colour := "${BOLD_GREEN}"
	if stat > 80.0*float64(multiplier) {
		colour = "${BOLD_RED}"
	} else if stat > 60.0*float64(multiplier) {
		colour = "${BOLD_YELLOW}"
	}
	printf("  ${BOLD_WHITE}%s:${RESET} %s%5.1f%%${RESET}", caption, colour, stat)
}

func recalcWindowSize(backend *cli.LogBackend) {
	rows, cols, _ := cli.WindowSize()
	backend.Lock()
	defer backend.Unlock()
	backend.Rows = rows - 4 // Give a little space at the edge for any off-by-ones
	backend.Cols = cols
	backend.RecalcLines()
}

// Limited-length printf that respects current window width.
// Output is truncated at the middle to fit within 'cols'.
func (d *displayer) printf(format string, args ...interface{}) {
	fmt.Fprint(os.Stderr, lprintfPrepare(d.maxCols, os.Expand(fmt.Sprintf(format, args...), replace)))
}

func lprintfPrepare(cols int, s string) string {
	if len(s) < cols {
		return s // it's short enough, nice and simple
	}
	// Okay, it's too long. Tricky thing: ANSI escape codes don't count for width
	// so we need to count without those. Bonus: make an effort to be unicode-aware.
	var b bytes.Buffer
	written := 0
	inAnsiCode := false
	for _, rune := range s {
		if inAnsiCode {
			b.WriteRune(rune)
			if rune == 'm' {
				inAnsiCode = false
			}
		} else if rune == '\x1b' {
			b.WriteRune(rune)
			inAnsiCode = true
		} else if rune == '\n' {
			b.WriteRune(rune)
		} else if written == cols-3 {
			b.WriteString("...")
			written += 3
		} else if written < cols-3 {
			b.WriteRune(rune)
			written++
		}
	}
	return b.String()
}

// setWindowTitle sets the title of the current shell window based on the current build state.
func setWindowTitle(state *core.BuildState, running bool) {
	if !state.Config.Display.UpdateTitle {
		return
	}
	if running {
		SetWindowTitle("plz: finishing up")
	} else {
		SetWindowTitle(fmt.Sprintf("plz: %d / %d tasks, %3.1fs", state.NumDone(), state.NumActive(), time.Since(state.StartTime).Seconds()))
	}
}

// SetWindowTitle sets the title of the current shell window.
func SetWindowTitle(title string) {
	if cli.StdErrIsATerminal && terminalClaimsToBeXterm {
		os.Stderr.Write([]byte(fmt.Sprintf("\033]0;%s\007", title)))
	}
}
