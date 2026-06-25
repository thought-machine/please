package export

import (
	"slices"
	"sort"

	"github.com/please-build/buildtools/build"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/parse/asp"
)

var passExpression = []byte("pass  # Trimmed during export")

// trimmer implements the filtering logic for statements in package files.
type trimmer struct {
	// origin are the bytes from the original package file.
	origin []byte
	// pkg references the package being trimmed.
	pkg *core.Package
	// bytes contain the content to be written after the trimming process.
	bytes []byte
	// exporter is used to lookup target related data from the export process, e.g. which targets are
	// required.
	exporter *trimmedExporter
}

// statementConsumer defines the type for methods used to visit each statement during a
// file walk. The method accepts the currently interpreted statement.
type statementConsumer func(*asp.Statement)

// walkFile will walk the file from the specified start to end, consuming the statements and optionally writing non-statement bytes found along the way (e.g. comments and blank space).
func (t *trimmer) walkFile(stmts []*asp.Statement, start, end asp.Position, consumer statementConsumer) {
	// cursor tracks the position in a block that's being interpreted.
	cursor := start
	for _, stmt := range stmts {
		// Write content that's between stmts (e.g. comments). We skip these while parsing so it won't
		// be included in "parsedStmts" but we want the resulting BUILD file to include these.
		if cursor < stmt.Pos {
			t.copy(cursor, stmt.Pos)
		}

		// Consume stmt with specified method
		consumer(stmt)

		// Move the cursor to the end of the processed statement. The cursor will enable writing of lines
		// that are not considered statements by the parser (e.g. comments, new lines).
		cursor = stmt.EndPos
	}

	// Write the rest of the original file (non build targets)
	t.copy(cursor, end)
}

// trimBlock visits all the statements in a block and trims undesired statements.
func (t *trimmer) trimBlock(stmts []*asp.Statement, blockStart, blockEnd asp.Position) bool {
	var written bool
	t.walkFile(stmts, blockStart, blockEnd, func(stmt *asp.Statement) {
		if stmt.If != nil {
			if t.trimIf(stmt) {
				written = true
			}
		} else if stmt.For != nil {
			if t.trimFor(stmt) {
				written = true
			}
		} else if stmt.Ident != nil && stmt.Ident.Name == "subinclude" {
			t.trimSubinclude(stmt)
			written = true
		} else if relatives := t.relatedTargets(stmt); len(relatives) > 0 {
			// Meaning it is a build statement that creates build targets.
			if t.anyExported(relatives) {
				t.copy(stmt.Pos, stmt.EndPos)
				written = true
			}
		} else {
			// Write every other statement.
			// If the statement didn't generate any targets (e.g. variable assignments, package() calls),
			// we keep it to ensure the BUILD file remains valid.
			t.copy(stmt.Pos, stmt.EndPos)
			written = true
		}
	})
	return written
}

// trimIf will trim an if-else statement by exporting only the required targets, but keeping the
// if-else primitive -- a implementation decision to help understand the changes caused by an export
// when using a source-control management (SCM) system.
func (t *trimmer) trimIf(stmt *asp.Statement) bool {
	type clause struct {
		hStart, hEnd asp.Position
		stmts        []*asp.Statement
	}

	clauses := []clause{
		{hStart: stmt.If.HeaderPos, hEnd: stmt.If.HeaderEndPos, stmts: stmt.If.Statements},
	}
	for _, elif := range stmt.If.Elif {
		clauses = append(clauses,
			clause{hStart: elif.HeaderPos, hEnd: elif.HeaderEndPos, stmts: elif.Statements})
	}
	if len(stmt.If.ElseStatements) > 0 {
		clauses = append(clauses,
			clause{hStart: stmt.If.ElseHeaderPos, hEnd: stmt.If.ElseHeaderEndPos, stmts: stmt.If.ElseStatements})
	}

	// In an if-else statement only the interpreted/evaluated block will generate targets, meaning
	// that normally only one of the clauses is interpreted, however an if stmt could be inside of
	// a loop where the clause condition depends on the iteration meaning more than one clause
	// could end up being interpreted. Because of that we lookup all the required clauses before
	// writing the statement.
	var requiredClauses = make([]bool, len(clauses))
	for i, c := range clauses {
		required := t.isRequiredStatements(c.stmts)
		requiredClauses[i] = required
	}
	// No clause is required, skip the if-else stmt entirely
	if !slices.Contains(requiredClauses, true) {
		return false
	}

	for i, c := range clauses {
		// Write clause header
		t.copy(c.hStart, c.hEnd)

		// Visit statements in block
		end := stmt.EndPos
		if i+1 < len(clauses) {
			end = clauses[i+1].hStart
		}
		if requiredClauses[i] {
			t.trimBlock(c.stmts, c.hEnd, end)
		} else {
			t.passBlock(c.stmts, c.hEnd, end)
		}
	}
	return true
}

