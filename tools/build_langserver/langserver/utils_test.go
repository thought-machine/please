package langserver

import (
	"context"
	"github.com/thought-machine/please/src/core"
	"fmt"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/tools/build_langserver/lsp"
)

func TestIsURL(t *testing.T) {
	currentFile, err := getFileinCwd("utils_test.go")
	assert.Equal(t, err, nil)

	assert.False(t, IsURL(currentFile))

	documentURI := "file://" + currentFile
	assert.True(t, IsURL(documentURI))
}

func TestGetPathFromURL(t *testing.T) {
	currentFile, err := getFileinCwd("utils_test.go")
	assert.Equal(t, err, nil)
	documentURI := lsp.DocumentURI("file://" + currentFile)

	// Test GetPathFromURL when documentURI passed in is a URI
	p, err := GetPathFromURL(documentURI, "file")
	assert.Equal(t, err, nil)
	assert.Equal(t, p, string(currentFile))

	// Test GetPathFromURL when documentURI passed in is a file path
	p2, err := GetPathFromURL(currentFile, "File")
	assert.Equal(t, err, nil)
	assert.Equal(t, p2, string(currentFile))

}

func TestGetPathFromURLFail(t *testing.T) {
	// Test invalid file fail with Bogus file
	bogusFile, err := getFileinCwd("BLAH.blah")
	assert.Equal(t, err, nil)

	p, err := GetPathFromURL(bogusFile, "file")
	assert.Equal(t, p, "")
	assert.Error(t, err)

	// Test invalid file not in repo root
	p2, err := GetPathFromURL(lsp.DocumentURI("/tmp"), "path")
	assert.Equal(t, p2, "")
	assert.Error(t, err)

	// Test invalid pathtype
	currentFile, err := getFileinCwd("utils_test.go")
	assert.Equal(t, err, nil)
	p3, err := GetPathFromURL(currentFile, "dir")
	assert.Equal(t, p3, "")
	assert.Error(t, err)
}

func TestLocalFilesFromURI(t *testing.T) {
	exampleBuildURI := lsp.DocumentURI("file://tools/build_langserver/langserver/test_data/example.build")
	files, err := LocalFilesFromURI(exampleBuildURI)
	assert.NoError(t, err)
	assert.True(t, StringInSlice(files, "foo.go"))
	assert.True(t, !StringInSlice(files, "example.go"))
}

func TestPackageLabelFromURI(t *testing.T) {
	filePath := path.Join(core.RepoRoot, "tools/build_langserver/langserver/BUILD")
	uri := lsp.DocumentURI("file://" + filePath)
	label, err := PackageLabelFromURI(uri)

	assert.Equal(t, err, nil)
	assert.Equal(t, "//tools/build_langserver/langserver", label)
}

func TestEnsureURL(t *testing.T) {
	currentFile, err := getFileinCwd("utils_test.go")
	assert.Equal(t, err, nil)

	uri, err := EnsureURL(currentFile, "file")
	assert.Equal(t, err, nil)
	assert.Equal(t, uri, lsp.DocumentURI("file://"+string(currentFile)))
}

func TestReadFile(t *testing.T) {
	ctx := context.Background()
	filepath := path.Join(core.RepoRoot, "tools/build_langserver/langserver/test_data/example.build")
	uri := lsp.DocumentURI("file://" + filepath)

	// Test ReadFile() with filepath as uri
	content, err := ReadFile(ctx, lsp.DocumentURI(filepath))
	// Type checking
	assert.Equal(t, fmt.Sprintf("%T", content), "[]string")
	assert.Equal(t, err, nil)
	// Content check
	assert.Equal(t, content[0], "go_library(")
	assert.Equal(t, strings.TrimSpace(content[1]), "name = \"langserver\",")

	// Test ReadFile() with uri as uri
	content1, err := ReadFile(ctx, uri)
	// Type checking
	assert.Equal(t, fmt.Sprintf("%T", content), "[]string")
	assert.Equal(t, err, nil)
	// Content check
	assert.Equal(t, content1[0], "go_library(")
	assert.Equal(t, strings.TrimSpace(content1[1]), "name = \"langserver\",")

}

func TestReadFileCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	filepath := path.Join(core.RepoRoot, "tools/build_langserver/langserver/test_data/example.build")

	cancel()
	content, err := ReadFile(ctx, lsp.DocumentURI(filepath))
	assert.Equal(t, content, []string(nil))
	assert.Equal(t, err, nil)
}

func TestGetLineContent(t *testing.T) {
	ctx := context.Background()
	filepath := path.Join(core.RepoRoot, "tools/build_langserver/langserver/test_data/example.build")
	pos := lsp.Position{
		Line:      0,
		Character: 0,
	}

	line, err := GetLineContent(ctx, lsp.DocumentURI(filepath), pos)
	assert.Equal(t, err, nil)
	assert.Equal(t, strings.TrimSpace(line[0]), "go_library(")
}

func TestTrimQoutes(t *testing.T) {
	trimed := TrimQuotes("\"blah\"")
	assert.Equal(t, trimed, "blah")

	trimed = TrimQuotes(`"//src/core",`)
	assert.Equal(t, "//src/core", trimed)

	trimed = TrimQuotes(`     "//src/core",`)
	assert.Equal(t, "//src/core", trimed)

	trimed = TrimQuotes("'blah")
	assert.Equal(t, "blah", trimed)

	// this is to make sure our regex works,
	// so it doesn't just match anything with a build label
	trimed = TrimQuotes(`visibility = ["//tools/build_langserver/...", "//src/core"]`)
	assert.Equal(t, `visibility = ["//tools/build_langserver/...", "//src/core"]`, trimed)

	trimed = TrimQuotes(`"//src/core`)
	assert.Equal(t, "//src/core", trimed)
}

func TestExtractStringVal(t *testing.T) {
	assert.Equal(t, "blah", ExtractStrTail(`    srcs=["blah"`))
	assert.Equal(t, "foo", ExtractStrTail(`    "foo"`))
	assert.Equal(t, "", ExtractStrTail(`"blah", srcs=`))
}

func TestLooksLikeAttribute(t *testing.T) {
	assert.True(t, LooksLikeAttribute("CONFIG."))
	assert.True(t, LooksLikeAttribute("CONFIG.J"))
	assert.True(t, LooksLikeAttribute("		CONFIG.J"))
	assert.True(t, LooksLikeAttribute("		\"blah\".for"))
	assert.True(t, LooksLikeAttribute("		\"blah\"."))
	assert.True(t, LooksLikeAttribute("	mystr = \"{time}-{message}\".fo"))

	assert.False(t, LooksLikeAttribute("		func(ca"))
	assert.False(t, LooksLikeAttribute("call_assign = subinclude(\"//build_defs:fpm\")"))
	assert.False(t, LooksLikeAttribute("     \"//tools/build_langserver/lsp\","))
	assert.False(t, LooksLikeAttribute("    augassign += len(replace_str)"))
}

func TestLooksLikeCONFIGAttr(t *testing.T) {
	assert.True(t, LooksLikeCONFIGAttr("CONFIG."))
	assert.True(t, LooksLikeCONFIGAttr("CONFIG.J"))
	assert.True(t, LooksLikeCONFIGAttr("		CONFIG.J"))
	assert.True(t, LooksLikeCONFIGAttr("		CONFIG.9"))

	assert.False(t, LooksLikeCONFIGAttr("CONFIG = BLAH."))
	assert.False(t, LooksLikeCONFIGAttr("CONFIG.BLAH = \"hello\""))
	assert.False(t, LooksLikeCONFIGAttr("func(ca"))
}

