package asp

import "strings"

// FindTarget returns the top-level call in a BUILD file that corresponds to a target
// of the given name (or nil if one does not exist).
func FindTarget(statements []*Statement, name string) *Statement {
	for _, statement := range statements {
		if ident := statement.Ident; ident != nil && ident.Action != nil && ident.Action.Call != nil {
			for _, arg := range ident.Action.Call.Arguments {
				if arg.Expr.Val != nil && arg.Expr.Val.Ident != nil && arg.Expr.Val.Ident.Name == "name" {
					if v := arg.Value; v != nil && v.Val.String != "" && strings.Trim(v.Val.String, `"`) == name {
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
