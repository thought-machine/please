package langserver

import (
	"context"
	"core"
	"os"
	"path"
	"strings"
	"testing"
	"tools/build_langserver/lsp"

	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	core.FindRepoRoot()
	retCode := m.Run()
	os.Exit(retCode)
}

var filePath = path.Join("tools/build_langserver/langserver/test_data/example.build")
var exampleBuildURI = lsp.DocumentURI("file://" + filePath)

var filePath2 = path.Join("tools/build_langserver/langserver/test_data/assignment.build")
var assignBuildURI = lsp.DocumentURI("file://" + filePath2)

var analyzer = newAnalyzer()

/***************************************
 *Tests for Build Definitions
 ***************************************/
func TestGetHoverContentOnBuildDefName(t *testing.T) {
	var ctx = context.Background()

	content, err := getHoverContent(ctx, analyzer, exampleBuildURI, lsp.Position{Line: 0, Character: 3})
	expected := analyzer.BuiltIns["go_library"].Header + "\n\n" + analyzer.BuiltIns["go_library"].Docstring

	assert.Equal(t, nil, err)
	assert.Equal(t, expected, content.Value)
}

func TestGetHoverContentOnArgument(t *testing.T) {
	var ctx = context.Background()

	// Test hovering over argument name
	content, err := getHoverContent(ctx, analyzer, exampleBuildURI, lsp.Position{Line: 7, Character: 7})
	assert.Equal(t, nil, err)
	assert.Equal(t, "deps required:false, type:list", content.Value)

	// Test hovering over argument name with nested call
	content, err = getHoverContent(ctx, analyzer, exampleBuildURI, lsp.Position{Line: 2, Character: 7})
	assert.Equal(t, nil, err)
	assert.Equal(t, "srcs required:true, type:list",
		content.Value)

	// Test hovering over argument content

	// when build label is a label for including all sub packages
	content, err = getHoverContent(ctx, analyzer, exampleBuildURI, lsp.Position{Line: 6, Character: 21})
	assert.Equal(t, nil, err)
	expected := "BuildLabel includes all subpackages in path: " +
		core.RepoRoot + "/tools/build_langserver"
	assert.Equal(t, expected, content.Value)

	// When build label is definitive, e.g. "//src/core" or "//src/core:core"
	content, err = getHoverContent(ctx, analyzer, exampleBuildURI, lsp.Position{Line: 6, Character: 57})
	expectedContent := []string{"go_library(", "    name = \"core\",", "    srcs = glob("}
	assert.Equal(t, nil, err)
	// Checking only the first 3 line
	splited := strings.Split(content.Value, "\n")
	assert.Equal(t, expectedContent, splited[:3])
}

func TestGetHoverContentOnNestedCall(t *testing.T) {
	var ctx = context.Background()

	// Test hovering over nested call on definition, e.g. glob(
	content, err := getHoverContent(ctx, analyzer, exampleBuildURI, lsp.Position{Line: 2, Character: 14})
	assert.Equal(t, nil, err)
	assert.Equal(t, "def glob(include:list, exclude:list&excludes=[], hidden:bool=False)",
		content.Value)

	// Test hovering over nested call on ending parenthese
	content, err = getHoverContent(ctx, analyzer, exampleBuildURI, lsp.Position{Line: 5, Character: 6})
	assert.Equal(t, nil, err)
	assert.Equal(t, "", content.Value)

	// Test hovering over argument assignment of nested call
	content, err = getHoverContent(ctx, analyzer, exampleBuildURI, lsp.Position{Line: 4, Character: 15})
	assert.Equal(t, nil, err)
	assert.Equal(t, "exclude required:false, type:list",
		content.Value)

	content, err = getHoverContent(ctx, analyzer, exampleBuildURI, lsp.Position{Line: 3, Character: 15})
	assert.Equal(t, nil, err)
	assert.Equal(t, "include required:true, type:list",
		content.Value)
}

func TestGetHoverContentOnEmptyContent(t *testing.T) {
	var ctx = context.Background()

	content, err := getHoverContent(ctx, analyzer, exampleBuildURI, lsp.Position{Line: 1, Character: 3})
	assert.Equal(t, nil, err)
	assert.Equal(t, "", content.Value)

	content, err = getHoverContent(ctx, analyzer, exampleBuildURI, lsp.Position{Line: 2, Character: 30})
	assert.Equal(t, nil, err)
	assert.Equal(t, "", content.Value)
}

func TestGetHoverContentOnBuildLabels(t *testing.T) {
	var ctx = context.Background()

	// Test hovering over buildlabels
	content, err := getHoverContent(ctx, analyzer, exampleBuildURI, lsp.Position{Line: 13, Character: 15})
	expected := "go_get(\n" +
		"    name = \"jsonrpc2\",\n" +
		"    get = \"github.com/sourcegraph/jsonrpc2\",\n" +
		"    revision = \"549eb959f029d014d623104d40ab966d159a92de\",\n" +
		")"
	assert.Equal(t, nil, err)
	assert.Equal(t, expected, content.Value)
}

func TestGetHoverContentOnNoneBuildLabelString(t *testing.T) {
	var ctx = context.Background()

	content, err := getHoverContent(ctx, analyzer, exampleBuildURI, lsp.Position{Line: 20, Character: 18})
	assert.Equal(t, nil, err)
	assert.Equal(t, "", content.Value)
}

