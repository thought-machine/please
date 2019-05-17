// Package scm abstracts operations on various tools like git
// Currently, only git is supported.
package scm

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

// git implements operations on a git repository.
type git struct {
	repoRoot string
}

// CurrentRevIdentifier returns the string that specifies what the current revision is.
func (g *git) CurrentRevIdentifier() string {
	out, err := exec.Command("git", "rev-parse", "HEAD").CombinedOutput()
	if err != nil {
		log.Fatalf("Failed to read HEAD: %s", err)
	}
	return strings.TrimSpace(string(out))
}

// ChangesIn returns a list of modified files in the given diffSpec.
func (g *git) ChangesIn(diffSpec string, relativeTo string) []string {
	if relativeTo == "" {
		relativeTo = g.repoRoot
	}
	files := make([]string, 0)
	command := []string{"diff-tree", "--no-commit-id", "--name-only", "-r", diffSpec}
	out, err := exec.Command("git", command...).CombinedOutput()
	if err != nil {
		log.Fatalf("unable to determine changes: %s", err)
	}
	output := strings.Split(string(out), "\n")
	for _, o := range output {
		files = append(files, g.fixGitRelativePath(strings.TrimSpace(o), relativeTo))
	}
	return files
}

// ChangedFiles returns a list of modified files since the given commit, optionally including untracked files.
func (g *git) ChangedFiles(fromCommit string, includeUntracked bool, relativeTo string) []string {
	if relativeTo == "" {
		relativeTo = g.repoRoot
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
		normalized = append(normalized, g.fixGitRelativePath(strings.TrimSpace(f), relativeTo))
	}
	return normalized
}

func (g *git) fixGitRelativePath(worktreePath, relativeTo string) string {
	p, err := filepath.Rel(relativeTo, path.Join(g.repoRoot, worktreePath))
	if err != nil {
		log.Fatalf("unable to determine relative path for %s and %s", g.repoRoot, relativeTo)
	}
	return p
}

func (g *git) IgnoreFile(name string) error {
	gitignore := path.Join(g.repoRoot, ".gitignore")
	b, err := ioutil.ReadFile(gitignore)
	if err != nil && !os.IsNotExist(err) { // Not an error for this not to exist.
		return err
	}
	if len(b) > 0 { // Don't append an initial newline if at the start of the file.
		b = append(b, '\n')
	}
	b = append(b, []byte("# Please output directory\nplz-out\n")...)
	return ioutil.WriteFile(gitignore, b, 0644)
}
