package langserver

import (
	"context"
	"github.com/thought-machine/please/src/core"
	"strings"
	"testing"

	"github.com/thought-machine/please/tools/build_langserver/lsp"

	"github.com/stretchr/testify/assert"
)

var ws = newWorkspaceStore(lsp.DocumentURI(core.RepoRoot))

func TestWorkspaceStore_Store(t *testing.T) {
	ctx := context.Background()
	content, err := ReadFile(ctx, completionURI)
	assert.Equal(t, nil, err)
	assert.Equal(t, 0, len(ws.documents))

	text := strings.Join(content, "\n")
	ws.Store(completionURI, text, 1)
	assert.Equal(t, 1, len(ws.documents))
	assert.Equal(t, "name = \"langserver_test\"\n", ws.documents[completionURI].text[0])
}

func TestWorkspaceStore_TrackEdit(t *testing.T) {
	// Test Add Character
	change := lsp.TextDocumentContentChangeEvent{
		Range: &lsp.Range{
			Start: lsp.Position{Line: 4, Character: 7},
			End:   lsp.Position{Line: 4, Character: 7},
		},
		RangeLength: 1,
		Text:        "s",
	}

	err := ws.TrackEdit(completionURI, []lsp.TextDocumentContentChangeEvent{change}, 33)
	assert.Equal(t, nil, err)
	assert.Equal(t, "    srcs\n", ws.documents[completionURI].textInEdit[4])

	// Test Remove line
	change = lsp.TextDocumentContentChangeEvent{
		Range: &lsp.Range{
			Start: lsp.Position{Line: 9, Character: 0},
			End:   lsp.Position{Line: 10, Character: 12},
		},
		RangeLength: 0,
		Text:        "",
	}

	err = ws.TrackEdit(completionURI, []lsp.TextDocumentContentChangeEvent{change}, 34)
	assert.Equal(t, nil, err)
	expected := []string{
		"go_test(\n",
		"    name = \"\",\n",
		"    srcs =[],\n",
		"    \"//src\"\n",
		")\n",
		"",
	}
	assert.Equal(t, expected, ws.documents[completionURI].textInEdit[6:])
	assert.Equal(t, 34, ws.documents[completionURI].version)
}

func TestWorkspaceStore_applyChangeAddChar(t *testing.T) {
	text := []string{
		"     \"//src/cli\",\n",
		" name = \"please\",\n",
		"\"//src/quer\"\n",
		"\"//src\"\n",
		"",
	}

	// add one character
	change := lsp.TextDocumentContentChangeEvent{
		Range: &lsp.Range{
			Start: lsp.Position{Line: 2, Character: 11},
			End:   lsp.Position{Line: 2, Character: 11},
		},
		RangeLength: 0,
		Text:        "y",
	}
	// Copy the text slice so it doesn't modify original
	inText := append([]string{}, text...)

	outText, err := ws.applyChange(inText, change)
	assert.Equal(t, nil, err)
	expected := []string{
		"     \"//src/cli\",\n",
		" name = \"please\",\n",
		"\"//src/query\"\n",
		"\"//src\"\n",
		"",
	}
	assert.Equal(t, expected, outText)

	// add a whole line
	change = lsp.TextDocumentContentChangeEvent{
		Range: &lsp.Range{
			Start: lsp.Position{Line: 1, Character: 0},
			End:   lsp.Position{Line: 1, Character: 0},
		},
		RangeLength: 0,
		Text:        "        \"//third_party/go:jsonrpc2\",\n",
	}
	// Copy the text slice so it doesn't modify original
	inText = append([]string{}, text...)

	outText, err = ws.applyChange(inText, change)
	assert.Equal(t, nil, err)
	expected = []string{
		"     \"//src/cli\",\n",
		"        \"//third_party/go:jsonrpc2\",\n",
		" name = \"please\",\n",
		"\"//src/quer\"\n",
		"\"//src\"\n",
		"",
	}
	assert.Equal(t, expected, outText)

	// apply changes with empty content
	change = lsp.TextDocumentContentChangeEvent{
		Range: &lsp.Range{
			Start: lsp.Position{Line: 0, Character: 0},
			End:   lsp.Position{Line: 0, Character: 0},
		},
		RangeLength: 0,
		Text:        "yh",
	}
	outText, err = ws.applyChange([]string{""}, change)
	assert.Equal(t, nil, err)
	assert.Equal(t, "yh", outText[0])
	assert.Equal(t, 1, len(outText))
}

