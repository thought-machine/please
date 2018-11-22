package langserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"tools/build_langserver/lsp"
)

func TestGetFormatEdits(t *testing.T) {
	analyzer.State.Config.Parse.BuildFileName = append(analyzer.State.Config.Parse.BuildFileName,
		[]string{"reformat.build", "example.build"}...)
	edits, err := handler.getFormatEdits(reformatURI)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(edits))

	expected := `python_library(
    name = "test",
    srcs = ["utils_test.go"],
)
`
	assert.Equal(t, expected, edits[0].NewText)

	edits, err = handler.getFormatEdits(exampleBuildURI)
	assert.NoError(t, err)
	expected = `blah = "blah"` + "\n"
	assert.Equal(t, expected, edits[len(edits)-1].NewText)

	expected = `
subinclude({
    "foo": "bar",
    "blah": 1,
})
`
	t.Log(edits[7].Range)
	assert.Equal(t, expected, edits[7].NewText)
}

func TestGetEdits(t *testing.T) {
	edits := getEdits(`blah="hello"`, `blah = "blah"`+"\n")
	expectedRange := lsp.Range{
		Start: lsp.Position{
			Line:      0,
			Character: 0,
		},
		End: lsp.Position{
			Line:      1,
			Character: 0,
		},
	}

	assert.Equal(t, expectedRange, edits[0].Range)

	after := `subinclude({
    "foo": "bar",
    "blah": 1,
})
`
	edits = getEdits(`subinclude({"foo": "bar", "blah":1})`, after)
	assert.Equal(t, expectedRange, edits[0].Range)
	assert.Equal(t, after, edits[0].NewText)
	t.Log(edits[0].NewText)
}
