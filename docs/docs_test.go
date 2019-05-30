package docs

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

func TestAllLinksAreLive(t *testing.T) {
	htmlfiles := map[string]bool{}
	allfiles := map[string]bool{}
	for _, datum := range strings.Split(os.Getenv("DATA"), " ") {
		datum = strings.TrimPrefix(datum, "docs/")
		allfiles[datum] = true
		if strings.HasSuffix(datum, ".html") {
			htmlfiles[datum] = true
		}
	}
	allnames := map[string]bool{}
	alllinks := []string{}
	for filename := range htmlfiles {
		f, err := os.Open("docs/" + filename)
		require.NoError(t, err)
		defer f.Close() // not until function exits, o well
		doc, err := html.Parse(f)
		require.NoError(t, err)
		var fn func(*html.Node)
		fn = func(n *html.Node) {
			if n.Type == html.ElementNode && n.Data == "a" {
				for _, attr := range n.Attr {
					if attr.Key == "href" && !strings.HasPrefix(attr.Val, "http") && !strings.HasPrefix(attr.Val, "about:") && attr.Val != "#" {
						if strings.HasPrefix(attr.Val, "#") {
							alllinks = append(alllinks, filename+attr.Val)
						} else {
							alllinks = append(alllinks, attr.Val)
						}
					} else if attr.Key == "name" {
						allnames[filename+"#"+attr.Val] = true
					}
				}
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				fn(c)
			}
		}
		fn(doc)
		allnames[filename] = true
	}
	for _, link := range alllinks {
		assert.Contains(t, allnames, link, "Broken link %s", link)
	}
}