func TestWorkspaceStore_applyChangeDelete(t *testing.T) {
	text := []string{
		"     \"//src/cli\",\n",
		" name = \"please\",\n",
		"\"//src/quer\"\n",
		"\"//src\"\n",
		"",
	}

	// delete one character
	change := lsp.TextDocumentContentChangeEvent{
		Range: &lsp.Range{
			Start: lsp.Position{Line: 2, Character: 10},
			End:   lsp.Position{Line: 2, Character: 11},
		},
		RangeLength: 0,
		Text:        "",
	}
	// Copy the text slice so it doesn't modify original
	inText := append([]string{}, text...)

	outText, err := ws.applyChange(inText, change)
	assert.Equal(t, nil, err)
	expected := []string{
		"     \"//src/cli\",\n",
		" name = \"please\",\n",
		"\"//src/que\"\n",
		"\"//src\"\n",
		"",
	}
	assert.Equal(t, expected, outText)

	// delete more than one char
	change = lsp.TextDocumentContentChangeEvent{
		Range: &lsp.Range{
			Start: lsp.Position{Line: 2, Character: 6},
			End:   lsp.Position{Line: 2, Character: 11},
		},
		RangeLength: 0,
		Text:        "",
	}

	inText = append([]string{}, text...)
	outText, err = ws.applyChange(inText, change)
	expected = []string{
		"     \"//src/cli\",\n",
		" name = \"please\",\n",
		"\"//src\"\n",
		"\"//src\"\n",
		"",
	}
	assert.Equal(t, expected, outText)

	// delete the whole line content
	change = lsp.TextDocumentContentChangeEvent{
		Range: &lsp.Range{
			Start: lsp.Position{Line: 2, Character: 0},
			End:   lsp.Position{Line: 2, Character: 12},
		},
		RangeLength: 0,
		Text:        "",
	}

	inText = append([]string{}, text...)
	outText, err = ws.applyChange(inText, change)
	expected = []string{
		"     \"//src/cli\",\n",
		" name = \"please\",\n",
		"\n",
		"\"//src\"\n",
		"",
	}
	assert.Equal(t, expected, outText)

	// delete single line content
	change = lsp.TextDocumentContentChangeEvent{
		Range: &lsp.Range{
			Start: lsp.Position{Line: 0, Character: 0},
			End:   lsp.Position{Line: 0, Character: 5},
		},
		RangeLength: 0,
		Text:        "",
	}

	outText, err = ws.applyChange([]string{"hello"}, change)
	assert.Equal(t, nil, err)
	assert.Equal(t, 1, len(outText))
	assert.Equal(t, "", outText[0])
}

