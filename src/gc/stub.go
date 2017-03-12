package gc

import "core"

// GarbageCollect is a stub used at initial bootstrap time to avoid requiring us to run go-bindata yet again.
func GarbageCollect(state *core.BuildState, targets []core.BuildLabel, keepLabels []string, conservative, targetsOnly, srcsOnly, noPrompt, dryRun, git bool) {
}
