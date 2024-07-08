package test

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/thought-machine/please/src/core"

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

// Config options we don't want to document for various reasons
var ignoreConfigFields = map[string]struct{}{
	// These are deprecated
	"build.pleasesandboxtool": {},
	// Just don't want to advertise these ones too much
	"bazel.compatibility": {},
	"featureflags":        {},
	"metrics":             {},
	// These aren't real config options
	"please.version.isgte":  {},
	"please.version.isset":  {},
	"homedir":               {},
	"profiling":             {},
	"buildenvstored":        {},
	"pleaselocation":        {},
	"plugin.extravalues":    {},
	"plugindefinition.name": {},
}

// IDs in the html that are for other purposes other than documenting config.
var nonConfigIds = map[string]struct{}{
	"menu-list":    {},
	"nav-graphic":  {},
	"side-images":  {},
	"menu-button":  {},
	"main-content": {},

	"config-file-reference": {}, // the top heading for the overall page
}

func TestConfigDocumented(t *testing.T) {
	findConfigFields("", reflect.TypeOf(core.Configuration{}))
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

	configFields := map[string]struct{}{}
	for _, configField := range findConfigFields("", reflect.TypeOf(core.Configuration{})) {
		configFields[configField] = struct{}{}
		if _, ok := ids[configField]; !ok {
			t.Logf("missing section with matching ID for config field %v", configField)
			t.Fail()
		}
	}

	for id := range ids {
		if _, ok := nonConfigIds[id]; ok {
			continue
		}
		if _, ok := configFields[id]; !ok {
			t.Logf("config section with id %v matches no config field", id)
			t.Fail()
		}
	}
}

func findConfigFields(path string, configType reflect.Type) []string {
	var fields []string
	for i := 0; i < configType.NumField(); i++ {
		field := configType.Field(i)

		if field.Tag.Get("help") == "" {
			continue
		}

		name := strings.ToLower(field.Name)
		if path != "" {
			name = fmt.Sprintf("%v.%v", path, name)
		}
		if _, ok := ignoreConfigFields[name]; ok {
			continue
		}
		fields = append(fields, name)

		t := fieldElem(field.Type)
		if t.Kind() == reflect.Struct {
			fields = append(fields, findConfigFields(name, t)...)
		}

	}
	return fields
}

func fieldElem(t reflect.Type) reflect.Type {
	kind := t.Kind()
	if kind == reflect.Ptr || kind == reflect.Map || kind == reflect.Array || kind == reflect.Slice {
		return fieldElem(t.Elem())
	}
	return t
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
