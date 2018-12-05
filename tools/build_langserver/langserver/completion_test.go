package langserver

import (
	"context"
	"strings"
	"testing"

	"github.com/thought-machine/please/tools/build_langserver/lsp"

	"github.com/stretchr/testify/assert"
)

/***************************************
 *Tests for Attributes
 ***************************************/
func TestCompletionWithCONFIG(t *testing.T) {
	ctx := context.Background()

	// Test completion on CONFIG with no starting character
	items, err := handler.getCompletionItemsList(ctx, completionPropURI, lsp.Position{Line: 4, Character: 7})
	assert.Equal(t, nil, err)
	assert.Equal(t, len(analyzer.State.Config.TagsToFields()), len(items))
	for _, i := range items {
		assert.Equal(t, i.Kind, lsp.Property)
	}

	// Test completion on CONFIG with 1 starting character
	items, err = handler.getCompletionItemsList(ctx, completionPropURI, lsp.Position{Line: 5, Character: 8})
	assert.Equal(t, nil, err)
	assert.True(t, len(analyzer.State.Config.TagsToFields()) > len(items))
	assert.True(t, itemInList(items, "JARCAT_TOOL"))
	assert.False(t, itemInList(items, "PLZ_VERSION"))

	// Test completion on CONFIG with a word
	items, err = handler.getCompletionItemsList(ctx, completionPropURI, lsp.Position{Line: 6, Character: 11})
	assert.Equal(t, nil, err)
	assert.True(t, len(analyzer.State.Config.TagsToFields()) > len(items))
	assert.True(t, itemInList(items, "JAVAC_TOOL"))
	for _, i := range items {
		assert.True(t, strings.Contains(i.Label, "JAVA"))
	}

	// Test completion with assignment
	items, err = handler.getCompletionItemsList(ctx, completionPropURI, lsp.Position{Line: 7, Character: 18})
	assert.Equal(t, nil, err)
	assert.True(t, len(analyzer.State.Config.TagsToFields()) > len(items))
	assert.True(t, itemInList(items, "JAVAC_TOOL"))
	for _, i := range items {
		assert.True(t, strings.Contains(i.Label, "JAVA"))
	}

	// Test completion on empty line
	items, err = handler.getCompletionItemsList(ctx, completionPropURI, lsp.Position{Line: 9, Character: 13})
	assert.Equal(t, nil, err)
	assert.Equal(t, 0, len(items))

	// Test config should be empty
	items, err = handler.getCompletionItemsList(ctx, completionPropURI, lsp.Position{Line: 8, Character: 14})
	assert.Equal(t, nil, err)
	assert.Equal(t, 0, len(items))
}

func TestCompletionWithStringMethods(t *testing.T) {
	ctx := context.Background()
	context.Background()

	// Tests completion on no letters follows after dot(.)
	items, err := handler.getCompletionItemsList(ctx, completionPropURI, lsp.Position{Line: 10, Character: 19})
	assert.Equal(t, nil, err)
	assert.Equal(t, len(analyzer.Attributes["str"]), len(items))
	assert.True(t, itemInList(items, "replace"))
	assert.True(t, itemInList(items, "format"))

	// Test completion with 1 starting character: f
	items, err = handler.getCompletionItemsList(ctx, completionPropURI, lsp.Position{Line: 11, Character: 20})
	assert.Equal(t, nil, err)
	assert.True(t, itemInList(items, "format"))
	assert.True(t, itemInList(items, "find"))
	assert.True(t, itemInList(items, "rfind"))

	// Test completion with a three letters: for
	items, err = handler.getCompletionItemsList(ctx, completionPropURI, lsp.Position{Line: 12, Character: 22})
	assert.Equal(t, nil, err)
	assert.Equal(t, 1, len(items))
	assert.Equal(t, "format", items[0].Label)

	// Test completion with assignment
	items, err = handler.getCompletionItemsList(ctx, completionPropURI, lsp.Position{Line: 13, Character: 19})
	assert.Equal(t, nil, err)
	assert.Equal(t, 1, len(items))
	assert.Equal(t, "format", items[0].Label)

	// Test str method completion with variables
	items, err = handler.getCompletionItemsList(ctx, completionPropURI, lsp.Position{Line: 14, Character: 16})
	assert.Equal(t, nil, err)
	assert.Equal(t, len(analyzer.Attributes["str"]), len(items))
	assert.True(t, itemInList(items, "replace"))
	assert.True(t, itemInList(items, "format"))
}

