// Package logging contains the singleton logger that we use globally.
// It deliberately has little else since it's a dependency everywhere.
package logging

import (
	"gopkg.in/op/go-logging.v1"
)

var Log = logging.MustGetLogger("plz")
