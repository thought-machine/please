package langserver

import (
	"context"
	"github.com/thought-machine/please/src/core"
	"strings"
	"testing"

	"github.com/thought-machine/please/tools/build_langserver/lsp"

	"github.com/stretchr/testify/assert"
)

/***************************************
 *Tests for Build Definitions
 ***************************************/
func TestGetHoverContentOnBuildDefName(t *testing.T) {
	var ctx = context.Background()

	content, err := handler.getHoverContent(ctx, exampleBuildURI, lsp.Position{Line: 0, Character: 3})
	expected := analyzer.BuiltIns["go_library"].Header + "\n\n" + analyzer.BuiltIns["go_library"].Docstring

	assert.Equal(t, nil, err)
	assert.Equal(t, expected, content)
}

func TestGetHoverContentOnArgument(t *testing.T) {
	var ctx = context.Background()

	// Test hovering over argument name
	content, err := handler.getHoverContent(ctx, exampleBuildURI, lsp.Position{Line: 7, Character: 7})
	assert.Equal(t, nil, err)
	assert.Equal(t, "deps required:false, type:list", content)

	// Test hovering over argument name with nested call
	content, err = handler.getHoverContent(ctx, exampleBuildURI, lsp.Position{Line: 2, Character: 7})
	assert.Equal(t, nil, err)
	assert.Equal(t, "srcs required:true, type:list",
		content)

	// Test hovering over argument content

	// when build label is a label for including all sub packages
	content, err = handler.getHoverContent(ctx, exampleBuildURI, lsp.Position{Line: 6, Character: 21})
	assert.Equal(t, nil, err)
	expected := "BuildLabel includes all subpackages in path: " +
		core.RepoRoot + "/tools/build_langserver"
	assert.Equal(t, expected, content)

	// When build label is definitive, e.g. "//src/core" or "//src/core:core"
	content, err = handler.getHoverContent(ctx, exampleBuildURI, lsp.Position{Line: 6, Character: 57})
	expectedContent := []string{"go_library(", "    name = \"core\",", "    srcs = glob("}
	assert.Equal(t, nil, err)
	// Checking only the first 3 line
	splited := strings.Split(content, "\n")
	assert.Equal(t, expectedContent, splited[:3])
}

func TestGetHoverContentOnNestedCall(t *testing.T) {
	var ctx = context.Background()

	// Test hovering over nested call on definition, e.g. glob(
	content, err := handler.getHoverContent(ctx, exampleBuildURI, lsp.Position{Line: 2, Character: 14})
	assert.Equal(t, nil, err)
	assert.Equal(t, "def glob(include:list, exclude:list&excludes=[], hidden:bool=False)",
		content)

	// Test hovering over nested call on ending parenthese
	content, err = handler.getHoverContent(ctx, exampleBuildURI, lsp.Position{Line: 5, Character: 6})
	assert.Equal(t, nil, err)
	assert.Equal(t, "", content)

	// Test hovering over argument assignment of nested call
	content, err = handler.getHoverContent(ctx, exampleBuildURI, lsp.Position{Line: 4, Character: 15})
	assert.Equal(t, nil, err)
	assert.Equal(t, "exclude required:false, type:list",
		content)

	content, err = handler.getHoverContent(ctx, exampleBuildURI, lsp.Position{Line: 3, Character: 15})
	assert.Equal(t, nil, err)
	assert.Equal(t, "include required:true, type:list",
		content)
}

func TestGetHoverContentOnEmptyContent(t *testing.T) {
	var ctx = context.Background()

	content, err := handler.getHoverContent(ctx, exampleBuildURI, lsp.Position{Line: 1, Character: 3})
	assert.Equal(t, nil, err)
	assert.Equal(t, "", content)

	content, err = handler.getHoverContent(ctx, exampleBuildURI, lsp.Position{Line: 2, Character: 30})
	assert.Equal(t, nil, err)
	assert.Equal(t, "", content)
}

func TestGetHoverContentOnBuildLabels(t *testing.T) {
	var ctx = context.Background()

	// Test hovering over buildlabels
	content, err := handler.getHoverContent(ctx, exampleBuildURI, lsp.Position{Line: 13, Character: 15})
	expected := "go_get(\n" +
		"    name = \"jsonrpc2\",\n" +
		"    get = \"github.com/sourcegraph/jsonrpc2\",\n" +
		"    revision = \"549eb959f029d014d623104d40ab966d159a92de\",\n" +
		")"
	assert.Equal(t, nil, err)
	assert.Equal(t, expected, content)
}

func TestGetHoverContentOnNoneBuildLabelString(t *testing.T) {
	var ctx = context.Background()

	content, err := handler.getHoverContent(ctx, exampleBuildURI, lsp.Position{Line: 20, Character: 18})
	assert.Equal(t, nil, err)
	assert.Equal(t, "", content)
}

