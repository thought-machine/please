package langserver

import (
	"context"
	"fmt"
	"github.com/thought-machine/please/src/core"

	"github.com/thought-machine/please/src/parse/asp"
	"github.com/thought-machine/please/tools/build_langserver/lsp"

	"github.com/Workiva/go-datastructures/queue"
	"github.com/sourcegraph/jsonrpc2"
)

func (h *LsHandler) publishDiagnostics(conn *jsonrpc2.Conn) error {
	ctx := context.Background()

	// Get task from the queue
	t, err := h.diagPublisher.queue.Get(1)
	if err != nil {
		log.Warning("fail to get diagnostic publishing task")
		return nil
	}
	if len(t) <= 0 {
		return nil
	}

	task := t[0].(taskDef)

	// exit if the uri is not in the list of documents
	if _, ok := h.workspace.documents[task.uri]; !ok {
		return nil
	}

	params := &lsp.PublishDiagnosticsParams{
		URI:         task.uri,
		Diagnostics: h.diagPublisher.diagnose(ctx, h.analyzer, task.content, task.uri),
	}

	log.Info("Diagnostics detected: %s", params.Diagnostics)

	return conn.Notify(ctx, "textDocument/publishDiagnostics", params)
}

type diagnosticsPublisher struct {
	queue *queue.PriorityQueue
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
		queue: queue.NewPriorityQueue(10000, true),
	}

	return publisher
}

func (dp *diagnosticsPublisher) diagnose(ctx context.Context, analyzer *Analyzer, content string, uri lsp.DocumentURI) []*lsp.Diagnostic {
	stmts := analyzer.AspStatementFromContent(content)

	diag := &diagnosticStore{
		uri:         uri,
		subincludes: analyzer.GetSubinclude(ctx, stmts, uri),
	}

	diag.storeDiagnostics(analyzer, stmts)

	return diag.stored
}

func (ds *diagnosticStore) storeDiagnostics(analyzer *Analyzer, stmts []*asp.Statement) {
	defer func() {
		if r := recover(); r != nil {
			log.Fatalf("Error storing diagnostics: %s", r)
		}
	}()
	ds.stored = []*lsp.Diagnostic{}

	var callback func(astStruct interface{}) interface{}

	callback = func(astStruct interface{}) interface{} {
		if stmt, ok := astStruct.(asp.Statement); ok {
			if stmt.Ident != nil {
				ds.diagnoseIdentStmt(analyzer, stmt.Ident, stmt.Pos, stmt.EndPos)
			}
		} else if expr, ok := astStruct.(asp.Expression); ok {
			ds.diagnoseExpression(analyzer, expr)
		} else if identExpr, ok := astStruct.(asp.IdentExpr); ok {
			ds.diagnoseIdentExpr(analyzer, identExpr, stmts)
		}

		return nil
	}
	asp.WalkAST(stmts, callback)
}

func (ds *diagnosticStore) diagnoseIdentStmt(analyzer *Analyzer, ident *asp.IdentStatement,
	pos asp.Position, endpos asp.Position) {

	if ident.Action != nil && ident.Action.Call != nil {

		funcRange := lsp.Range{
			Start: aspPositionToLsp(pos),
			End:   aspPositionToLsp(endpos),
		}
		ds.diagnoseFuncCall(analyzer, ident.Name,
			ident.Action.Call.Arguments, funcRange)
	}
}

func (ds *diagnosticStore) diagnoseExpression(analyzer *Analyzer, expr asp.Expression) {
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
}

func (ds *diagnosticStore) diagnoseIdentExpr(analyzer *Analyzer, identExpr asp.IdentExpr, stmts []*asp.Statement) {

	if identExpr.Action == nil {
		// Check if variable has been defined
		pos := aspPositionToLsp(identExpr.Pos)
		variables := analyzer.VariablesFromStatements(stmts, &pos)

		if _, ok := variables[identExpr.Name]; !ok && !StringInSlice(analyzer.GetConfigNames(), identExpr.Name) {
			diag := &lsp.Diagnostic{
				Range:    getNameRange(aspPositionToLsp(identExpr.Pos), identExpr.Name),
				Severity: lsp.Error,
				Source:   "build",
				Message:  fmt.Sprintf("unexpected variable or config property '%s'", identExpr.Name),
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
			ds.diagnoseFuncCall(analyzer, identExpr.Name, action.Call.Arguments, identRange)
		}
	}
}

// diagnoseFuncCall checks if the function call's argument name and type are correct
// Store a *lsp.diagnostic if found
func (ds *diagnosticStore) diagnoseFuncCall(analyzer *Analyzer, funcName string,
	callArgs []asp.CallArgument, funcRange lsp.Range) {

	// Check if the funcDef is defined
	def := analyzer.GetBuildRuleByName(funcName, ds.subincludes)
	if def == nil {
		diagRange := getNameRange(funcRange.Start, funcName)
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
		if def.Object != "" && arg.Name == "self" && i == 0 {
			continue
		}

		// Diagnostics for the cases when there are not enough argument passed to the function
		if len(callArgs)-1 < i {
			if def.ArgMap[arg.Name].Required == true {
				diag := &lsp.Diagnostic{
					Range:    getCallRange(funcRange, funcName),
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
		argDef := arg

		argRange := lsp.Range{
			Start: aspPositionToLsp(callArg.Value.Pos),
			End:   aspPositionToLsp(callArg.Value.EndPos),
		}
		// Check if the argument is a valid keyword arg
		// **Ignore the builtins listed in excludedBuiltins, as the args are not definite
		if callArg.Name != "" && !StringInSlice(BuiltInsWithIrregularArgs, def.Name) {
			if _, present := def.ArgMap[callArg.Name]; !present {
				argRange = getNameRange(aspPositionToLsp(callArg.Pos), callArg.Name)
				diag := &lsp.Diagnostic{
					Range:    argRange,
					Severity: lsp.Error,
					Source:   "build",
					Message:  fmt.Sprintf("unexpected argument %s", callArg.Name),
				}
				ds.stored = append(ds.stored, diag)
				continue
			}

			// ensure we are checking the correct argument definition
			// As keyword args can be in any order
			if callArg.Name != arg.Name {
				argDef = *def.ArgMap[callArg.Name].Argument
			}

		}
		// Check if the argument value type is correct
		if diag := ds.diagnosticFromCallArgType(analyzer, argDef, callArg); diag != nil {
			ds.stored = append(ds.stored, diag)
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
func getCallRange(funcRange lsp.Range, funcName string) lsp.Range {
	return lsp.Range{
		Start: lsp.Position{Line: funcRange.Start.Line,
			Character: funcRange.Start.Character + len(funcName)},
		End: funcRange.End,
	}

}

func getNameRange(pos lsp.Position, name string) lsp.Range {
	return lsp.Range{
		Start: pos,
		End: lsp.Position{Line: pos.Line,
			Character: pos.Character + len(name)},
	}
}
