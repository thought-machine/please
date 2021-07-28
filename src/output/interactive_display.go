package output

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
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
	buf, lineBuf                                        bytes.Buffer
}

func display(ctx context.Context, state *core.BuildState, buildingTargets []buildingTarget) {
	cli.CurrentBackend.SetPassthrough(false, state.Config.Display.MaxWorkers)
	defer cli.CurrentBackend.SetPassthrough(true, state.Config.Display.MaxWorkers)

	d := &displayer{
		state:      state,
		targets:    buildingTargets,
		numWorkers: state.Config.Please.NumThreads,
		maxWorkers: state.Config.Display.MaxWorkers,
		numRemote:  state.Config.NumRemoteExecutors(),
		stats:      state.Config.Display.SystemStats,
	}

	d.printLines()
	d.run(ctx)
	setWindowTitle(state, false)
	// Clear it all out.
	d.moveToFirstLine()
	d.printf("${CLEAR_END}")
}

func (d *displayer) run(ctx context.Context) {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	done := ctx.Done()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			d.maxRows, d.maxCols = cli.CurrentBackend.MaxDimensions()
			d.moveToFirstLine()
			d.printLines()
			for _, line := range cli.CurrentBackend.Output() {
				d.printf("${ERASE_AFTER}%s\n", line)
				d.lines++
			}
			// Clean out any lines that were visible last time but are not now.
			if d.lines < d.lastLines {
				for i := d.lines; i < d.lastLines; i++ {
					d.printf("${ERASE_AFTER}\n")
				}
				d.printf("\x1b[%dA", d.lastLines-d.lines) // Move back up again
			}
			setWindowTitle(d.state, true)
			os.Stderr.Write(d.buf.Bytes())
			os.Stderr.Sync()
			d.buf.Reset()
		}
	}
}

// moveToFirstLine resets back to the first line.
func (d *displayer) moveToFirstLine() {
	d.printf("\x1b[%dA", d.lines)
	d.lastLines = d.lines
	d.lines = 0
}

func (d *displayer) printLines() {
	now := time.Now()
	d.printf("Building [%d/%d, %3.1fs]:\n", d.state.NumDone(), d.state.NumActive(), time.Since(d.state.StartTime).Seconds())
	d.lines++
	if d.stats {
		d.printStat("CPU use", d.state.Stats.CPU.Used, d.state.Stats.CPU.Count)
		d.printStat("I/O", d.state.Stats.CPU.IOWait, d.state.Stats.CPU.Count)
		d.printStat("Mem use", d.state.Stats.Memory.UsedPercent, 1)
		if d.state.Stats.NumWorkerProcesses > 0 {
			d.printf("  ${BOLD_WHITE}Worker processes: %d${RESET}", d.state.Stats.NumWorkerProcesses)
		}
		if d.state.RemoteClient != nil {
			in, out, _, _ := d.state.RemoteClient.DataRate()
			d.printf("  ${BOLD_WHITE}RPC data in: %6s/s out: %6s/s${RESET}", humanize.Bytes(uint64(in)), humanize.Bytes(uint64(out)))
		}
		d.printf("${ERASE_AFTER}\n")
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
		d.printf("Remote processes [%3d/%3d active]:   ${ERASE_AFTER}\n", active, d.numRemote)
		d.lines++
		for i := 0; i < d.numRemote && d.lines < d.maxRows && workers < d.maxWorkers; i++ {
			workers += d.printRow(d.numWorkers+i, now, true)
			d.lines++
		}
		if workers < active {
			d.printf("${RESET}   [%2d more...]${ERASE_AFTER}\n", active-workers)
			d.lines++
		}
	}
	d.printf("${RESET}")
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
		d.printf("${BOLD_GREY}=|${ERASE_AFTER}\n")
	} else {
		d.lines-- // Didn't print it
		return 0
	}
	return 1
}

// printStat prints a single statistic with appropriate colours.
func (d *displayer) printStat(caption string, stat float64, multiplier int) {
	colour := "${BOLD_GREEN}"
	if stat > 80.0*float64(multiplier) {
		colour = "${BOLD_RED}"
	} else if stat > 60.0*float64(multiplier) {
		colour = "${BOLD_YELLOW}"
	}
	d.printf("  ${BOLD_WHITE}%s:${RESET} %s%5.1f%%${RESET}", caption, colour, stat)
}

// Limited-length printf that respects current window width.
// Output is truncated at the middle to fit within 'cols'.
func (d *displayer) printf(format string, args ...interface{}) {

	fmt.Fprint(&d.buf, d.lprintfPrepare(d.maxCols, os.Expand(fmt.Sprintf(format, args...), replace)))
}

func (d *displayer) lprintfPrepare(cols int, s string) string {
	if len(s) < cols {
		return s // it's short enough, nice and simple
	}
	// Okay, it's too long. Tricky thing: ANSI escape codes don't count for width
	// so we need to count without those. Bonus: make an effort to be unicode-aware.
	written := 0
	inAnsiCode := false
	d.lineBuf.Reset()
	for _, rune := range s {
		if inAnsiCode {
			d.lineBuf.WriteRune(rune)
			if rune == 'm' {
				inAnsiCode = false
			}
		} else if rune == '\x1b' {
			d.lineBuf.WriteRune(rune)
			inAnsiCode = true
		} else if rune == '\n' {
			d.lineBuf.WriteRune(rune)
		} else if written == cols-3 {
			d.lineBuf.WriteString("...")
			written += 3
		} else if written < cols-3 {
			d.lineBuf.WriteRune(rune)
			written++
		}
	}
	return d.lineBuf.String()
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
