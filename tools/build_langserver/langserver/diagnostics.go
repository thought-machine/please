package langserver

import (
	"context"
	"fmt"
	"parse/asp"
	"strings"

	"tools/build_langserver/lsp"

	"github.com/sourcegraph/jsonrpc2"
)

// TODO(bnm): use conn.Notify
func (h *LsHandler) publishDiagnostics(ctx context.Context, conn *jsonrpc2.Conn, uri lsp.DocumentURI) error {
	//doc, present := h.workspace.documents[uri]
	//if !present {
	//	log.Warning("document not found at %s", uri)
	//	return nil
	//}
	//
	//diag := doc.diagnostics

	return nil
}

type diagnosticsStore struct {
	uri lsp.DocumentURI

	analyzer *Analyzer

	diagnostics map[lsp.Range]*lsp.Diagnostic
}

func newDiagnostics(analyzer *Analyzer, content string, uri lsp.DocumentURI) *diagnosticsStore {

	// Making a new context as this has nothing to do with the requests we get from the client
	ctx := context.Background()

	store := &diagnosticsStore{
		uri:         uri,
		diagnostics: make(map[lsp.Range]*lsp.Diagnostic),
		analyzer:    analyzer,
	}

	stmts := analyzer.AspStatementFromContent(content)
	subincludes := analyzer.GetSubinclude(ctx, stmts, uri)

	var callback func(astStruct interface{}) interface{}

	callback := func(astStruct interface{}) interface{} {
		if stmt, ok := astStruct.(asp.Statement); ok {
			if stmt.Ident != nil {
				asp.WalkAST(stmt.Ident, callback)

				if stmt.Ident.Action != nil && stmt.Ident.Action.Call != nil {
					ruleDef := analyzer.GetBuildRuleByName(stmt.Ident.Name, subincludes)
					if ruleDef == nil {

						diagRange := getNameRange(stmt.Pos, stmt.Ident.Name)
						store.diagnostics[diagRange] = &lsp.Diagnostic{
							Range:    diagRange,
							Severity: lsp.Error,
							Source:   "build",
							Message:  fmt.Sprintf("Invalid function name, %s", stmt.Ident.Name),
						}
						return nil

					} else {
						callRange := getStmtCallRange(stmt)
					}
				}
			}

		}
		return nil
	}

	asp.WalkAST(stmts, callback)
	//asp.WalkAST(callback, stmts)
	return store
}

func (ds *diagnosticsStore) storeFuncCallDiagnostics(def *RuleDef, callArgs []asp.CallArgument, callRange lsp.Range) {
	for i, arg := range def.Arguments {
		// Diagnostics for the cases when there are not enough argument passed to the function
		if len(callArgs)-1 < i && def.ArgMap[arg.Name].Required == true {
			ds.diagnostics[callRange] = &lsp.Diagnostic{
				Range:    callRange,
				Severity: lsp.Error,
				Source:   "build",
				Message:  fmt.Sprintf("not enough arguments in call to %s", def.Name),
			}
			break
		}

		callArg := callArgs[i]

		argRange := lsp.Range{
			Start: aspPositionToLsp(callArg.Value.Pos),
			End:   aspPositionToLsp(callArg.Value.EndPos),
		}
		var msg string

		if callArg.Value.Val == nil {
			msg = "expression expected"
		} else if !StringInSlice(arg.Type, GetValType(callArg.Value.Val)) {
			msg = fmt.Sprintf("invalid type for argument type for %s. %s",
				callArg.Name, def.ArgMap[arg.Name].Definition)
		} else if callArg.Value.Val.String != "" {
			// TODO(bnm): check for valid build label
		}

		if callArg.Name != "" {
			if _, present := def.ArgMap[callArg.Name]; !present {
				argRange = getNameRange(callArg.Pos, callArg.Name)
				msg = fmt.Sprintf("unexpected argument %s", callArg.Name)
			}

		}

		if msg != "" {
			ds.diagnostics[argRange] = &lsp.Diagnostic{
				Range:    argRange,
				Severity: lsp.Error,
				Source:   "build",
				Message:  msg,
			}
		}
	}
}

func (ds *diagnosticsStore) diagnosticFromBuildLabel(labelStr string) *lsp.Diagnostic {
	trimmed := TrimQuotes(labelStr)

	ctx := context.Background()
	label, err := ds.analyzer.BuildLabelFromString(ctx, ds.uri, labelStr)
	if err != nil {
		return
	}

	return nil
}

/************************
 * Helper functions
 ************************/
// TODO(bnm): will need one for IdentExpr as well
func getStmtCallRange(callStruct interface{}) *lsp.Range {
	if stmt, ok := callStruct.(asp.Statement); ok {
		if stmt.Ident == nil {
			return nil
		}

		return &lsp.Range{
			Start: lsp.Position{Line: stmt.Pos.Line - 1,
				Character: stmt.Pos.Column - len(stmt.Ident.Name) - 1},
			End: aspPositionToLsp(stmt.EndPos),
		}
	}

	return nil
}

func getNameRange(Pos asp.Position, name string) lsp.Range {
	return lsp.Range{
		Start: aspPositionToLsp(Pos),
		End: lsp.Position{Line: Pos.Line - 1,
			Character: Pos.Column - len(name) - 1},
	}
}
