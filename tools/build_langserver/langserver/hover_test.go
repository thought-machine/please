package langserver

import (
	"context"
	"core"
	"path"
	"testing"
	"tools/build_langserver/lsp"

	"github.com/stretchr/testify/assert"
	"fmt"
)

func TestGetHoverContent(t *testing.T) {
	core.FindRepoRoot()
	ctx := context.Background()
	filepath := path.Join(core.RepoRoot, "tools/build_langserver/langserver/test_data/example.build")
	uri := lsp.DocumentURI("file://" + filepath)

	analyzer := newAnalyzer()
	// Test hovering on the function call
	content, err := getHoverContent(ctx, analyzer, uri, lsp.Position{Line: 0, Character: 3})
	assert.Equal(t, nil, err)
	expected := "def go_library(name:str, srcs:list, asm_srcs:list=None, hdrs:list=None, out:str=None, deps:list=[],\n" +
		"               visibility:list=None, test_only:bool&testonly=False, complete:bool=True,\n" +
		"               _needs_transitive_deps=False, _all_srcs=False, cover:bool=True,\n" +
		"               filter_srcs:bool=True, _link_private:bool=False, _link_extra:bool=True)\n\n" +
		"\"Generates a Go library which can be reused by other rules.\n\n" +
		"    Args:\n" +
		"      name (str): Name of the rule.\n" +
		"      srcs (list): Go source files to compile.\n" +
		"      asm_srcs (list): Source files to assemble with `go tool assemble`.\n" +
		"      hdrs (list): Header files needed for assembly. Has no effect if asm_srcs is not given.\n" +
		"      out (str): Name of the output library to compile (defaults to name suffixed with .a)\n" +
		"      deps (list): Dependencies\n" +
		"      visibility (list): Visibility specification\n" +
		"      test_only (bool): If True, is only visible to test rules.\n" +
		"      complete (bool): Indicates whether the library is complete or not (ie. buildable with\n" +
		"                       `go tool build -complete`). In nearly all cases this is True (the main\n" +
		"                       exception being for cgo).\n" +
		"      cover (bool): Indicates whether this library should be considered for coverage annotations.\n" +
		"                    Libraries are only annotated when using `plz cover` (or `plz build -c cover`),\n" +
		"                    but if this is false they never will be. Can be useful for e.g. third-party\n" +
		"                    code that you never want to be instrumented.\n" +
		"      filter_srcs (bool): If True, filters source files through Go's standard build constraints.\n" +
		"    \""

	assert.Equal(t, expected, content.Value)

	// Test hovering over arguments
	content, err = getHoverContent(ctx, analyzer, uri, lsp.Position{Line:7, Character:7})
	assert.Equal(t, nil, err)
	assert.Equal(t, "deps required:false, type:list", content.Value)

	// Test hovering over arguments
	content2, err := getHoverContent(ctx, analyzer, uri, lsp.Position{Line:5, Character:10})
	assert.Equal(t, nil, err)
	fmt.Println(content2.Value)
	//assert.Equal(t, "deps required:false, type:list", content.Value)
}
