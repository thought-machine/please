// +build bootstrap

package gc

import "github.com/thought-machine/please/src/core"

// GarbageCollect is a stub used at initial bootstrap time to avoid requiring us to run go-bindata yet again.
func GarbageCollect(state *core.BuildState, filter, targets, keepTargets []core.BuildLabel, keepLabels []string, conservative, targetsOnly, srcsOnly, noPrompt, dryRun, git bool) {
}

// RewriteFile is also a stub used at boostrap time that does nothing.
func RewriteFile(state *core.BuildState, filename string, targets []string) error { return nil }
