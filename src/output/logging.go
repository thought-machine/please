// Contains various utility functions related to logging.

package output

import (
	"bytes"
	"container/list"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"

	"gopkg.in/op/go-logging.v1"

	"core"
)

var log = logging.MustGetLogger("output")

var logLevel = logging.WARNING
var fileLogLevel = logging.WARNING
var fileBackend *logging.LogBackend = nil

type logFileWriter struct {
	file io.Writer
}

func (writer logFileWriter) Write(p []byte) (n int, err error) {
	return writer.file.Write(stripAnsi.ReplaceAllLiteral(p, []byte{}))
}

// translateLogLevel translates our verbosity flags to logging levels.
func translateLogLevel(verbosity int) logging.Level {
	if verbosity <= 0 {
		return logging.ERROR
	} else if verbosity == 1 {
		return logging.WARNING
	} else if verbosity == 2 {
		return logging.NOTICE
	} else if verbosity == 3 {
		return logging.INFO
	} else {
		return logging.DEBUG
	}
}

// InitLogging initialises logging backends. verbosity controls output to shell,
// logFile and logFileLevel what's (optionally) logged to a file as well.
func InitLogging(verbosity int, logFile string, logFileLevel int) {
	logLevel = translateLogLevel(verbosity)
	fileLogLevel = translateLogLevel(logFileLevel)
	logging.SetFormatter(logFormatter())
	setLogBackend(logging.NewLogBackend(os.Stderr, "", 0))

	if logFile != "" {
		if err := os.MkdirAll(path.Dir(logFile), core.DirPermissions); err != nil {
			log.Fatalf("Error creating log file directory: %s", err)
		}
		if file, err := os.Create(logFile); err != nil {
			log.Fatalf("Error opening log file: %s", err)
		} else {
			fileBackend = logging.NewLogBackend(logFileWriter{file: file}, "", 0)
			setLogBackend(logging.NewLogBackend(os.Stderr, "", 0))
		}
	}
}

func logFormatter() logging.Formatter {
	formatStr := "%{time:15:04:05.000} %{level:7s}: %{message}"
	if StdErrIsATerminal {
		formatStr = "%{color}" + formatStr + "%{color:reset}"
	}
	return logging.MustStringFormatter(formatStr)
}

func setLogBackend(backend logging.Backend) {
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
	realBackend *logBackend // To work around the logging interface requiring us to pass by value.
}

func (backend logBackendFacade) Log(level logging.Level, calldepth int, rec *logging.Record) error {
	var b bytes.Buffer
	backend.realBackend.Formatter.Format(calldepth, rec, &b)
	if rec.Level <= logging.CRITICAL {
		fmt.Print(b.String()) // Don't capture critical messages, just die immediately.
		os.Exit(1)
	}
	backend.realBackend.mutex.Lock()
	defer backend.realBackend.mutex.Unlock()
	backend.realBackend.LogMessages.PushBack(strings.TrimSpace(b.String()))
	backend.realBackend.RecalcLines()
	return nil
}

type logBackend struct {
	Rows, Cols, MaxRecords, InteractiveRows, MaxInteractiveRows, maxLines int
	Output                                                                []string
	LogMessages                                                           *list.List
	mutex                                                                 sync.Mutex        // Protects access to LogMessages
	Formatter                                                             logging.Formatter // TODO(pebers): seems a bit weird that we have to have this here, but it doesn't
} //               seem to be possible to retrieve the formatter from outside the package?

func (backend *logBackend) RecalcLines() {
	for backend.LogMessages.Len() >= backend.MaxRecords {
		backend.LogMessages.Remove(backend.LogMessages.Front())
	}
	backend.maxLines = backend.Rows - backend.InteractiveRows - 1
	if backend.maxLines > 15 {
		backend.maxLines = 15 // Cap it here so we don't log too much
	} else if backend.maxLines <= 0 {
		backend.maxLines = 1 // Set a minimum so we don't have negative indices later.
	}
	backend.Output = backend.calcOutput()
	backend.MaxInteractiveRows = backend.Rows - len(backend.Output) - 1
}

func (backend *logBackend) calcOutput() []string {
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

// Wraps a string across multiple lines. Returned slice is reversed.
func (backend *logBackend) lineWrap(msg string) []string {
	lines := strings.Split(msg, "\n")
	wrappedLines := make([]string, 0, len(lines))
	for _, line := range lines {
		for i := 0; i < len(line); {
			split := i + findSplit(line[i:len(line)], backend.Cols)
			wrappedLines = append(wrappedLines, line[i:split])
			i = split
		}
	}
	if len(wrappedLines) > backend.maxLines {
		return reverse(wrappedLines[:backend.maxLines])
	} else {
		return reverse(wrappedLines)
	}
}

func reverse(s []string) []string {
	if len(s) > 1 {
		r := []string{}
		for i := len(s) - 1; i >= 0; i-- {
			r = append(r, s[i])
		}
		return r
	} else {
		return s
	}
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
