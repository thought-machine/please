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
	callback := func(astStruct interface{}) interface{} {
		if expr, ok := astStruct.(Expression); ok {
			if withInRange(expr.Pos, expr.EndPos, position) {
				return expr
			}
		} else if stmt, ok := astStruct.(Statement); ok {
			if withInRange(stmt.Pos, stmt.EndPos, position) {
				// get function call, assignment, and property access
				if stmt.Ident != nil {
					return stmt
				}
			}
		}

		return nil
	}

	item := WalkAST(stmts, callback)
	if item != nil {
		if expr, ok := item.(Expression); ok {
			return nil, &expr
		} else if stmt, ok := item.(Statement); ok {
			return &stmt, nil
		}
	}

	return nil, nil
}

// WalkAST is a generic function that walks through the ast recursively,
// astStruct can be anything inside of the AST, such as asp.Statement, asp.Expression
// it accepts a callback for any operations
func WalkAST(astStruct interface{}, callback func(astStruct interface{}) interface{}) interface{} {
	if astStruct == nil {
		return nil
	}

	item := callback(astStruct)
	if item != nil {
		return item
	}

	v, ok := astStruct.(reflect.Value)
	if !ok {
		v = reflect.ValueOf(astStruct)
	}

	if v.Kind() == reflect.Ptr && !v.IsNil() {
		return WalkAST(v.Elem().Interface(), callback)
	} else if v.Kind() == reflect.Slice {
		for i := 0; i < v.Len(); i++ {
			item = WalkAST(v.Index(i).Interface(), callback)
			if item != nil {
				return item
			}
		}
	} else if v.Kind() == reflect.Struct {
		for i := 0; i < v.NumField(); i++ {
			item = WalkAST(v.Field(i).Interface(), callback)
			if item != nil {
				return item
			}
		}
	}
	return nil

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

	if pos.Line == exprPos.Line {
		return pos.Column >= exprPos.Column
	}

	if pos.Line == exprEndPos.Line {
		return pos.Column <= exprEndPos.Column
	}

	return true
}