func TestLooksLikeStringAttr(t *testing.T) {
	assert.True(t, LooksLikeStringAttr("\"{time}-{message}\"."))
	assert.True(t, LooksLikeStringAttr("'{time}-{message}'."))
	assert.True(t, LooksLikeStringAttr("	'message'."))
	assert.True(t, LooksLikeStringAttr("blah = 'message'."))

	// Test fail when quotes style don't match
	assert.False(t, LooksLikeStringAttr("\"foo'."))
}

func TestLooksLikeDictAttr(t *testing.T) {
	assert.True(t, LooksLikeDictAttr("{\"foo\":2, \"bar\":\"baz\"}."))
	assert.True(t, LooksLikeDictAttr("{\"foo\":2}."))
	assert.True(t, LooksLikeDictAttr("{\"foo\":2}.k"))
	assert.True(t, LooksLikeDictAttr("{\"foo\":2}.keys"))

	// Ensure completed call does not get triggered
	assert.False(t, LooksLikeDictAttr("{\"foo\":2}.keys()"))
}

func TestExtractBuildLabel(t *testing.T) {
	label := ExtractBuildLabel(`target = "//src/cache/blah:hello`)
	assert.Equal(t, "//src/cache/blah:hello", label)
	t.Log(label)

	label = ExtractBuildLabel(`target = "//src/cache/blah:hello"`)
	assert.Equal(t, "//src/cache/blah:hello", label)

	label = ExtractBuildLabel(`		"//src/cache:`)
	assert.Equal(t, "//src/cache:", label)

	label = ExtractBuildLabel(`		"//src/cache/blah`)
	assert.Equal(t, "//src/cache/blah", label)

	label = ExtractBuildLabel(`		"//src/cache/blah/`)
	assert.Equal(t, "//src/cache/blah/", label)

	// no match
	label = ExtractBuildLabel("blah")
	assert.Equal(t, "", label)

	label = ExtractBuildLabel(`"//src/cache/blah//`)
	assert.Equal(t, "", label)

	label = ExtractBuildLabel(`"//src/cache/blah/:`)
	assert.Equal(t, "", label)

	label = ExtractBuildLabel(`"//src/ca`)
	assert.Equal(t, "//src/ca", label)
}

func TestExtractLiteral(t *testing.T) {
	lit := ExtractLiteral(`blah = "go_librar`)
	assert.Equal(t, "", lit)

	lit = ExtractLiteral(`blah = go_librar`)
	assert.Equal(t, "go_librar", lit)

	lit = ExtractLiteral(`go_librar`)
	assert.Equal(t, "go_librar", lit)

	lit = ExtractLiteral(`blah = " = go_librar`)
	assert.Equal(t, "", lit)

	lit = ExtractLiteral(`blah = "hello", hi = go_lib`)
	assert.Equal(t, "go_lib", lit)

	lit = ExtractLiteral(`blah = 'hello', hi = go_lib`)
	assert.Equal(t, "go_lib", lit)

	lit = ExtractLiteral(`"blah = 'hello, hi = go_lib`)
	assert.Equal(t, "", lit)

	// Tests for extracting attribute literals
	lit = ExtractLiteral(`blah.form`)
	assert.Equal(t, "blah.form", lit)

	lit = ExtractLiteral(`hello = blah.form`)
	assert.Equal(t, "blah.form", lit)

	lit = ExtractLiteral(`hello = "blah.form`)
	assert.Equal(t, "", lit)

	lit = ExtractLiteral(`blah = 'hello', hi = blah.form`)
	assert.Equal(t, "blah.form", lit)

	assert.Equal(t, ".format", ExtractLiteral(`"blah".format`))
}

/*
 * Utilities function for tests in this file
 */
func getFileinCwd(name string) (lsp.DocumentURI, error) {
	core.FindRepoRoot()
	filePath := path.Join(core.RepoRoot, "tools/build_langserver/langserver/"+name)

	return lsp.DocumentURI(filePath), nil
}