func (t *trimmer) trimFor(stmt *asp.Statement) bool {
	if len(stmt.For.Statements) == 0 {
		return false
	}

	hStart, hEnd := stmt.Pos, stmt.For.Statements[0].Pos
	t.copy(hStart, hEnd)

	written := t.trimBlock(stmt.For.Statements, hEnd, stmt.EndPos)
	if !written {
		t.write(passExpression)
	}
	return true
}

func (t *trimmer) trimSubinclude(stmt *asp.Statement) {
	bStmt := asp.NewBuildStatement(stmt)
	stmtLabels := t.pkg.Metadata.GetSubincludedLabels(bStmt)
	subStmt := t.minimalSubincludeStatement(stmtLabels)
	t.write([]byte(subStmt))
}

// passBlock skips all ASP statements (keeping comments and blank space) and writes a single "pass"
// primitive.
func (t *trimmer) passBlock(stmts []*asp.Statement, blockStart, blockEnd asp.Position) {
	var passWritten bool
	t.walkFile(stmts, blockStart, blockEnd, func(s *asp.Statement) {
		// When the trim results in an empty block, i.e. no statement are written, we write
		// the "pass" primitive. This is useful when parsing inner blocks (e.g. if-else stmts).
		if !passWritten {
			passWritten = true
			t.write(passExpression)
		}
	})
}

func (t *trimmer) isRequiredStatements(stmts []*asp.Statement) bool {
	return slices.ContainsFunc(stmts, t.isRequiredStatement)
}

func (t *trimmer) isRequiredStatement(stmt *asp.Statement) bool {
	if stmt.If != nil {
		// If
		if t.isRequiredStatements(stmt.If.Statements) {
			return true
		}
		// Elif
		if len(stmt.If.Elif) > 0 {
			for _, elif := range stmt.If.Elif {
				if t.isRequiredStatements(elif.Statements) {
					return true
				}
			}
		}
		// Else
		if len(stmt.If.ElseStatements) > 0 && t.isRequiredStatements(stmt.If.ElseStatements) {
			return true
		}
	} else if stmt.For != nil {
		return t.isRequiredStatements(stmt.For.Statements)
	}
	return t.anyExported(t.relatedTargets(stmt))
}

func (t *trimmer) relatedTargets(stmt *asp.Statement) []*core.BuildTarget {
	bStmt := asp.NewBuildStatement(stmt)
	return t.pkg.Metadata.FindTargets(bStmt)
}

func (t *trimmer) anyExported(targets []*core.BuildTarget) bool {
	required := slices.ContainsFunc(targets, func(target *core.BuildTarget) bool {
		return t.exporter.exportedTargets[target.Label]
	})
	return required
}

// minimalSubincludeStatement generates a subinclude statement containing only the required labels.
func (t *trimmer) minimalSubincludeStatement(available core.BuildLabels) string {
	var filteredLabels core.BuildLabels
	for _, required := range t.exporter.requiredSubincludes[t.pkg.Label()] {
		if slices.Contains(available, required) {
			filteredLabels = append(filteredLabels, required)
		}
	}

	if len(filteredLabels) == 0 {
		return ""
	}

	sort.Sort(filteredLabels)

	call := &build.CallExpr{
		X: &build.Ident{Name: "subinclude"},
	}
	for _, label := range filteredLabels {
		call.List = append(call.List, &build.StringExpr{Value: label.ShortString(t.pkg.Label())})
	}

	return build.FormatString(call)
}

func (t *trimmer) copy(start, end asp.Position) {
	if start < 0 || start > end || int(end) > len(t.origin) {
		return
	}
	t.bytes = append(t.bytes, t.origin[start:end]...)
}

func (t *trimmer) write(bytes []byte) {
	t.bytes = append(t.bytes, bytes...)
}
