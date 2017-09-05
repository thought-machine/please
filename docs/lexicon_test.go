package docs

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

// undocumentedFunctions is a list of whitelisted functions that we're not
// publicly documenting (yet?).
var undocumentedFunctions = map[string]bool{
	"grpc_languages":  true,
	"proto_languages": true,
	"proto_language":  true,
	// Right now c_library etc are duplicates of cc_library and friends.
	// For now we're skipping here, in future we may add a separate section for documentation.
	"c_binary":         true,
	"c_library":        true,
	"c_test":           true,
	"c_embed_binary":   true,
	"c_shared_object":  true,
	"c_static_library": true,
	// These are deprecated and will be removed from builtins in the next major version.
	"go_yacc":     true,
	"git_repo":    true,
	"github_file": true,
	"fpm_package": true,
	"fpm_deb":     true,
	// This guy is documented but I don't want to add a single-line table for this test.
	"decompose": true,
}

type ruleArgs struct {
	Functions map[string]struct {
		Args []ruleArg `json:"args"`
	} `json:"functions"`
}

type ruleArg struct {
	Deprecated bool     `json:"deprecated"`
	Required   bool     `json:"required"`
	Name       string   `json:"name"`
	Types      []string `json:"types"`
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func loadRuleArgs() map[string]map[string]ruleArg {
	b, err := ioutil.ReadFile("src/parse/rule_args.json")
	must(err)
	args := &ruleArgs{}
	must(json.Unmarshal(b, &args))
	ret := map[string]map[string]ruleArg{}
	for name, f := range args.Functions {
		a := map[string]ruleArg{}
		for _, arg := range f.Args {
			a[arg.Name] = arg
		}
		ret[name] = a
	}
	return ret
}

// loadHTML loads lexicon.html and returns a mapping of function name -> arg name -> declared types.
func loadHTML() map[string]map[string]ruleArg {
	ret := map[string]map[string]ruleArg{}
	f, err := os.Open("docs/lexicon.html")
	must(err)
	defer f.Close()
	tree, err := html.Parse(f)
	must(err)
	// This assumes fairly specific knowledge about the structure of the HTML.
	// Note that it's not a trivial parse since the rules & their tables are all siblings.
	lastAName := ""
	tree = tree.FirstChild.FirstChild.NextSibling // Walk through structural elements that parser inserts
	for node := tree.FirstChild; node != nil; node = node.NextSibling {
		if node.Type == html.ElementNode && node.Data == "h3" {
			if a := node.FirstChild; a != nil && a.Type == html.ElementNode && a.Data == "a" {
				for _, attr := range a.Attr {
					if attr.Key == "name" {
						lastAName = attr.Val
						break
					}
				}
			}
		} else if node.Type == html.ElementNode && node.Data == "table" {
			if tbody := allChildren(node, "tbody"); len(tbody) == 1 {
				args := map[string]ruleArg{}
				trs := allChildren(tbody[0], "tr")
				for _, tr := range trs {
					if tr.Type == html.ElementNode && tr.Data == "tr" {
						tds := allChildren(tr, "td")
						name := innerText(tds[0])
						args[name] = ruleArg{
							Name:  name,
							Types: strings.Split(innerText(tds[2]), " or "),
						}
					}
				}
				ret[lastAName] = args
			}
		}
	}
	return ret
}

func allChildren(node *html.Node, tag string) []*html.Node {
	ret := []*html.Node{}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.Data == tag {
			ret = append(ret, child)
		}
	}
	return ret
}

func innerText(node *html.Node) string {
	if child := node.FirstChild; child != nil {
		return child.Data
	}
	return ""
}

func TestAllArgsArePresentInHTML(t *testing.T) {
	args := loadRuleArgs()
	html := loadHTML()
	for name, arg := range args {
		if undocumentedFunctions[name] {
			continue // Function isn't expected to be documented right now.
		}
		t.Run(name, func(t *testing.T) {
			htmlArg, present := html[name]
			require.True(t, present, "Built-in function %s is not documented in lexicon", name)
			for argName, arg2 := range arg {
				if arg2.Deprecated {
					continue // Deprecated args aren't required to be documented.
				} else if argName == "size" {
					continue // Currently unsure whether I want to document or remove this.
				}
				htmlArg2, present := htmlArg[argName]
				assert.True(t, present, "Built-in function %s is lacking documentation for argument %s", name, argName)
				if present {
					assert.EqualValues(t, arg2.Types, htmlArg2.Types, "Built-in function %s, argument %s declares different types to documentation", name, argName)
				}
			}
		})
	}
}
