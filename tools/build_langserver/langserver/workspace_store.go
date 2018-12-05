package langserver

import (
	"fmt"
	"strings"
	"sync"

	"github.com/thought-machine/please/tools/build_langserver/lsp"
)

type workspaceStore struct {
	rootURI lsp.DocumentURI
	mu      sync.Mutex

	documents map[lsp.DocumentURI]*document
}

type document struct {
	// text content of the document from the last time when saved
	text []string
	// test content of the document while in editing(not been saved)
	textInEdit []string
	version    int
}

func newWorkspaceStore(rootURI lsp.DocumentURI) *workspaceStore {
	return &workspaceStore{
		rootURI:   rootURI,
		documents: make(map[lsp.DocumentURI]*document),
	}
}

// Store method is generally used to correspond to "textDocument/didOpen",
// this stores the initial state of the document when opened
func (ws *workspaceStore) Store(uri lsp.DocumentURI, content string, version int) {
	text := SplitLines(content, true)
	ws.documents[uri] = &document{
		text:       text,
		textInEdit: text,
		version:    version,
	}
}

// Update method corresponds to "textDocument/didSave".
// This updates the existing uri document in the store
func (ws *workspaceStore) Update(uri lsp.DocumentURI, content string) error {
	text := SplitLines(content, true)
	if _, ok := ws.documents[uri]; !ok {
		return fmt.Errorf("document %s did not open", uri)
	}
	ws.documents[uri].text = text
	ws.documents[uri].textInEdit = text

	return nil
}

// Close method corresponds to "textDocument/didClose"
// This removes the document stored in workspaceStore
func (ws *workspaceStore) Close(uri lsp.DocumentURI) error {
	if _, ok := ws.documents[uri]; !ok {
		return fmt.Errorf("document %s did not open", uri)
	}

	ws.mu.Lock()
	defer ws.mu.Unlock()

	delete(ws.documents, uri)

	return nil
}

// TrackEdit tracks the changes of the content for the targeting uri, and update the corresponding
func (ws *workspaceStore) TrackEdit(uri lsp.DocumentURI, contentChanges []lsp.TextDocumentContentChangeEvent, version int) error {
	doc, ok := ws.documents[uri]
	if !ok {
		log.Error("document '%s' is not opened, edit did not apply", uri)
		return nil
	}

	ws.mu.Lock()
	defer ws.mu.Unlock()

	for _, change := range contentChanges {
		newText, err := ws.applyChange(doc.textInEdit, change)
		if err != nil {
			return err
		}
		doc.textInEdit = newText
	}
	doc.version = version

	return nil
}

func (ws *workspaceStore) applyChange(text []string, change lsp.TextDocumentContentChangeEvent) ([]string, error) {
	if change.Range == nil && change.RangeLength == 0 {
		return text, nil // new full content
	}

	changeText := change.Text

	startLine := change.Range.Start.Line
	endLine := change.Range.End.Line
	startCol := change.Range.Start.Character
	endCol := change.Range.End.Character

	var newText string

	for i, line := range text {
		if i < startLine || i > endLine {
			newText += line
			continue
		}

		if i == startLine {
			newText += line[:startCol] + changeText
		}

		if i == endLine {
			// Apparently, when you delete a whole line, intellij plugin sometimes sends the range like so:
			// {startline: deletedline_index, startcol: 0}, {endline: nextline, endcol: len_of_deleted_line}...
			if len(line)-1 < endCol && (len(text) != 1 && len(text[i-1])-1 == endCol) {
				newText += line
			} else {
				newText += line[endCol:]
			}
		}
	}

	return SplitLines(newText, true), nil
}

// SplitLines splits a content with \n characters, and returns a slice of string
// if keepEnds is true, all lines will keep it's original splited character
func SplitLines(content string, keepEnds bool) []string {
	splited := strings.Split(content, "\n")
	if !keepEnds {
		return splited
	}

	for i := range splited {
		// Do not add endline character on the last empty line
		if (i == len(splited)-1 && splited[i] == "") || len(splited) <= 1 {
			continue
		}
		splited[i] += "\n"
	}

	return splited
}

// JoinLines concatenate a slice of string, removes the trailing "\n" if hadEnds is true
func JoinLines(text []string, hasEnds bool) string {
	if !hasEnds {
		concat := strings.Join(text, "\n")
		return concat
	}

	newText := make([]string, len(text))
	for i := range text {
		if i == len(text)-1 && text[i] == "" {
			newText[i] = text[i]
			continue
		}
		newText[i] = strings.TrimSuffix(text[i], "\n")
	}

	return strings.Join(newText, "\n")
}
