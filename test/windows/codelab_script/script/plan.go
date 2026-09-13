package script

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Plan is what the runner consumes: every codelab reduced to an ordered list of steps.
type Plan struct {
	Codelabs []PlannedCodelab `json:"codelabs"`
}

// PlannedCodelab is one codelab's steps, plus a census of what its blocks turned out to be.
type PlannedCodelab struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Source string `json:"source"`
	// Set when the codelab has nothing anyone runs locally. github_actions teaches CI
	// configuration and is entirely YAML.
	NotRunnable string `json:"not_runnable,omitempty"`
	// How many blocks there were and what they were. The report reconciles against this: a
	// summary that counts only what it ran cannot tell you it ran almost nothing.
	Blocks map[string]int `json:"blocks"`
	Steps  []Step         `json:"steps"`
}

// Step is one thing for the runner to do.
type Step struct {
	Key string `json:"key"`
	// "run", "file", "chdir" or "skip".
	Kind    string `json:"kind"`
	Line    int    `json:"line"`
	Section string `json:"section,omitempty"`

	// kind=run.
	Command string `json:"command,omitempty"`
	// What the codelab shows this command printing. Advisory: the corpus is full of timings,
	// incrementality percentages and a randomly chosen greeting, so asserting on it would
	// produce flakes that discredit the whole check. Promoted to a requirement only by the
	// sidecar's assert directive.
	ExpectedOutput []string `json:"expected_output,omitempty"`
	Assert         string   `json:"assert,omitempty"`
	// Tools this step needs, checked on the machine at run time.
	Needs   []string `json:"needs,omitempty"`
	Timeout int      `json:"timeout,omitempty"`
	// A failure here does not mark the rest of the codelab blocked. True for a command that
	// only displays something, and wherever the sidecar says so.
	NonBlocking bool `json:"non_blocking,omitempty"`

	// kind=file.
	Path    string `json:"path,omitempty"`
	Mode    string `json:"mode,omitempty"`
	Content string `json:"content,omitempty"`

	// kind=chdir.
	Dir string `json:"dir,omitempty"`

	// kind=skip.
	Reason string `json:"reason,omitempty"`
	Detail string `json:"detail,omitempty"`
}

// A command line that only changes directory. The runner owns the working directory across steps,
// because a `cd` in a child process is lost the moment it exits - and the codelabs open with
// "mkdir getting_started_go && cd getting_started_go", with every later step depending on it.
var chdirRe = regexp.MustCompile(`^cd\s+([^\s;|&]+)\s*$`)

// BuildPlan classifies every block of every codelab and returns the plan.
//
// It fails rather than guessing. An unclassified block, a sidecar stanza whose text no longer
// matches the block it names, and a stanza that names nothing at all are all errors, reported
// together so that one pass over the output fixes all of them.
func BuildPlan(codelabs []Codelab, side *Sidecar) (*Plan, []error) {
	plan := &Plan{Codelabs: []PlannedCodelab{}}
	var errs []error

	for _, c := range codelabs {
		planned := PlannedCodelab{
			ID:     c.ID,
			Title:  c.Title,
			Source: c.Source,
			Blocks: map[string]int{},
			Steps:  []Step{},
		}
		if entry, ok := side.Lookup(c.ID); ok {
			planned.NotRunnable = entry.NotRunnable
		}

		for _, b := range c.Blocks {
			key := Key(c, b)
			kind, path, err := decide(c, b, key, side, planned.NotRunnable != "")
			if err != nil {
				errs = append(errs, err)
				continue
			}
			planned.Blocks["total"]++
			planned.Blocks[string(kind)]++

			entry, hasEntry := side.Lookup(key)
			if hasEntry {
				if err := checkMatches(c, b, entry, kind, path); err != nil {
					errs = append(errs, err)
					continue
				}
			}

			switch kind {
			case KindUnclassified:
				errs = append(errs, unclassifiedError(c, b, key))
			case KindIllustration, KindIgnore:
				// Carried in the census and nowhere else. Not a step, so it never
				// appears in the pass, fail or skip tallies.
			case KindFile:
				planned.Steps = append(planned.Steps, fileStep(b, key, path, entry))
			case KindCommand, KindTranscript:
				if hasEntry && entry.Skip != "" {
					planned.Steps = append(planned.Steps, Step{
						Key: key, Kind: "skip", Line: b.Line, Section: b.Section,
						Reason: entry.Skip, Detail: entry.Reason,
					})
					continue
				}
				steps, stepErrs := commandSteps(c, b, key, entry, side)
				planned.Steps = append(planned.Steps, steps...)
				errs = append(errs, stepErrs...)
			}
		}
		plan.Codelabs = append(plan.Codelabs, planned)
	}

	for _, key := range side.Unresolved() {
		errs = append(errs, fmt.Errorf("codelab_steps.conf: %s names no block in any codelab; the codelab it refers to has been edited, so the decision recorded there needs revisiting rather than dropping", key))
	}
	return plan, errs
}

