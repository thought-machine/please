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
	// DescribeIdentifier returns the string that is a "human-readable" identifier of the given revision.
	DescribeIdentifier(revision string) string
	// CurrentRevIdentifier returns the string that specifies what the current revision is.
	CurrentRevIdentifier() string
	// ChangesIn returns a list of modified files in the given diffSpec.
	ChangesIn(diffSpec string, relativeTo string) []string
	// ChangedFiles returns a list of modified files since the given commit, optionally including untracked files.
	ChangedFiles(fromCommit string, includeUntracked bool, relativeTo string) []string
	// IgnoreFile marks a file to be ignored by the SCM.
	IgnoreFile(name string) error
	// Remove deletes the given files from the SCM.
	Remove(names []string) error
	// ChangedLines returns the set of lines that have been modified,
	// as a map of filename -> affected line numbers.
	ChangedLines() (map[string][]int, error)
	// Checkout checks out the given revision.
	Checkout(revision string) error
}

// New returns a new SCM instance for this repo root.
// It returns nil if there is no known implementation there.
func New(repoRoot string) SCM {
	if fs.PathExists(path.Join(repoRoot, ".git")) {
		return &git{repoRoot: repoRoot}
	}
	return nil
}

// NewFallback returns a new SCM instance for this repo root.
// If there is no known implementation it returns a stub.
func NewFallback(repoRoot string) SCM {
	if scm := New(repoRoot); scm != nil {
		return scm
	}
	log.Warning("Cannot determine SCM, revision identifiers will be unavailable and `plz query changes/changed` will not work correctly.")
	return &stub{}
}

// MustNew returns a new SCM instance for this repo root. It dies on any errors.
func MustNew(repoRoot string) SCM {
	scm := New(repoRoot)
	if scm == nil {
		log.Fatalf("Cannot determine SCM implementation")
	}
	return scm
}
