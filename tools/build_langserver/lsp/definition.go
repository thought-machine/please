package lsp

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/sourcegraph/go-lsp"

	"github.com/thought-machine/please/rules"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/parse/asp"
	"github.com/thought-machine/please/tools/build_langserver/lsp/astutils"
)

// definition implements 'go-to-definition' support.
// It is also used for go-to-declaration since we do not make a distinction between the two.
func (h *Handler) definition(params *lsp.TextDocumentPositionParams) ([]lsp.Location, error) {
	doc := h.doc(params.TextDocument.URI)
	ast := h.parseIfNeeded(doc)

	var locs []lsp.Location
	pos := aspPos(params.Position)
	asp.WalkAST(ast, func(expr *asp.Expression) bool {
		if !asp.WithinRange(pos, expr.Pos, expr.EndPos) {
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
	// It might also be a statement.
	asp.WalkAST(ast, func(stmt *asp.Statement) bool {
		if stmt.Ident != nil {
			endPos := stmt.Pos
			// TODO(jpoole): The AST should probably just have this information
			endPos.Column += len(stmt.Ident.Name)

			if !asp.WithinRange(pos, stmt.Pos, endPos) {
				return false
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
	uri := lsp.DocumentURI("file://" + path.Join(h.root, pkg.Filename))
	loc := lsp.Location{URI: uri}
	doc, err := h.maybeOpenDoc(uri)
	if err != nil {
		log.Warningf("failed to open doc for completion: %v", err)
		return loc
	}
	ast := h.parseIfNeeded(doc)

	// Try and find expression function calls
	asp.WalkAST(ast, func(expr *asp.Expression) bool {
		if expr.Val != nil && expr.Val.Call != nil {
			if findName(expr.Val.Call.Arguments) == l.Name {
				loc.Range = lsp.Range{
					Start: lsp.Position{Line: expr.Pos.Line, Character: expr.Pos.Column},
					End:   lsp.Position{Line: expr.EndPos.Line, Character: expr.EndPos.Column},
				}
			}
			return false
		}
		return true
	})

	asp.WalkAST(ast, func(stmt *asp.Statement) bool {
		if stmt.Ident != nil && stmt.Ident.Action != nil && stmt.Ident.Action.Call != nil {
			if findName(stmt.Ident.Action.Call.Arguments) == l.Name {
				loc.Range = lsp.Range{
					Start: lsp.Position{Line: stmt.Pos.Line, Character: stmt.Pos.Column},
					End:   lsp.Position{Line: stmt.EndPos.Line, Character: stmt.EndPos.Column},
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
			if data, err := rules.ReadAsset(f.Pos.Filename); err != nil {
				log.Warning("Failed to extract builtin rules for %s: %s", name, err)
				return lsp.Location{}
			} else if err := os.WriteFile(dest, data, 0644); err != nil {
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
