package utils

import "gopkg.in/op/go-logging.v1"

// HTTPLogWrapper wraps the standard logger to provide a Printf function
// for use with retryablehttp.
type HTTPLogWrapper struct {
	*logging.Logger
}

// Error logs at error level
func (w *HTTPLogWrapper) Error(msg string, keysAndValues ...interface{}) {
	w.Errorf("%v: %v", msg, keysAndValues)
}

// Info logs at info level
func (w *HTTPLogWrapper) Info(msg string, keysAndValues ...interface{}) {
	w.Infof("%v: %v", msg, keysAndValues)
}

// Debug logs at debug level
func (w *HTTPLogWrapper) Debug(msg string, keysAndValues ...interface{}) {
	w.Debugf("%v: %v", msg, keysAndValues)
}

// Warn logs at warning level
func (w *HTTPLogWrapper) Warn(msg string, keysAndValues ...interface{}) {
	w.Warningf("%v: %v", msg, keysAndValues)
}
