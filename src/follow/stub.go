// +build bootstrap

// Package follow implements remote communication between plz instances.
// This file is a stub used only for initial bootstrap.
package follow

import (
	"time"

	"core"
)

// InitialiseServer is a stub that does nothing.
func InitialiseServer(state *core.BuildState, port int) func() {
	return func() {}
}

// ConnectClient is a stub that always returns false immediately.
func ConnectClient(state *core.BuildState, url string, retries int, delay time.Duration) bool {
	return false
}

// UpdateResources is a stub that also does nothing.
func UpdateResources(state *core.BuildState) {
}
