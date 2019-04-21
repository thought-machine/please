// Package scm abstracts operations on various tools like git
// Currently, only git is supported.
package scm

import (
	"path"

	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/fs"
)

var log = logging.MustGetLogger("scm")

// An SCM represents an SCM implementation that we can ask for various things.
type SCM interface {
	// CurrentRevIdentifier returns the string that specifies what the current revision is.
	CurrentRevIdentifier() string
	// ChangesIn returns a list of modified files in the given diffSpec.
	ChangesIn(diffSpec string, relativeTo string) []string
	// ChangedFiles returns a list of modified files since the given commit, optionally including untracked files.
	ChangedFiles(fromCommit string, includeUntracked bool, relativeTo string) []string
	// IgnoreFile marks a file to be ignored by the SCM.
	IgnoreFile(name string) error
}

// New returns a new SCM instance for this repo root.
// It returns nil if there is no known implementation there.
func New(repoRoot string) SCM {
	if fs.PathExists(path.Join(repoRoot, ".git")) {
		return &git{repoRoot: repoRoot}
	}
	return nil
}

// MustNew returns a new SCM instance for this repo root.
// It dies if a supported one cannot be determined.
func MustNew(repoRoot string) SCM {
	s := New(repoRoot)
	if s == nil {
		log.Fatalf("Cannot determine SCM implementation for this repo")
	}
	return s
}
