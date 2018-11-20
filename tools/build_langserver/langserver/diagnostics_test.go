package langserver

import (
	"context"
	"testing"
	"tools/build_langserver/lsp"

	"github.com/stretchr/testify/assert"
)

func TestDiagnose(t *testing.T) {
	analyzer.State.Config.Parse.BuildFileName = append(analyzer.State.Config.Parse.BuildFileName, "example.build")
	ds := &diagnosticStore{
		uri:    exampleBuildURI,
		stored: []*lsp.Diagnostic{},
	}

	stmts, err := analyzer.AspStatementFromFile(exampleBuildURI)
	assert.NoError(t, err)

	ds.storeDiagnostics(analyzer, stmts)

	assert.Equal(t, 11, len(ds.stored))
	assert.NotNil(t, DiagnosticStored(ds.stored, "Invalid build label //dummy/buildlabels:foo"))
	assert.NotNil(t, DiagnosticStored(ds.stored, "unexpected argument foo"))
	assert.NotNil(t, DiagnosticStored(ds.stored,
		"invalid type for argument type 'dict' for target, expecting one of [str]"))
	assert.NotNil(t, DiagnosticStored(ds.stored, "unexpected variable 'baz'"))
	assert.NotNil(t, DiagnosticStored(ds.stored, "unexpected variable 'bar'"))
	assert.NotNil(t, DiagnosticStored(ds.stored, "function undefined: blah"))

	for _, d := range ds.stored {
		t.Log(d)
	}
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
	assert.NotNil(t, DiagnosticStored(ds.stored, "unexpected variable 'blah'"))
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

	ds.storeFuncCallDiagnostics(analyzer, "go_test",
		buildDef.Action.Call.Arguments, callRange)
	expectedRange := lsp.Range{
		Start: lsp.Position{Line: 25, Character: 4},
		End:   lsp.Position{Line: 25, Character: 7},
	}
	diag := DiagnosticStored(ds.stored,
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
	ds.storeFuncCallDiagnostics(analyzer, "subinclude", stmt.Ident.Action.Call.Arguments, callRange)
	assert.Zero(t, len(ds.stored))

	// Test for function call with incorrect argument value type
	callRange = lsp.Range{
		Start: lsp.Position{Line: 50, Character: 0},
		End:   lsp.Position{Line: 50, Character: 33},
	}
	stmt = analyzer.StatementFromPos(stmts, callRange.Start)
	ds.storeFuncCallDiagnostics(analyzer, "subinclude", stmt.Ident.Action.Call.Arguments, callRange)

	expectedRange := lsp.Range{
		Start: lsp.Position{Line: 50, Character: 11},
		End:   lsp.Position{Line: 50, Character: 35},
	}
	diag := DiagnosticStored(ds.stored,
		"invalid type for argument type 'dict' for target, expecting one of [str]")
	assert.Equal(t, expectedRange, diag.Range)

}

func TestDiagnosticFromBuildLabel(t *testing.T) {
	analyzer.State.Config.Parse.BuildFileName = append(analyzer.State.Config.Parse.BuildFileName, "example.build")
	ds := &diagnosticStore{
		uri: exampleBuildURI,
	}
	t.Log(ds.uri)
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
	assert.Equal(t, "Invalid build label //src/blah:foo", diag.Message)

	// Tests for invisible labels
	diag = ds.diagnosticFromBuildLabel(analyzer, "//src/output:interactive_display_test", dummyRange)
	assert.Equal(t, "build label //src/output:interactive_display_test is not visible to current package",
		diag.Message)
}

/************************
 * Helper functions
 ************************/
func DiagnosticStored(diag []*lsp.Diagnostic, message string) *lsp.Diagnostic {
	for _, v := range diag {
		if v.Message == message {
			return v
		}
	}
	return nil
}