func TestCompletionWithDictMethods(t *testing.T) {
	ctx := context.Background()

	// Tests completion on no letters follows after dot(.)
	items, err := handler.getCompletionItemsList(ctx, completionPropURI, lsp.Position{Line: 17, Character: 25})
	assert.Equal(t, nil, err)
	assert.Equal(t, len(analyzer.Attributes["dict"]), len(items))
	assert.True(t, itemInList(items, "get"))
	assert.True(t, itemInList(items, "keys"))
	assert.True(t, itemInList(items, "items"))

	// Test completion with 1 starting character: k
	items, err = handler.getCompletionItemsList(ctx, completionPropURI, lsp.Position{Line: 18, Character: 16})
	assert.Equal(t, nil, err)
	assert.Equal(t, 1, len(items))
	assert.Equal(t, "keys", items[0].Label)

	// Test completion with a three letters: get
	items, err = handler.getCompletionItemsList(ctx, completionPropURI, lsp.Position{Line: 19, Character: 18})
	assert.Equal(t, nil, err)
	assert.Equal(t, 1, len(items))
	assert.Equal(t, "get", items[0].Label)

	// Test dict method completion with variables
	items, err = handler.getCompletionItemsList(ctx, completionPropURI, lsp.Position{Line: 20, Character: 17})
	assert.Equal(t, nil, err)
	assert.Equal(t, len(analyzer.Attributes["dict"]), len(items))
	assert.True(t, itemInList(items, "get"))
	assert.True(t, itemInList(items, "keys"))
	assert.True(t, itemInList(items, "items"))
}

/***************************************
 *Tests for Build label completions
 ***************************************/
func TestCompletionWithBuildLabels(t *testing.T) {
	ctx := context.Background()

	items, err := handler.getCompletionItemsList(ctx, completionLabelURI, lsp.Position{Line: 0, Character: 6})
	assert.Equal(t, nil, err)
	assert.True(t, itemInList(items, "src/cache"))
	for _, i := range items {
		assert.True(t, strings.HasPrefix(i.Label, "src"))
	}

	items, err = handler.getCompletionItemsList(ctx, completionLabelURI, lsp.Position{Line: 1, Character: 13})
	assert.Equal(t, nil, err)
	assert.Equal(t, 1, len(items))
	assert.Equal(t, "query", items[0].Label)

	items, err = handler.getCompletionItemsList(ctx, completionLabelURI, lsp.Position{Line: 2, Character: 14})
	assert.Equal(t, nil, err)
	assert.Equal(t, 1, len(items))
	assert.Equal(t, "query", items[0].Label)
	t.Log(items[0].Label)

	// Ensure handling of partially completed labels
	items, err = handler.getCompletionItemsList(ctx, completionLabelURI, lsp.Position{Line: 4, Character: 8})
	assert.Equal(t, nil, err)
	assert.Equal(t, 2, len(items))
	assert.True(t, itemInList(items, "query"))
	assert.True(t, itemInList(items, "query:query"))
	assert.Equal(t, items[1].Detail, " BUILD Label: //src/query")
}

func TestCompletionWithBuildLabels2(t *testing.T) {
	ctx := context.Background()

	analyzer.State.Config.Parse.BuildFileName = []string{"completion2.build", "BUILD"}
	// Ensure relative label working correctly
	items, err := handler.getCompletionItemsList(ctx, completion2URI, lsp.Position{Line: 15, Character: 11})
	assert.Equal(t, nil, err)
	assert.Equal(t, 4, len(items))

	assert.True(t, itemInList(items, "my_binary"))
	assert.True(t, itemInList(items, "langserver_test"))
}

/***************************************
 *Tests for Builtin rules
 ***************************************/
func TestCompletionWithBuiltins(t *testing.T) {
	ctx := context.Background()

	items, err := handler.getCompletionItemsList(ctx, completionLiteralURI, lsp.Position{Line: 2, Character: 3})
	assert.Equal(t, nil, err)
	assert.True(t, itemInList(items, "go_library"))
	assert.True(t, itemInList(items, "go_binary"))
	assert.True(t, itemInList(items, "go_test"))
	assert.True(t, itemInList(items, "go_get"))
	for _, item := range items {
		if item.Label == "go_library" {
			expectedDetail := "(name:str, srcs:list, asm_srcs:list=None, hdrs:list=None, out:str=None, deps:list=[],\n" +
				"               visibility:list=None, test_only:bool&testonly=False, complete:bool=True, cover:bool=True,\n" +
				"               filter_srcs:bool=True)"
			assert.Equal(t, expectedDetail, item.Detail)
		}
	}

	items, err = handler.getCompletionItemsList(ctx, completionLiteralURI, lsp.Position{Line: 4, Character: 8})
	assert.Equal(t, nil, err)
	assert.Equal(t, 1, len(items))
	assert.True(t, itemInList(items, "python_library"))
}

/***************************************
 *Tests for Variables
 ***************************************/
