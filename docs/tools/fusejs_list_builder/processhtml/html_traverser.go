package processhtml

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/k3a/html2text"

	"github.com/thought-machine/please/docs/tools/fusejs_list_builder/fusejslist"
)

func nodeText(node *html.Node) (string, error) {
	var sb strings.Builder
	if err := html.Render(&sb, node); err != nil {
		return "", fmt.Errorf("%w: %w", ErrRenderingHTML, err)
	}

	renderedHTML := sb.String()
	htmlText := html2text.HTML2TextWithOptions(renderedHTML, html2text.WithUnixLineBreaks())
	return strings.TrimSpace(htmlText), nil
}

func idAttr(node *html.Node) string {
	for _, attr := range node.Attr {
		if attr.Key == "id" {
			return attr.Val
		}
	}
	return ""
}

// contentPart represents a heading or a section of text on the page
type contentPart struct {
	Heading          bool
	HeadingAnchorTag string
	Text             string
}

// HTMLTraverser traverses an HTML document, producing a list of entries for Fuse.js
type HTMLTraverser struct {
	PagePath string

	pageTitle    string
	contentParts []*contentPart
}

func (t *HTMLTraverser) processTextNode(node *html.Node) error {
	if node.Type != html.TextNode {
		panic(fmt.Sprintf(
			"processTextNode() called on html.NodeType %v (it should only be called on a html.TextNode)",
			node.Type,
		))
	}

	nodeText, err := nodeText(node)
	if err != nil {
		return err
	}

	t.contentParts = append(t.contentParts, &contentPart{
		Text: nodeText,
	})
	return nil
}

func (t *HTMLTraverser) processElementNode(node *html.Node) error {
	if node.Type != html.ElementNode {
		panic(fmt.Sprintf(
			"processElementNode() called on html.NodeType %v (it should only be called on a html.ElementNode)",
			node.Type,
		))
	}

	if node.DataAtom == atom.H1 {
		nodeText, err := nodeText(node)
		if err != nil {
			return err
		}

		if t.pageTitle != "" {
			t.pageTitle = fmt.Sprintf("%s, %s", t.pageTitle, nodeText)
		} else {
			t.pageTitle = nodeText
		}
	}

	if node.DataAtom == atom.H1 || node.DataAtom == atom.H2 {
		nodeText, err := nodeText(node)
		if err != nil {
			return err
		}

		anchorTag := idAttr(node)
		if anchorTag == "" {
			return fmt.Errorf("%w: heading '%s' on page '%s'", ErrHeadingMissingIDAttr, nodeText, t.pageTitle)
		}

		t.contentParts = append(t.contentParts, &contentPart{
			Heading:          true,
			HeadingAnchorTag: anchorTag,
			Text:             nodeText,
		})
		return nil
	}

	return t.traverseChildren(node)
}

func (t *HTMLTraverser) traverseChildren(node *html.Node) error {
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		if err := t.traverse(c); err != nil {
			return err
		}
	}
	return nil
}

func (t *HTMLTraverser) traverse(node *html.Node) error {
	switch node.Type {
	case html.TextNode:
		return t.processTextNode(node)
	case html.ElementNode:
		return t.processElementNode(node)
	}

	return t.traverseChildren(node)
}

// TraverseDocument produces a list of entries for Fuse.js from a HTML document.
// The PageTitle field is inferred from the <h1> elements in the document.
func (t *HTMLTraverser) TraverseDocument(node *html.Node) (fusejslist.List, error) {
	if node.Type != html.DocumentNode {
		return nil, ErrNotDocumentNode
	}

	if err := t.traverse(node); err != nil {
		return nil, err
	}

	var fusejsListItems fusejslist.List

	currHeading := ""
	currHeadingTag := ""
	var currTextContentParts []string

	for i := 0; i <= len(t.contentParts); i++ {
		if i < len(t.contentParts) && !t.contentParts[i].Heading {
			if t.contentParts[i].Text != "" {
				currTextContentParts = append(currTextContentParts, t.contentParts[i].Text)
			}
			continue
		}

		if len(currTextContentParts) > 0 {
			fusejsListItems = append(fusejsListItems, &fusejslist.ListItem{
				PagePath:         t.PagePath,
				PageTitle:        t.pageTitle,
				Heading:          currHeading,
				HeadingAnchorTag: currHeadingTag,
				TextContent:      strings.Join(currTextContentParts, " "),
			})
		}

		if i < len(t.contentParts) {
			currHeading = t.contentParts[i].Text
			currHeadingTag = t.contentParts[i].HeadingAnchorTag
			currTextContentParts = []string{}
		}
	}
	return fusejsListItems, nil
}
