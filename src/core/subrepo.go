package core

import (
	"cli"
	"fmt"
	"path"
	"strings"
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
}

// SubrepoForArch creates a new subrepo for the given architecture.
func SubrepoForArch(state *BuildState, arch cli.Arch) *Subrepo {
	return &Subrepo{
		Name:  arch.String(),
		State: state.ForArch(arch),
	}
}

// MakeRelative makes a build label that is within this subrepo relative to it (i.e. strips the leading name part).
// The caller should already know that it is within this repo, otherwise this will panic.
func (s *Subrepo) MakeRelative(label BuildLabel) BuildLabel {
	return BuildLabel{s.MakeRelativeName(label.PackageName), label.Name}
}

// MakeRelativeName is as MakeRelative but operates only on the package name.
func (s *Subrepo) MakeRelativeName(name string) string {
	// Check for nil, makes it easier to call this without having so many conditionals.
	if s == nil {
		return name
	} else if !strings.HasPrefix(name, s.Name) {
		panic(fmt.Errorf("cannot make label %s relative, it is not within this subrepo (%s)", name, s.Name))
	}
	return strings.TrimPrefix(name[len(s.Name):], "/")
}

// Dir returns the directory for a package of this name.
func (s *Subrepo) Dir(dir string) string {
	return path.Join(s.Root, s.MakeRelativeName(dir))
}
