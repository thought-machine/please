package script

import (
	"fmt"
	"regexp"
	"strings"
)

// Kind is what a fenced block turned out to be.
type Kind string

const (
	// A file to write, with a path and contents.
	KindFile Kind = "file"
	// Commands to run.
	KindCommand Kind = "command"
	// Commands shown with the output they produced. Runnable, but the output is what the
	// codelab saw on its author's machine and is advisory here; see plan.go.
	KindTranscript Kind = "transcript"
	// Shown for reference and never run: a `tree -a` listing, an expected build result, the
	// YAML of a CI workflow, a walk-through of an interactive session.
	KindIllustration Kind = "illustration"
	// The sidecar says to leave this block alone entirely, with a reason.
	KindIgnore Kind = "ignore"
	// No rule decided. Always an error; see Classify.
	KindUnclassified Kind = "unclassified"
)

// A heading whose entire text is a backticked path, e.g. "### `src/BUILD`". The dominant
// convention: genrule, go_intro, python_intro and using_plugins introduce every file this way.
var fileHeadingRe = regexp.MustCompile("^#{2,6}\\s+`([^`]+)`\\s*$")

// The same thing said in prose and ending in a colon, which is how puku and k8s do it throughout:
// "Create a file `hello_service/service.go`:". Those two use no file headings at all.
var fileProseRe = regexp.MustCompile("`([^`]+)`[^`]*:\\s*$")

// The name half of an inline environment assignment.
var envPrefixRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)

// A transcript line: the command follows the prompt, and the rest of the block is its output.
var promptRe = regexp.MustCompile(`^\$\s+(.*)$`)

// An introducing line promising output rather than asking for anything to be run. Several codelabs
// tag such a block ```bash anyway - using_plugins shows two `tree -a` listings that way, k8s a
// job-control trace - and running those lines would fail on every platform, Linux included.
var outputIntroRe = regexp.MustCompile(`(?i)\b(output|should look like|should see|will see|prints|looks like|similar to)\b`)

// Fence languages that mean "this is the content of a file", given a path can be found for it.
var fileLangs = map[string]bool{
	"python": true, "go": true, "golang": true, "yaml": true,
	"ini": true, "shell script": true, "dockerfile": true,
}

// Fence languages that might hold commands.
var commandLangs = map[string]bool{"bash": true, "sh": true, "shell": true, "text": true, "": true}

// The first word of a line that is plausibly a command. Deliberately a list rather than a pattern:
// the codelabs' output blocks are full of lines that look like commands to a pattern, and a wrong
// guess here becomes a step that fails on every platform for reasons that have nothing to do with
// Windows. Anything not listed is unclassified, which asks rather than guesses.
var commandVerbs = map[string]bool{
	"plz": true, "./pleasew": true, "pleasew": true, "./plz": true,
	"go": true, "git": true, "puku": true, "pip": true, "pip3": true,
	"python": true, "python3": true, "docker": true, "kubectl": true, "minikube": true,
	"mkdir": true, "cd": true, "echo": true, "cat": true, "ls": true, "rm": true,
	"touch": true, "cp": true, "mv": true, "curl": true, "wget": true, "tree": true,
	"which": true, "pkill": true, "eval": true, "export": true, "source": true,
	"printf": true, "wc": true, "sort": true, "sed": true, "grep": true, "chmod": true,
}

// Classify works out what a block is, consulting the sidecar first.
//
// The order matters and each step earns its place:
//
//  1. The sidecar, which overrides everything and always carries a reason.
//  2. A heading immediately above whose whole text is a backticked path: a file.
//  3. A command-shaped fence, checked before the prose rule because a sentence naming a file is
//     as often followed by the command that creates it as by its contents.
//  4. A file-shaped fence with a path findable in the prose before or after it.
//  5. Nothing: KindUnclassified, which the caller must treat as fatal.
//
// Returning KindUnclassified rather than quietly bucketing into "other" is the whole design. It is
// what makes a codelab edit that introduces an unreadable block go red on Linux, in an ordinary
// unit test, instead of silently shrinking what the Windows job checks.
func Classify(c Codelab, b Block, key string, side *Sidecar) (Kind, string, error) {
	if entry, ok := side.Lookup(key); ok {
		kind, path, err := entry.Kind(b)
		if err != nil {
			return KindUnclassified, "", fmt.Errorf("%s:%d: %s: %w", c.Source, b.Line, key, err)
		}
		if kind != "" {
			return kind, path, nil
		}
	}

	if m := fileHeadingRe.FindStringSubmatch(b.Intro); m != nil {
		return KindFile, m[1], nil
	}

	if commandLangs[b.Lang] {
		if hasPrompt(b.Body) {
			return KindTranscript, "", nil
		}
		if outputIntroRe.MatchString(b.Intro) {
			return KindIllustration, "", nil
		}
		if isCommandish(b.Body) {
			return KindCommand, "", nil
		}
	}

	// Only the prose before a block is read for a path. The line after was tried and named the
	// wrong file on its first outing: puku's "Add a filegroup for go.mod at `BUILD`:" is
	// followed by "Update your `.plzconfig`:", which introduces the next block, not this one.
	if fileLangs[b.Lang] || commandLangs[b.Lang] {
		if path, ok := proseFilePath(b.Intro); ok {
			return KindFile, path, nil
		}
	}

	return KindUnclassified, "", nil
}