func TestCompletionWithVars(t *testing.T) {
	ctx := context.Background()

	items, err := handler.getCompletionItemsList(ctx, completionLiteralURI, lsp.Position{Line: 7, Character: 3})
	assert.Equal(t, nil, err)
	assert.Equal(t, 1, len(items))
	assert.Equal(t, "my_str", items[0].Label)

	// Test none existing variable
	items, err = handler.getCompletionItemsList(ctx, completionLiteralURI, lsp.Position{Line: 6, Character: 10})
	assert.Equal(t, nil, err)
	assert.Equal(t, 0, len(items))
}

/***************************************
 *Tests for Statements
 ***************************************/
func TestCompletionWithIf(t *testing.T) {
	ctx := context.Background()

	// test within if statement
	items, err := handler.getCompletionItemsList(ctx, completionStmtURI, lsp.Position{Line: 7, Character: 7})
	assert.Equal(t, nil, err)
	assert.True(t, itemInList(items, "go_library"))

	items, err = handler.getCompletionItemsList(ctx, completionStmtURI, lsp.Position{Line: 1, Character: 21})
	assert.Equal(t, nil, err)
	assert.Equal(t, len(items), 2)
	assert.True(t, itemInList(items, "query"))
	assert.True(t, itemInList(items, "query:query"))

	// test at the beginning of if statement
	items, err = handler.getCompletionItemsList(ctx, completionStmtURI, lsp.Position{Line: 9, Character: 6})
	assert.Equal(t, nil, err)
	assert.True(t, itemInList(items, "python_library"))

	// test else statement
	items, err = handler.getCompletionItemsList(ctx, completionStmtURI, lsp.Position{Line: 4, Character: 17})
	assert.Equal(t, nil, err)
	assert.True(t, itemInList(items, "format"))
	assert.True(t, itemInList(items, "find"))
}

/***************************************
 *Tests for Subincludes
 ***************************************/
func TestCompletionSubinclude(t *testing.T) {
	ctx := context.Background()

	// test within if statement
	items, err := handler.getCompletionItemsList(ctx, subincludeURI, lsp.Position{Line: 6, Character: 4})
	assert.Equal(t, nil, err)
	assert.Equal(t, len(items), 1)
	assert.True(t, itemInList(items, "plz_e2e_test"))
}

/***************************************
 *Tests for argument name completion
 ***************************************/
func TestCompletionArgNameBuild(t *testing.T) {
	ctx := context.Background()
	analyzer.State.Config.Parse.BuildFileName = []string{"completion2.build", "BUILD"}

	// test completion for a "visibility" arg
	items, err := handler.getCompletionItemsList(ctx, completion2URI, lsp.Position{Line: 28, Character: 6})
	assert.Equal(t, nil, err)
	assert.Equal(t, 1, len(items))

	items, err = handler.getCompletionItemsList(ctx, completion2URI, lsp.Position{Line: 29, Character: 6})
	assert.Equal(t, nil, err)
	assert.Equal(t, 0, len(items))

	items, err = handler.getCompletionItemsList(ctx, completion2URI, lsp.Position{Line: 30, Character: 6})
	assert.Equal(t, nil, err)
	assert.Equal(t, 2, len(items))
	assert.True(t, itemInList(items, "filter_srcs="))
	assert.True(t, itemInList(items, "asm_srcs="))

}

func TestCompletionArgSrcLocalFiles(t *testing.T) {
	ctx := context.Background()
	analyzer.State.Config.Parse.BuildFileName = []string{"completion2.build", "BUILD"}

	items, err := handler.getCompletionItemsList(ctx, completion2URI, lsp.Position{Line: 26, Character: 11})
	assert.Equal(t, nil, err)
	assert.Equal(t, 3, len(items))
	assert.True(t, itemInList(items, "foo.go"))

	// test completion with non-existent srcs
	items, err = handler.getCompletionItemsList(ctx, completion2URI, lsp.Position{Line: 25, Character: 12})
	assert.Equal(t, nil, err)
	assert.Nil(t, items)

	// test completion with string without ','
	items, err = handler.getCompletionItemsList(ctx, completion2URI, lsp.Position{Line: 36, Character: 6})
	assert.Equal(t, nil, err)
	assert.Equal(t, 7, len(items))
	assert.True(t, itemInList(items, "foo.go"))

	items, err = handler.getCompletionItemsList(ctx, completion2URI, lsp.Position{Line: 42, Character: 15})
	assert.Equal(t, nil, err)
	assert.Equal(t, 2, len(items))
	assert.True(t, itemInList(items, "reformat.build"))
	assert.True(t, itemInList(items, "reformat2.build"))
}

/***************************************
 * Helpers
 ***************************************/
func itemInList(itemList []*lsp.CompletionItem, targetLabel string) bool {
	for _, item := range itemList {
		if item.Label == targetLabel {
			return true
		}
	}
	return false
}
