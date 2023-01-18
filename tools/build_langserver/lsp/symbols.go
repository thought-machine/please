package lsp

import (
	"sort"
	"strconv"
	"strings"

	"github.com/sourcegraph/go-lsp"

	"github.com/thought-machine/please/src/parse/asp"
)

// symbols implements the basic form of textDocument/symbols, i.e. it returns a list of
// symbols rather than the hierarchical version using DocumentSymbol.
// We could potentially do that but it's a lot more work, and is unclear how much
// value there is until we know more about how the client will use this information -
// also oddly there seems to be little support for describing statements.
func (h *Handler) symbols(params *lsp.DocumentSymbolParams) ([]*lsp.SymbolInformation, error) {
	doc := h.doc(params.TextDocument.URI)
	f := doc.AspFile()
	stmts := h.parseIfNeeded(doc)
	syms := []*lsp.SymbolInformation{}
	addSym := func(name string, kind lsp.SymbolKind, pos, endPos asp.Position) {
		if kind != 0 {
			sym := &lsp.SymbolInformation{Name: name, Kind: kind}
			sym.Location = lsp.Location{
				URI:   params.TextDocument.URI,
				Range: rng(f.Pos(pos), f.Pos(endPos)),
			}
			syms = append(syms, sym)
		}
	}
	asp.WalkAST(stmts, func(stmt *asp.Statement) bool {
		name, kind := stmtToSymbol(stmt)
		addSym(name, kind, stmt.Pos, stmt.EndPos)
		return true
	})
	asp.WalkAST(stmts, func(expr *asp.Expression) bool {
		name, kind := exprToSymbol(expr)
		addSym(name, kind, expr.Pos, expr.EndPos)
		return true
	})
	asp.WalkAST(stmts, func(arg *asp.CallArgument) bool {
		if arg.Name != "" {
			addSym(arg.Name, lsp.SKKey, arg.Pos, arg.Pos + asp.Position(len(arg.Name)))
		}
		return true
	})
	sort.Slice(syms, func(i, j int) bool { return compareRanges(syms[i].Location.Range, syms[j].Location.Range) })
	return syms, nil
}

func exprToSymbol(expr *asp.Expression) (string, lsp.SymbolKind) {
	if v := expr.Val; v == nil {
		return "", 0
	} else if v.String != "" {
		return stringLiteral(v.String), lsp.SKString
	} else if v.FString != nil {
		return reconstructFString(v.FString), lsp.SKString
	} else if v.IsInt {
		return strconv.Itoa(v.Int), lsp.SKNumber
	} else if v.True {
		return "True", lsp.SKBoolean
	} else if v.False {
		return "False", lsp.SKBoolean
	} else if v.None {
		return "None", lsp.SKConstant
	} else if v.List != nil || v.Tuple != nil {
		return "list", lsp.SKArray
	} else if v.Dict != nil {
		return "dict", lsp.SKObject
	} else if v.Ident != nil {
		if len(v.Ident.Action) > 0 && v.Ident.Action[0].Call != nil {
			return v.Ident.Name, lsp.SKFunction
		}
		return v.Ident.Name, lsp.SKVariable
	}
	return "", 0
}

func reconstructFString(f *asp.FString) string {
	var b strings.Builder
	for _, v := range f.Vars {
		b.WriteString(v.Prefix)
		b.WriteByte('{')
		b.WriteString(strings.Join(v.Var, "."))
		b.WriteByte('}')
	}
	b.WriteString(f.Suffix)
	return b.String()
}

func stmtToSymbol(stmt *asp.Statement) (string, lsp.SymbolKind) {
	if stmt.Ident != nil {
		if stmt.Ident.Action != nil && stmt.Ident.Action.Call != nil {
			return stmt.Ident.Name, lsp.SKFunction
		}
		return stmt.Ident.Name, lsp.SKVariable
	} else if stmt.FuncDef != nil {
		return stmt.FuncDef.Name, lsp.SKFunction
	}
	return "", 0
}

// pos converts an asp Position into an LSP one.
// N.B. asp positions are 1-indexed whereas LSP ones are zero-indexed.
func pos(pos asp.FilePosition) lsp.Position {
	return lsp.Position{Line: pos.Line - 1, Character: pos.Column - 1}
}

// aspPos converts an LSP position into an asp one.
// Note 1 vs 0-indexing again.
func aspPos(pos lsp.Position) asp.FilePosition {
	return asp.FilePosition{Line: pos.Line + 1, Column: pos.Character + 1}
}

// rng converts a pair of asp positions into an LSP range.
func rng(start, end asp.FilePosition) lsp.Range {
	return lsp.Range{Start: pos(start), End: pos(end)}
}

// compareRanges compares two lsp.Ranges and returns true if the first starts before
// the second in a document. If equal the end positions are considered.
func compareRanges(a, b lsp.Range) bool {
	if comparePositions(a.Start, b.Start) {
		return true
	} else if comparePositions(b.Start, a.Start) {
		return false
	}
	return comparePositions(a.End, b.End)
}

// comparePositions compares two lsp.Positions and returns true if the first
// is before the second in a document.
func comparePositions(a, b lsp.Position) bool {
	return a.Line < b.Line || (a.Line == b.Line && a.Character < b.Character)
}
