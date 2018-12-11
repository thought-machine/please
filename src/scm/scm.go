// Package scm abstracts operations on various tools like git
// Currently, only git
package scm

import (
	"github.com/thought-machine/please/src/core"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/op/go-logging.v1"
)

var log = logging.MustGetLogger("scm")

// CurrentRevIdentifier returns the string that specifies what the current revision is.
func CurrentRevIdentifier() string {
	return "HEAD"
}

// ChangesIn returns a list of modified files in the given diffSpec.
func ChangesIn(diffSpec string, relativeTo string) []string {
	if relativeTo == "" {
		relativeTo = core.RepoRoot
	}
	files := make([]string, 0)
	command := []string{"diff-tree", "--no-commit-id", "--name-only", "-r", diffSpec}
	out, err := exec.Command("git", command...).CombinedOutput()
	if err != nil {
		log.Fatalf("unable to determine changes: %s", err)
	}
	output := strings.Split(string(out), "\n")
	for _, o := range output {
		files = append(files, fixGitRelativePath(strings.TrimSpace(o), relativeTo))
	}
	return files
}

// ChangedFiles returns a list of modified files since the given commit, optionally including untracked files.
func ChangedFiles(fromCommit string, includeUntracked bool, relativeTo string) []string {
	if relativeTo == "" {
		relativeTo = core.RepoRoot
	}
	relSuffix := []string{"--", relativeTo}
	command := []string{"diff", "--name-only", "HEAD"}

	out, err := exec.Command("git", append(command, relSuffix...)...).CombinedOutput()
	if err != nil {
		log.Fatalf("unable to find changes: %s", err)
	}
	files := strings.Split(string(out), "\n")

	if fromCommit != "" {
		// Grab the diff from the merge-base to HEAD using ... syntax.  This ensures we have just
		// the changes that have occurred on the current branch.
		command = []string{"diff", "--name-only", fromCommit + "...HEAD"}
		out, err = exec.Command("git", append(command, relSuffix...)...).CombinedOutput()
		if err != nil {
			log.Fatalf("unable to check current branch: %s", err)
		}
		committedChanges := strings.Split(string(out), "\n")
		files = append(files, committedChanges...)
	}
	if includeUntracked {
		command = []string{"ls-files", "--other", "--exclude-standard"}
		out, err = exec.Command("git", append(command, relSuffix...)...).CombinedOutput()
		if err != nil {
			log.Fatalf("unable to determine untracked files: %s", err)
		}
		untracked := strings.Split(string(out), "\n")
		files = append(files, untracked...)
	}
	// git will report changed files relative to the worktree: re-relativize to relativeTo
	normalized := make([]string, 0)
	for _, f := range files {
		normalized = append(normalized, fixGitRelativePath(strings.TrimSpace(f), relativeTo))
	}
	return normalized
}

func fixGitRelativePath(worktreePath, relativeTo string) string {
	p, err := filepath.Rel(relativeTo, path.Join(core.RepoRoot, worktreePath))
	if err != nil {
		log.Fatalf("unable to determine relative path for %s and %s", core.RepoRoot, relativeTo)
	}
	return p
}
