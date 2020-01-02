// +build bootstrap

// Used at initial bootstrap time to cut down on dependencies.

package watch

import (
	"context"

	"github.com/thought-machine/please/src/core"
)

// A CallbackFunc is supplied to Watch in order to trigger a build.
type CallbackFunc func(*core.BuildState, []core.BuildLabel)

// Watch is a stub implementation of the real function in watch.go, this one does nothing.
func Watch(ctx context.Context, state *core.BuildState, labels core.BuildLabels, callback CallbackFunc) {
}
