package langserver

import (
	"context"
	"core"
	"fmt"

	"parse/asp"
	"tools/build_langserver/lsp"

	"github.com/Workiva/go-datastructures/queue"
	"github.com/sourcegraph/jsonrpc2"
)

func (h *LsHandler) publishDiagnostics(conn *jsonrpc2.Conn, content string, uri lsp.DocumentURI) error {
	ctx := context.Background()

	if _, ok := h.workspace.documents[uri]; !ok {
		return nil
	}

	h.diagPublisher.diagnose(ctx, h.analyzer, content, uri)

	params := &lsp.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: h.diagPublisher.stored[uri].stored,
	}
	log.Info("Diagnostics detected: %s", params.Diagnostics)

	if err := conn.Notify(ctx, "textDocument/publishDiagnostics", params); err != nil {
		return err
	}
	return nil
}

type diagnosticsPublisher struct {
	queue *queue.PriorityQueue

	stored map[lsp.DocumentURI]*diagnosticStore
}

type diagnosticStore struct {
	uri lsp.DocumentURI
	// Subincludes of the file if any
	subincludes map[string]*RuleDef

	stored []*lsp.Diagnostic
}

type taskDef struct {
	uri     lsp.DocumentURI
	content string
}

func (td taskDef) Compare(other queue.Item) int {
	otherTask := other.(taskDef)
	if otherTask.uri != td.uri || otherTask.content == td.content {
		return 0
	}

	return 1
}

func newDiagnosticsPublisher() *diagnosticsPublisher {
	publisher := &diagnosticsPublisher{
		stored: make(map[lsp.DocumentURI]*diagnosticStore),
		queue:  queue.NewPriorityQueue(10000, true),
	}

	return publisher
}

func (dp *diagnosticsPublisher) diagnose(ctx context.Context, analyzer *Analyzer, content string, uri lsp.DocumentURI) {
	stmts := analyzer.AspStatementFromContent(content)

	if _, ok := dp.stored[uri]; !ok {
		dp.stored[uri] = &diagnosticStore{
			uri:         uri,
			subincludes: analyzer.GetSubinclude(ctx, stmts, uri),
		}
	}

	dp.stored[uri].storeDiagnostics(analyzer, stmts)
}

