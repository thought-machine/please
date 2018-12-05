package output

import (
	"bytes"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
)

// We only set the terminal title for terminals that at least claim to be xterm
// (note that most terminals do for compatibility; some report as xterm-color, hence HasPrefix)
var terminalClaimsToBeXterm = strings.HasPrefix(os.Getenv("TERM"), "xterm")

func display(state *core.BuildState, buildingTargets *[]buildingTarget, stop <-chan struct{}, done chan<- struct{}) {
	backend := cli.NewLogBackend(len(*buildingTargets))
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

	printLines(state, *buildingTargets, backend.MaxInteractiveRows, backend.Cols)
	outputLines := len(backend.Output)
	ticker := time.NewTicker(50 * time.Millisecond)
loop:
	for {
		select {
		case <-stop:
			break loop
		case <-ticker.C:
			moveToFirstLine(*buildingTargets, outputLines, backend.MaxInteractiveRows, state.Config.Display.SystemStats)
			printLines(state, *buildingTargets, backend.MaxInteractiveRows, backend.Cols)
			for _, line := range backend.Output {
				printf("\x1b[2K%s\n", line) // erase each line as we go
			}
			outputLines = len(backend.Output)
			setWindowTitle(state, true)
		}
	}
	ticker.Stop()
	setWindowTitle(state, false)
	// Clear it all out.
	moveToFirstLine(*buildingTargets, outputLines, backend.MaxInteractiveRows, state.Config.Display.SystemStats)
	printf("\x1b[0J") // Clear out to end of screen.
	backend.Deactivate()
	done <- struct{}{}
}

// moveToFirstLine resets back to the first line. Not as easy as you might think.
func moveToFirstLine(buildingTargets []buildingTarget, outputLines, maxInteractiveRows int, showingStats bool) {
	if maxInteractiveRows > len(buildingTargets) {
		maxInteractiveRows = len(buildingTargets)
	}
	if showingStats {
		maxInteractiveRows++
	}
	printf("\x1b[%dA", maxInteractiveRows+1+outputLines)
}

func printLines(state *core.BuildState, buildingTargets []buildingTarget, maxLines, cols int) {
	now := time.Now()
	printf("Building [%d/%d, %3.1fs]:\n", state.NumDone(), state.NumActive(), time.Since(state.StartTime).Seconds())
	if state.Config.Display.SystemStats {
		printStat("CPU use", state.Stats.CPU.Used, state.Stats.CPU.Count)
		printStat("I/O", state.Stats.CPU.IOWait, state.Stats.CPU.Count)
		printStat("Mem use", state.Stats.Memory.UsedPercent, 1)
		if state.Stats.NumWorkerProcesses > 0 {
			printf("  ${BOLD_WHITE}Worker processes: %d${RESET}", state.Stats.NumWorkerProcesses)
		}
		printf("${ERASE_AFTER}\n")
	}
	for i := 0; i < len(buildingTargets) && i < maxLines; i++ {
		buildingTargets[i].Lock()
		// Take a local copy of the structure, which isn't *that* big, so we don't need to retain the lock
		// while we do potentially blocking things like printing.
		target := buildingTargets[i].buildingTargetData
		buildingTargets[i].Unlock()
		label := target.Label.Parent()
		duration := now.Sub(target.Started).Seconds()
		if target.Active && target.Target != nil && target.Target.ShowProgress && target.Target.Progress > 0.0 {
			if target.Target.Progress > 1.0 && target.Target.Progress < 100.0 && target.Target.Progress != target.LastProgress {
				proportionDone := target.Target.Progress / 100.0
				perPercent := float32(duration) / proportionDone
				buildingTargets[i].Eta = time.Duration(perPercent * (1.0 - proportionDone) * float32(time.Second)).Truncate(time.Second)
				buildingTargets[i].LastProgress = target.Target.Progress
			}
			if target.Eta > 0 {
				lprintf(cols, "${BOLD_WHITE}=> [%4.1fs] ${RESET}%s%s ${BOLD_WHITE}%s${RESET} (%.1f%%%%, est %s remaining)${ERASE_AFTER}\n",
					duration, target.Colour, label, target.Description, target.Target.Progress, target.Eta)
			} else {
				lprintf(cols, "${BOLD_WHITE}=> [%4.1fs] ${RESET}%s%s ${BOLD_WHITE}%s${RESET} (%.1f%%%% complete)${ERASE_AFTER}\n",
					duration, target.Colour, label, target.Description, target.Target.Progress)
			}
		} else if target.Active {
			lprintf(cols, "${BOLD_WHITE}=> [%4.1fs] ${RESET}%s%s ${BOLD_WHITE}%s${ERASE_AFTER}\n",
				duration, target.Colour, label, target.Description)
		} else if time.Since(target.Finished).Seconds() < 0.5 {
			// Only display finished targets for half a second after they're done.
			duration := target.Finished.Sub(target.Started).Seconds()
			if target.Failed {
				lprintf(cols, "${BOLD_RED}=> [%4.1fs] ${RESET}%s%s ${BOLD_RED}Failed${ERASE_AFTER}\n",
					duration, target.Colour, label)
			} else if target.Cached {
				lprintf(cols, "${BOLD_WHITE}=> [%4.1fs] ${RESET}%s%s ${BOLD_GREY}%s${ERASE_AFTER}\n",
					duration, target.Colour, label, target.Description)
			} else {
				lprintf(cols, "${BOLD_WHITE}=> [%4.1fs] ${RESET}%s%s ${WHITE}%s${ERASE_AFTER}\n",
					duration, target.Colour, label, target.Description)
			}
		} else {
			printf("${BOLD_GREY}=|${ERASE_AFTER}\n")
		}
	}
	printf("${RESET}")
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
func lprintf(cols int, format string, args ...interface{}) {
	printf(lprintfPrepare(cols, format, args...))
}

func lprintfPrepare(cols int, format string, args ...interface{}) string {
	s := fmt.Sprintf(format, args...)
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
