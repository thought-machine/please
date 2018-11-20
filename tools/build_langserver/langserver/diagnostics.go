package langserver

import (
	"context"
	"core"
	"fmt"
	"parse/asp"
	"tools/build_langserver/lsp"

	"github.com/sourcegraph/jsonrpc2"
)

// TODO(bnm): use conn.Notify
// TODO(bnm): below function can be moved to text_document.go
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
	uri      lsp.DocumentURI
	analyzer *Analyzer
	// Subincludes of the file if any
	subincludes map[string]*RuleDef

	diagnostics map[lsp.Range]*lsp.Diagnostic
}

func newDiagnostics(ctx context.Context, analyzer *Analyzer, content string, uri lsp.DocumentURI) *diagnosticsStore {
	stmts := analyzer.AspStatementFromContent(content)

	store := &diagnosticsStore{
		uri:         uri,
		diagnostics: make(map[lsp.Range]*lsp.Diagnostic),
		analyzer:    analyzer,
		subincludes: analyzer.GetSubinclude(ctx, stmts, uri),
	}

	store.diagnose(stmts)

	return store
}

func (ds *diagnosticsStore) diagnose(stmts []*asp.Statement) {

	var callback func(astStruct interface{}) interface{}

	callback = func(astStruct interface{}) interface{} {
		if stmt, ok := astStruct.(asp.Statement); ok {
			if stmt.Ident != nil {
				asp.WalkAST(stmt.Ident, callback)

				if stmt.Ident.Action != nil && stmt.Ident.Action.Call != nil {
					callRange := getStmtCallRange(stmt)
					ds.storeFuncCallDiagnostics(stmt.Ident.Name, stmt.Ident.Action.Call.Arguments, *callRange)
				}
			}
		} else if expr, ok := astStruct.(asp.Expression); ok {
			exprRange := lsp.Range{
				Start: aspPositionToLsp(expr.Pos),
				End:   aspPositionToLsp(expr.EndPos),
			}

			if expr.Val == nil {
				ds.diagnostics[exprRange] = &lsp.Diagnostic{
					Range:    exprRange,
					Severity: lsp.Error,
					Source:   "build",
					Message:  "expression expected",
				}
			} else if expr.Val.String != "" && core.LooksLikeABuildLabel(TrimQuotes(expr.Val.String)) {
				if diag := ds.diagnosticFromBuildLabel(expr.Val.String, exprRange); diag != nil {
					ds.diagnostics[exprRange] = diag
				}
			}
		} else if identExpr, ok := astStruct.(asp.IdentExpr); ok {
			identRange := lsp.Range{
				Start: aspPositionToLsp(identExpr.Pos),
				End:   aspPositionToLsp(identExpr.EndPos),
			}
			if identExpr.Action == nil {
				// Check if variable has been defined
				pos := aspPositionToLsp(identExpr.Pos)
				variables := ds.analyzer.VariablesFromStatements(stmts, &pos)

				if _, ok := variables[identExpr.Name]; !ok {
					ds.diagnostics[identRange] = &lsp.Diagnostic{
						Range:    identRange,
						Severity: lsp.Error,
						Source:   "build",
						Message:  fmt.Sprintf("unexpected variable '%s'", identExpr.Name),
					}
				}
			}
			for _, action := range identExpr.Action {
				if action.Call != nil {
					ds.storeFuncCallDiagnostics(identExpr.Name, action.Call.Arguments, identRange)
				} else if action.Property != nil {
					asp.WalkAST(action.Property, callback)
				}
			}
		}

		return nil
	}

	asp.WalkAST(stmts, callback)
}

