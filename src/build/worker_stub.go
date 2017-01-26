// Contains a stub for the buildMaybeRemotely function which is used during the
// initial bootstrap when we don't have protobufs etc available.

package build

import (
	"fmt"

	"core"
)

// buildMaybeRemotely builds a target, either sending it to a remote worker if needed,
// or locally if not.
func buildMaybeRemotely(state *core.BuildState, target *core.BuildTarget, inputHash []byte) ([]byte, error) {
	worker, _, localCmd := workerCommandAndArgs(target)
	if worker == "" {
		return runBuildCommand(state, target, localCmd, inputHash)
	}
	return nil, fmt.Errorf("Remote worker support has not been compiled in")
}

// StopWorkers does nothing, because in the stub we don't have any workers.
func StopWorkers() {}
