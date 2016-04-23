package output

import (
	"bytes"
	"container/list"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"gopkg.in/op/go-logging.v1"

	"core"
)

// We only set the terminal title for terminals that at least claim to be xterm
// (note that most terminals do for compatibility; some report as xterm-color, hence HasPrefix)
var terminalClaimsToBeXterm = strings.HasPrefix(os.Getenv("TERM"), "xterm")

func display(state *core.BuildState, buildingTargets *[]buildingTarget, stop <-chan interface{}, done chan<- interface{}) {
	backend := logBackend{InteractiveRows: len(*buildingTargets), MaxRecords: 10, LogMessages: list.New(), Formatter: logFormatter()}
	go func() {
		sig := make(chan os.Signal, 10)
		signal.Notify(sig, syscall.SIGWINCH)
		for {
			<-sig
			recalcWindowSize(&backend)
		}
	}()
	recalcWindowSize(&backend)
	setLogBackend(logBackendFacade{&backend})

	printLines(state, *buildingTargets, backend.MaxInteractiveRows, backend.Cols)
	outputLines := len(backend.Output)
	ticker := time.NewTicker(50 * time.Millisecond)
loop:
	for {
		select {
		case <-stop:
			break loop
		case <-ticker.C:
			moveToFirstLine(*buildingTargets, outputLines, backend.MaxInteractiveRows)
			printLines(state, *buildingTargets, backend.MaxInteractiveRows, backend.Cols)
			for _, line := range backend.Output {
				printf("\x1b[2K%s\n", line) // erase each line as we go
			}
			outputLines = len(backend.Output)
			setWindowTitle(state)
		}
	}
	ticker.Stop()
	setWindowTitle(nil)
	// Clear it all out.
	moveToFirstLine(*buildingTargets, outputLines, backend.MaxInteractiveRows)
	printf("\x1b[0J") // Clear out to end of screen.
	setLogBackend(logging.NewLogBackend(os.Stderr, "", 0))
	done <- struct{}{}
}

// moveToFirstLine resets back to the first line. Not as easy as you might think.
func moveToFirstLine(buildingTargets []buildingTarget, outputLines, maxInteractiveRows int) {
	if maxInteractiveRows > len(buildingTargets) {
		maxInteractiveRows = len(buildingTargets)
	}
	printf("\x1b[%dF", maxInteractiveRows+1+outputLines)
}

func printLines(state *core.BuildState, buildingTargets []buildingTarget, maxLines, cols int) {
	now := time.Now()
	printf("Building [%d/%d, %3.1fs]:\n", state.NumDone(), state.NumActive(), time.Since(startTime).Seconds())
	for i := 0; i < len(buildingTargets) && i < maxLines; i++ {
		buildingTargets[i].Lock()
		// Take a local copy of the structure, which isn't *that* big, so we don't need to retain the lock
		// while we do potentially blocking things like printing.
		target := buildingTargets[i].buildingTargetData
		buildingTargets[i].Unlock()
		label := displayLabel(target.Label)
		if target.Active {
			lprintf(cols, "${BOLD_WHITE}=> [%4.1fs] ${RESET}%s%s ${BOLD_WHITE}%s${ERASE_AFTER}\n",
				now.Sub(target.Started).Seconds(), target.Colour, label, target.Description)
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

// displayLabel returns the label that we'll display to the user; this is often just the label
// but in some cases rules define extra rules in the form _<name>#thing. For anything matching that
// we strip the leading _ and trailing hashtag to make it look like it's all the same rule.
// Note that this is only a nicety for display, all log messages etc will still display the real names.
func displayLabel(label core.BuildLabel) string {
	if len(label.Name) > 0 && label.Name[0] == '_' {
		if index := strings.IndexRune(label.Name, '#'); index != -1 {
			return core.BuildLabel{PackageName: label.PackageName, Name: strings.TrimLeft(label.Name[1:index], "_")}.String()
		}
	}
	return label.String()
}

// For calculating the size of the console window; this is pretty important when we're writing
// arbitrary-length log messages around the interactive display.
type winsize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

func windowSize() (int, int) {
	ws := new(winsize)
	if ret, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(syscall.Stderr),
		uintptr(magicNumber()),
		uintptr(unsafe.Pointer(ws)),
	); int(ret) == -1 {
		log.Errorf("error %d getting window size", int(errno))
		return 80, 25
	} else {
		return int(ws.Row), int(ws.Col)
	}
}

func magicNumber() int {
	// Found these on an internet somewhere.
	if runtime.GOOS == "darwin" {
		return 1074295912
	} else {
		return 0x5413
	}
}

func recalcWindowSize(backend *logBackend) {
	rows, cols := windowSize()
	backend.mutex.Lock()
	defer backend.mutex.Unlock()
	backend.Rows = rows - 4 // Give a little space at the edge for any off-by-ones
	backend.Cols = cols
	backend.RecalcLines()
}

// Limited-length printf that respects current window width.
// Output is truncated at the middle to fit within 'cols'.
func lprintf(cols int, format string, args ...interface{}) {
	printf(lprintfPrepare(cols, format, args...))
}

// Implementation of above. Split out to make testing easier.
var stripAnsi = regexp.MustCompile("\x1b[^m]+m")

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

// setWindowTitle sets the title of the current shell window.
func setWindowTitle(state *core.BuildState) {
	if !StdErrIsATerminal || !terminalClaimsToBeXterm {
		return // Terminal doesn't seem to support it.
	} else if state == nil {
		os.Stderr.Write([]byte(fmt.Sprintf("\033]0;plz: finishing up\007")))
	} else {
		os.Stderr.Write([]byte(fmt.Sprintf("\033]0;plz: %d / %d tasks, %3.1fs\007", state.NumDone(), state.NumActive(), time.Since(startTime).Seconds())))
	}
}
