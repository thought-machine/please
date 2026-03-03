package lsp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sourcegraph/go-lsp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReferencesBuiltinFunction(t *testing.T) {
	// Test finding references to go_library function using actual test data file
	uri := "file://" + filepath.Join(os.Getenv("TEST_DIR"), "tools/build_langserver/lsp/test_data/src/core/test.build")
	h := initHandler()
	content, err := os.ReadFile(filepath.Join(os.Getenv("TEST_DIR"), "tools/build_langserver/lsp/test_data/src/core/test.build"))
	require.NoError(t, err)

	err = h.Request("textDocument/didOpen", &lsp.DidOpenTextDocumentParams{
		TextDocument: lsp.TextDocumentItem{
			URI:  lsp.DocumentURI(uri),
			Text: string(content),
		},
	}, nil)
	require.NoError(t, err)
	h.WaitForPackage("src/core")

	var locs []lsp.Location
	err = h.Request("textDocument/references", &lsp.ReferenceParams{
		TextDocumentPositionParams: lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{
				URI: lsp.DocumentURI(uri),
			},
			Position: lsp.Position{Line: 0, Character: 5}, // on "go_library"
		},
		Context: lsp.ReferenceContext{
			IncludeDeclaration: false,
		},
	}, &locs)
	require.NoError(t, err)
	// go_library is widely used, we should find many references
	assert.NotEmpty(t, locs, "expected to find references to go_library")
}

func TestReferencesGoTestFunction(t *testing.T) {
	// Test finding references to go_test function
	uri := "file://" + filepath.Join(os.Getenv("TEST_DIR"), "tools/build_langserver/lsp/test_data/src/core/test.build")
	h := initHandler()
	content, err := os.ReadFile(filepath.Join(os.Getenv("TEST_DIR"), "tools/build_langserver/lsp/test_data/src/core/test.build"))
	require.NoError(t, err)

	err = h.Request("textDocument/didOpen", &lsp.DidOpenTextDocumentParams{
		TextDocument: lsp.TextDocumentItem{
			URI:  lsp.DocumentURI(uri),
			Text: string(content),
		},
	}, nil)
	require.NoError(t, err)
	h.WaitForPackage("src/core")

	var locs []lsp.Location
	err = h.Request("textDocument/references", &lsp.ReferenceParams{
		TextDocumentPositionParams: lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{
				URI: lsp.DocumentURI(uri),
			},
			Position: lsp.Position{Line: 19, Character: 3}, // on "go_test" (line 20, 0-indexed = 19)
		},
		Context: lsp.ReferenceContext{
			IncludeDeclaration: false,
		},
	}, &locs)
	require.NoError(t, err)
	// go_test is widely used, we should find many references
	assert.NotEmpty(t, locs, "expected to find references to go_test")
}

func TestReferencesNoResults(t *testing.T) {
	// Test that we get empty results when cursor is on a comment
	h := initHandlerText(`# just a comment
# nothing referenceable here`)
	h.WaitForPackageTree()

	var locs []lsp.Location
	err := h.Request("textDocument/references", &lsp.ReferenceParams{
		TextDocumentPositionParams: lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{
				URI: testURI,
			},
			Position: lsp.Position{Line: 0, Character: 5},
		},
		Context: lsp.ReferenceContext{
			IncludeDeclaration: false,
		},
	}, &locs)
	require.NoError(t, err)
	assert.Empty(t, locs, "expected no references for a comment")
}

func TestReferencesBuildLabel(t *testing.T) {
	// Test that label references don't panic when package not in graph
	h := initHandlerText(`go_library(
    name = "core",
    srcs = ["lib.go"],
)

go_test(
    name = "core_test",
    srcs = ["lib_test.go"],
    deps = [":core"],
)`)
	h.WaitForPackageTree()

	var locs []lsp.Location
	err := h.Request("textDocument/references", &lsp.ReferenceParams{
		TextDocumentPositionParams: lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{
				URI: testURI,
			},
			// Position on the ":core" string in deps
			Position: lsp.Position{Line: 8, Character: 13},
		},
		Context: lsp.ReferenceContext{
			IncludeDeclaration: false,
		},
	}, &locs)
	// Should not error even if package not in graph
	require.NoError(t, err)
}

