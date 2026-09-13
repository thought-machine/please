// Package script turns the published codelabs into something a machine can replay.
//
// The codelabs at https://please.build/codelabs.html are the front door for new users, and nothing
// anywhere has ever executed a line of them. This package reads docs/codelabs/*.md and produces an
// ordered plan of the files each one tells you to create and the commands it tells you to run;
// test/windows/run_codelabs.ps1 replays that plan on a real Windows machine.
//
// The plan is derived from the Markdown rather than transcribed into fixtures, so that the check
// and the published page cannot drift apart. That means living with conventions the codelabs were
// never written to satisfy. Where a convention runs out, the answer is never to guess: an
// unclassified block is a hard error, and the decision gets written down in codelab_steps.conf
// with a reason. A tolerant parser that silently ignored what it could not read would emit a
// three-step plan for a thirty-step codelab, pass, and tell nobody anything.
//
// Parsed line by line with no Markdown library, like docs/codelabs/codelab_template.go, which
// reads the same front matter for the index page. A line scanner gives exact line numbers, which
// is what the error messages here are made of.
package script

import (
	"regexp"
	"strings"
)

// Codelab is one .md file: its front matter and every fenced block in it.
type Codelab struct {
	ID     string
	Title  string
	Status string
	// Path as given, so a failure can be traced back to a file.
	Source string
	Blocks []Block
}

// Block is one fenced block, with the context needed to work out what it is.
type Block struct {
	// The fence's language tag, empty for a bare ```.
	Lang string
	Body []string
	// 1-based line of the opening fence.
	Line int
	// Nearest non-blank line above the fence: a heading or a sentence that says what the block
	// is, as in "Add the following to `common/docker/BUILD`:".
	Intro string
	// Text of the enclosing "##" heading, and its slug. Blocks are keyed by section rather than
	// by position in the file so that an edit in one section does not renumber another.
	Section     string
	SectionSlug string
	// Ordinal of this block within its section, from 1.
	Ordinal int
}

var (
	headingRe  = regexp.MustCompile(`^(#{1,6})\s+(.*?)\s*$`)
	nonSlugRe  = regexp.MustCompile(`[^a-z0-9]+`)
	frontKeyRe = regexp.MustCompile(`^([A-Za-z ]+):\s*(.*)$`)
)

// ParseCodelab reads one codelab into its blocks. It does not classify them; see Classify.
func ParseCodelab(filename, content string) Codelab {
	lines := strings.Split(content, "\n")
	codelab := Codelab{Source: filename}
	readFrontMatter(&codelab, lines)

	section, slug := "", ""
	ordinal := 0

	for i := 0; i < len(lines); i++ {
		if m := headingRe.FindStringSubmatch(lines[i]); m != nil {
			// Only "##" starts a new section. The codelabs use "###" for sub-steps and for
			// file headings, both of which belong to the section around them.
			if len(m[1]) == 2 {
				section = m[2]
				slug = slugify(m[2])
				ordinal = 0
			}
			continue
		}
		if !strings.HasPrefix(lines[i], "```") {
			continue
		}
		// An unterminated fence takes the rest of the file. The codelabs have none, but a
		// half-written one should say so rather than silently swallowing every block after it.
		end := i + 1
		for end < len(lines) && !strings.HasPrefix(lines[end], "```") {
			end++
		}
		ordinal++
		codelab.Blocks = append(codelab.Blocks, Block{
			Lang:        strings.TrimSpace(strings.TrimPrefix(lines[i], "```")),
			Body:        lines[i+1 : end],
			Line:        i + 1,
			Intro:       nearestNonBlank(lines, i, -1),
			Section:     section,
			SectionSlug: slug,
			Ordinal:     ordinal,
		})
		i = end
	}
	return codelab
}

// readFrontMatter reads the "key: value" header the codelabs open with, which runs until the first
// blank line. The same shape codelab_template.go reads, and only the fields this needs.
func readFrontMatter(codelab *Codelab, lines []string) {
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			return
		}
		m := frontKeyRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(m[1])) {
		case "id":
			codelab.ID = strings.TrimSpace(m[2])
		case "summary":
			codelab.Title = strings.TrimSpace(m[2])
		case "status":
			codelab.Status = strings.TrimSpace(m[2])
		}
	}
}

// nearestNonBlank walks from i in the given direction and returns the first non-blank line, or ""
// if there is none. Blank lines are skipped and nothing else is.
func nearestNonBlank(lines []string, i, step int) string {
	for j := i + step; j >= 0 && j < len(lines); j += step {
		if strings.TrimSpace(lines[j]) != "" {
			return lines[j]
		}
	}
	return ""
}

// slugify turns a heading into the section part of a step key: lowercase, words joined by
// hyphens. Readable, because these keys end up in codelab_known_failures.txt, which is a findings
// record someone has to read.
func slugify(s string) string {
	s = nonSlugRe.ReplaceAllString(strings.ToLower(s), "-")
	return strings.Trim(s, "-")
}
