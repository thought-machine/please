package core

import (
	"path"
	"path/filepath"
	"sync"

	"github.com/thought-machine/please/src/cli"
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
	// The build state instance that tracks this subrepo (it's different from the host one if
	// this subrepo is for a different architecture)
	State *BuildState
	// Architecture for this subrepo.
	Arch cli.Arch
	// True if this subrepo was created for a different architecture
	IsCrossCompile bool
	// loadConfig is used to control when we load plugin configuration. We need access to the subrepo itself to do this
	// so it happens at build time.
	loadConfig sync.Once
}

// SubrepoForArch creates a new subrepo for the given architecture.
func SubrepoForArch(state *BuildState, arch cli.Arch) *Subrepo {
	return &Subrepo{
		Name:           arch.String(),
		State:          state.ForArch(arch),
		Arch:           arch,
		IsCrossCompile: true,
	}
}

// Dir returns the directory for a package of this name.
func (s *Subrepo) Dir(dir string) string {
	return path.Join(s.Root, dir)
}

// LoadSubrepoConfig will load the .plzconfig from the subrepo. We can only do this once the subrepo is built hence why
// it's not done up front.
func (s *Subrepo) LoadSubrepoConfig() (err error) {
	s.loadConfig.Do(func() {
		err = readConfigFile(s.State.Config, filepath.Join(s.Root, ".plzconfig"))
	})
	return
}
