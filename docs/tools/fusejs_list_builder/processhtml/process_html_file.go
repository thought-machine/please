package processhtml

import (
	"fmt"
	"os"

	"golang.org/x/net/html"

	"github.com/thought-machine/please/docs/tools/fusejs_list_builder/fusejslist"
)

func ProcessHTMLFile(f *os.File, path string) (fusejslist.List, error) {
	docNode, err := html.Parse(f)
	if err != nil {
		return nil, fmt.Errorf("%w - %s: %w", ErrParsingHTMLFile, f.Name(), err)
	}

	t := &HTMLTraverser{
		PagePath: path,
	}
	return t.TraverseDocument(docNode)
}