func TestWorkspaceStore_applyChangeDeleteCrossLine(t *testing.T) {
	text := []string{
		"     \"//src/cli\",\n",
		" name = \"please\",\n",
		"\"//src/quer\"\n",
		"\"//blah\"\n",
		"",
	}

	// delete the whole line including newline char
	change := lsp.TextDocumentContentChangeEvent{
		Range: &lsp.Range{
			Start: lsp.Position{Line: 3, Character: 0},
			End:   lsp.Position{Line: 4, Character: 0},
		},
		RangeLength: 0,
		Text:        "",
	}

	inText := append([]string{}, text...)
	outText, err := ws.applyChange(inText, change)
	assert.Equal(t, nil, err)
	expected := []string{
		"     \"//src/cli\",\n",
		" name = \"please\",\n",
		"\"//src/quer\"\n",
		"",
	}
	assert.Equal(t, expected, outText)

	// Intellij's weird range for deleting the whole line...
	change = lsp.TextDocumentContentChangeEvent{
		Range: &lsp.Range{
			Start: lsp.Position{Line: 3, Character: 0},
			End:   lsp.Position{Line: 4, Character: 8},
		},
		RangeLength: 0,
		Text:        "",
	}

	inText = append([]string{}, text...)
	outText, err = ws.applyChange(inText, change)
	assert.Equal(t, nil, err)
	expected = []string{
		"     \"//src/cli\",\n",
		" name = \"please\",\n",
		"\"//src/quer\"\n",
		"",
	}
	assert.Equal(t, expected, outText)

	// and we hope this does not interfere with the same thing
	// if we delete two lines but leave one of the line breaks
	change = lsp.TextDocumentContentChangeEvent{
		Range: &lsp.Range{
			Start: lsp.Position{Line: 2, Character: 0},
			End:   lsp.Position{Line: 3, Character: 12},
		},
		RangeLength: 0,
		Text:        "",
	}

	inText = []string{
		"     \"//src/cli\",\n",
		" name = \"please\",\n",
		"\"//src/quer\"\n",
		"\"//src/quer\"\n",
		"\"//blah\"\n",
		"",
	}
	outText, err = ws.applyChange(inText, change)
	assert.Equal(t, nil, err)
	expected = []string{
		"     \"//src/cli\",\n",
		" name = \"please\",\n",
		"\n",
		"\"//blah\"\n",
		"",
	}
	assert.Equal(t, expected, outText)

	// delete some the previous line, but leaving parts of the next line
	change = lsp.TextDocumentContentChangeEvent{
		Range: &lsp.Range{
			Start: lsp.Position{Line: 2, Character: 0},
			End:   lsp.Position{Line: 3, Character: 5},
		},
		RangeLength: 0,
		Text:        "",
	}

	inText = append([]string{}, text...)
	outText, err = ws.applyChange(inText, change)
	assert.Equal(t, nil, err)
	expected = []string{
		"     \"//src/cli\",\n",
		" name = \"please\",\n",
		"ah\"\n",
		"",
	}
	assert.Equal(t, expected, outText)

	change = lsp.TextDocumentContentChangeEvent{
		Range: &lsp.Range{
			Start: lsp.Position{Line: 1, Character: 17},
			End:   lsp.Position{Line: 3, Character: 8},
		},
		RangeLength: 0,
		Text:        "",
	}

	inText = append([]string{}, text...)
	outText, err = ws.applyChange(inText, change)
	assert.Equal(t, nil, err)
	expected = []string{
		"     \"//src/cli\",\n",
		" name = \"please\",\n",
		"",
	}
	assert.Equal(t, expected, outText)
}

func TestSplitLines(t *testing.T) {
	content := "     \"//src/cli\",\n" +
		" name = \"please\",\n" +
		"\"//src/quer\"\n"

	text := SplitLines(content, true)
	expected := []string{
		"     \"//src/cli\",\n",
		" name = \"please\",\n",
		"\"//src/quer\"\n",
		"",
	}

	assert.Equal(t, expected, text)

	// Test with empty content
	content = ""
	assert.Equal(t, []string{""}, SplitLines(content, true))
}

func TestJoinLines(t *testing.T) {
	text := []string{
		"     \"//src/cli\",\n",
		" name = \"please\",\n",
		"\"//src/quer\"\n",
		"",
	}
	expected := "     \"//src/cli\",\n" +
		" name = \"please\",\n" +
		"\"//src/quer\"\n"
	content := JoinLines(text, true)

	assert.Equal(t, expected, content)

	stmts, _ := analyzer.AspStatementFromFile(exampleBuildURI)
	for _, stmt := range stmts {
		t.Log(stmt)
	}

}
