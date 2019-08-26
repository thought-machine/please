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

// WalkAST is a generic function that walks through the ast recursively,
// It accepts a function to look for a particular grammar object; it will be called on
// each instance of that type, and returns a bool - for example
// WalkAST(ast, func(expr *Expression) bool { ... })
// If the callback returns true, the node will be further visited; if false it (and
// all children) will be skipped.
func WalkAST(ast []*Statement, callback interface{}) {
	cb := reflect.ValueOf(callback)
	typ := cb.Type().In(0)
	for _, node := range ast {
		walkAST(reflect.ValueOf(node), typ, cb)
	}
}

func walkAST(v reflect.Value, nodeType reflect.Type, callback reflect.Value) {
	call := func(v reflect.Value) bool {
		if v.Type() == nodeType {
			vs := callback.Call([]reflect.Value{v})
			return vs[0].Bool()
		}
		return true
	}

	if v.Kind() == reflect.Ptr && !v.IsNil() {
		walkAST(v.Elem(), nodeType, callback)
	} else if v.Kind() == reflect.Slice {
		for i := 0; i < v.Len(); i++ {
			walkAST(v.Index(i), nodeType, callback)
		}
	} else if v.Kind() == reflect.Struct {
		if call(v.Addr()) {
			for i := 0; i < v.NumField(); i++ {
				walkAST(v.Field(i), nodeType, callback)
			}
		}
	}
}

// WithinRange returns true if the input position is within the range of the given positions.
func WithinRange(needle, start, end Position) bool {
	if needle.Line < start.Line || needle.Line > end.Line {
		return false
	} else if needle.Line == start.Line && needle.Column < start.Column {
		return false
	} else if needle.Line == end.Line && needle.Column > end.Column {
		return false
	}
	return true
}