func TestGetHoverContentOnArgumentWithProperty(t *testing.T) {
	var ctx = context.Background()

	// Hover on argument name
	content, err := handler.getHoverContent(ctx, exampleBuildURI, lsp.Position{Line: 35, Character: 6})
	assert.Equal(t, nil, err)
	assert.Equal(t, "name required:true, type:str", content)

	// Hover on property name
	content, err = handler.getHoverContent(ctx, exampleBuildURI, lsp.Position{Line: 35, Character: 20})
	assert.Equal(t, nil, err)
	assert.Equal(t, "", content)

	// Hover on property call
	content, err = handler.getHoverContent(ctx, exampleBuildURI, lsp.Position{Line: 35, Character: 34})
	assert.Equal(t, nil, err)
	assert.Equal(t, "str.format()", content)
}

func TestGetHoverContentArgOnTheSameLine(t *testing.T) {
	var ctx = context.Background()

	content, err := handler.getHoverContent(ctx, exampleBuildURI, lsp.Position{Line: 43, Character: 17})
	assert.Equal(t, nil, err)
	assert.Equal(t, "name required:true, type:str", content)
}

/***************************************
*Tests for Variable assignments
***************************************/
func TestGetHoverContentOnPropertyAssignment(t *testing.T) {
	var ctx = context.Background()

	//Hover on assignment with properties
	content, err := handler.getHoverContent(ctx, assignBuildURI, lsp.Position{Line: 0, Character: 30})
	assert.Equal(t, nil, err)
	assert.Equal(t, "str.format()", content)

	content, err = handler.getHoverContent(ctx, assignBuildURI, lsp.Position{Line: 2, Character: 18})
	assert.Equal(t, nil, err)
	assert.Equal(t, "str.replace(old:str, new:str)", content)

	//Hover on argument of assignment property
	content, err = handler.getHoverContent(ctx, assignBuildURI, lsp.Position{Line: 0, Character: 36})
	assert.Equal(t, nil, err)
	assert.Equal(t, "", content)

	content, err = handler.getHoverContent(ctx, assignBuildURI, lsp.Position{Line: 2, Character: 25})
	assert.Equal(t, nil, err)
	assert.Equal(t, "old required:true, type:str", content)

	content, err = handler.getHoverContent(ctx, assignBuildURI, lsp.Position{Line: 2, Character: 29})
	assert.Equal(t, nil, err)
	assert.Equal(t, "new required:true, type:str", content)

	// Hover on assignment with function call
	content, err = handler.getHoverContent(ctx, assignBuildURI, lsp.Position{Line: 6, Character: 18})
	assert.Equal(t, nil, err)
	assert.Equal(t, "def subinclude(target:str, hash:str=None)", content)
}

func TestGetHoverAssignmentBuildLabel(t *testing.T) {
	var ctx = context.Background()

	content, err := handler.getHoverContent(ctx, assignBuildURI, lsp.Position{Line: 25, Character: 13})
	expected := []string{"go_library(", "    name = \"fs\",", "    srcs = ["}
	t.Log(content)
	assert.Equal(t, nil, err)
	assert.Equal(t, strings.Split(content, "\n")[:3], expected)
}

func TestGetHoverContentOnUnaryAssignment(t *testing.T) {
	var ctx = context.Background()

	// Hover on assignment with unary op
	content, err := handler.getHoverContent(ctx, assignBuildURI, lsp.Position{Line: 4, Character: 13})
	assert.Equal(t, nil, err)
	assert.Equal(t, "def len(obj:list|dict|str)", content)
}

func TestGetHoverContentOnListAssignment(t *testing.T) {
	var ctx = context.Background()

	coreContent := []string{"go_library(", "    name = \"core\",", "    srcs = glob("}
	fsContent := []string{"go_library(", "    name = \"fs\",", "    srcs = ["}

	// Hover on assignment with Multiline list
	content, err := handler.getHoverContent(ctx, assignBuildURI, lsp.Position{Line: 10, Character: 11})
	assert.Equal(t, nil, err)
	splited := strings.Split(content, "\n")
	assert.Equal(t, coreContent, splited[:3])

	// Hover on assignment with single line list
	content, err = handler.getHoverContent(ctx, assignBuildURI, lsp.Position{Line: 19, Character: 26})
	assert.Equal(t, nil, err)
	splited = strings.Split(content, "\n")
	assert.Equal(t, fsContent, splited[:3])

	content, err = handler.getHoverContent(ctx, assignBuildURI, lsp.Position{Line: 19, Character: 39})
	assert.Equal(t, nil, err)
	splited = strings.Split(content, "\n")
	assert.Equal(t, coreContent, splited[:3])

	// Hover on empty space assignment
	content, err = handler.getHoverContent(ctx, assignBuildURI, lsp.Position{Line: 12, Character: 52})
	assert.Equal(t, nil, err)
	assert.Equal(t, "", content)
}

