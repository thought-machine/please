// Package logging contains the singleton logger that we use globally.
// It deliberately has little else since it's a dependency everywhere.
package logging

import (
	"gopkg.in/op/go-logging.v1"
)

// Log is the singleton logger instance.
// We never alter individual levels and don't log the module name, so there
// is no need to have more than one, and it helps avoid race conditions.
var Log = logging.MustGetLogger("plz")

// Level is a re-export of the library type.
type Level = logging.Level

// Re-exports of various log levels.
const (
	CRITICAL = logging.CRITICAL
	ERROR    = logging.ERROR
	WARNING  = logging.WARNING
	NOTICE   = logging.NOTICE
	INFO     = logging.INFO
	DEBUG    = logging.DEBUG
)
