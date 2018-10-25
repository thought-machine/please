package asp

import (
	"reflect"
	"strings"
)

// FindTarget returns the top-level call in a BUILD file that corresponds to a target
// of the given name (or nil if one does not exist).
func FindTarget(statements []*Statement, name string) *Statement {
	for _, statement := range statements {
		if ident := statement.Ident; ident != nil && ident.Action != nil && ident.Action.Call != nil {
			for _, arg := range ident.Action.Call.Arguments {
				if arg.Name == "name" {
					if arg.Value.Val != nil && arg.Value.Val.String != "" && strings.Trim(arg.Value.Val.String, `"`) == name {
						return statement
					}
				}
			}
		}
	}
	return nil
}

// NextStatement finds the statement that follows the given one.
// This is often useful to find the extent of a statement in source code.
// It will return nil if there is not one following it.
func NextStatement(statements []*Statement, statement *Statement) *Statement {
	for i, s := range statements {
		if s == statement && i < len(statements)-1 {
			return statements[i+1]
		}
	}
	return nil
}

// GetExtents returns the "extents" of a statement, i.e. the lines that it covers in source.
// The caller must pass a value for the maximum extent of the file; we can't detect it here
// because the AST only contains positions for the beginning of the statements.
func GetExtents(statements []*Statement, statement *Statement, max int) (int, int) {
	next := NextStatement(statements, statement)
	if next == nil {
		// Assume it reaches to the end of the file
		return statement.Pos.Line, max
	}
	return statement.Pos.Line, next.Pos.Line - 1
}

// FindArgument finds an argument of any one of the given names, or nil if there isn't one.
// The statement must be a function call (e.g. as returned by FindTarget).
func FindArgument(statement *Statement, args ...string) *CallArgument {
	for i, a := range statement.Ident.Action.Call.Arguments {
		for _, arg := range args {
			if a.Name == arg {
				return &statement.Ident.Action.Call.Arguments[i]
			}
		}
	}
	return nil
}

// StatementOrExpressionFromAst recursively finds asp.IdentStatement and asp.Expression in the ast
// and returns a valid statement pointer if within range
func StatementOrExpressionFromAst(stmts []*Statement, position Position) (statement *Statement, expression *Expression) {
	return getStatementOrExpressionFromAst(reflect.ValueOf(stmts), position)
}

func getStatementOrExpressionFromAst(v reflect.Value, position Position) (statement *Statement, expression *Expression) {
	if v.Type() == reflect.TypeOf(Expression{}) {
		expr := v.Interface().(Expression)
		if withInRange(expr.Pos, expr.EndPos, position) {
			return nil, &expr
		}
	} else if v.Type() == reflect.TypeOf([]*Statement{}) && v.Len() != 0 {
		stmts := v.Interface().([]*Statement)
		for _, stmt := range stmts {
			if withInRange(stmt.Pos, stmt.EndPos, position) {
				// get function call, assignment, and property access
				if stmt.Ident != nil {
					return stmt, nil
				}
				return getStatementOrExpressionFromAst(reflect.ValueOf(stmt), position)
			}
		}
	} else if v.Kind() == reflect.Ptr && !v.IsNil() {
		return getStatementOrExpressionFromAst(v.Elem(), position)
	} else if v.Kind() == reflect.Slice {
		for i := 0; i < v.Len(); i++ {
			stmt, expr := getStatementOrExpressionFromAst(v.Index(i), position)
			if expr != nil || stmt != nil {
				return stmt, expr
			}
		}
	} else if v.Kind() == reflect.Struct {
		for i := 0; i < v.NumField(); i++ {
			stmt, expr := getStatementOrExpressionFromAst(v.Field(i), position)
			if expr != nil || stmt != nil {
				return stmt, expr
			}
		}
	}
	return nil, nil
}

// withInRange checks if the input position is within the range of the Expression
func withInRange(exprPos Position, exprEndPos Position, pos Position) bool {
	withInLineRange := pos.Line >= exprPos.Line &&
		pos.Line <= exprEndPos.Line

	withInColRange := pos.Column >= exprPos.Column &&
		pos.Column <= exprEndPos.Column

	onTheSameLine := pos.Line == exprEndPos.Line &&
		pos.Line == exprPos.Line

	if !withInLineRange || (onTheSameLine && !withInColRange) {
		return false
	}

	return true
}
