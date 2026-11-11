package query

import (
	"cmp"
	"encoding/base64"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"io"
	"iter"
	"os"
	"path/filepath"
	"runtime"
	"slices"

	"github.com/thought-machine/please/src/build"
	"github.com/thought-machine/please/src/core"
)

// graphEncodeOptions are the options used to encode the graph.
var graphEncodeOptions = json.JoinOptions(
	jsontext.WithIndent("    "),
	json.Deterministic(true), // Sort map keys so the output is stable
	// Paths and commands aren't guaranteed to be valid UTF-8; replace bad bytes with U+FFFD rather
	// than failing (which would leave a truncated document since we're streaming).
	jsontext.AllowInvalidUTF8(true),
	jsontext.AllowDuplicateNames(true), // We construct all the keys ourselves; this saves tracking every name in each object
)

// Graph prints a representation of the build graph as JSON.
func Graph(state *core.BuildState, targets []core.BuildLabel) {
	log.Notice("Generating graph...")
	if err := writeGraph(os.Stdout, state, targets); err != nil {
		log.Fatalf("Failed to serialise JSON: %s\n", err)
	}
	log.Notice("Done")
}

// writeGraph writes the JSON representation of the build graph to the given writer.
// When writing the entire graph, it is streamed out one package at a time rather than
// being constructed in memory first.
func writeGraph(w io.Writer, state *core.BuildState, targets []core.BuildLabel) error {
	enc := jsontext.NewEncoder(w, graphEncodeOptions)
	if len(targets) != 0 {
		return json.MarshalEncode(enc, makeJSONGraph(state, targets))
	}
	return writeAllPackages(enc, state)
}

// writeAllPackages streams the entire graph to the given encoder. The output is identical to
// encoding a JSONGraph containing every package.
func writeAllPackages(enc *jsontext.Encoder, state *core.BuildState) error {
	// Sort the packages so the output is deterministic; this matches the order of the keys in
	// JSONGraph (packages in the root repo, then each subrepo in turn, then by name within those).
	pkgs := slices.SortedFunc(state.Graph.AllPackages(), func(a, b *core.Package) int {
		return cmp.Or(cmp.Compare(a.SubrepoName, b.SubrepoName), cmp.Compare(a.Name, b.Name))
	})
	w := tokenWriter{enc: enc}
	w.Write(jsontext.BeginObject, jsontext.String("packages"), jsontext.BeginObject)
	subrepo := ""
	// The window is fairly generous since per-package cost varies a lot and a slow one blocks
	// everything queued behind it. Completed packages are small relative to the whole graph.
	window := 100 * runtime.GOMAXPROCS(0)
	for pkg := range orderedParallel(pkgs, window, func(pkg *core.Package) JSONPackage {
		return makeJSONPackage(state, pkg)
	}) {
		if pkg.subrepo != subrepo {
			if subrepo == "" {
				// End of the root packages, start of the subrepos
				w.Write(jsontext.EndObject, jsontext.String("subrepos"), jsontext.BeginObject)
			} else {
				// End of the previous subrepo's packages and the subrepo itself
				w.Write(jsontext.EndObject, jsontext.EndObject)
			}
			w.Write(jsontext.String(pkg.subrepo), jsontext.BeginObject, jsontext.String("packages"), jsontext.BeginObject)
			subrepo = pkg.subrepo
		}
		w.Write(jsontext.String(pkg.name))
		if w.err == nil {
			w.err = json.MarshalEncode(enc, pkg)
		}
		if w.err != nil {
			return w.err
		}
	}
	if subrepo != "" {
		w.Write(jsontext.EndObject, jsontext.EndObject) // the last subrepo's packages and the subrepo itself
	}
	w.Write(jsontext.EndObject, jsontext.EndObject) // packages / subrepos, and the root object
	return w.err
}

// A tokenWriter writes tokens to an encoder, retaining the first error that occurs.
type tokenWriter struct {
	enc *jsontext.Encoder
	err error
}

