//+build bootstrap

package test

import "core"

func runPossiblyContainerisedTest(tid int, state *core.BuildState, target *core.BuildTarget) ([]byte, error) {
	return runTest(state, target) // Containerisation not supported (but we don't run any tests during bootstrap anyway)
}