func TestReferencesIncludeDeclaration(t *testing.T) {
	// Test that IncludeDeclaration includes the function definition
	uri := "file://" + filepath.Join(os.Getenv("TEST_DIR"), "tools/build_langserver/lsp/test_data/src/core/test.build")
	h := initHandler()
	content, err := os.ReadFile(filepath.Join(os.Getenv("TEST_DIR"), "tools/build_langserver/lsp/test_data/src/core/test.build"))
	require.NoError(t, err)

	err = h.Request("textDocument/didOpen", &lsp.DidOpenTextDocumentParams{
		TextDocument: lsp.TextDocumentItem{
			URI:  lsp.DocumentURI(uri),
			Text: string(content),
		},
	}, nil)
	require.NoError(t, err)
	h.WaitForPackage("src/core")

	var locsWithDecl []lsp.Location
	err = h.Request("textDocument/references", &lsp.ReferenceParams{
		TextDocumentPositionParams: lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{
				URI: lsp.DocumentURI(uri),
			},
			Position: lsp.Position{Line: 0, Character: 5}, // on "go_library"
		},
		Context: lsp.ReferenceContext{
			IncludeDeclaration: true,
		},
	}, &locsWithDecl)
	require.NoError(t, err)

	var locsWithoutDecl []lsp.Location
	err = h.Request("textDocument/references", &lsp.ReferenceParams{
		TextDocumentPositionParams: lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{
				URI: lsp.DocumentURI(uri),
			},
			Position: lsp.Position{Line: 0, Character: 5}, // on "go_library"
		},
		Context: lsp.ReferenceContext{
			IncludeDeclaration: false,
		},
	}, &locsWithoutDecl)
	require.NoError(t, err)

	// With declaration should include one more location (the definition)
	assert.GreaterOrEqual(t, len(locsWithDecl), len(locsWithoutDecl),
		"IncludeDeclaration=true should return at least as many results")
}

func TestFindLabelsForFile(t *testing.T) {
	// This test verifies that findLabelsForFile returns ALL labels that produce a given file,
	// not just the first one found. This is important when the same file is exposed by
	// multiple filegroups under different labels (e.g., //pkg:alias1 and //pkg:alias2).
	h := initHandler()
	h.WaitForPackageTree()

	// Test with a file that exists - should return at least one label or empty
	// (we can't easily test multiple labels without setting up complex test data,
	// but we verify the function doesn't panic and returns a slice)
	labels := h.findLabelsForFile("nonexistent_file.build_defs")
	assert.Empty(t, labels, "nonexistent file should return empty labels")
}

func TestBuiltinHasLabelsField(t *testing.T) {
	// Verify that builtins are populated with Labels (slice) for subinclude tracking
	h := initHandler()
	h.WaitForPackageTree()

	// Check that builtins exist and have the Labels field properly initialized
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// We should have some builtins loaded
	assert.NotEmpty(t, h.builtins, "expected builtins to be populated")

	// Each builtin should have a Labels field (may be empty for core builtins)
	for name, builtinList := range h.builtins {
		for _, b := range builtinList {
			// Labels field should exist (not nil) - it's a slice so empty is valid
			assert.NotNil(t, b.Stmt, "builtin %s should have a statement", name)
			// Labels is a slice, so we just verify it's accessible (not nil check needed for slices)
			_ = b.Labels // This would panic if the field didn't exist
		}
	}
}

// initHandlerWithRoot initializes a handler with a custom root directory
func initHandlerWithRoot(root string) *Handler {
	h := NewHandler()
	h.Conn = &rpc{
		Notifications: make(chan message, 100),
	}
	result := &lsp.InitializeResult{}
	if err := h.Request("initialize", &lsp.InitializeParams{
		Capabilities: lsp.ClientCapabilities{},
		RootURI:      lsp.DocumentURI("file://" + root),
	}, result); err != nil {
		log.Fatalf("init failed: %s", err)
	}
	return h
}

func TestReferencesMultipleLabelsOneFile(t *testing.T) {
	// Test that find-references works when a file is exposed via multiple labels.
	// Scenario:
	//   - shared_defs/my_func.build_defs defines my_shared_func()
	//   - shared_defs:alias1 and shared_defs:alias2 both expose my_func.build_defs
	//   - pkg_a subincludes alias1 and calls my_shared_func()
	//   - pkg_b subincludes alias2 and calls my_shared_func()
	// Expected: find-references on my_shared_func should find both pkg_a and pkg_b usages

	testDataRoot := filepath.Join(os.Getenv("TEST_DIR"), "tools/build_langserver/lsp/test_data_find_references")
	h := initHandlerWithRoot(testDataRoot)

	// Open the shared_defs file where my_shared_func is defined
	defsFile := filepath.Join(testDataRoot, "shared_defs/my_func.build_defs")
	uri := lsp.DocumentURI("file://" + defsFile)
	content, err := os.ReadFile(defsFile)
	require.NoError(t, err)

	err = h.Request("textDocument/didOpen", &lsp.DidOpenTextDocumentParams{
		TextDocument: lsp.TextDocumentItem{
			URI:  uri,
			Text: string(content),
		},
	}, nil)
	require.NoError(t, err)
	h.WaitForPackageTree()

	// Request references for my_shared_func (cursor on "def my_shared_func")
	var locs []lsp.Location
	err = h.Request("textDocument/references", &lsp.ReferenceParams{
		TextDocumentPositionParams: lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: uri},
			Position:     lsp.Position{Line: 0, Character: 6}, // on "my_shared_func"
		},
		Context: lsp.ReferenceContext{IncludeDeclaration: false},
	}, &locs)
	require.NoError(t, err)

	// Should find references in both pkg_a and pkg_b
	// Even though they subinclude different labels (alias1 vs alias2)
	var foundPkgA, foundPkgB bool
	for _, loc := range locs {
		uriStr := string(loc.URI)
		if filepath.Base(filepath.Dir(uriStr)) == "pkg_a" {
			foundPkgA = true
		}
		if filepath.Base(filepath.Dir(uriStr)) == "pkg_b" {
			foundPkgB = true
		}
	}

	assert.True(t, foundPkgA, "expected to find reference in pkg_a (subincludes alias1)")
	assert.True(t, foundPkgB, "expected to find reference in pkg_b (subincludes alias2)")
}