func (w *tokenWriter) Write(tokens ...jsontext.Token) {
	for _, tok := range tokens {
		if w.err != nil {
			return
		}
		w.err = w.enc.WriteToken(tok)
	}
}

// orderedParallel applies f to each element of in, running up to roughly n of them concurrently,
// and yields the results in the same order as the input.
// Limiting the window bounds how many results can be waiting in memory behind a slow one.
func orderedParallel[T, U any](in []T, n int, f func(T) U) iter.Seq[U] {
	return func(yield func(U) bool) {
		results := make(chan chan U, n)
		done := make(chan struct{})
		defer close(done)
		go func() {
			defer close(results)
			for _, x := range in {
				ch := make(chan U, 1)
				select {
				case results <- ch:
				case <-done:
					return
				}
				go func() { ch <- f(x) }()
			}
		}()
		for ch := range results {
			if !yield(<-ch) {
				return
			}
		}
	}
}

// JSONGraph is an alternate representation of our build graph; will contain different information
// to the structures in core (also those ones can't be printed as JSON directly).
type JSONGraph struct {
	Packages map[string]JSONPackage `json:"packages"`
	Subrepos map[string]*JSONGraph  `json:"subrepos,omitempty"`
}

// JSONPackage is an alternate representation of a build package
type JSONPackage struct {
	name    string
	subrepo string
	Targets map[string]JSONTarget `json:"targets"`
}

// JSONTarget is an alternate representation of a build target
type JSONTarget struct {
	Inputs   []string    `json:"inputs,omitempty" note:"declared inputs of target"`
	Outputs  []string    `json:"outs,omitempty" note:"corresponds to outs in rule declaration"`
	Sources  interface{} `json:"srcs,omitempty" note:"corresponds to srcs in rule declaration"`
	Tools    interface{} `json:"tools,omitempty" note:"corresponds to tools in rule declaration"`
	Deps     []string    `json:"deps,omitempty" note:"corresponds to deps in rule declaration"`
	Data     []string    `json:"data,omitempty" note:"corresponds to data in rule declaration"`
	Labels   []string    `json:"labels,omitempty" note:"corresponds to labels in rule declaration"`
	Requires []string    `json:"requires,omitempty" note:"corresponds to requires in rule declaration"`
	Command  string      `json:"command,omitempty" note:"the currently active command of the target. not present on filegroup or remote_file actions"`
	Hash     string      `json:"hash" note:"partial hash of target, does not include source hash"`
	Test     bool        `json:"test,omitzero" note:"true if target is a test"`
	Binary   bool        `json:"binary,omitzero" note:"true if target is a binary"`
	TestOnly bool        `json:"test_only,omitzero" note:"true if target should be restricted to test code"`
}

func makeJSONGraph(state *core.BuildState, targets []core.BuildLabel) *JSONGraph {
	ret := JSONGraph{
		Packages: map[string]JSONPackage{},
		Subrepos: map[string]*JSONGraph{},
	}
	if len(targets) == 0 {
		for pkg := range state.Graph.AllPackages() {
			ret.Subrepo(pkg.SubrepoName).Packages[pkg.Name] = makeJSONPackage(state, pkg)
		}
	} else {
		done := map[core.BuildLabel]struct{}{}
		for _, target := range targets {
			addJSONTarget(state, &ret, target, done)
		}
	}
	return &ret
}

// Subrepo returns a subrepo for the given name. If it's empty the top-level repo is returned.
func (graph *JSONGraph) Subrepo(name string) *JSONGraph {
	if name == "" {
		return graph
	} else if subrepo, present := graph.Subrepos[name]; present {
		return subrepo
	}
	subrepo := &JSONGraph{
		Packages: map[string]JSONPackage{},
		Subrepos: map[string]*JSONGraph{},
	}
	graph.Subrepos[name] = subrepo
	return subrepo
}