func TestGetHoverContentOnIfAssignment(t *testing.T) {
	var ctx = context.Background()

	// Hover on if statement assignment empty
	content, err := handler.getHoverContent(ctx, assignBuildURI, lsp.Position{Line: 21, Character: 33})
	assert.Equal(t, nil, err)
	assert.Equal(t, "", content)

	// Hover on else statement assignment
	content, err = handler.getHoverContent(ctx, assignBuildURI, lsp.Position{Line: 21, Character: 40})
	assert.Equal(t, nil, err)
	assert.Equal(t, "def subinclude(target:str, hash:str=None)",
		content)
}

/***************************************
*Tests for Variable augAssign
***************************************/
func TestGetHoverContentAugAssign(t *testing.T) {
	var ctx = context.Background()

	// Hover on assignment with call
	content, err := handler.getHoverContent(ctx, assignBuildURI, lsp.Position{Line: 23, Character: 14})
	assert.Equal(t, nil, err)
	assert.Equal(t, "def len(obj:list|dict|str)", content)

	// Hover on empty space
	content, err = handler.getHoverContent(ctx, assignBuildURI, lsp.Position{Line: 23, Character: 56})
	assert.Equal(t, nil, err)
	assert.Equal(t, "", content)
}

/***************************************
*Tests for property
***************************************/
func TestGetHoverContentProperty(t *testing.T) {
	var ctx = context.Background()

	// Hover on CONFIG property
	content, err := handler.getHoverContent(ctx, propURI, lsp.Position{Line: 0, Character: 4})
	assert.Equal(t, nil, err)
	assert.Equal(t, "", content)

	content, err = handler.getHoverContent(ctx, propURI, lsp.Position{Line: 2, Character: 4})
	assert.Equal(t, nil, err)
	assert.Equal(t, "str.replace(old:str, new:str)", content)
}

/***************************************
*Tests for ast expression statements
***************************************/
func TestGetHoverContentAst(t *testing.T) {
	var ctx = context.Background()

	// Test for statement
	content, err := handler.getHoverContent(ctx, miscURI, lsp.Position{Line: 0, Character: 11})
	assert.Equal(t, nil, err)
	assert.Equal(t, "def len(obj:list|dict|str)", content)

	// Test inner For statement
	content, err = handler.getHoverContent(ctx, miscURI, lsp.Position{Line: 1, Character: 11})
	assert.Equal(t, nil, err)
	assert.Equal(t, "str.replace(old:str, new:str)", content)

	// Test Assert For statement
	content, err = handler.getHoverContent(ctx, miscURI, lsp.Position{Line: 2, Character: 17})
	assert.Equal(t, nil, err)
	assert.Equal(t, "def subinclude(target:str, hash:str=None)", content)
}

func TestGetHoverContentAst2(t *testing.T) {
	var ctx = context.Background()

	// Test if statement
	content, err := handler.getHoverContent(ctx, miscURI, lsp.Position{Line: 4, Character: 7})
	assert.Equal(t, nil, err)
	assert.Equal(t, "str.find(needle:str)", content)

	// Test elif statement
	content, err = handler.getHoverContent(ctx, miscURI, lsp.Position{Line: 6, Character: 8})
	assert.Equal(t, nil, err)
	assert.Equal(t, "str.count(needle:str)", content)

	// Test return statement
	content, err = handler.getHoverContent(ctx, miscURI, lsp.Position{Line: 5, Character: 17})
	assert.Equal(t, nil, err)
	assert.Equal(t, "def subinclude(target:str, hash:str=None)", content)

	content, err = handler.getHoverContent(ctx, miscURI, lsp.Position{Line: 9, Character: 17})
	assert.Equal(t, nil, err)
	assert.Equal(t, "str.lower()", content)
}

func TestGetHoverContentSubinclude(t *testing.T) {
	var ctx = context.Background()

	// hover on subincluded rule name
	content, err := handler.getHoverContent(ctx, subincludeURI, lsp.Position{Line: 2, Character: 8})
	assert.Equal(t, nil, err)

	expected := "def plz_e2e_test(name, cmd, pre_cmd=None, expected_output=None, expected_failure=False,\n" +
		"                 expect_output_contains=None, expect_output_doesnt_contain=None,\n" +
		"                 deps=None, data=[], labels=None, sandbox=None,\n" +
		"                 expect_file_exists=None, expect_file_doesnt_exist=None)"
	assert.Equal(t, expected, content)

	// hover on subincluded arg
	content, err = handler.getHoverContent(ctx, subincludeURI, lsp.Position{Line: 3, Character: 7})
	assert.Equal(t, nil, err)
	assert.Equal(t, "name required:true", content)
}
