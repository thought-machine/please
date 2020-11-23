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

const messageHistoryMaxSize = 100

var log = logging.MustGetLogger("cli")

// StdErrIsATerminal is true if the process' stderr is an interactive TTY.
var StdErrIsATerminal = terminal.IsTerminal(int(os.Stderr.Fd()))

// StdOutIsATerminal is true if the process' stdout is an interactive TTY.
var StdOutIsATerminal = terminal.IsTerminal(int(os.Stdout.Fd()))

// ShowColouredOutput tracks whether we are displaying coloured output or not.
var ShowColouredOutput = StdErrIsATerminal

// StripAnsi is a regex to find & replace ANSI console escape sequences.
var StripAnsi = regexp.MustCompile("\x1b[^m]+m")

// logLevel is the current verbosity level that is set.
var logLevel = logging.WARNING

var fileLogLevel = logging.WARNING
var fileBackend logging.Backend

// A Verbosity is used as a flag to define logging verbosity.
type Verbosity = cli.Verbosity

// CurrentBackend is the current interactive logging backend.
var CurrentBackend *LogBackend

// InitLogging initialises logging backends.
func InitLogging(verbosity Verbosity) {
	logLevel = logging.Level(verbosity)
	setLogBackend(logging.NewLogBackend(os.Stderr, "", 0))
}

// InitFileLogging initialises an optional logging backend to a file.
func InitFileLogging(logFile string, logFileLevel Verbosity, append bool) {
	fileLogLevel = logging.Level(logFileLevel)
	if err := os.MkdirAll(path.Dir(logFile), os.ModeDir|0775); err != nil {
		log.Fatalf("Error creating log file directory: %s", err)
	}
	flags := os.O_RDWR | os.O_CREATE | os.O_TRUNC
	if append {
		flags = os.O_RDWR | os.O_CREATE | os.O_APPEND
	}
	file, err := os.OpenFile(logFile, flags, 0666)
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
	if fileBackend == nil {
		logging.SetBackend(newLogBackend(backend))
	} else {
		fileBackendLeveled := logging.AddModuleLevel(fileBackend)
		fileBackendLeveled.SetLevel(fileLogLevel, "")
		logging.SetBackend(newLogBackend(backend), fileBackendLeveled)
	}
}

type logBackendFacade struct {
	realBackend *LogBackend // To work around the logging interface requiring us to pass by value.
}

func (backend logBackendFacade) Log(level logging.Level, calldepth int, rec *logging.Record) error {
	return backend.realBackend.Log(level, calldepth+1, rec)
}

// LogBackend is the backend we use for logging during the interactive console display.
type LogBackend struct {
	mutex                                                                 sync.Mutex
	rows, cols, maxRecords, interactiveRows, maxInteractiveRows, maxLines int
	output                                                                []string
	logMessages                                                           *list.List
	messageHistory                                                        *list.List
	messageCount                                                          int
	formatter                                                             logging.Formatter
	origBackend                                                           logging.Backend
	passthrough                                                           bool
}

// Log implements the logging.Backend interface.
func (backend *LogBackend) Log(level logging.Level, calldepth int, rec *logging.Record) error {
	backend.mutex.Lock()
	defer backend.mutex.Unlock()
	if backend.passthrough || rec.Level <= logging.CRITICAL {
		backend.origBackend.Log(level, calldepth, rec)
		return nil
	}
	var b bytes.Buffer
	backend.formatter.Format(calldepth, rec, &b)
	msg := strings.TrimSpace(b.String())
	backend.logMessages.PushBack(msg)

	// Add the messages to the history so we may output them again after the build has finished
	if backend.messageCount < messageHistoryMaxSize {
		backend.messageHistory.PushBack(msg)
	}
	backend.messageCount++

	backend.RecalcLines()
	return nil
}

// RecalcLines recalculates how many lines we have available, typically in response to the window size changing
func (backend *LogBackend) RecalcLines() {
	for backend.logMessages.Len() >= backend.maxRecords {
		backend.logMessages.Remove(backend.logMessages.Front())
	}
	backend.maxLines = backend.rows - backend.interactiveRows - 1
	if backend.maxLines > 15 {
		backend.maxLines = 15 // Cap it here so we don't log too much
	} else if backend.maxLines <= 0 {
		backend.maxLines = 3 // Set a minimum so we don't have negative indices later.
	}
	backend.output = backend.calcOutput()
	backend.maxInteractiveRows = backend.rows - len(backend.output) - 1
}

// newLogBackend constructs a new logging backend.
func newLogBackend(origBackend logging.Backend) logging.LeveledBackend {
	b := &LogBackend{
		interactiveRows: 10,
		maxRecords:      10,
		logMessages:     list.New(),
		messageHistory:  list.New(),
		formatter:       logFormatter(StdErrIsATerminal),
		origBackend:     origBackend,
		passthrough:     true,
	}
	CurrentBackend = b
	l := logging.AddModuleLevel(logBackendFacade{realBackend: b})
	l.SetLevel(logLevel, "")
	return l
}

func (backend *LogBackend) calcOutput() []string {
	ret := []string{}
	for e := backend.logMessages.Back(); e != nil; e = e.Prev() {
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

// GetMessageHistory returns the history of log messages. The message history is limited to messageHistoryMaxSize so
// this method returns the total amount of messages logged and the amount actually retained as well.
func (backend *LogBackend) GetMessageHistory() ([]string, int, int) {
	ret := make([]string, 0, backend.messageHistory.Len())
	for e := backend.messageHistory.Front(); e != nil; e = e.Next() {
		msg := backend.lineWrap(e.Value.(string))
		ret = append(ret, msg...)
	}
	if backend.messageCount > messageHistoryMaxSize {
		return ret, backend.messageCount, messageHistoryMaxSize
	}
	return ret, backend.messageCount, backend.messageCount
}

// SetPassthrough sets whether we are "passing through" log messages or not, i.e. whether they go straight to
// the normal log output or are stored in here.
func (backend *LogBackend) SetPassthrough(passthrough bool, interactiveRows int) {
	backend.mutex.Lock()
	backend.passthrough = passthrough
	backend.interactiveRows = interactiveRows
	backend.mutex.Unlock()
	if passthrough {
		go notifyOnWindowResize(backend.recalcWindowSize)
	}
	backend.recalcWindowSize()
}

func (backend *LogBackend) recalcWindowSize() {
	rows, cols, _ := WindowSize()
	backend.mutex.Lock()
	defer backend.mutex.Unlock()
	backend.rows = rows - 4 // Give a little space at the edge for any off-by-ones
	backend.cols = cols
	backend.RecalcLines()
}

// MaxDimensions returns the maximum number of rows / columns available in the display.
func (backend *LogBackend) MaxDimensions() (rows int, cols int) {
	backend.mutex.Lock()
	defer backend.mutex.Unlock()
	return backend.maxInteractiveRows, backend.cols
}

// Output returns the current set of preformatted log messages.
func (backend *LogBackend) Output() []string {
	backend.mutex.Lock()
	defer backend.mutex.Unlock()
	return backend.output[:]
}

// Wraps a string across multiple lines. Returned slice is reversed.
func (backend *LogBackend) lineWrap(msg string) []string {
	lines := strings.Split(msg, "\n")
	wrappedLines := make([]string, 0, len(lines))
	for _, line := range lines {
		for i := 0; i < len(line); {
			split := i + findSplit(line[i:], backend.cols)
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