// decide classifies a block the way the plan and the census both need it, so that the two cannot
// disagree about what a block is. A codelab declared not runnable has its reason recorded once, at
// the top, rather than a stanza per block restating it, so what no rule decides there is shown.
func decide(c Codelab, b Block, key string, side *Sidecar, notRunnable bool) (Kind, string, error) {
	kind, path, err := Classify(c, b, key, side)
	if err == nil && kind == KindUnclassified && notRunnable {
		kind = KindIllustration
	}
	return kind, path, err
}

func fileStep(b Block, key, path string, entry *Entry) Step {
	mode := "write"
	if entry != nil && entry.Mode != "" {
		mode = entry.Mode
	}
	return Step{
		Key: key, Kind: "file", Line: b.Line, Section: b.Section,
		Path: path, Mode: mode, Content: strings.Join(b.Body, "\n"),
	}
}

// Commands that only display something. Nothing later in a codelab can depend on one succeeding,
// so a failure - `tree -a` has no Windows counterpart that takes that flag - is recorded without
// marking every later step blocked.
var displayVerbs = map[string]bool{"tree": true, "cat": true, "which": true, "ls": true, "printenv": true}

// commandSteps turns a block into one step per command, splitting "a && b" so that a `cd` can
// become a chdir the runner applies to itself.
//
// A stanza can name a single command as well as a whole block, as "<key>.<n>", for the cases where
// one line of a block needs a decision the others do not: python_intro builds a pex and then runs
// it in the same block, and only the second of those depends on a shebang.
func commandSteps(c Codelab, b Block, key string, entry *Entry, side *Sidecar) ([]Step, []error) {
	commands, expect := Commands(b)
	var steps []Step
	var errs []error
	n := 0
	for _, command := range commands {
		for _, part := range splitChain(command) {
			n++
			stepKey := fmt.Sprintf("%s.%d", key, n)
			if m := chdirRe.FindStringSubmatch(part); m != nil {
				steps = append(steps, Step{
					Key: stepKey, Kind: "chdir", Line: b.Line,
					Section: b.Section, Dir: m[1],
				})
				continue
			}
			step := Step{
				Key: stepKey, Kind: "run", Line: b.Line,
				Section: b.Section, Command: part,
			}
			verb, _, _ := strings.Cut(part, " ")
			step.NonBlocking = displayVerbs[verb]
			applyEntry(&step, entry)
			if own, ok := side.Lookup(stepKey); ok {
				if own.Matches != "" && own.Matches != part {
					errs = append(errs, fmt.Errorf("%s:%d: %s: codelab_steps.conf expects %q here, but the command now says %q. The decision recorded there was made about different text, so re-read it before updating the stanza",
						c.Source, b.Line, stepKey, own.Matches, part))
					continue
				}
				if own.Skip != "" {
					steps = append(steps, Step{
						Key: stepKey, Kind: "skip", Line: b.Line, Section: b.Section,
						Reason: own.Skip, Detail: own.Reason,
					})
					continue
				}
				applyEntry(&step, own)
			}
			steps = append(steps, step)
		}
	}
	// The shown output belongs to the block, so it is attached to the last step of it: that is
	// the one whose output the codelab is displaying.
	if len(expect) > 0 && len(steps) > 0 {
		steps[len(steps)-1].ExpectedOutput = expect
	}
	return steps, errs
}

