package langserver

import (
	"context"
	"testing"
	"tools/build_langserver/lsp"

	"github.com/stretchr/testify/assert"
)

func TestDiagnose(t *testing.T) {
	analyzer.State.Config.Parse.BuildFileName = append(analyzer.State.Config.Parse.BuildFileName, "example.build")
	ds := &diagnosticsStore{
		uri:         exampleBuildURI,
		analyzer:    analyzer,
		diagnostics: make(map[lsp.Range]*lsp.Diagnostic),
	}

	stmts, err := analyzer.AspStatementFromFile(exampleBuildURI)
	assert.NoError(t, err)

	ds.diagnose(stmts)

	assert.Equal(t, 11, len(ds.diagnostics))
	assert.True(t, DiagnosticStored(ds.diagnostics, "Invalid build label //dummy/buildlabels:foo"))
	assert.True(t, DiagnosticStored(ds.diagnostics, "unexpected argument foo"))
	assert.True(t, DiagnosticStored(ds.diagnostics,
		"invalid type for argument type 'dict' for target, expecting one of [str]"))
	assert.True(t, DiagnosticStored(ds.diagnostics, "unexpected variable 'baz'"))
	assert.True(t, DiagnosticStored(ds.diagnostics, "unexpected variable 'bar'"))
	assert.True(t, DiagnosticStored(ds.diagnostics, "function undefined: blah"))

	for _, d := range ds.diagnostics {
		t.Log(d)
	}
}

func TestDiagnoseOutOfScope(t *testing.T) {
	analyzer.State.Config.Parse.BuildFileName = append(analyzer.State.Config.Parse.BuildFileName, "out_of_scope.build")
	ds := &diagnosticsStore{
		uri:         OutScopeURI,
		analyzer:    analyzer,
		diagnostics: make(map[lsp.Range]*lsp.Diagnostic),
	}

	stmts, err := analyzer.AspStatementFromFile(OutScopeURI)
	assert.NoError(t, err)

	ds.diagnose(stmts)
	assert.Equal(t, 1, len(ds.diagnostics))
	assert.True(t, DiagnosticStored(ds.diagnostics, "unexpected variable 'blah'"))
	for _, d := range ds.diagnostics {
		t.Log(d)
	}
}

func TestStoreFuncCallDiagnosticsBuildDef(t *testing.T) {
	ctx := context.Background()
	ds := &diagnosticsStore{
		uri:         exampleBuildURI,
		analyzer:    analyzer,
		diagnostics: make(map[lsp.Range]*lsp.Diagnostic),
	}

	buildDefs, err := analyzer.BuildDefsFromURI(ctx, exampleBuildURI)
	assert.NoError(t, err)

	// Tests for build def
	buildDef := buildDefs["langserver_test"]
	callRange := lsp.Range{
		Start: lsp.Position{Line: 19, Character: 1},
		End:   lsp.Position{Line: 32, Character: 1},
	}

	ds.storeFuncCallDiagnostics("go_test", buildDef.Action.Call.Arguments,
		callRange)
	expectedRange := lsp.Range{
		Start: lsp.Position{Line: 25, Character: 4},
		End:   lsp.Position{Line: 25, Character: 7},
	}
	diag, ok := ds.diagnostics[expectedRange]
	assert.True(t, ok)
	assert.Equal(t, lsp.Error, diag.Severity)
	assert.Equal(t, "unexpected argument foo", diag.Message)
}

func TestStoreFuncCallDiagnosticsFuncCall(t *testing.T) {
	ds := &diagnosticsStore{
		uri:         exampleBuildURI,
		analyzer:    analyzer,
		diagnostics: make(map[lsp.Range]*lsp.Diagnostic),
	}

	stmts, err := analyzer.AspStatementFromFile(exampleBuildURI)
	assert.NoError(t, err)

	// Test for regular function call with correct argument
	callRange := lsp.Range{
		Start: lsp.Position{Line: 49, Character: 0},
		End:   lsp.Position{Line: 49, Character: 33},
	}
	stmt := analyzer.StatementFromPos(stmts, callRange.Start)
	ds.storeFuncCallDiagnostics("subinclude", stmt.Ident.Action.Call.Arguments, callRange)
	assert.Zero(t, len(ds.diagnostics))

	// Test for function call with incorrect argument value type
	callRange = lsp.Range{
		Start: lsp.Position{Line: 50, Character: 0},
		End:   lsp.Position{Line: 50, Character: 33},
	}
	stmt = analyzer.StatementFromPos(stmts, callRange.Start)
	ds.storeFuncCallDiagnostics("subinclude", stmt.Ident.Action.Call.Arguments, callRange)

	expectedRange := lsp.Range{
		Start: lsp.Position{Line: 50, Character: 11},
		End:   lsp.Position{Line: 50, Character: 35},
	}
	diag, ok := ds.diagnostics[expectedRange]
	assert.True(t, ok)
	assert.Equal(t, "invalid type for argument type 'dict' for target, expecting one of [str]",
		diag.Message)

}

func TestDiagnosticFromBuildLabel(t *testing.T) {
	analyzer.State.Config.Parse.BuildFileName = append(analyzer.State.Config.Parse.BuildFileName, "example.build")
	ds := &diagnosticsStore{
		analyzer: analyzer,
		uri:      exampleBuildURI,
	}
	t.Log(ds.uri)
	dummyRange := lsp.Range{
		Start: lsp.Position{Line: 19, Character: 1},
		End:   lsp.Position{Line: 32, Character: 1},
	}

	// Tests for valid labels
	diag := ds.diagnosticFromBuildLabel("//src/query", dummyRange)
	assert.Nil(t, diag)
	diag = ds.diagnosticFromBuildLabel("//src/query:query", dummyRange)
	assert.Nil(t, diag)
	diag = ds.diagnosticFromBuildLabel(":langserver_test", dummyRange)
	assert.Nil(t, diag)
	diag = ds.diagnosticFromBuildLabel(":langserver", dummyRange)
	assert.Nil(t, diag)
	diag = ds.diagnosticFromBuildLabel("//third_party/go:jsonrpc2", dummyRange)
	assert.Nil(t, diag)

	// Tests for invalid labels
	diag = ds.diagnosticFromBuildLabel("//src/blah:foo", dummyRange)
	assert.Equal(t, "Invalid build label //src/blah:foo", diag.Message)

	// Tests for invisible labels
	diag = ds.diagnosticFromBuildLabel("//src/output:interactive_display_test", dummyRange)
	assert.Equal(t, "build label //src/output:interactive_display_test is not visible to current package",
		diag.Message)
}

/************************
 * Helper functions
 ************************/
func DiagnosticStored(diag map[lsp.Range]*lsp.Diagnostic, message string) bool {
	for _, v := range diag {
		if v.Message == message {
			return true
		}
	}
	return false
}