func addJSONTarget(state *core.BuildState, graph *JSONGraph, label core.BuildLabel, done map[core.BuildLabel]struct{}) {
	if _, present := done[label]; present {
		return
	}
	done[label] = struct{}{}
	if label.IsAllTargets() {
		pkg := state.Graph.PackageOrDie(label)
		for _, target := range pkg.AllTargets() {
			addJSONTarget(state, graph, target.Label, done)
		}
		return
	}
	target := state.Graph.TargetOrDie(label)
	repo := graph.Subrepo(label.Subrepo)
	if _, present := repo.Packages[label.PackageName]; present {
		repo.Packages[label.PackageName].Targets[label.Name] = makeJSONTarget(state, target)
	} else {
		repo.Packages[label.PackageName] = JSONPackage{
			Targets: map[string]JSONTarget{
				label.Name: makeJSONTarget(state, target),
			},
		}
	}
	deps, unresolved := target.Dependencies(state.Graph)
	if len(unresolved) > 0 {
		log.Fatalf("Can't generate graph for %s; dependencies not in build graph: %s", target.Label, unresolved)
	}
	for _, dep := range deps {
		addJSONTarget(state, graph, dep.Label, done)
	}
}

func makeJSONPackage(state *core.BuildState, pkg *core.Package) JSONPackage {
	targets := map[string]JSONTarget{}
	for _, target := range pkg.AllTargets() {
		targets[target.Label.Name] = makeJSONTarget(state, target)
	}
	return JSONPackage{name: pkg.Name, subrepo: pkg.SubrepoName, Targets: targets}
}

func makeJSONTarget(state *core.BuildState, target *core.BuildTarget) JSONTarget {
	t := JSONTarget{
		Sources: makeJSONInputField(state.Graph, target.AllSourcePaths(state.Graph), target.NamedSources),
		Tools:   makeJSONInputField(state.Graph, buildInputsToStrings(state.Graph, target.AllTools()), target.AllNamedTools()),
	}
	for in := range core.IterSources(state, state.Graph, target, false) {
		t.Inputs = append(t.Inputs, in)
	}
	for _, out := range target.Outputs(state.Graph) {
		t.Outputs = append(t.Outputs, filepath.Join(target.Label.PackageName, out))
	}
	deps, unresolved := target.Dependencies(state.Graph)
	if len(unresolved) > 0 {
		log.Fatalf("Can't generate graph for %s; dependencies not in build graph: %s", target.Label, unresolved)
	}
	for _, dep := range deps {
		t.Deps = append(t.Deps, dep.Label.String())
	}
	// just use run 1 as this is only used to print the test dir
	for data := range core.IterRuntimeFiles(state.Graph, target, false, target.TestDir(1)) {
		t.Data = append(t.Data, data)
	}
	t.Labels = target.Labels
	t.Requires = target.Requires
	if !target.IsFilegroup && !target.IsRemoteFile {
		t.Command = target.GetCommand(state)
	}
	rawHash := append(build.RuleHash(state, target, true, false), state.Hashes.Config...)
	t.Hash = base64.RawStdEncoding.EncodeToString(rawHash)
	t.Test = target.IsTest()
	t.Binary = target.IsBinary
	t.TestOnly = target.TestOnly
	return t
}

// makeJSONInputField takes a named and unnamed field (e.g. srcs or tools) and returns an
// appropriate representation.
func makeJSONInputField(graph *core.BuildGraph, unnamed []string, named map[string][]core.BuildInput) interface{} {
	if len(named) == 0 {
		if len(unnamed) == 0 {
			return nil
		}
		return unnamed
	}
	namedInputs := make(map[string][]string, len(named))
	for name, srcs := range named {
		namedInputs[name] = buildInputsToStrings(graph, srcs)
	}
	return namedInputs
}

func buildInputsToStrings(graph *core.BuildGraph, inputs []core.BuildInput) []string {
	s := make([]string, 0, len(inputs))
	for _, x := range inputs {
		s = append(s, x.Paths(graph)...)
	}
	return s
}
