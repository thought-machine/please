package langserver

import (
	"context"
	"testing"
	"github.com/thought-machine/please/tools/build_langserver/lsp"

	"github.com/stretchr/testify/assert"
)

func TestDiagnose(t *testing.T) {
	ds := getDefaultDS(t)
	assert.Equal(t, 13, len(ds.stored))
}

func TestDiagnosisInvalidBuildLabel(t *testing.T) {
	ds := getDefaultDS(t)

	// Test for invalid build label
	diag := FindDiagnosticByMsg(ds.stored,
		"Invalid build label //dummy/buildlabels:foo. error: cannot find the path for build label //dummy/buildlabels:foo")
	assert.NotNil(t, diag)
	expected := lsp.Range{
		Start: lsp.Position{Line: 30, Character: 8},
		End:   lsp.Position{Line: 30, Character: 33},
	}
	assert.Equal(t, expected, diag.Range)

	diag = FindDiagnosticByMsg(ds.stored,
		"Invalid build label //sr. error: cannot find the path for build label //sr:sr")
	assert.NotNil(t, diag)
	expected = lsp.Range{
		Start: lsp.Position{Line: 45, Character: 0},
		End:   lsp.Position{Line: 45, Character: 6},
	}
	assert.Equal(t, expected, diag.Range)

	// Test for invisible build label
	expected = lsp.Range{
		Start: lsp.Position{Line: 12, Character: 8},
		End:   lsp.Position{Line: 12, Character: 27},
	}
	diag = FindDiagnosticByRange(ds.stored, expected)
	assert.NotNil(t, diag)
	assert.Equal(t, "build label //src/parse/rules is not visible to current package", diag.Message)

}

func TestDiagnosisFuncArgument(t *testing.T) {
	ds := getDefaultDS(t)

	// Test for unexpected argument
	diag := FindDiagnosticByMsg(ds.stored, "unexpected argument foo")
	assert.NotNil(t, diag)
	expected := lsp.Range{
		Start: lsp.Position{Line: 25, Character: 4},
		End:   lsp.Position{Line: 25, Character: 7},
	}
	assert.Equal(t, expected, diag.Range)

	// Test for invalid argument type
	diag = FindDiagnosticByMsg(ds.stored,
		"invalid type for argument type 'dict' for target, expecting one of [str]")
	assert.NotNil(t, diag)
	expected = lsp.Range{
		Start: lsp.Position{Line: 50, Character: 11},
		End:   lsp.Position{Line: 50, Character: 35},
	}
	assert.Equal(t, expected, diag.Range)

}

func TestDiagnosisVariable(t *testing.T) {
	ds := getDefaultDS(t)

	// Test for undefined variable, variable later defined
	diag := FindDiagnosticByMsg(ds.stored, "unexpected variable or config property 'baz'")
	assert.NotNil(t, diag)
	expected := lsp.Range{
		Start: lsp.Position{Line: 55, Character: 8},
		End:   lsp.Position{Line: 55, Character: 11},
	}
	assert.Equal(t, expected, diag.Range)

	// Test for undefined variable in function
	diag = FindDiagnosticByMsg(ds.stored, "unexpected variable or config property 'bar'")
	assert.NotNil(t, diag)
	expected = lsp.Range{
		Start: lsp.Position{Line: 51, Character: 4},
		End:   lsp.Position{Line: 51, Character: 7},
	}
	assert.Equal(t, expected, diag.Range)

}

func TestDiagnosisFunction(t *testing.T) {
	ds := getDefaultDS(t)

	// Test for undefined function
	diag := FindDiagnosticByMsg(ds.stored, "function undefined: blah")
	assert.NotNil(t, diag)
	expected := lsp.Range{
		Start: lsp.Position{Line: 53, Character: 0},
		End:   lsp.Position{Line: 53, Character: 4},
	}
	assert.Equal(t, expected, diag.Range)

	// Test for not enough argument
	diag = FindDiagnosticByMsg(ds.stored, "not enough arguments in call to add_dep")
	assert.NotNil(t, diag)
	expected = lsp.Range{
		Start: lsp.Position{Line: 62, Character: 7},
		End:   lsp.Position{Line: 62, Character: 15},
	}
	assert.Equal(t, expected, diag.Range)

	for _, d := range ds.stored {
		t.Log(d)
	}
}

func TestDiagnosticsProperty(t *testing.T) {
	ds := getDefaultDS(t)

	// Test for invalid string call
	diag := FindDiagnosticByMsg(ds.stored, "function undefined: foo")
	assert.NotNil(t, diag)
	expected := lsp.Range{
		Start: lsp.Position{Line: 64, Character: 30},
		End:   lsp.Position{Line: 64, Character: 33},
	}
	assert.Equal(t, expected, diag.Range)

	// Test for invalid config property
	diag = FindDiagnosticByMsg(ds.stored, "unexpected variable or config property 'BLAH'")
	assert.NotNil(t, diag)
	expected = lsp.Range{
		Start: lsp.Position{Line: 66, Character: 13},
		End:   lsp.Position{Line: 66, Character: 17},
	}
	assert.Equal(t, expected, diag.Range)

	// Ensure correct config property is not being listed in diagnostics
	configRange := lsp.Range{
		Start: lsp.Position{Line: 64, Character: 58},
		End:   lsp.Position{Line: 64, Character: 84},
	}
	assert.Nil(t, FindDiagnosticByRange(ds.stored, configRange))

}

