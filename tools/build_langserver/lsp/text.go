package lsp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/bazelbuild/buildtools/build"
	"github.com/sourcegraph/go-lsp"

	"github.com/thought-machine/please/src/parse/asp"
)

// A doc is a representation of a document that's opened by the editor.
type doc struct {
	// The filename of the document.
	Filename string
	// The Please package for this file
	PkgName string
	// The raw content of the document.
	Content []string
	// Parsed version of it
	AST   []*asp.Statement
	Mutex sync.Mutex
	// Channel for diagnostic requests.
	Diagnostics chan []*asp.Statement
}

func (d *doc) Text() string {
	d.Mutex.Lock()
	defer d.Mutex.Unlock()
	return strings.Join(d.Content, "\n")
}

func (d *doc) Lines() []string {
	d.Mutex.Lock()
	defer d.Mutex.Unlock()
	return d.Content
}

func (d *doc) SetText(text string) {
	d.Mutex.Lock()
	defer d.Mutex.Unlock()
	d.Content = strings.Split(text, "\n")
}

func (d *doc) AspFile() *asp.File {
	d.Mutex.Lock()
	defer d.Mutex.Unlock()
	return asp.NewFile(d.Filename, []byte(strings.Join(d.Content, "\n")))
}

func (h *Handler) didOpen(params *lsp.DidOpenTextDocumentParams) error {
	_, err := h.open(params.TextDocument.URI, params.TextDocument.Text)
	return err
}

func (h *Handler) open(uri lsp.DocumentURI, content string) (*doc, error) {
	filename := fromURI(uri)
	d := &doc{
		Filename:    filename,
		Diagnostics: make(chan []*asp.Statement, 100),
	}
	if path, err := filepath.Rel(h.root, filename); err == nil {
		d.Filename = path
	} else {
		log.Warningf("failed to figure out rel path: %v", err)
	}
	d.PkgName = filepath.Dir(d.Filename)

	d.SetText(content)
	go h.parse(d, content)
	go h.diagnose(d)
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.docs[filename] = d
	return d, nil
}

// maybeOpenDoc will open a doc unless it is already open. It will load
// the doc from disk to initialise its content
func (h *Handler) maybeOpenDoc(uri lsp.DocumentURI) (*doc, error) {
	filename := fromURI(uri)
	if doc, ok := h.docs[filename]; ok {
		return doc, nil
	}

	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return h.open(uri, string(content))
}

// parse parses the given document and updates its statements.
func (h *Handler) parse(d *doc, content string) {
	defer func() {
		recover()
	}()
	// Ignore errors, it will often fail if the file is partially complete, so
	// just take whatever we've got.
	stmts, _ := h.parser.ParseData([]byte(content), d.Filename)
	d.Mutex.Lock()
	defer d.Mutex.Unlock()
	d.AST = stmts
	d.Diagnostics <- stmts
}

// parseIfNeeded parses the document if it hasn't been done yet.
func (h *Handler) parseIfNeeded(d *doc) []*asp.Statement {
	d.Mutex.Lock()
	ast := d.AST[:] //nolint:ifshort
	d.Mutex.Unlock()
	if len(ast) != 0 {
		return ast
	}
	stmts, _ := h.parser.ParseData([]byte(d.Text()), d.Filename)
	d.Mutex.Lock()
	defer d.Mutex.Unlock()
	d.AST = stmts
	return stmts
}

// doc returns a document of the given URI, or panics if one doesn't exist.
func (h *Handler) doc(uri lsp.DocumentURI) *doc {
	filename := fromURI(uri)
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if doc := h.docs[filename]; doc != nil {
		return doc
	}
	// Theoretically at least this shouldn't happen - it indicates we are getting
	// requests for a document without a didOpen first.
	panic("Unknown document " + string(uri))
}

func (h *Handler) didChange(params *lsp.DidChangeTextDocumentParams) error {
	doc := h.doc(params.TextDocument.URI)
	// Synchronise changes into the doc's contents
	for _, change := range params.ContentChanges {
		if change.Range != nil {
			return fmt.Errorf("non-incremental change received")
		}
		doc.SetText(change.Text)
		go h.parse(doc, change.Text)
	}
	return nil
}

func (h *Handler) didSave(params *lsp.DidSaveTextDocumentParams) error {
	// TODO(peterebden): There should be a 'Text' property on the params that we can
	//                   sync from. It's in the spec but doesn't seem to be in go-lsp.
	return nil
}

func (h *Handler) didClose(params *lsp.DidCloseTextDocumentParams) error {
	d := h.doc(params.TextDocument.URI)
	h.mutex.Lock()
	defer h.mutex.Unlock()
	delete(h.docs, d.Filename)
	close(d.Diagnostics)
	// TODO(peterebden): At this point we should re-parse this package into the graph.
	return nil
}

func (h *Handler) formatting(params *lsp.DocumentFormattingParams) ([]*lsp.TextEdit, error) {
	doc := h.doc(params.TextDocument.URI)
	// Ignore formatting options, BUILD files are always canonically formatted at 4-space tabs.
	fn := build.ParseDefault
	if h.state.Config.IsABuildFile(filepath.Base(doc.Filename)) {
		fn = build.ParseBuild
	}
	f, err := fn(doc.Filename, []byte(doc.Text()))
	if err != nil {
		return nil, err
	}
	after := string(build.Format(f))
	if before := doc.Text(); before == after {
		return []*lsp.TextEdit{}, nil // Already formatted - great!
	}
	linesBefore := doc.Lines()
	doc.SetText(after)
	linesAfter := doc.Lines()
	// TODO(peterebden): Could do cleverer matching here...
	edits := []*lsp.TextEdit{}
	for i, line := range linesAfter {
		if i >= len(linesBefore) {
			// Gone off the end of the previous lines, insert all the rest in one go.
			edits = append(edits, &lsp.TextEdit{
				Range: lsp.Range{
					Start: lsp.Position{Line: i, Character: 0},
					End:   lsp.Position{Line: i, Character: 0},
				},
				NewText: strings.Join(linesAfter[i:], "\n"),
			})
			break
		} else if line != linesBefore[i] {
			edits = append(edits, &lsp.TextEdit{
				Range: lsp.Range{
					Start: lsp.Position{Line: i, Character: 0},
					End:   lsp.Position{Line: i, Character: len(linesBefore[i])},
				},
				NewText: line,
			})
		}
	}
	return edits, nil
}