func (ds *diagnosticStore) storeDiagnostics(analyzer *Analyzer, stmts []*asp.Statement) {
	ds.stored = []*lsp.Diagnostic{}

	var callback func(astStruct interface{}) interface{}
	callback = func(astStruct interface{}) interface{} {
		if stmt, ok := astStruct.(asp.Statement); ok {
			if stmt.Ident != nil {
				if stmt.Ident.Action != nil && stmt.Ident.Action.Call != nil {
					callRange := lsp.Range{
						Start: aspPositionToLsp(stmt.Pos),
						End:   aspPositionToLsp(stmt.EndPos),
					}
					ds.storeFuncCallDiagnostics(analyzer, stmt.Ident.Name,
						stmt.Ident.Action.Call.Arguments, callRange)
				}
			}
		} else if expr, ok := astStruct.(asp.Expression); ok {
			exprRange := lsp.Range{
				Start: aspPositionToLsp(expr.Pos),
				End:   aspPositionToLsp(expr.EndPos),
			}

			if expr.Val == nil {
				diag := &lsp.Diagnostic{
					Range:    exprRange,
					Severity: lsp.Error,
					Source:   "build",
					Message:  "expression expected",
				}
				ds.stored = append(ds.stored, diag)
			} else if expr.Val.String != "" && core.LooksLikeABuildLabel(TrimQuotes(expr.Val.String)) {
				if diag := ds.diagnosticFromBuildLabel(analyzer, expr.Val.String, exprRange); diag != nil {
					ds.stored = append(ds.stored, diag)
				}
			}
		} else if identExpr, ok := astStruct.(asp.IdentExpr); ok {

			if identExpr.Action == nil {
				// Check if variable has been defined
				pos := aspPositionToLsp(identExpr.Pos)
				variables := analyzer.VariablesFromStatements(stmts, &pos)

				if _, ok := variables[identExpr.Name]; !ok {
					nameRange := getNameRange(aspPositionToLsp(identExpr.Pos), identExpr.Name)

					diag := &lsp.Diagnostic{
						Range:    nameRange,
						Severity: lsp.Error,
						Source:   "build",
						Message:  fmt.Sprintf("unexpected variable '%s'", identExpr.Name),
					}
					ds.stored = append(ds.stored, diag)
				}
			}
			for _, action := range identExpr.Action {
				if action.Call != nil {
					identRange := lsp.Range{
						Start: aspPositionToLsp(identExpr.Pos),
						End:   aspPositionToLsp(identExpr.EndPos),
					}
					ds.storeFuncCallDiagnostics(analyzer, identExpr.Name, action.Call.Arguments, identRange)
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
func (ds *diagnosticStore) storeFuncCallDiagnostics(analyzer *Analyzer, funcName string,
	callArgs []asp.CallArgument, callRange lsp.Range) {

	excludedBuiltins := []string{"format", "zip", "package", "join_path"}

	// Check if the funcDef is defined
	def := analyzer.GetBuildRuleByName(funcName, ds.subincludes)
	if def == nil {
		diagRange := getNameRange(callRange.Start, funcName)
		diag := &lsp.Diagnostic{
			Range:    diagRange,
			Severity: lsp.Error,
			Source:   "build",
			Message:  fmt.Sprintf("function undefined: %s", funcName),
		}
		ds.stored = append(ds.stored, diag)
		return
	}

	for i, arg := range def.Arguments {
		// Diagnostics for the cases when there are not enough argument passed to the function
		if len(callArgs)-1 < i {
			if def.ArgMap[arg.Name].Required == true {
				diag := &lsp.Diagnostic{
					Range:    callRange,
					Severity: lsp.Error,
					Source:   "build",
					Message:  fmt.Sprintf("not enough arguments in call to %s", def.Name),
				}
				ds.stored = append(ds.stored, diag)
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
		if diag := ds.diagnosticFromCallArgType(analyzer, arg, callArg); diag != nil {
			ds.stored = append(ds.stored, diag)
		}

		// Check if the argument is a valid keyword arg
		// **Ignore the builtins listed in excludedBuiltins, as the args are not definite
		if callArg.Name != "" && !StringInSlice(excludedBuiltins, def.Name) {
			if _, present := def.ArgMap[callArg.Name]; !present {
				argRange = getNameRange(aspPositionToLsp(callArg.Pos), callArg.Name)
				diag := &lsp.Diagnostic{
					Range:    argRange,
					Severity: lsp.Error,
					Source:   "build",
					Message:  fmt.Sprintf("unexpected argument %s", callArg.Name),
				}
				ds.stored = append(ds.stored, diag)
			}

		}

	}
}

func (ds *diagnosticStore) diagnosticFromCallArgType(analyzer *Analyzer, argDef asp.Argument, callArg asp.CallArgument) *lsp.Diagnostic {
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
				vars, err := analyzer.VariablesFromURI(ds.uri, &argRange.Start)
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
				if retType := ds.getIdentExprReturnType(analyzer, ident); retType != "" {
					varType = retType
				}
			}
		}

		if varType != "" && len(argDef.Type) != 0 && !StringInSlice(argDef.Type, varType) {
			msg = fmt.Sprintf("invalid type for argument type '%s' for %s, expecting one of %s",
				varType, argDef.Name, argDef.Type)
		}
	}

	if msg != "" {
		return &lsp.Diagnostic{
			Range:    argRange,
			Severity: lsp.Error,
			Source:   "build",
			Message:  msg,
		}
	}
	return nil
}

func (ds *diagnosticStore) diagnosticFromBuildLabel(analyzer *Analyzer, labelStr string, valRange lsp.Range) *lsp.Diagnostic {
	trimmed := TrimQuotes(labelStr)

	ctx := context.Background()
	label, err := analyzer.BuildLabelFromString(ctx, ds.uri, trimmed)
	if err != nil {
		return &lsp.Diagnostic{
			Range:    valRange,
			Severity: lsp.Error,
			Source:   "build",
			Message:  fmt.Sprintf("Invalid build label %s. error: %s", trimmed, err),
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

func (ds *diagnosticStore) getIdentExprReturnType(analyzer *Analyzer, ident *asp.IdentExpr) string {
	if ident.Action == nil {
		return ""
	}

	for _, action := range ident.Action {
		if action.Call != nil {
			if def := analyzer.GetBuildRuleByName(ident.Name, ds.subincludes); def != nil {
				return def.Return
			}
		} else if action.Property != nil {
			return ds.getIdentExprReturnType(analyzer, action.Property)
		}
	}

	return ""
}

/************************
 * Helper functions
 ************************/
func getCallRange(pos asp.Position, endpos asp.Position, funcName string) *lsp.Range {
	return &lsp.Range{
		Start: lsp.Position{Line: pos.Line - 1,
			Character: pos.Column + len(funcName) - 1},
		End: aspPositionToLsp(endpos),
	}

}

func getNameRange(pos lsp.Position, name string) lsp.Range {
	return lsp.Range{
		Start: pos,
		End: lsp.Position{Line: pos.Line,
			Character: pos.Character + len(name)},
	}
}