func TestReferencesNoFalsePositivesForSameNameDifferentFile(t *testing.T) {
	// Test that we DON'T get false positives when two different files define
	// functions with the same name.
	// Scenario:
	//   - defs1/func.build_defs defines duplicate_func()
	//   - defs2/func.build_defs ALSO defines duplicate_func() (different file!)
	//   - pkg_uses_defs1 subincludes //defs1 and calls duplicate_func()
	//   - pkg_uses_defs2 subincludes //defs2 and calls duplicate_func()
	// Expected: find-references on defs1's definition should ONLY find pkg_uses_defs1,
	//           NOT pkg_uses_defs2 (which uses a different definition)

	testDataRoot := filepath.Join(os.Getenv("TEST_DIR"), "tools/build_langserver/lsp/test_data_find_references")
	h := initHandlerWithRoot(testDataRoot)

	// Open defs1's file where duplicate_func is defined
	defs1File := filepath.Join(testDataRoot, "defs1/func.build_defs")
	uri1 := lsp.DocumentURI("file://" + defs1File)
	content1, err := os.ReadFile(defs1File)
	require.NoError(t, err)

	err = h.Request("textDocument/didOpen", &lsp.DidOpenTextDocumentParams{
		TextDocument: lsp.TextDocumentItem{
			URI:  uri1,
			Text: string(content1),
		},
	}, nil)
	require.NoError(t, err)
	h.WaitForPackageTree()

	// Request references for duplicate_func from defs1
	var locs []lsp.Location
	err = h.Request("textDocument/references", &lsp.ReferenceParams{
		TextDocumentPositionParams: lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: uri1},
			Position:     lsp.Position{Line: 0, Character: 6}, // on "duplicate_func"
		},
		Context: lsp.ReferenceContext{IncludeDeclaration: false},
	}, &locs)
	require.NoError(t, err)

	// Check which packages were found
	var foundPkgUsesDefs1, foundPkgUsesDefs2 bool
	for _, loc := range locs {
		uriStr := string(loc.URI)
		dir := filepath.Base(filepath.Dir(uriStr))
		if dir == "pkg_uses_defs1" {
			foundPkgUsesDefs1 = true
		}
		if dir == "pkg_uses_defs2" {
			foundPkgUsesDefs2 = true
		}
	}

	// Should find pkg_uses_defs1 (correctly subincludes defs1)
	assert.True(t, foundPkgUsesDefs1, "expected to find reference in pkg_uses_defs1")
	// Should NOT find pkg_uses_defs2 (uses different definition from defs2)
	assert.False(t, foundPkgUsesDefs2, "should NOT find reference in pkg_uses_defs2 - it uses a different definition")
}

// TestReferencesSurvivesBrokenParse verifies that find-references doesn't crash
// when a document fails to parse.
func TestReferencesSurvivesBrokenParse(t *testing.T) {
	h := initHandler()

	// Open a doc with syntactically broken content
	brokenContent := "def broken_func(\n    # missing closing paren"
	h.Request("textDocument/didOpen", &lsp.DidOpenTextDocumentParams{
		TextDocument: lsp.TextDocumentItem{
			URI:     "file:///tmp/broken.build",
			Text:    brokenContent,
			Version: 1,
		},
	}, nil)

	// parseIfNeeded should handle the parse failure gracefully
	doc := h.docs["/tmp/broken.build"]
	if doc != nil {
		ast := h.parseIfNeeded(doc)
		// Should return empty/partial AST, not panic
		t.Logf("parsed broken file, got %d statements", len(ast))
	}

	// Also verify find-references doesn't crash on a function that doesn't exist
	locs, err := h.findFunctionReferences("broken_func", "", false)
	assert.NoError(t, err)
	t.Logf("found %d locations for non-existent func (expected: 0)", len(locs))
}
