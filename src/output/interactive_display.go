package output

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/dustin/go-humanize"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
)

// We only set the terminal title for terminals that at least claim to be xterm
// (note that most terminals do for compatibility; some report as xterm-color, hence HasPrefix)
var terminalClaimsToBeXterm = strings.HasPrefix(os.Getenv("TERM"), "xterm")

// A displayer is the interface to display things on screen while a build is running.
type displayer interface {
	Update(targets []buildingTarget)
	Close()
	Frequency() time.Duration
}

func setupDisplayer(state *core.BuildState, plain bool) displayer {
	if plain {
		return &plainDisplay{state: state}
	}
	cli.CurrentBackend.SetPassthrough(false, state.Config.Display.MaxWorkers, state.Watch)
	return &interactiveDisplay{
		state:      state,
		numWorkers: state.Config.Please.NumThreads,
		maxWorkers: state.Config.Display.MaxWorkers,
		numRemote:  state.Config.NumRemoteExecutors(),
		stats:      state.Config.Display.SystemStats,
	}
}

type plainDisplay struct {
	state *core.BuildState
}

func (d *plainDisplay) Update(targets []buildingTarget) {
	localbusy, remotebusy := countActive(targets)
	log.Notice("Build running for %s, %d / %d tasks done, %s busy, parsing %d BUILD files", time.Since(d.state.StartTime).Round(time.Second), d.state.NumDone(), d.state.NumActive(), pluralise(localbusy+remotebusy, "worker", "workers"), d.state.Parses().Load())
}

func countActive(targets []buildingTarget) (local int, remote int) {
	for _, t := range targets {
		if t.Active {
			if t.Remote {
				remote++
			} else {
				local++
			}
		}
	}
	return local, remote
}

func (d *plainDisplay) Frequency() time.Duration {
	return 10 * time.Second
}

func (d *plainDisplay) Close() {}

type interactiveDisplay struct {
	state                                               *core.BuildState
	numWorkers, maxWorkers, numRemote, maxRows, maxCols int
	stats                                               bool
	lines, lastLines                                    int // mutable - records how many rows we've printed this time
	buf, lineBuf                                        bytes.Buffer
}

func (d *interactiveDisplay) Close() {
	setWindowTitle(d.state, false)
	d.moveToFirstLine()
	d.printf("${CLEAR_END}")
	d.flush()
	cli.CurrentBackend.SetPassthrough(true, d.state.Config.Display.MaxWorkers, d.state.Watch)
}

func (d *interactiveDisplay) Frequency() time.Duration {
	return 50 * time.Millisecond
}

func (d *interactiveDisplay) Update(targets []buildingTarget) {
	d.maxRows, d.maxCols = cli.CurrentBackend.MaxDimensions()
	d.moveToFirstLine()
	d.printLines(targets)
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
	d.flush()
}

// moveToFirstLine resets back to the first line.
func (d *interactiveDisplay) moveToFirstLine() {
	if d.lines > 0 {
		d.printf("\x1b[%dA", d.lines)
		d.lastLines = d.lines
		d.lines = 0
	}
}

func (d *interactiveDisplay) printLines(targets []buildingTarget) {
	now := time.Now()
	localActive, remoteActive := countActive(targets)
	totalActive := localActive + remoteActive
	if d.numRemote > 0 {
		d.printf("Building [%d/%d, %2d/%d local, %3d/%d remote, %3.1fs]:\n", d.state.NumDone(), d.state.NumActive(), localActive, d.numWorkers, remoteActive, d.numRemote, time.Since(d.state.StartTime).Seconds())
	} else {
		d.printf("Building [%d/%d, %3.1fs]:\n", d.state.NumDone(), d.state.NumActive(), time.Since(d.state.StartTime).Seconds())
	}
	d.lines++
	if d.stats {
		stats := d.state.SystemStats()
		d.printStat("CPU use", stats.CPU.Used, stats.CPU.Count)
		d.printStat("I/O", stats.CPU.IOWait, stats.CPU.Count)
		d.printStat("Mem use", stats.Memory.UsedPercent, 1)
		if stats.NumWorkerProcesses > 0 {
			d.printf("  ${BOLD_WHITE}Worker processes: %d${RESET}", stats.NumWorkerProcesses)
		}
		if d.state.RemoteClient != nil {
			in, out, _, _ := d.state.RemoteClient.DataRate()
			d.printf("  ${BOLD_WHITE}RPC data in: %6s/s out: %6s/s${RESET}", humanize.Bytes(uint64(in)), humanize.Bytes(uint64(out)))
		}
		d.printf("${ERASE_AFTER}\n")
		d.lines++
	}
	workers := 0
	for i := 0; i < len(targets) && d.lines < d.maxRows && workers < d.maxWorkers; i++ {
		workers += d.printRow(&targets[i], now)
		d.lines++
	}
	if workers < totalActive {
		d.printf("${RESET}   [%2d more...]${ERASE_AFTER}\n", totalActive-workers)
		d.lines++
	}
	d.printf("${RESET}")
}

