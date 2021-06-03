package test

import (
	"fmt"
	"github.com/thought-machine/please/src/core"
	"os"
	"reflect"
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
			if n.Type == html.ElementNode {
				for _, attr := range n.Attr {
					if n.Data == "a" {
						if attr.Key == "href" && !strings.HasPrefix(attr.Val, "http") && !strings.HasPrefix(attr.Val, "about:") && attr.Val != "#" {
							if strings.HasPrefix(attr.Val, "/codelabs") {
								continue
							}
							if strings.HasPrefix(attr.Val, "#") {
								alllinks = append(alllinks, filename+attr.Val)
							} else {
								alllinks = append(alllinks, attr.Val)
							}
						} else if attr.Key == "name" {
							allnames[filename+"#"+attr.Val] = true
						}
					} else {
						if attr.Key == "id" {
							allnames[filename+"#"+attr.Val] = true
						}
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
		assert.Contains(t, allnames, strings.TrimPrefix(link, "/"), "Broken link %s", link)
	}
}

var ignoreConfigFields = map[string]struct{} {
	// These are covered by the documentation of their parent
	"please.version.version": {},
	"build.arch.os": {},
	"build.arch.arch": {},
	"cpp.testmain.packagename": {},
	"cpp.testmain.name": {},
	"cpp.testmain.subrepo": {},
	// These are deprecated
	"build.pleasesandboxtool": {},
	// Just don't want to advertise these ones too much
	"build.passunsafeenv": {},
	"bazel.compatibility": {},
	"featureflags": {},
	"metrics": {},
	// These aren't real config options
	"please.version.isgte": {},
	"please.version.isset": {},
	"homedir": {},
	"profiling": {},
	"buildenvstored": {},
	"pleaselocation": {},
}

func TestConfigDocumented(t *testing.T) {
	configHTML, err := os.Open("docs/config.html")
	if err != nil {
		t.Fatalf("couldn't read docs/config.html: %v", err)
	}

	doc, err := html.Parse(configHTML)
	if err != nil {
		t.Fatalf("couldn't parse docs/config.html: %v", err)
	}

	ids := map[string]struct{}{}
	findIDs(doc, ids)

	for _, configField := range findConfigFields("", reflect.TypeOf(core.Configuration{})) {
		if _, ok := ids[configField]; !ok {
			t.Logf("missing section with matching ID for config field %v", configField)
			t.Fail()
		}
	}

}

func findConfigFields(path string, configType reflect.Type) []string {
	var fields []string
	for i := 0; i < configType.NumField(); i++ {
		field := configType.Field(i)

		name := strings.ToLower(field.Name)
		if path != "" {
			name = fmt.Sprintf("%v.%v", path, name)
		}
		if _, ok := ignoreConfigFields[name]; ok {
			continue
		}
		fields = append(fields, name)

		if field.Type.Kind() == reflect.Struct {
			fields = append(fields, findConfigFields(name, field.Type)...)
		}
	}
	return fields
}

func findIDs(node *html.Node, ids map[string]struct{}) map[string]struct{} {
	for _, attr := range node.Attr {
		if attr.Key == "id" {
			ids[attr.Val] = struct{}{}
			break
		}
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		findIDs(child, ids)
	}
	return ids
}