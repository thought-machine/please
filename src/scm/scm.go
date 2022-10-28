// Package scm abstracts operations on various tools like git
// Currently, only git is supported.
package scm

import (
	"path/filepath"

	"github.com/thought-machine/please/src/cli/logging"
	"github.com/thought-machine/please/src/fs"
)

var log = logging.Log

// An SCM represents an SCM implementation that we can ask for various things.
type SCM interface {
	// DescribeIdentifier returns the string that is a "human-readable" identifier of the given revision.
	DescribeIdentifier(revision string) string
	// CurrentRevIdentifier returns a string that specifies what the current revision is. If
	// "permanent" is true, this string will permanently identify the revision; otherwise, the string
	// may only be a transient identifier.
	CurrentRevIdentifier(permanent bool) string
	// ChangesIn returns a list of modified files in the given diffSpec.
	ChangesIn(diffSpec string, relativeTo string) []string
	// ChangedFiles returns a list of modified files since the given commit, optionally including untracked files.
	ChangedFiles(fromCommit string, includeUntracked bool, relativeTo string) []string
	// IgnoreFiles marks a file to be ignored by the SCM.
	IgnoreFiles(gitignore string, files []string) error
	// FindOrCreateIgnoreFile gets the ignore file name for the version control system
	FindOrCreateIgnoreFile(path string) (string, error)
	// Remove deletes the given files from the SCM.
	Remove(names []string) error
	// ChangedLines returns the set of lines that have been modified,
	// as a map of filename -> affected line numbers.
	ChangedLines() (map[string][]int, error)
	// Checkout checks out the given revision.
	Checkout(revision string) error
	// CurrentRevDate returns the commit date of the current revision, formatted according to the given format string.
	CurrentRevDate(format string) string
	// AreIgnored returns whether the files are all ignored or not
	AreIgnored(files ...string) bool
}

// New returns a new SCM instance for this repo root.
// It returns nil if there is no known implementation there.
func New(repoRoot string) SCM {
	if fs.PathExists(filepath.Join(repoRoot, ".git")) {
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