func TestDiagnoseOutOfScope(t *testing.T) {
	analyzer.State.Config.Parse.BuildFileName = append(analyzer.State.Config.Parse.BuildFileName, "out_of_scope.build")
	ds := &diagnosticStore{
		uri:    OutScopeURI,
		stored: []*lsp.Diagnostic{},
	}

	stmts, err := analyzer.AspStatementFromFile(OutScopeURI)
	assert.NoError(t, err)

	ds.storeDiagnostics(analyzer, stmts)
	assert.Equal(t, 1, len(ds.stored))
	assert.NotNil(t, FindDiagnosticByMsg(ds.stored, "unexpected variable or config property 'blah'"))
	for _, d := range ds.stored {
		t.Log(d)
	}
}

func TestStoreFuncCallDiagnosticsBuildDef(t *testing.T) {
	ctx := context.Background()
	ds := &diagnosticStore{
		uri: exampleBuildURI,
	}

	buildDefs, err := analyzer.BuildDefsFromURI(ctx, exampleBuildURI)
	assert.NoError(t, err)

	// Tests for build def
	buildDef := buildDefs["langserver_test"]
	callRange := lsp.Range{
		Start: lsp.Position{Line: 19, Character: 1},
		End:   lsp.Position{Line: 32, Character: 1},
	}

	ds.diagnoseFuncCall(analyzer, "go_test",
		buildDef.Action.Call.Arguments, callRange)
	expectedRange := lsp.Range{
		Start: lsp.Position{Line: 25, Character: 4},
		End:   lsp.Position{Line: 25, Character: 7},
	}
	diag := FindDiagnosticByMsg(ds.stored,
		"unexpected argument foo")
	assert.Equal(t, expectedRange, diag.Range)
}

func TestStoreFuncCallDiagnosticsFuncCall(t *testing.T) {
	ds := &diagnosticStore{
		uri:    exampleBuildURI,
		stored: []*lsp.Diagnostic{},
	}

	stmts, err := analyzer.AspStatementFromFile(exampleBuildURI)
	assert.NoError(t, err)

	// Test for regular function call with correct argument
	callRange := lsp.Range{
		Start: lsp.Position{Line: 49, Character: 0},
		End:   lsp.Position{Line: 49, Character: 33},
	}
	stmt := analyzer.StatementFromPos(stmts, callRange.Start)
	ds.diagnoseFuncCall(analyzer, "subinclude", stmt.Ident.Action.Call.Arguments, callRange)
	assert.Zero(t, len(ds.stored))

	// Test for function call with incorrect argument value type
	callRange = lsp.Range{
		Start: lsp.Position{Line: 50, Character: 0},
		End:   lsp.Position{Line: 50, Character: 33},
	}
	stmt = analyzer.StatementFromPos(stmts, callRange.Start)
	ds.diagnoseFuncCall(analyzer, "subinclude", stmt.Ident.Action.Call.Arguments, callRange)

	expectedRange := lsp.Range{
		Start: lsp.Position{Line: 50, Character: 11},
		End:   lsp.Position{Line: 50, Character: 35},
	}
	diag := FindDiagnosticByMsg(ds.stored,
		"invalid type for argument type 'dict' for target, expecting one of [str]")
	assert.Equal(t, expectedRange, diag.Range)

}

func TestDiagnosticFromBuildLabel(t *testing.T) {
	ds := getDefaultDS(t)

	dummyRange := lsp.Range{
		Start: lsp.Position{Line: 19, Character: 1},
		End:   lsp.Position{Line: 32, Character: 1},
	}

	// Tests for valid labels
	diag := ds.diagnosticFromBuildLabel(analyzer, "//src/query", dummyRange)
	assert.Nil(t, diag)
	diag = ds.diagnosticFromBuildLabel(analyzer, "//src/query:query", dummyRange)
	assert.Nil(t, diag)
	diag = ds.diagnosticFromBuildLabel(analyzer, ":langserver_test", dummyRange)
	assert.Nil(t, diag)
	diag = ds.diagnosticFromBuildLabel(analyzer, ":langserver", dummyRange)
	assert.Nil(t, diag)
	diag = ds.diagnosticFromBuildLabel(analyzer, "//third_party/go:jsonrpc2", dummyRange)
	assert.Nil(t, diag)

	// Tests for invalid labels
	diag = ds.diagnosticFromBuildLabel(analyzer, "//src/blah:foo", dummyRange)
	assert.Equal(t, "Invalid build label //src/blah:foo. error: cannot find the path for build label //src/blah:foo", diag.Message)

	// Tests for invisible labels
	diag = ds.diagnosticFromBuildLabel(analyzer, "//src/output:interactive_display_test", dummyRange)
	assert.Equal(t, "build label //src/output:interactive_display_test is not visible to current package",
		diag.Message)
}

/************************
 * Helper functions
 ************************/
func getDefaultDS(t testing.TB) *diagnosticStore {
	if !StringInSlice(analyzer.State.Config.Parse.BuildFileName, "example.build") {
		analyzer.State.Config.Parse.BuildFileName = append(analyzer.State.Config.Parse.BuildFileName, "example.build")
	}

	ds := &diagnosticStore{
		uri:    exampleBuildURI,
		stored: []*lsp.Diagnostic{},
	}

	stmts, err := analyzer.AspStatementFromFile(exampleBuildURI)
	assert.NoError(t, err)

	ds.storeDiagnostics(analyzer, stmts)

	return ds
}

func FindDiagnosticByMsg(diag []*lsp.Diagnostic, message string) *lsp.Diagnostic {
	for _, v := range diag {
		if v.Message == message {
			return v
		}
	}
	return nil
}

func FindDiagnosticByRange(diag []*lsp.Diagnostic, DRange lsp.Range) *lsp.Diagnostic {
	for _, v := range diag {
		if v.Range == DRange {
			return v
		}
	}
	return nil
}
