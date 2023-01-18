package hashes

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/thought-machine/please/src/cli/logging"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/parse/asp"
)

var log = logging.Log

// RewriteHashes rewrites the hashes in a BUILD file.
func RewriteHashes(state *core.BuildState, labels []core.BuildLabel) {
	// Collect the targets per-package so we only rewrite each file once.
	m := map[*core.Package]map[string]string{}
	for _, l := range labels {
		pkg := state.Graph.PackageOrDie(l)
		for _, target := range pkg.AllChildren(state.Graph.TargetOrDie(l)) {
			// Ignore targets with no hash specified.
			if len(target.Hashes) == 0 {
				continue
			}
			h, err := state.TargetHasher.OutputHash(target)
			if err != nil {
				log.Fatalf("%s\n", err)
			}
			// Interior targets won't appear in the BUILD file directly, look for their parent instead.
			l := target.Label.Parent()
			hashStr := hex.EncodeToString(h)
			if m2, present := m[pkg]; present {
				m2[l.Name] = hashStr
			} else {
				m[pkg] = map[string]string{l.Name: hashStr}
			}
		}
	}
	for pkg, hashes := range m {
		if err := rewriteHashes(state, pkg.Filename, runtime.GOOS+"_"+runtime.GOARCH, hashes); err != nil {
			log.Fatalf("%s\n", err)
		}
	}
}

// rewriteHashes rewrites hashes in a single file.
func rewriteHashes(state *core.BuildState, filename, platform string, hashes map[string]string) error {
	log.Notice("Rewriting hashes in %s...", filename)
	p := asp.NewParser(state)
	stmts, err := p.ParseFileOnly(filename)
	if err != nil {
		return err
	}
	b, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	f := asp.NewFile(filename, b)
	lines := bytes.Split(b, []byte{'\n'})
	for k, v := range hashes {
		if err := rewriteHash(f, lines, stmts, platform, k, v); err != nil {
			return err
		}
	}
	return os.WriteFile(filename, bytes.Join(lines, []byte{'\n'}), 0664)
}

// rewriteHash rewrites a single hash on a statement.
func rewriteHash(f *asp.File, lines [][]byte, stmts []*asp.Statement, platform, name, hash string) error {
	stmt := asp.FindTarget(stmts, name)
	if stmt == nil {
		return fmt.Errorf("Can't find target %s to rewrite", name)
	} else if arg := asp.FindArgument(stmt, "hash", "hashes"); arg != nil {
		if arg.Value.Val != nil && arg.Value.Val.List != nil {
			for _, h := range arg.Value.Val.List.Values {
				pos := f.Pos(h.Pos)
				if line, ok := rewriteLine(lines[pos.Line-1], pos.Column, platform, h.Val.String, hash); ok {
					lines[pos.Line-1] = line
					return nil
				}
			}
		} else if arg.Value.Val != nil && arg.Value.Val.String != "" {
			h := arg.Value
			pos := f.Pos(h.Pos)
			if line, ok := rewriteLine(lines[pos.Line-1], pos.Column, platform, h.Val.String, hash); ok {
				lines[pos.Line-1] = line
				return nil
			}
		}
	}
	if platform != "" {
		return rewriteHash(f, lines, stmts, "", name, hash)
	}
	return fmt.Errorf("Can't find hash or hashes argument on %s", name)
}

// rewriteLine implements the rewriting logic within a single line.
// It returns the new line and true if it should be replaced, or false if not.
func rewriteLine(line []byte, start int, platform, current, new string) ([]byte, bool) {
	current = strings.Trim(current, `"`) // asp string literals are surrounded by quotes
	if strings.HasPrefix(current, platform) {
		if platform != "" {
			new = platform + ": " + new
		}
		return bytes.Join([][]byte{line[:start], []byte(new), line[start+len(current):]}, nil), true
	}
	return nil, false
}