// Key is the identifier a step is known by, in codelab_steps.conf, in
// codelab_known_failures.txt and in the runner's report.
//
// "<codelab>::<section-slug>/b<n>", with the command's position appended for a block holding
// several. Readable rather than hashed, because the failures file is a findings record someone
// has to read; an edit in one section does not renumber another, which a whole-file ordinal could
// not promise, and codelab_steps.conf pins the text of what it names so an edit within a section
// cannot silently move a decision onto a different block.
func Key(c Codelab, b Block) string {
	return fmt.Sprintf("%s::%s/b%d", c.ID, b.SectionSlug, b.Ordinal)
}

func hasPrompt(body []string) bool {
	for _, line := range body {
		if promptRe.MatchString(line) {
			return true
		}
	}
	return false
}

// isCommandish says whether the first non-blank line of a block starts with a word we recognise as
// a command, or with an inline environment assignment such as GODEBUG="installgoroot=all".
func isCommandish(body []string) bool {
	for _, line := range body {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		first, _, _ := strings.Cut(line, " ")
		if commandVerbs[first] {
			return true
		}
		// An inline environment prefix: VAR=value cmd. Bash syntax, so these are exactly the
		// steps most likely to fail on Windows - but they are commands, and saying so is what
		// lets them run and be reported rather than sit unreadable. The value may be quoted,
		// as in GODEBUG="installgoroot=all", so only the name is checked.
		return envPrefixRe.MatchString(first)
	}
	return false
}

// Commands splits a block into the commands to run and, for each, the output shown after it.
//
// In a transcript the output belongs to the command above it, not to the block. genrule shows
// `$ plz build` with its build summary and then `$ cat` with a word count; attaching both to the
// last command made the first native run report that none of the cat's lines appeared.
func Commands(b Block) (commands []string, expect [][]string) {
	prompted := hasPrompt(b.Body)
	for _, line := range b.Body {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if !prompted {
			commands = append(commands, line)
			expect = append(expect, nil)
			continue
		}
		if m := promptRe.FindStringSubmatch(line); m != nil {
			commands = append(commands, m[1])
			expect = append(expect, nil)
		} else if len(expect) > 0 {
			expect[len(expect)-1] = append(expect[len(expect)-1], line)
		}
	}
	return commands, expect
}

// proseFilePath pulls a file path out of a sentence such as
// "Add the following to `common/docker/BUILD`:".
func proseFilePath(line string) (string, bool) {
	m := fileProseRe.FindStringSubmatch(line)
	if m == nil || !isRepoPath(m[1]) {
		return "", false
	}
	return m[1], true
}

// isRepoPath says whether a backticked token is plausibly a file in the repo being built.
//
// The exclusions are not hypothetical. Each one is a line in the codelabs as they stand that would
// otherwise be read as a file to create:
//   - a space or a URL scheme: prose, or a link
//   - a leading slash: a [build] path value, such as `/usr/local/go/bin/go` in puku
//   - a dot in the first segment: a module path, such as `github.com/stretchr/testify` in go_intro
func isRepoPath(s string) bool {
	if s == "" || strings.ContainsAny(s, " \t") || strings.Contains(s, "://") {
		return false
	}
	if strings.HasPrefix(s, "/") {
		return false
	}
	first, _, _ := strings.Cut(s, "/")
	if strings.Contains(first, ".") && !strings.HasPrefix(first, ".") {
		return false
	}
	// A bare BUILD is the one file name without a slash or a dot that the codelabs create.
	return s == "BUILD" || strings.Contains(s, "/") || strings.Contains(s, ".")
}