// storeFuncCallDiagnostics checks if the function call's argument name and type are correct
// Store a *lsp.diagnostic if found
func (ds *diagnosticsStore) storeFuncCallDiagnostics(funcName string, callArgs []asp.CallArgument, callRange lsp.Range) {
	excludedBuiltins := []string{"format", "zip", "package", "join_path"}

	// Check if the funcDef is defined
	def := ds.analyzer.GetBuildRuleByName(funcName, ds.subincludes)
	if def == nil {
		diagRange := getNameRange(callRange.Start, funcName)
		ds.diagnostics[diagRange] = &lsp.Diagnostic{
			Range:    diagRange,
			Severity: lsp.Error,
			Source:   "build",
			Message:  fmt.Sprintf("function undefined: %s", funcName),
		}
		return
	}

	for i, arg := range def.Arguments {
		// Diagnostics for the cases when there are not enough argument passed to the function
		if len(callArgs)-1 < i {
			if def.ArgMap[arg.Name].Required == true {
				ds.diagnostics[callRange] = &lsp.Diagnostic{
					Range:    callRange,
					Severity: lsp.Error,
					Source:   "build",
					Message:  fmt.Sprintf("not enough arguments in call to %s", def.Name),
				}
				break
			}
			continue
		}

		callArg := callArgs[i]

		argRange := lsp.Range{
			Start: aspPositionToLsp(callArg.Value.Pos),
			End:   aspPositionToLsp(callArg.Value.EndPos),
		}

		// Check if the argument value type is correct
		var msg string
		if callArg.Value.Val == nil {
			msg = "expression expected"
		} else {
			var varType string

			if GetValType(callArg.Value.Val) != "" {
				varType = GetValType(callArg.Value.Val)
			} else if callArg.Value.Val.Ident != nil {
				ident := callArg.Value.Val.Ident

				if ident.Action == nil {
					vars, err := ds.analyzer.VariablesFromURI(ds.uri, &argRange.Start)
					if err != nil {
						log.Warning("fail to get variables from %s, skipping", ds.uri)
					}
					// We don't have to worry about the case when the variable does not exist,
					// as it has been taken care of in diagnose
					if variable, ok := vars[ident.Name]; ok {
						varType = variable.Type
					}
				} else {
					// Check return types of call if variable is being assigned to a call
					if retType := ds.getIdentExprReturnType(ident); retType != "" {
						varType = retType
					}
				}
			}

			if varType != "" && len(arg.Type) != 0 && !StringInSlice(arg.Type, varType) {
				msg = fmt.Sprintf("invalid type for argument type '%s' for %s, expecting one of %s",
					varType, arg.Name, arg.Type)
			}
		}

		// Check if the argument is a valid keyword arg
		// **Ignore the builtins listed in excludedBuiltins, as the args are not definite
		if callArg.Name != "" && !StringInSlice(excludedBuiltins, def.Name) {
			if _, present := def.ArgMap[callArg.Name]; !present {
				argRange = getNameRange(aspPositionToLsp(callArg.Pos), callArg.Name)
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

func (ds *diagnosticsStore) diagnosticFromBuildLabel(labelStr string, valRange lsp.Range) *lsp.Diagnostic {
	trimmed := TrimQuotes(labelStr)

	ctx := context.Background()
	label, err := ds.analyzer.BuildLabelFromString(ctx, ds.uri, trimmed)
	if err != nil {
		return &lsp.Diagnostic{
			Range:    valRange,
			Severity: lsp.Error,
			Source:   "build",
			Message:  fmt.Sprintf("Invalid build label %s", trimmed),
		}
	}
	currentPkg, err := PackageLabelFromURI(ds.uri)

	if label.BuildDef != nil && !isVisible(label.BuildDef, currentPkg) {
		return &lsp.Diagnostic{
			Range:    valRange,
			Severity: lsp.Error,
			Source:   "build",
			Message:  fmt.Sprintf("build label %s is not visible to current package", trimmed),
		}
	}
	return nil
}

func (ds *diagnosticsStore) getIdentExprReturnType(ident *asp.IdentExpr) string {
	if ident.Action == nil {
		return ""
	}

	for _, action := range ident.Action {
		if action.Call != nil {
			if def := ds.analyzer.GetBuildRuleByName(ident.Name, ds.subincludes); def != nil {
				return def.Return
			}
		} else if action.Property != nil {
			return ds.getIdentExprReturnType(action.Property)
		}
	}

	return ""
}

/************************
 * Helper functions
 ************************/
// TODO(bnm): will need one for IdentExpr as well
func getStmtCallRange(stmt asp.Statement) *lsp.Range {
	if stmt.Ident == nil {
		return nil
	}

	return &lsp.Range{
		Start: lsp.Position{Line: stmt.Pos.Line - 1,
			Character: stmt.Pos.Column + len(stmt.Ident.Name) - 1},
		End: aspPositionToLsp(stmt.EndPos),
	}

	return nil
}

func getNameRange(pos lsp.Position, name string) lsp.Range {
	return lsp.Range{
		Start: pos,
		End: lsp.Position{Line: pos.Line,
			Character: pos.Character + len(name)},
	}
}
