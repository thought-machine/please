package langserver

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetFormatEdits(t *testing.T) {
	analyzer.State.Config.Parse.BuildFileName = append(analyzer.State.Config.Parse.BuildFileName,
		[]string{"reformat.build", "example.build"}...)
	edits, err := handler.getFormatEdits(reformatURI)
	assert.NoError(t, err)
	assert.Equal(t, 5, len(edits))

	assert.Equal(t, "python_library(\n", edits[0].NewText)
	assert.Equal(t, `    name = "test",`+"\n", edits[1].NewText)
	assert.Equal(t, `    srcs = ["utils_test.go"],`+"\n", edits[2].NewText)
	assert.Equal(t, ")\n", edits[3].NewText)
	assert.Equal(t, "", edits[4].NewText)
	for _, edit := range edits {
		t.Log(edit.NewText)
		t.Log(edit.Range)
	}

	edits, err = handler.getFormatEdits(exampleBuildURI)
	assert.NoError(t, err)
	expected := `blah = "blah"` + "\n"
	assert.Equal(t, expected, edits[len(edits)-2].NewText)
}

func TestGetFormatEdits2(t *testing.T) {
	analyzer.State.Config.Parse.BuildFileName = append(analyzer.State.Config.Parse.BuildFileName,
		[]string{"reformat2.build"}...)
	edits, err := handler.getFormatEdits(reformat2URI)
	assert.NoError(t, err)
	assert.Equal(t, 14, len(edits))

	assert.Equal(t, "    srcs = [\n", edits[0].NewText)
	assert.Equal(t, `        "handler_test.go",`+"\n", edits[1].NewText)
	assert.Equal(t, `        "analyzer_test.go",`+"\n", edits[2].NewText)
	assert.Equal(t, `        "hover_test.go",`+"\n", edits[3].NewText)
	assert.Equal(t, "    ],\n", edits[4].NewText)
}

func TestGetEdits(t *testing.T) {
	// test cases for replacement
	edits := getEdits(`blah="hello"`, `blah = "blah"`+"\n")
	assert.Equal(t, 2, len(edits))
	assert.Equal(t, 0, edits[0].Range.Start.Line, edits[0].Range.End.Line)
	assert.Equal(t, 0, edits[0].Range.Start.Character)
	assert.Equal(t, 11, edits[0].Range.End.Character)

	after := `subinclude({
    "foo": "bar",
    "blah": 1,
})
`
	edits = getEdits(`subinclude({"foo": "bar", "blah":1})`, after)
	assert.Equal(t, "subinclude({\n", edits[0].NewText)
	assert.Equal(t, 35, edits[0].Range.End.Character)

	assert.Equal(t, `    "foo": "bar",`+"\n", edits[1].NewText)
	assert.Equal(t, 1, edits[1].Range.End.Line, edits[1].Range.Start.Line)

	assert.Equal(t, `    "blah": 1,`+"\n", edits[2].NewText)
	assert.Equal(t, 2, edits[2].Range.End.Line, edits[2].Range.Start.Line)

	assert.Equal(t, `    "blah": 1,`+"\n", edits[2].NewText)
	assert.Equal(t, 2, edits[2].Range.End.Line, edits[2].Range.Start.Line)

	// test cases for deletion
	edits = getEdits(`blah="hello"`, "")
	assert.Equal(t, 0, edits[0].Range.End.Line, edits[0].Range.Start.Line)
	assert.Equal(t, 11, edits[0].Range.End.Character)
	assert.Equal(t, "", edits[0].NewText)

	edits = getEdits(`blah="hello"`+"\n", "")
	assert.Equal(t, 0, edits[0].Range.Start.Line)
	assert.Equal(t, 1, edits[0].Range.End.Line)
	assert.Equal(t, 0, edits[0].Range.End.Character)
	assert.Equal(t, "", edits[0].NewText)

	edits = getEdits(`blah="hello"`+"\n"+"blah", `blah="hello"`)
	assert.Equal(t, `blah="hello"`, edits[0].NewText)
	assert.Equal(t, 0, edits[0].Range.Start.Line)
	assert.Equal(t, 1, edits[0].Range.End.Line)

	// Test for insertion
	edits = getEdits("", "\n"+"bar")

	assert.Equal(t, "\n", edits[0].NewText)
	assert.Equal(t, 0, edits[0].Range.Start.Line)
	assert.Equal(t, 0, edits[0].Range.End.Line)
	assert.Equal(t, "bar\n", edits[1].NewText)
	assert.Equal(t, 1, edits[1].Range.Start.Line)
	assert.Equal(t, 1, edits[1].Range.End.Line)
}
