package godeps

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

type Target struct {
	Labels []string `json:"labels"`
	X map[string]interface{} `json:"-"` // Rest of the fields should go here.
}

type Package struct {
	Targets map[string]*Target `json:"targets"`
}

type Graph struct {
	Pkgs map[string]*Package `json:"packages"`
}


type Resolver interface {
	Resolve(importPath string) (string, error)
}

type trie struct {
	// TODO(jpoole): this could be quite large. Consider sync.Map and making this split workers per package.
	nodes map[string]*trie
	target string
	matchAll bool
}

func (t *trie) insert(path []string, value string) {
	if len(path) == 0 {
		log.Fatalf("CRITICAL: failed to insert %v, called insert with empty path", value)
	}
	nodeName := path[0]
	node, ok := t.nodes[nodeName]
	if !ok {
		node = &trie{nodes: map[string]*trie{}}
		t.nodes[nodeName] = node
	}

	if len(path) == 1 {
		node.target = value
		return
	}

	if len(path) == 2 && path[1] == "..." {
		node.target = value
		node.matchAll = true
		return
	}

	node.insert(path[1:], value)
}

func (t *trie) find(soFar []string, path []string) (string, error) {
	if len(path) == 0 {
		return t.target, nil
	}
	if t.matchAll {
		return t.target, nil
	}
	node, ok := t.nodes[path[0]]
	if !ok {
		soFarStr := strings.Join(soFar, "/")
		pathStr :=  strings.Join(append(soFar, path...), "/")
		return "", fmt.Errorf("cant resolve %s in graph, %s doesn't contain %s", pathStr, soFarStr, path[0])
	}

	return node.find(append(soFar, path[0]), path[1:])
}

func (t *trie) Resolve(importPath string) (string, error) {
	return t.find([]string{}, strings.Split(importPath, "/"))
}

// GoDeps resolves the given imports and prints a space separated lines mapping packages to targets
func GoDeps(plz string, targets, imports []string) {
	resolver, err := BuildResolver(plz, targets)
	if err != nil {
		log.Fatal(err)
	}

	for _, i := range imports {
		target, err := resolver.Resolve(i)
		if err != nil {
			//TODO(jpoole): handle this
			log.Fatal(err)
		}
		fmt.Printf("%s %s\n", i, target)
	}
}


// BuildResolver build a resolver from the build graph. For performance, it might be desired to only process part of the
// build graph. For this reason, you can pass wildcard build labels to targets e.g. //src/..., otherwise the whole graph
// is processed
func BuildResolver(plz string, targets []string) (*trie, error) {
	cmd := exec.Command(plz, append([]string{"query", "graph"}, targets...)...)
	stdOut := &bytes.Buffer{}
	cmd.Stdout = stdOut
	cmd.Stderr = stdOut

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("error building import mapping: %w\nOutput: %s", err, stdOut.String())
	}

	graph := new(Graph)
	if err := json.Unmarshal(stdOut.Bytes(), &graph); err != nil {
		panic(err)
	}

	t := &trie{nodes: map[string]*trie{}}
	for pkg, target := range graph.goPackages() {
		t.insert(strings.Split(pkg, "/"), target)
	}
	return t, nil
}

func (pkg *Graph) goPackages() map[string]string {
	ret := make(map[string]string)
	for name, pkg := range pkg.Pkgs {
		for t, pkg := range pkg.goPackages(name) {
			ret[t] = pkg
		}
	}
	return ret
}

func (pkg *Package) goPackages(name string) map[string]string {
	ret := make(map[string]string)
	for targetName, t := range pkg.Targets {
		for _, l := range t.Labels {
			if strings.HasPrefix(l, "go_package:") {
				label := fmt.Sprintf("//%s:%s", name, targetName)
				ret[strings.TrimPrefix(l, "go_package:")] = label
			}
		}
	}
	return ret
}
