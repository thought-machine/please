package langserver

import (
	"os"
	"testing"

	"core"
	"tools/build_langserver/lsp"
)

// TODO(bnm): cleanup setup
// TestMain runs the setup for the tests for all the tests relating to langserver
func TestMain(m *testing.M) {
	core.FindRepoRoot()
	dummyBuildFiles := []string{
		"completion2.build",
	}

	for _, i := range dummyBuildFiles {
		analyzer.State.Config.Parse.BuildFileName = append(analyzer.State.Config.Parse.BuildFileName, i)
	}

	retCode := m.Run()
	os.Exit(retCode)
}

var exampleBuildURI = lsp.DocumentURI("file://tools/build_langserver/langserver/test_data/example.build")
var assignBuildURI = lsp.DocumentURI("file://tools/build_langserver/langserver/test_data/assignment.build")
var propURI = lsp.DocumentURI("file://tools/build_langserver/langserver/test_data/property.build")
var miscURI = lsp.DocumentURI("file://tools/build_langserver/langserver/test_data/misc.build")
var completionURI = lsp.DocumentURI("file://tools/build_langserver/langserver/test_data/completion.build")
var completion2URI = lsp.DocumentURI("file://tools/build_langserver/langserver/test_data/completion2.build")
var completionPropURI = lsp.DocumentURI("file://tools/build_langserver/langserver/test_data/completion_props.build")
var completionLabelURI = lsp.DocumentURI("file://tools/build_langserver/langserver/test_data/completion_buildlabels.build")
var completionLiteralURI = lsp.DocumentURI("file://tools/build_langserver/langserver/test_data/completion_literal.build")

var analyzer, _ = newAnalyzer()

var handler = LsHandler{
	repoRoot:  core.RepoRoot,
	workspace: newWorkspaceStore(lsp.DocumentURI(core.RepoRoot)),
	analyzer:  analyzer,
	init: &lsp.InitializeParams{
		RootURI: lsp.DocumentURI(core.RepoRoot),
		Capabilities: lsp.ClientCapabilities{
			TextDocument: lsp.TextDocumentClientCapabilities{
				Completion: lsp.Completion{
					CompletionItem: struct {
						SnippetSupport bool `json:"snippetSupport,omitempty"`
					}{SnippetSupport: true}},
			},
		},
	},
}
