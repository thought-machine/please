package core

import (
	"cli"
	"path"
)

// A Subrepo stores information about a registered subrepository, typically one
// that we have downloaded somehow to bring in third-party deps.
type Subrepo struct {
	// The name of the subrepo.
	Name string
	// The root directory to load it from.
	Root string
	// If this repo is output by a target, this is the target that creates it. Can be nil.
	Target *BuildTarget
	// If this repo has a different configuration (e.g. it's for a different architecture), it's stored here
	State *BuildState
	// True if this subrepo was created for a different architecture
	IsCrossCompile bool
}

// SubrepoForArch creates a new subrepo for the given architecture.
func SubrepoForArch(state *BuildState, arch cli.Arch) *Subrepo {
	return &Subrepo{
		Name:           arch.String(),
		State:          state.ForArch(arch),
		IsCrossCompile: true,
	}
}

// Dir returns the directory for a package of this name.
func (s *Subrepo) Dir(dir string) string {
	return path.Join(s.Root, dir)
}
