package lsp

import (
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/sourcegraph/go-lsp"

	"github.com/thought-machine/please/rules"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/parse/asp"
)

// definition implements 'go-to-definition' support.
// It is also used for go-to-declaration since we do not make a distinction between the two.
func (h *Handler) definition(params *lsp.TextDocumentPositionParams) ([]lsp.Location, error) {
	doc := h.doc(params.TextDocument.URI)
	ast := h.parseIfNeeded(doc)
	locs := []lsp.Location{}
	pos := aspPos(params.Position)
	asp.WalkAST(ast, func(expr *asp.Expression) bool {
		if asp.WithinRange(pos, expr.Pos, expr.EndPos) {
			if expr.Val.Ident != nil {
				if loc := h.findDefinition(expr.Val.Ident.Name); loc.URI != "" {
					locs = append(locs, loc)
					return false
				}
			}
			return true
		}
		return false
	})
	// It might also be a statement.
	asp.WalkAST(ast, func(stmt *asp.Statement) bool {
		if asp.WithinRange(pos, stmt.Pos, stmt.EndPos) {
			if stmt.Ident != nil {
				if loc := h.findDefinition(stmt.Ident.Name); loc.URI != "" {
					locs = append(locs, loc)
					return false
				}
			}
			return true
		}
		return false
	})
	return locs, nil
}

// findDefinition returns the location of a global of the given name.
func (h *Handler) findDefinition(name string) lsp.Location {
	if f, present := h.builtins[name]; present {
		if f.FuncDef.IsBuiltin && !strings.Contains(f.Pos.Filename, "/") {
			// Extract the builtin to a temporary location so the user can see it.
			dir, err := os.UserCacheDir()
			if err != nil {
				log.Warning("Cannot determine user cache dir: %s", err)
				return lsp.Location{}
			} else if err := os.MkdirAll(path.Join(dir, "please"), core.DirPermissions); err != nil {
				log.Warning("Cannot create cache dir: %s", err)
				return lsp.Location{}
			}
			dest := path.Join(dir, "please", f.Pos.Filename)
			if data, err := rules.Asset(f.Pos.Filename); err != nil {
				log.Warning("Failed to extract builtin rules for %s: %s", name, err)
				return lsp.Location{}
			} else if err := ioutil.WriteFile(dest, data, 0644); err != nil {
				log.Warning("Failed to extract builtin rules for %s: %s", name, err)
				return lsp.Location{}
			}
			f.Pos.Filename = dest
		}
		file := f.Pos.Filename
		if !path.IsAbs(file) {
			file = path.Join(h.root, f.Pos.Filename)
		}
		return lsp.Location{
			URI:   lsp.DocumentURI("file://" + file),
			Range: rng(f.Pos, f.EndPos),
		}
	}
	return lsp.Location{}
}