func TestGetHoverContentOnArgumentWithProperty(t *testing.T) {
	var ctx = context.Background()

	// Hover on argument name
	content, err := getHoverContent(ctx, analyzer, exampleBuildURI, lsp.Position{Line: 34, Character: 6})
	assert.Equal(t, nil, err)
	assert.Equal(t, "name required:true, type:str", content.Value)

	// Hover on property name
	content, err = getHoverContent(ctx, analyzer, exampleBuildURI, lsp.Position{Line: 34, Character: 20})
	assert.Equal(t, nil, err)
	assert.Equal(t, "", content.Value)

	// Hover on property call
	content, err = getHoverContent(ctx, analyzer, exampleBuildURI, lsp.Position{Line: 34, Character: 34})
	assert.Equal(t, nil, err)
	assert.Equal(t, "str.format()", content.Value)
}


func TestGetHoverContentArgOnTheSameLine(t *testing.T) {
	var ctx = context.Background()

	content, err := getHoverContent(ctx, analyzer, exampleBuildURI, lsp.Position{Line: 42, Character: 17})
	assert.Equal(t, nil, err)
	assert.Equal(t, "name required:true, type:str", content.Value)
}

/***************************************
 *Tests for Variable assignments
 ***************************************/
func TestGetHoverContentOnPropertyAssignment(t *testing.T) {
	var ctx = context.Background()

	//Hover on assignment with properties
	content, err := getHoverContent(ctx, analyzer, assignBuildURI, lsp.Position{Line: 0, Character: 30})
	assert.Equal(t, nil, err)
	assert.Equal(t, "str.format()", content.Value)

	content, err = getHoverContent(ctx, analyzer, assignBuildURI, lsp.Position{Line: 2, Character: 18})
	assert.Equal(t, nil, err)
	assert.Equal(t, "str.replace(old:str, new:str)", content.Value)

	//Hover on argument of assignment property
	content, err = getHoverContent(ctx, analyzer, assignBuildURI, lsp.Position{Line: 0, Character: 36})
	assert.Equal(t, nil, err)
	assert.Equal(t, "", content.Value)

	content, err = getHoverContent(ctx, analyzer, assignBuildURI, lsp.Position{Line: 2, Character: 25})
	assert.Equal(t, nil, err)
	assert.Equal(t, "old required:true, type:str", content.Value)

	content, err = getHoverContent(ctx, analyzer, assignBuildURI, lsp.Position{Line: 2, Character: 29})
	assert.Equal(t, nil, err)
	assert.Equal(t, "new required:true, type:str", content.Value)

	// Hover on assignment with function call
	content, err = getHoverContent(ctx, analyzer, assignBuildURI, lsp.Position{Line: 6, Character: 18})
	assert.Equal(t, nil, err)
	assert.Equal(t, "def subinclude(target:str, hash:str=None)", content.Value)
}

func TestGetHoverContentOnUnaryAssignment(t *testing.T) {
	var ctx = context.Background()

	// Hover on assignment with unary op
	content, err := getHoverContent(ctx, analyzer, assignBuildURI, lsp.Position{Line: 4, Character: 13})
	assert.Equal(t, nil, err)
	assert.Equal(t, "def len(obj)", content.Value)
}

func TestGetHoverContentOnListAssignment(t *testing.T) {
	var ctx = context.Background()

	coreContent := []string{"go_library(", "    name = \"core\",", "    srcs = glob("}
	fsContent := []string{"go_library(", "    name = \"fs\",", "    srcs = ["}

	// Hover on assignment with Multiline list
	content, err := getHoverContent(ctx, analyzer, assignBuildURI, lsp.Position{Line: 10, Character: 11})
	assert.Equal(t, nil, err)
	splited := strings.Split(content.Value, "\n")
	assert.Equal(t, coreContent, splited[:3])

	// Hover on assignment with single line list
	content, err = getHoverContent(ctx, analyzer, assignBuildURI, lsp.Position{Line: 19, Character: 26})
	assert.Equal(t, nil, err)
	splited = strings.Split(content.Value, "\n")
	assert.Equal(t, fsContent, splited[:3])

	content, err = getHoverContent(ctx, analyzer, assignBuildURI, lsp.Position{Line: 19, Character: 39})
	assert.Equal(t, nil, err)
	splited = strings.Split(content.Value, "\n")
	assert.Equal(t, coreContent, splited[:3])

	// Hover on empty space assignment
	content, err = getHoverContent(ctx, analyzer, assignBuildURI, lsp.Position{Line: 12, Character: 52})
	assert.Equal(t, nil, err)
	assert.Equal(t, "", content.Value)
}

func TestGetHoverContentOnIfAssignment(t *testing.T) {
	var ctx = context.Background()

	// Hover on if statement assignment empty
	content, err := getHoverContent(ctx, analyzer, assignBuildURI, lsp.Position{Line: 21, Character: 33})
	assert.Equal(t, nil, err)
	assert.Equal(t, "", content.Value)

	// Hover on else statement assignment
	content, err = getHoverContent(ctx, analyzer, assignBuildURI, lsp.Position{Line: 21, Character: 40})
	assert.Equal(t, nil, err)
	assert.Equal(t, "def subinclude(target:str, hash:str=None)",
		content.Value)
}


/***************************************
 *Tests for Variable augAssign
 ***************************************/

func TestGetHoverContentAugAssign(t *testing.T) {
	var ctx = context.Background()

	content, err := getHoverContent(ctx, analyzer, assignBuildURI, lsp.Position{Line: 23, Character: 14})
	assert.Equal(t, nil, err)
	assert.Equal(t, "def len(obj)", content.Value)
}