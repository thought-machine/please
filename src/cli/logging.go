// Contains various utility functions related to logging.

package cli

import (
	"bytes"
	"container/list"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"

	"github.com/peterebden/go-cli-init"
	"golang.org/x/crypto/ssh/terminal"
	"gopkg.in/op/go-logging.v1"
)

var log = logging.MustGetLogger("cli")

// StdErrIsATerminal is true if the process' stderr is an interactive TTY.
var StdErrIsATerminal = terminal.IsTerminal(int(os.Stderr.Fd()))

// StdOutIsATerminal is true if the process' stdout is an interactive TTY.
var StdOutIsATerminal = terminal.IsTerminal(int(os.Stdout.Fd()))

// StripAnsi is a regex to find & replace ANSI console escape sequences.
var StripAnsi = regexp.MustCompile("\x1b[^m]+m")

// logLevel is the current verbosity level that is set.
var logLevel = logging.WARNING

var fileLogLevel = logging.WARNING
var fileBackend logging.Backend

// A Verbosity is used as a flag to define logging verbosity.
type Verbosity = cli.Verbosity

// InitLogging initialises logging backends.
func InitLogging(verbosity Verbosity) {
	logLevel = logging.Level(verbosity)
	setLogBackend(logging.NewLogBackend(os.Stderr, "", 0))
}

// InitFileLogging initialises an optional logging backend to a file.
func InitFileLogging(logFile string, logFileLevel Verbosity) {
	fileLogLevel = logging.Level(logFileLevel)
	if err := os.MkdirAll(path.Dir(logFile), os.ModeDir|0775); err != nil {
		log.Fatalf("Error creating log file directory: %s", err)
	}
	file, err := os.Create(logFile)
	if err != nil {
		log.Fatalf("Error opening log file: %s", err)
	}
	fileBackend = logging.NewLogBackend(file, "", 0)
	fileBackend = logging.NewBackendFormatter(fileBackend, logFormatter(false))
	setLogBackend(logging.NewLogBackend(os.Stderr, "", 0))
	AtExit(func() {
		fileBackend = nil
		setLogBackend(logging.NewLogBackend(os.Stderr, "", 0))
		file.Close()
	})
}

func logFormatter(coloured bool) logging.Formatter {
	formatStr := "%{time:15:04:05.000} %{level:7s}: %{message}"
	if coloured {
		formatStr = "%{color}" + formatStr + "%{color:reset}"
	}
	return logging.MustStringFormatter(formatStr)
}

func setLogBackend(backend logging.Backend) {
	backend = logging.NewBackendFormatter(backend, logFormatter(StdErrIsATerminal))
	backendLeveled := logging.AddModuleLevel(backend)
	backendLeveled.SetLevel(logLevel, "")
	if fileBackend == nil {
		logging.SetBackend(backendLeveled)
	} else {
		fileBackendLeveled := logging.AddModuleLevel(fileBackend)
		fileBackendLeveled.SetLevel(fileLogLevel, "")
		logging.SetBackend(backendLeveled, fileBackendLeveled)
	}
}

type logBackendFacade struct {
	realBackend *LogBackend // To work around the logging interface requiring us to pass by value.
}

func (backend logBackendFacade) Log(level logging.Level, calldepth int, rec *logging.Record) error {
	var b bytes.Buffer
	backend.realBackend.Formatter.Format(calldepth, rec, &b)
	if rec.Level <= logging.CRITICAL {
		fmt.Print(b.String()) // Don't capture critical messages, just die immediately.
		os.Exit(1)
	}
	backend.realBackend.Lock()
	defer backend.realBackend.Unlock()
	backend.realBackend.LogMessages.PushBack(strings.TrimSpace(b.String()))
	backend.realBackend.RecalcLines()
	return nil
}

// LogBackend is the backend we use for logging during the interactive console display.
type LogBackend struct {
	sync.Mutex                                                            // Protects access to LogMessages
	Rows, Cols, MaxRecords, InteractiveRows, MaxInteractiveRows, maxLines int
	Output                                                                []string
	LogMessages                                                           *list.List
	Formatter                                                             logging.Formatter // TODO(pebers): seems a bit weird that we have to have this here, but it doesn't
} //               seem to be possible to retrieve the formatter from outside the package?

// RecalcLines recalculates how many lines we have available, typically in response to the window size changing
func (backend *LogBackend) RecalcLines() {
	for backend.LogMessages.Len() >= backend.MaxRecords {
		backend.LogMessages.Remove(backend.LogMessages.Front())
	}
	backend.maxLines = backend.Rows - backend.InteractiveRows - 1
	if backend.maxLines > 15 {
		backend.maxLines = 15 // Cap it here so we don't log too much
	} else if backend.maxLines <= 0 {
		backend.maxLines = 3 // Set a minimum so we don't have negative indices later.
	}
	backend.Output = backend.calcOutput()
	backend.MaxInteractiveRows = backend.Rows - len(backend.Output) - 1
}

// NewLogBackend constructs a new logging backend.
func NewLogBackend(interactiveRows int) *LogBackend {
	return &LogBackend{
		InteractiveRows: interactiveRows,
		MaxRecords:      10,
		LogMessages:     list.New(),
		Formatter:       logFormatter(StdErrIsATerminal),
	}
}

func (backend *LogBackend) calcOutput() []string {
	ret := []string{}
	for e := backend.LogMessages.Back(); e != nil; e = e.Prev() {
		new := backend.lineWrap(e.Value.(string))
		if len(ret)+len(new) <= backend.maxLines {
			ret = append(ret, new...)
		}
	}
	if len(ret) > 0 {
		ret = append(ret, "Messages:")
	}
	return reverse(ret)
}

// SetActive sets this backend as the currently active log backend.
func (backend *LogBackend) SetActive() {
	setLogBackend(logBackendFacade{backend})
}

// Deactivate removes this backend as the currently active log backend.
func (backend *LogBackend) Deactivate() {
	setLogBackend(logging.NewLogBackend(os.Stderr, "", 0))
}

// Wraps a string across multiple lines. Returned slice is reversed.
func (backend *LogBackend) lineWrap(msg string) []string {
	lines := strings.Split(msg, "\n")
	wrappedLines := make([]string, 0, len(lines))
	for _, line := range lines {
		for i := 0; i < len(line); {
			split := i + findSplit(line[i:], backend.Cols)
			wrappedLines = append(wrappedLines, line[i:split])
			i = split
		}
	}
	if len(wrappedLines) > backend.maxLines {
		return reverse(wrappedLines[:backend.maxLines])
	}
	return reverse(wrappedLines)
}

func reverse(s []string) []string {
	if len(s) > 1 {
		r := []string{}
		for i := len(s) - 1; i >= 0; i-- {
			r = append(r, s[i])
		}
		return r
	}
	return s
}

// Tries to find an appropriate point to word wrap line, taking shell escape codes into account.
// (Note that because the escape codes are not visible, we can run past the max length for one of them)
func findSplit(line string, guess int) int {
	if guess >= len(line) {
		return len(line)
	}
	r := regexp.MustCompilePOSIX(fmt.Sprintf(".{%d,%d}(\\x1b[^m]+m)?", guess/2, guess))
	m := r.FindStringIndex(line)
	if m != nil {
		return m[1] // second element in slice is the end index
	}
	return guess // Dunno what to do at this point. It's probably unlikely to happen often though.
}
