package script

import (
	"fmt"
	"strconv"
	"strings"
)

// Sidecar is test/windows/codelab_steps.conf: the decisions about what the codelabs mean that the
// Markdown cannot express, kept out of the prose because the codelabs are documentation and have
// to read as documentation.
//
// Every stanza carries a reason, enforced rather than encouraged: a stanza with no comment above it
// is a parse error. This file is read later by whoever decides what to do about the codelabs that
// cannot work on Windows, and a bare directive would tell them nothing.
type Sidecar struct {
	Entries map[string]*Entry
	// Order the stanzas appeared in, for stable reporting.
	Order []string
}

// Entry is one stanza.
type Entry struct {
	Key string
	// The comment above the stanza. Required.
	Reason string
	// The command or path this stanza expects to find at Key. Extraction fails if the block
	// there no longer says this, so an edit to a codelab cannot silently move a decision onto
	// a different block. This is the drift guard; the keys themselves are readable, not hashed.
	Matches string
	// "file:<path>", "command", "transcript", "illustration" or "ignore". Overrides the
	// heuristics outright.
	KindDirective string
	// "write" (the default) or "merge", for a file. Several codelabs show a .plzconfig under a
	// heading that names the whole file when what they mean is a fragment to add to what
	// plz init already wrote. Writing those verbatim drops the earlier keys and manufactures a
	// failure that has nothing to do with Windows.
	Mode string
	// A reason class saying this step cannot run here at all, e.g. "unix-shell".
	Skip string
	// Tools the step needs, checked on the machine at run time: docker, kubectl, minikube,
	// network, github-api, interactive.
	Needs []string
	// A substring that must appear in the output, promoting one line from advisory to required.
	Assert string
	// Seconds; 0 means the runner's default.
	Timeout int
	// Codelab-level: this codelab has nothing to run, with this as the reason.
	NotRunnable string
	// Set by "blocking = false": a failure here does not mark the rest of the codelab blocked.
	// For a step nothing later depends on, so that a bash-only line does not hide every
	// finding after it.
	NonBlocking bool
	// Set when something resolved this stanza against a real block, so ParseSidecar's caller can
	// report the ones that matched nothing.
	Resolved bool
}

// Kind returns the kind this stanza forces, if any, and the path for a file.
func (e *Entry) Kind(b Block) (Kind, string, error) {
	if e.KindDirective == "" {
		return "", "", nil
	}
	directive, path, hasPath := strings.Cut(e.KindDirective, ":")
	switch Kind(directive) {
	case KindFile:
		if !hasPath || path == "" {
			return "", "", fmt.Errorf(`kind = file needs a path, as "file:src/BUILD"`)
		}
		return KindFile, path, nil
	case KindCommand, KindTranscript, KindIllustration, KindIgnore:
		if hasPath {
			return "", "", fmt.Errorf("kind = %s takes no path", directive)
		}
		return Kind(directive), "", nil
	}
	return "", "", fmt.Errorf("unknown kind %q", e.KindDirective)
}

// Lookup finds the stanza for a key and marks it resolved.
func (s *Sidecar) Lookup(key string) (*Entry, bool) {
	e, ok := s.Entries[key]
	if ok {
		e.Resolved = true
	}
	return e, ok
}

// Unresolved lists stanzas that matched no block, in file order. A stanza that names nothing is
// a decision about a codelab that has since been edited, and is reported rather than ignored.
func (s *Sidecar) Unresolved() []string {
	var out []string
	for _, key := range s.Order {
		if !s.Entries[key].Resolved {
			out = append(out, key)
		}
	}
	return out
}

// ParseSidecar reads codelab_steps.conf.
//
// .plzconfig-flavoured: ";" comments, "[stanza]" headers, "key = value" directives. That is this
// repo's idiom for a file a person maintains by hand, and it puts the reason on the line above the
// decision where it belongs.
func ParseSidecar(filename, content string) (*Sidecar, error) {
	side := &Sidecar{Entries: map[string]*Entry{}}
	var reason []string
	var current *Entry

	for i, line := range strings.Split(content, "\n") {
		lineno := i + 1
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			// A blank line separates the file's own header from the first stanza, and one
			// stanza from the next. It also discards a comment, so that a reason cannot
			// drift away from what it explains.
			reason = nil
			continue
		}
		if strings.HasPrefix(trimmed, ";") || strings.HasPrefix(trimmed, "#") {
			reason = append(reason, strings.TrimSpace(strings.TrimLeft(trimmed, ";# ")))
			continue
		}
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			key := strings.TrimSpace(trimmed[1 : len(trimmed)-1])
			if key == "" {
				return nil, fmt.Errorf("%s:%d: empty stanza name", filename, lineno)
			}
			if _, dup := side.Entries[key]; dup {
				return nil, fmt.Errorf("%s:%d: %s appears twice", filename, lineno, key)
			}
			if len(reason) == 0 {
				return nil, fmt.Errorf("%s:%d: %s has no reason above it; every stanza here needs one, because this file is what the decision about the codelabs will be taken from", filename, lineno, key)
			}
			current = &Entry{Key: key, Reason: strings.Join(reason, " ")}
			side.Entries[key] = current
			side.Order = append(side.Order, key)
			reason = nil
			continue
		}

		name, value, ok := strings.Cut(trimmed, "=")
		if !ok {
			return nil, fmt.Errorf("%s:%d: expected a stanza, a comment, or 'name = value'", filename, lineno)
		}
		if current == nil {
			return nil, fmt.Errorf("%s:%d: %s appears before any stanza", filename, lineno, strings.TrimSpace(name))
		}
		if err := current.set(strings.TrimSpace(name), strings.TrimSpace(value)); err != nil {
			return nil, fmt.Errorf("%s:%d: %s: %w", filename, lineno, current.Key, err)
		}
	}
	return side, nil
}

func (e *Entry) set(name, value string) error {
	switch strings.ToLower(name) {
	case "matches":
		e.Matches = value
	case "kind":
		e.KindDirective = value
	case "mode":
		if value != "write" && value != "merge" {
			return fmt.Errorf("mode is write or merge, not %q", value)
		}
		e.Mode = value
	case "skip":
		e.Skip = value
	case "needs":
		for _, need := range strings.Split(value, ",") {
			if need = strings.TrimSpace(need); need != "" {
				e.Needs = append(e.Needs, need)
			}
		}
	case "assert":
		e.Assert = value
	case "timeout":
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("timeout is a number of seconds: %w", err)
		}
		e.Timeout = n
	case "not-runnable":
		e.NotRunnable = value
	case "blocking":
		b, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("blocking is true or false: %w", err)
		}
		e.NonBlocking = !b
	default:
		return fmt.Errorf("unknown directive %q", name)
	}
	return nil
}
