package lsp

import (
	"path/filepath"

	"github.com/sourcegraph/go-lsp"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/parse/asp"
	"github.com/thought-machine/please/src/query"
	"github.com/thought-machine/please/tools/build_langserver/lsp/astutils"
)

// references implements 'find all references' support.
func (h *Handler) references(params *lsp.ReferenceParams) ([]lsp.Location, error) {
	doc := h.doc(params.TextDocument.URI)
	ast := h.parseIfNeeded(doc)
	f := doc.AspFile()
	pos := aspPos(params.Position)

	// Check if cursor is on a function definition (def funcname(...))
	var funcName string
	asp.WalkAST(ast, func(stmt *asp.Statement) bool {
		if stmt.FuncDef != nil {
			stmtStart := f.Pos(stmt.Pos)
			// Check if cursor is on the function name
			nameEnd := stmtStart
			nameEnd.Column += len("def ") + len(stmt.FuncDef.Name)
			if asp.WithinRange(pos, stmtStart, nameEnd) {
				funcName = stmt.FuncDef.Name
				return false
			}
		}
		return true
	})

	// If we found a function definition, find all calls to it
	if funcName != "" {
		return h.findFunctionReferences(funcName, params.Context.IncludeDeclaration)
	}

	// Otherwise, look for build label references
	return h.findLabelReferences(doc, ast, f, pos, params.Context.IncludeDeclaration)
}

// findFunctionReferences finds all calls to a function across all BUILD files.
func (h *Handler) findFunctionReferences(funcName string, includeDeclaration bool) ([]lsp.Location, error) {
	locs := []lsp.Location{}

	// Search all packages for calls to this function
	for _, pkg := range h.state.Graph.PackageMap() {
		uri := lsp.DocumentURI("file://" + filepath.Join(h.root, pkg.Filename))
		refDoc, err := h.maybeOpenDoc(uri)
		if err != nil {
			continue
		}
		refAst := h.parseIfNeeded(refDoc)
		refFile := refDoc.AspFile()

		// Find all statement calls to the function (e.g., go_library(...))
		asp.WalkAST(refAst, func(stmt *asp.Statement) bool {
			if stmt.Ident != nil && stmt.Ident.Name == funcName {
				start := refFile.Pos(stmt.Pos)
				end := start
				end.Column += len(funcName)
				locs = append(locs, lsp.Location{
					URI: uri,
					Range: lsp.Range{
						Start: lsp.Position{Line: start.Line - 1, Character: start.Column - 1},
						End:   lsp.Position{Line: end.Line - 1, Character: end.Column - 1},
					},
				})
			}
			return true
		})

		// Find expression calls (e.g., x = go_library(...))
		asp.WalkAST(refAst, func(expr *asp.Expression) bool {
			if expr.Val.Ident != nil && expr.Val.Ident.Name == funcName && len(expr.Val.Ident.Action) > 0 && expr.Val.Ident.Action[0].Call != nil {
				start := refFile.Pos(expr.Pos)
				end := start
				end.Column += len(funcName)
				locs = append(locs, lsp.Location{
					URI: uri,
					Range: lsp.Range{
						Start: lsp.Position{Line: start.Line - 1, Character: start.Column - 1},
						End:   lsp.Position{Line: end.Line - 1, Character: end.Column - 1},
					},
				})
			}
			return true
		})
	}

	// Include the definition itself if requested
	if includeDeclaration {
		h.mutex.Lock()
		if builtin, ok := h.builtins[funcName]; ok {
			filename := builtin.Pos.Filename
			if !filepath.IsAbs(filename) {
				filename = filepath.Join(h.root, filename)
			}
			locs = append(locs, lsp.Location{
				URI:   lsp.DocumentURI("file://" + filename),
				Range: rng(builtin.Pos, builtin.EndPos),
			})
		}
		h.mutex.Unlock()
	}

	return locs, nil
}

// findLabelReferences finds all references to a build label.
func (h *Handler) findLabelReferences(doc *doc, ast []*asp.Statement, f *asp.File, pos asp.FilePosition, includeDeclaration bool) ([]lsp.Location, error) {
	var targetLabel core.BuildLabel
	var targetName string

	// Check if cursor is on a string (build label)
	asp.WalkAST(ast, func(expr *asp.Expression) bool {
		exprStart := f.Pos(expr.Pos)
		exprEnd := f.Pos(expr.EndPos)
		if !asp.WithinRange(pos, exprStart, exprEnd) {
			return false
		}
		if expr.Val.String != "" {
			label := astutils.TrimStrLit(expr.Val.String)
			if l, err := core.TryParseBuildLabel(label, doc.PkgName, ""); err == nil {
				targetLabel = l
			}
			return false
		}
		return true
	})

	// Check if cursor is on a target definition (name = "...")
	if targetLabel.IsEmpty() {
		asp.WalkAST(ast, func(stmt *asp.Statement) bool {
			if stmt.Ident != nil && stmt.Ident.Action != nil && stmt.Ident.Action.Call != nil {
				stmtStart := f.Pos(stmt.Pos)
				stmtEnd := f.Pos(stmt.EndPos)
				if asp.WithinRange(pos, stmtStart, stmtEnd) {
					if name := findName(stmt.Ident.Action.Call.Arguments); name != "" {
						targetLabel = core.BuildLabel{PackageName: doc.PkgName, Name: name}
						targetName = name
					}
				}
				return false
			}
			return true
		})
	}

	if targetLabel.IsEmpty() {
		return []lsp.Location{}, nil
	}

	// Use query.FindRevdeps to find all reverse dependencies
	// Parameters: hidden=false, followSubincludes=true, includeSubrepos=true, depth=-1 (unlimited)
	revdeps := query.FindRevdeps(h.state, core.BuildLabels{targetLabel}, false, true, true, -1)

	locs := []lsp.Location{}

	// For each reverse dependency, find the exact location of the reference in its BUILD file
	for target := range revdeps {
		pkg := h.state.Graph.PackageByLabel(target.Label)
		if pkg == nil {
			continue
		}

		uri := lsp.DocumentURI("file://" + filepath.Join(h.root, pkg.Filename))
		refDoc, err := h.maybeOpenDoc(uri)
		if err != nil {
			continue
		}
		refAst := h.parseIfNeeded(refDoc)
		refFile := refDoc.AspFile()

		// Find all string literals that reference our target
		labelStr := targetLabel.String()
		shortLabelStr := ":" + targetLabel.Name // For same-package references

		asp.WalkAST(refAst, func(expr *asp.Expression) bool {
			if expr.Val.String != "" {
				str := astutils.TrimStrLit(expr.Val.String)
				// Check if this string matches our target label
				if str == labelStr || (refDoc.PkgName == targetLabel.PackageName && str == shortLabelStr) {
					// Also try parsing it as a label to handle relative references
					if l, err := core.TryParseBuildLabel(str, refDoc.PkgName, ""); err == nil && l == targetLabel {
						start := refFile.Pos(expr.Pos)
						end := refFile.Pos(expr.EndPos)
						locs = append(locs, lsp.Location{
							URI: uri,
							Range: lsp.Range{
								Start: lsp.Position{Line: start.Line - 1, Character: start.Column - 1},
								End:   lsp.Position{Line: end.Line - 1, Character: end.Column - 1},
							},
						})
					}
				}
			}
			return true
		})
	}

	// Optionally include the definition itself if requested
	if includeDeclaration && targetName != "" {
		if defLoc := h.findLabel(doc.PkgName, targetLabel.String()); defLoc.URI != "" {
			locs = append(locs, defLoc)
		}
	}

	return locs, nil
}
