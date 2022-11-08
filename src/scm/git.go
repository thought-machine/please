// Package scm abstracts operations on various tools like git
// Currently, only git is supported.
package scm

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/sourcegraph/go-diff/diff"
)

const pleaseDoNotEdit = "# Entries below this point are managed by Please (DO NOT EDIT)"

var defaultIgnoredFiles = []string{"plz-out", ".plzconfig.local"}

const ignoreFileName = ".gitignore"

// git implements operations on a git repository.
type git struct {
	repoRoot string
}

type gitIgnore struct {
	*os.File
	entries      map[string]struct{}
	hasDoNotEdit bool
}

// DescribeIdentifier returns the string that is a "human-readable" identifier of the given revision.
func (g *git) DescribeIdentifier(revision string) string {
	out, err := exec.Command("git", "describe", "--always", revision).CombinedOutput()
	if err != nil {
		log.Fatalf("Failed to read %s: %s\nOutput:\n%s", revision, err, string(out))
	}
	return strings.TrimSpace(string(out))
}

// CurrentRevIdentifier returns the string that specifies what the current revision is.
//
// If "permanent" is true, CurrentRevIdentifier returns the current revision's commit hash; this
// should be used to uniquely and permanently identify the revision. If "permanent" is false,
// CurrentRevIdentifier returns the name of a branch if the revision is the HEAD of that branch,
// and the current revision's commit hash otherwise; this will not permanently identify the current
// revision, as the HEAD of the branch may change in future.
func (g *git) CurrentRevIdentifier(permanent bool) string {
	if !permanent {
		out, err := exec.Command("git", "symbolic-ref", "-q", "--short", "HEAD").CombinedOutput()
		if err == nil {
			return strings.TrimSpace(string(out))
		}
	}
	out, err := exec.Command("git", "rev-parse", "HEAD").CombinedOutput()
	if err != nil {
		log.Fatalf("Failed to read HEAD: %s\nOutput:\n%s", err, string(out))
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
		log.Fatalf("unable to determine changes: %s\nOutput:\n%s", err, string(out))
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
		log.Fatalf("unable to find changes: %s\nOutput:\n%s)", err, string(out))
	}
	files := strings.Split(string(out), "\n")

	if fromCommit != "" {
		// Grab the diff from the merge-base to HEAD using ... syntax.  This ensures we have just
		// the changes that have occurred on the current branch.
		command = []string{"diff", "--name-only", fromCommit + "...HEAD"}
		command = append(command, relSuffix...)
		out, err = exec.Command("git", command...).CombinedOutput()
		if err != nil {
			log.Fatalf("unable to check diff vs. %s: %s\nOutput:\n%s", fromCommit, err, string(out))
		}
		committedChanges := strings.Split(string(out), "\n")
		files = append(files, committedChanges...)
	}
	if includeUntracked {
		command = []string{"ls-files", "--other", "--exclude-standard"}
		out, err = exec.Command("git", append(command, relSuffix...)...).CombinedOutput()
		if err != nil {
			log.Fatalf("unable to determine untracked files: %s\nOutput:\n%s", err, string(out))
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
	p, err := filepath.Rel(relativeTo, filepath.Join(g.repoRoot, worktreePath))
	if err != nil {
		log.Fatalf("unable to determine relative path for %s and %s", g.repoRoot, relativeTo)
	}
	return p
}

func openGitignore(file string) (*gitIgnore, error) {
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	ignoreFile := &gitIgnore{
		File:    f,
		entries: map[string]struct{}{},
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == pleaseDoNotEdit {
			ignoreFile.hasDoNotEdit = true
			continue
		}
		ignoreFile.entries[line] = struct{}{}
	}
	return ignoreFile, nil
}

func (g *git) IgnoreFiles(path string, files []string) error {
	// If we're generating the ignore in the root of the project, we should ignore some Please stuff too
	if filepath.Dir(path) == "." && files == nil {
		files = defaultIgnoredFiles
	}

	ignore, err := openGitignore(filepath.Join(g.repoRoot, path))
	if err != nil {
		return err
	}

	defer ignore.Close()

	newLines := make([]string, 0, len(files))
	for _, file := range files {
		if _, ok := ignore.entries[file]; ok {
			continue
		}
		newLines = append(newLines, file)
	}

	if len(newLines) > 0 && !ignore.hasDoNotEdit {
		if _, err := fmt.Fprintln(ignore, "\n"+pleaseDoNotEdit); err != nil {
			return err
		}
	}
	for _, line := range newLines {
		if _, err := fmt.Fprintln(ignore, line); err != nil {
			return err
		}
	}
	return nil
}

func (g *git) GetIgnoreFile(path string) string {
	return filepath.Join(path, ignoreFileName)
}

func (g *git) Remove(names []string) error {
	cmd := exec.Command("git", append([]string{"rm", "-q"}, names...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git rm failed: %s\nOutput:\n%s", err, string(out))
	}
	return nil
}

func (g *git) ChangedLines() (map[string][]int, error) {
	cmd := exec.Command("git", "diff", "origin/master", "--unified=0", "--no-color", "--no-ext-diff")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git diff failed: %s\nOutput:\n%s", err, string(out))
	}
	return g.parseChangedLines(out)
}

func (g *git) parseChangedLines(input []byte) (map[string][]int, error) {
	m := map[string][]int{}
	fds, err := diff.ParseMultiFileDiff(input)
	for _, fd := range fds {
		m[strings.TrimPrefix(fd.NewName, "b/")] = g.parseHunks(fd.Hunks)
	}
	return m, err
}

func (g *git) parseHunks(hunks []*diff.Hunk) []int {
	ret := []int{}
	for _, hunk := range hunks {
		for i := 0; i < int(hunk.NewLines); i++ {
			ret = append(ret, int(hunk.NewStartLine)+i)
		}
	}
	return ret
}

func (g *git) Checkout(revision string) error {
	if out, err := exec.Command("git", "checkout", revision).CombinedOutput(); err != nil {
		return fmt.Errorf("git checkout of %s failed: %s\nOutput:\n%s", revision, err, string(out))
	}
	return nil
}

func (g *git) CurrentRevDate(format string) string {
	out, err := exec.Command("git", "show", "-s", "--format=%ct").CombinedOutput()
	if err != nil {
		return "Unknown"
	}
	timestamp, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		return err.Error()
	}
	t := time.Unix(timestamp, 0)
	return t.Format(format)
}

func (g *git) AreIgnored(files ...string) bool {
	if unignored := g.getUnIgnored(files...); len(unignored) == 0 {
		return true
	}
	return false
}

func (g *git) getUnIgnored(files ...string) []string {
	args := append([]string{"check-ignore"}, files...)
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		// if exit code is 1 none are ignored. else there was a fatal error so we assume none are ignored. either way return all files
		return files
	}
	ignored := make(map[string]bool, len(files))
	for _, f := range strings.Split(string(out), "\n") {
		if len(f) == 0 {
			continue
		}
		ignored[f] = true
	}
	unignored := make([]string, 0, len(files)-len(ignored))
	for _, f := range files {
		if !ignored[f] {
			unignored = append(unignored, f)
		}
	}
	return unignored
}