// applyEntry copies a stanza's run-time directives onto a step. Directives a stanza leaves unset
// leave the step's own values alone, so a command-level stanza refines a block-level one.
func applyEntry(step *Step, e *Entry) {
	if e == nil {
		return
	}
	if len(e.Needs) > 0 {
		step.Needs = e.Needs
	}
	if e.Assert != "" {
		step.Assert = e.Assert
	}
	if e.Timeout != 0 {
		step.Timeout = e.Timeout
	}
	if e.NonBlocking {
		step.NonBlocking = true
	}
}

// splitChain splits "mkdir x && cd x" into its parts. Only "&&" is split: the codelabs use "|" to
// build real pipelines, which have to reach the shell intact.
func splitChain(command string) []string {
	var out []string
	for _, part := range strings.Split(command, "&&") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	if len(out) == 0 {
		return []string{command}
	}
	return out
}

// checkMatches enforces the sidecar's drift guard: the stanza says what it expects to find, and
// extraction fails if the block no longer says it.
func checkMatches(c Codelab, b Block, entry *Entry, kind Kind, path string) error {
	if entry.Matches == "" {
		return nil
	}
	var got []string
	switch kind {
	case KindFile:
		// The path alone pins little when the stanza is what chose the path, so the first
		// line of the contents is accepted too: `[Alias "puku"]` says which block is meant.
		got = []string{path, firstLine(b.Body)}
	case KindCommand, KindTranscript:
		commands, _ := Commands(b)
		got = commands
	default:
		// An illustration or an ignored block is pinned by its first line, which is what the
		// census prints and so what a person writing the stanza has in front of them.
		got = []string{firstLine(b.Body)}
	}
	for _, g := range got {
		if strings.TrimSpace(g) == entry.Matches {
			return nil
		}
	}
	return fmt.Errorf("%s:%d: %s: codelab_steps.conf expects %q here, but the block now says %q. The decision recorded there was made about different text, so re-read it before updating the stanza",
		c.Source, b.Line, entry.Key, entry.Matches, strings.Join(got, " / "))
}

// unclassifiedError is the authoring experience for codelab_steps.conf, so it says what to write.
func unclassifiedError(c Codelab, b Block, key string) error {
	first := ""
	for _, line := range b.Body {
		if strings.TrimSpace(line) != "" {
			first = strings.TrimSpace(line)
			break
		}
	}
	if len(first) > 60 {
		first = first[:60] + "..."
	}
	return fmt.Errorf("%s:%d: cannot tell what this block is (fence %q, introduced by %q, starting %q).\n"+
		"Decide in test/windows/codelab_steps.conf, with the reason above it:\n\n"+
		"; why this block is what it is\n[%s]\nmatches = %s\nkind = illustration",
		c.Source, b.Line, b.Lang, truncate(b.Intro, 60), first, key, first)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// Census renders one line per block: the key, what it was decided to be, and its first line. This
// is how codelab_steps.conf is authored from nothing, and the golden file the unit test diffs -
// a compact one-line-per-block record stays reviewable in a way a golden JSON plan would not.
func Census(codelabs []Codelab, side *Sidecar) string {
	var b strings.Builder
	for _, c := range codelabs {
		entry, ok := side.Entries[c.ID]
		notRunnable := ok && entry.NotRunnable != ""
		for _, block := range c.Blocks {
			key := Key(c, block)
			kind, path, err := decide(c, block, key, side, notRunnable)
			detail := path
			if err != nil {
				kind, detail = KindUnclassified, err.Error()
			}
			if detail == "" {
				detail = firstLine(block.Body)
			}
			fmt.Fprintf(&b, "%-56s %-13s %s\n", key, kind, truncate(detail, 60))
		}
	}
	return b.String()
}

func firstLine(body []string) string {
	for _, line := range body {
		if strings.TrimSpace(line) != "" {
			return strings.TrimSpace(line)
		}
	}
	return ""
}

// StepKeys lists every step key in the plan, sorted. Used to check that nothing in
// codelab_known_failures.txt names a step that no longer exists.
func (p *Plan) StepKeys() []string {
	var keys []string
	for _, c := range p.Codelabs {
		for _, s := range c.Steps {
			keys = append(keys, s.Key)
		}
	}
	sort.Strings(keys)
	return keys
}
