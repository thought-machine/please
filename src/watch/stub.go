// +build bootstrap

// Used at initial bootstrap time to cut down on dependencies.

package watch

import "core"

// Watch is a stub implementation of the real function in watch.go, this one does nothing.
func Watch(state *core.BuildState, labels []core.BuildLabel, run bool) {}
