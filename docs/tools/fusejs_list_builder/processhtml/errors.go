package processhtml

import "errors"

var (
	ErrHeadingMissingIDAttr = errors.New("heading tag missing 'id' attribute")
	ErrNotDocumentNode      = errors.New("not an HTML document node")
	ErrParsingHTMLFile      = errors.New("parsing HTML file")
	ErrRenderingHTML        = errors.New("rendering HTML")
)
