package asp

import (
	"reflect"
	"strings"
)

// FindTarget returns the statement in a BUILD file that corresponds to a target
// of the given name (or nil if one does not exist).
func FindTarget(statements []*Statement, name string) (target *Statement) {
	WalkAST(statements, func(stmt *Statement) bool {
		if arg := FindArgument(stmt, "name"); arg != nil && arg.Value.Val != nil && arg.Value.Val.String != "" && strings.Trim(arg.Value.Val.String, `"`) == name {
			target = stmt
		}
		return false // FindArgument is recursive so we never need to visit more deeply.
	})
	return
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
func GetExtents(file *File, statements []*Statement, statement *Statement, max int) (int, int) {
	next := NextStatement(statements, statement)
	if next == nil {
		// Assume it reaches to the end of the file
		return file.Pos(statement.Pos).Line, max
	}
	return file.Pos(statement.Pos).Line, file.Pos(next.Pos).Line - 1
}

// FindArgument finds an argument of any one of the given names, or nil if there isn't one.
// The statement must be a function call (e.g. as returned by FindTarget).
func FindArgument(statement *Statement, args ...string) (argument *CallArgument) {
	WalkAST([]*Statement{statement}, func(arg *CallArgument) bool {
		for _, a := range args {
			if arg.Name == a {
				argument = arg
				break
			}
		}
		return false // CallArguments can't contain other arguments so no point recursing further.
	})
	return
}

// WalkAST is a generic function that walks through the ast recursively,
// It accepts a function to look for a particular grammar object; it will be called on
// each instance of that type, and returns a bool - for example
// WalkAST(ast, func(expr *Expression) bool { ... })
// If the callback returns true, the node will be further visited; if false it (and
// all children) will be skipped.
func WalkAST[T any](ast []*Statement, callback func(*T) bool) {
	var t T
	for _, node := range ast {
		walkAST(reflect.ValueOf(node), reflect.TypeOf(t), callback)
	}
}

func walkAST[T any](v reflect.Value, t reflect.Type, callback func(*T) bool) {
	if v.Kind() == reflect.Ptr && !v.IsNil() {
		walkAST(v.Elem(), t, callback)
	} else if v.Kind() == reflect.Slice {
		for i := 0; i < v.Len(); i++ {
			walkAST(v.Index(i), t, callback)
		}
	} else if v.Kind() == reflect.Struct {
		if v.Type() != t || callback(v.Addr().Interface().(*T)) {
			for i := 0; i < v.NumField(); i++ {
				walkAST(v.Field(i), t, callback)
			}
		}
	}
}

// WithinRange returns true if the input position is within the range of the given positions.
func WithinRange(needle, start, end FilePosition) bool {
	if needle.Line < start.Line || needle.Line > end.Line {
		return false
	} else if needle.Line == start.Line && needle.Column < start.Column {
		return false
	} else if needle.Line == end.Line && needle.Column > end.Column {
		return false
	}
	return true
}
