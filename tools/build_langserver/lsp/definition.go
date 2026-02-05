package lsp

import (
	"os"
	"path/filepath"

	"github.com/sourcegraph/go-lsp"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/parse/asp"
	"github.com/thought-machine/please/tools/build_langserver/lsp/astutils"
)

// definition implements 'go-to-definition' support.
// It is also used for go-to-declaration since we do not make a distinction between the two.
func (h *Handler) definition(params *lsp.TextDocumentPositionParams) ([]lsp.Location, error) {
	doc := h.doc(params.TextDocument.URI)
	ast := h.parseIfNeeded(doc)
	f := doc.AspFile()

	locs := []lsp.Location{}
	pos := aspPos(params.Position)
	asp.WalkAST(ast, func(expr *asp.Expression) bool {
		exprStart := f.Pos(expr.Pos)
		exprEnd := f.Pos(expr.EndPos)
		if !asp.WithinRange(pos, exprStart, exprEnd) {
			return false
		}
		if expr.Val.Ident != nil {
			if loc := h.findGlobal(expr.Val.Ident.Name); loc.URI != "" {
				locs = append(locs, loc)
			}
			return false
		}
		if expr.Val.String != "" {
			label := astutils.TrimStrLit(expr.Val.String)
			if loc := h.findLabel(doc.PkgName, label); loc.URI != "" {
				locs = append(locs, loc)
			}
			return false
		}
		return true
	})
	// It might also be a statement (e.g. a function call like go_library(...))
	asp.WalkAST(ast, func(stmt *asp.Statement) bool {
		if stmt.Ident != nil {
			stmtStart := f.Pos(stmt.Pos)
			endPos := stmtStart
			// TODO(jpoole): The AST should probably just have this information
			endPos.Column += len(stmt.Ident.Name)

			if !asp.WithinRange(pos, stmtStart, endPos) {
				return true // continue to other statements
			}
			if loc := h.findGlobal(stmt.Ident.Name); loc.URI != "" {
				locs = append(locs, loc)
			}
			return false
		}
		return true
	})
	return locs, nil
}

// findLabel will attempt to parse the package containing the label to determine the position within that build
// file that that rule exists
func (h *Handler) findLabel(currentPath, label string) lsp.Location {
	l, err := core.TryParseBuildLabel(label, currentPath, "")

	// If we can't parse this as a build label, it might be a file on disk
	if err != nil {
		p := filepath.Join(h.root, currentPath, label)
		if _, err := os.Lstat(p); err == nil {
			return lsp.Location{URI: lsp.DocumentURI("file://" + p)}
		}
		return lsp.Location{}
	}

	pkg := h.state.Graph.PackageByLabel(l)
	if pkg == nil {
		return lsp.Location{}
	}
	uri := lsp.DocumentURI("file://" + filepath.Join(h.root, pkg.Filename))
	loc := lsp.Location{URI: uri}
	doc, err := h.maybeOpenDoc(uri)
	if err != nil {
		log.Warningf("failed to open doc for completion: %v", err)
		return loc
	}
	ast := h.parseIfNeeded(doc)
	f := doc.AspFile()

	// Try and find expression function calls
	asp.WalkAST(ast, func(expr *asp.Expression) bool {
		if expr.Val != nil && expr.Val.Call != nil {
			if findName(expr.Val.Call.Arguments) == l.Name {
				start := f.Pos(expr.Pos)
				end := f.Pos(expr.EndPos)
				loc.Range = lsp.Range{
					Start: lsp.Position{Line: start.Line, Character: start.Column},
					End:   lsp.Position{Line: end.Line, Character: end.Column},
				}
			}
			return false
		}
		return true
	})

	asp.WalkAST(ast, func(stmt *asp.Statement) bool {
		if stmt.Ident != nil && stmt.Ident.Action != nil && stmt.Ident.Action.Call != nil {
			if findName(stmt.Ident.Action.Call.Arguments) == l.Name {
				start := f.Pos(stmt.Pos)
				end := f.Pos(stmt.EndPos)
				loc.Range = lsp.Range{
					Start: lsp.Position{Line: start.Line, Character: start.Column},
					End:   lsp.Position{Line: end.Line, Character: end.Column},
				}
			}
			return false
		}
		return true
	})

	return loc
}

// findName finds the name arguments to a function call. The name must be a simple string lit as we don't evaluate the
// package to deal with more complex expressions.
func findName(args []asp.CallArgument) string {
	for _, arg := range args {
		if arg.Name == "name" {
			if arg.Value.Val != nil && arg.Value.Val.String != "" {
				return astutils.TrimStrLit(arg.Value.Val.String)
			}
		}
	}
	return ""
}

// findGlobal returns the location of a global of the given name.
func (h *Handler) findGlobal(name string) lsp.Location {
	h.mutex.Lock()
	f, present := h.builtins[name]
	h.mutex.Unlock()
	if present {
		filename := f.Pos.Filename
		// Make path absolute if it's relative
		if !filepath.IsAbs(filename) {
			filename = filepath.Join(h.root, filename)
		}
		return lsp.Location{
			URI:   lsp.DocumentURI("file://" + filename),
			Range: rng(f.Pos, f.EndPos),
		}
	}
	return lsp.Location{}
}
