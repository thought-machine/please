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

	cli "github.com/peterebden/go-cli-init/v5/logging"
	"github.com/peterebden/go-deferred-regex"
	"golang.org/x/term"
	"gopkg.in/op/go-logging.v1"

	logger "github.com/thought-machine/please/src/cli/logging"
)

const messageHistoryMaxSize = 100

var log = logger.Log

// StdErrIsATerminal is true if the process' stderr is an interactive TTY.
var StdErrIsATerminal = IsATerminal(os.Stderr)

// ShowColouredOutput tracks whether we are displaying coloured output or not.
var ShowColouredOutput = StdErrIsATerminal

// StripAnsi is a regex to find & replace ANSI console escape sequences.
var StripAnsi = deferredregex.DeferredRegex{Re: "\x1b[^m]+m"}

// logLevel is the current verbosity level that is set.
var logLevel = logging.WARNING

var fileLogLevel = logging.WARNING
var fileBackend logging.Backend

// A Verbosity is used as a flag to define logging verbosity.
type Verbosity = cli.Verbosity

// CurrentBackend is the current interactive logging backend.
var CurrentBackend = &LogBackend{
	interactiveRows: 10,
	maxRecords:      10,
	logMessages:     list.New(),
	messageHistory:  list.New(),
	formatter:       logFormatter(StdErrIsATerminal),
	passthrough:     true,
}

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
		log.SetBackend(newLogBackend(backend))
	} else {
		fileBackendLeveled := logging.AddModuleLevel(fileBackend)
		fileBackendLeveled.SetLevel(fileLogLevel, "")
		log.SetBackend(logging.AddModuleLevel(multiBackend(newLogBackend(backend), fileBackendLeveled)))
	}
}

func multiBackend(backends ...logging.Backend) logging.Backend {
	if len(backends) == 1 {
		return backends[0]
	}
	return logging.MultiLogger(backends...)
}

type logBackendFacade struct {
	realBackend *LogBackend // To work around the logging interface requiring us to pass by value.
}

func (backend logBackendFacade) Log(level logging.Level, calldepth int, rec *logging.Record) error {
	return backend.realBackend.Log(level, calldepth+1, rec)
}

// LogBackend is the backend we use for logging during the interactive console display.
type LogBackend struct {
	logMessages                                                           *list.List
	messageHistory                                                        *list.List
	formatter                                                             logging.Formatter
	origBackend                                                           logging.Backend
	output                                                                []string
	mutex                                                                 sync.Mutex
	rows, cols, maxRecords, interactiveRows, maxInteractiveRows, maxLines int
	messageCount                                                          int
	passthrough                                                           bool
	lineWrapRe                                                            *regexp.Regexp
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
	CurrentBackend.origBackend = origBackend
	l := logging.AddModuleLevel(logBackendFacade{realBackend: CurrentBackend})
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
		msg := reverse(backend.lineWrap(e.Value.(string)))
		ret = append(ret, msg...)
	}
	if backend.messageCount > messageHistoryMaxSize {
		return ret, backend.messageCount, messageHistoryMaxSize
	}
	return ret, backend.messageCount, backend.messageCount
}

// SetPassthrough sets whether we are "passing through" log messages or not, i.e. whether they go straight to
// the normal log output or are stored in here.
func (backend *LogBackend) SetPassthrough(passthrough bool, interactiveRows int, clearMessageHistory bool) {
	backend.mutex.Lock()
	backend.passthrough = passthrough
	backend.interactiveRows = interactiveRows
	if clearMessageHistory {
		backend.messageHistory = list.New()
		backend.messageCount = 0
	}
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
	backend.lineWrapRe = regexp.MustCompilePOSIX(fmt.Sprintf(".{%d,%d}(\\x1b[^m]+m)?", cols/2, cols))
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
			split := i + backend.findSplit(line[i:], backend.cols)
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
func (backend *LogBackend) findSplit(line string, guess int) int {
	if guess >= len(line) {
		return len(line)
	}
	if m := backend.lineWrapRe.FindStringIndex(line); m != nil {
		return m[1] // second element in slice is the end index
	}
	return guess // Dunno what to do at this point. It's probably unlikely to happen often though.
}

// HTTPLogWrapper wraps the standard logger to implement the LeveledLogger interface from retryablehttp.
type HTTPLogWrapper struct {
	Log *logging.Logger
}

// Error logs at error level
func (w *HTTPLogWrapper) Error(msg string, keysAndValues ...interface{}) {
	w.Log.Errorf("%v: %v", msg, keysAndValues)
}

// Info logs at info level
func (w *HTTPLogWrapper) Info(msg string, keysAndValues ...interface{}) {
	w.Log.Infof("%v: %v", msg, keysAndValues)
}

// Debug logs at debug level
func (w *HTTPLogWrapper) Debug(msg string, keysAndValues ...interface{}) {
	w.Log.Debugf("%v: %v", msg, keysAndValues)
}

// Warn logs at warning level
func (w *HTTPLogWrapper) Warn(msg string, keysAndValues ...interface{}) {
	w.Log.Warningf("%v: %v", msg, keysAndValues)
}

// IsATerminal returns true if the given file is an interactive TTY.
func IsATerminal(file *os.File) bool {
	return term.IsTerminal(int(file.Fd()))
}