func (d *interactiveDisplay) printRow(target *buildingTarget, now time.Time) int {
	label := target.Label.Parent()
	duration := now.Sub(target.Started).Seconds()
	progress := float32(0.0)
	if target.Target != nil {
		progress = target.Target.Progress.Load()
	}
	c := '='
	if target.Remote {
		c = '-'
	}
	if target.Active && target.Target.ShouldShowProgress() && progress > 0.0 {
		if progress > 1.0 && progress < 100.0 && progress != target.LastProgress {
			proportionDone := progress / 100.0
			perPercent := float32(duration) / proportionDone
			target.Eta = time.Duration(perPercent * (1.0 - proportionDone) * float32(time.Second)).Truncate(time.Second)
			target.LastProgress = progress
			fileSize := atomic.LoadUint64(&target.Target.FileSize)
			if fileSize > 0 {
				// Round the download speed to a multiple of 10kB which makes the display jitter around less
				const quantum = 10.0 * 1000.0
				bps := float64(fileSize) * float64(proportionDone) / duration
				target.BPS = float32(math.Round(bps/quantum) * quantum)
			}
		}
		if target.Eta > 0 {
			if target.BPS != 0.0 {
				d.printf("${BOLD_WHITE}%c> [%4.1fs] ${RESET}%s%s ${BOLD_WHITE}%s${RESET} (%.1f%%, %s/s, est %s remaining)${ERASE_AFTER}\n",
					c, duration, target.Colour, label, target.Description, target.Target.Progress.Load(), humanize.Bytes(uint64(target.BPS)), target.Eta)
			} else {
				d.printf("${BOLD_WHITE}%c> [%4.1fs] ${RESET}%s%s ${BOLD_WHITE}%s${RESET} (%.1f%%, est %s remaining)${ERASE_AFTER}\n",
					c, duration, target.Colour, label, target.Description, target.Target.Progress.Load(), target.Eta)
			}
		} else {
			d.printf("${BOLD_WHITE}%c> [%4.1fs] ${RESET}%s%s ${BOLD_WHITE}%s${RESET} (%.1f%% complete)${ERASE_AFTER}\n",
				c, duration, target.Colour, label, target.Description, target.Target.Progress.Load())
		}
	} else if target.Active {
		d.printf("${BOLD_WHITE}%c> [%4.1fs] ${RESET}%s%s ${BOLD_WHITE}%s${ERASE_AFTER}\n",
			c, duration, target.Colour, label, target.Description)
	} else if time.Since(target.Finished).Seconds() < 0.5 {
		// Only display finished targets for half a second after they're done.
		duration := target.Finished.Sub(target.Started).Seconds()
		if target.Failed {
			d.printf("${BOLD_RED}%c> [%4.1fs] ${RESET}%s%s ${BOLD_RED}Failed${ERASE_AFTER}\n",
				c, duration, target.Colour, label)
		} else if target.Cached {
			d.printf("${BOLD_WHITE}%c> [%4.1fs] ${RESET}%s%s ${BOLD_GREY}%s${ERASE_AFTER}\n",
				c, duration, target.Colour, label, target.Description)
		} else {
			d.printf("${BOLD_WHITE}%c> [%4.1fs] ${RESET}%s%s ${WHITE}%s${ERASE_AFTER}\n",
				c, duration, target.Colour, label, target.Description)
		}
	} else {
		d.lines-- // Didn't print it
		return 0
	}
	return 1
}

// printStat prints a single statistic with appropriate colours.
func (d *interactiveDisplay) printStat(caption string, stat float64, multiplier int) {
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
func (d *interactiveDisplay) printf(format string, args ...interface{}) {
	fmt.Fprint(&d.buf, d.lprintfPrepare(d.maxCols, os.Expand(fmt.Sprintf(format, args...), replace)))
}

// flush prints the current buffer to stderr and resets it.
func (d *interactiveDisplay) flush() {
	os.Stderr.Write(d.buf.Bytes())
	d.buf.Reset()
}

func (d *interactiveDisplay) lprintfPrepare(cols int, s string) string {
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
