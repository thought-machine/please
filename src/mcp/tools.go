package mcp

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/parse"
	"github.com/thought-machine/please/src/query"
)

// A targetsResult carries structured definitions of a set of targets, in the same
// format as plz query print --json.
type targetsResult struct {
	Targets map[string]map[string]any `json:"targets" jsonschema:"Target definitions keyed by build label, in plz query print --json format."`
}

// A pathResult is a targetsResult with an ordered path through the graph.
type pathResult struct {
	Path    []string                  `json:"path" jsonschema:"The dependency path, in order from the first target to the second."`
	Targets map[string]map[string]any `json:"targets" jsonschema:"Definitions of the targets on the path, keyed by build label."`
}

// A filesResult is a targetsResult with a mapping from query files to matching labels.
type filesResult struct {
	Files   map[string][]string       `json:"files" jsonschema:"Build labels matched for each queried file. Files matching no target map to an empty list."`
	Targets map[string]map[string]any `json:"targets" jsonschema:"Definitions of the matched targets, keyed by build label."`
}

// addTool registers a tool whose handler returns plain text.
func addTool[In any](
	srv *sdk.Server,
	name, description string,
	h func(ctx context.Context, in In) (string, error),
) {
	tool := &sdk.Tool{Name: name, Description: description}
	sdk.AddTool(srv, tool, func(
		ctx context.Context,
		req *sdk.CallToolRequest,
		in In,
	) (*sdk.CallToolResult, any, error) {
		out, err := h(ctx, in)
		if err != nil {
			return nil, nil, err
		}
		return textResult(out), nil, nil
	})
}

// addStructuredTool registers a tool whose handler returns a structured result;
// the SDK serialises it into the result's structured content.
func addStructuredTool[In, Out any](
	srv *sdk.Server,
	name, description string,
	h func(ctx context.Context, in In) (Out, error),
) {
	tool := &sdk.Tool{Name: name, Description: description}
	sdk.AddTool(srv, tool, func(
		ctx context.Context,
		req *sdk.CallToolRequest,
		in In,
	) (*sdk.CallToolResult, Out, error) {
		out, err := h(ctx, in)
		return nil, out, err
	})
}

type depsArgs struct {
	Targets []string `json:"targets" jsonschema:"Build labels to query, e.g. //src/core:core. Pseudo-targets like //src/... and //src/core:all are supported."`
	Hidden  bool     `json:"hidden,omitempty" jsonschema:"Include hidden targets (names beginning with an underscore)."`
	Level   int      `json:"level,omitempty" jsonschema:"Maximum depth to traverse; omit or 0 for unlimited."`
	Fields  []string `json:"fields,omitempty" jsonschema:"Restrict the returned target definitions to these fields (e.g. srcs, deps, outs). Omit for all fields."`
}

type revdepsArgs struct {
	Targets []string `json:"targets" jsonschema:"Build labels to query, e.g. //src/core:core. Pseudo-targets like //src/... and //src/core:all are supported."`
	Hidden  bool     `json:"hidden,omitempty" jsonschema:"Include hidden targets (names beginning with an underscore)."`
	Level   int      `json:"level,omitempty" jsonschema:"Levels of reverse dependencies to include; -1 for the full transitive set. Omitting it or 0 means 1 level, like plz query revdeps."`
	Fields  []string `json:"fields,omitempty" jsonschema:"Restrict the returned target definitions to these fields (e.g. srcs, deps, outs). Omit for all fields."`
}

type somepathArgs struct {
	From   string   `json:"from" jsonschema:"Build label to start from."`
	To     string   `json:"to" jsonschema:"Build label to find a path to."`
	Except []string `json:"except,omitempty" jsonschema:"Build labels to exclude from the path."`
	Hidden bool     `json:"hidden,omitempty" jsonschema:"Include hidden targets in the path."`
	Fields []string `json:"fields,omitempty" jsonschema:"Restrict the returned target definitions to these fields (e.g. srcs, deps, outs). Omit for all fields."`
}

type printArgs struct {
	Targets []string `json:"targets" jsonschema:"Build labels to print, e.g. //src/core:core."`
	Fields  []string `json:"fields,omitempty" jsonschema:"Restrict the returned target definitions to these fields (e.g. srcs, deps, outs). Omit for all fields."`
}

type alltargetsArgs struct {
	Targets []string `json:"targets,omitempty" jsonschema:"Packages to list targets in, e.g. //src/... . Omit to list the entire graph."`
	Hidden  bool     `json:"hidden,omitempty" jsonschema:"Include hidden targets (names beginning with an underscore)."`
	Fields  []string `json:"fields,omitempty" jsonschema:"Restrict the returned target definitions to these fields (e.g. srcs, deps, outs). Omit for all fields."`
}

type filterArgs struct {
	Targets []string `json:"targets,omitempty" jsonschema:"Build labels to filter. Omit to filter the entire graph."`
	Include []string `json:"include,omitempty" jsonschema:"Only include targets with at least one of these labels/tags."`
	Exclude []string `json:"exclude,omitempty" jsonschema:"Exclude targets with any of these labels/tags."`
	Hidden  bool     `json:"hidden,omitempty" jsonschema:"Include hidden targets (names beginning with an underscore)."`
	Fields  []string `json:"fields,omitempty" jsonschema:"Restrict the returned target definitions to these fields (e.g. srcs, deps, outs). Omit for all fields."`
}

type labelsArgs struct {
	Targets []string `json:"targets" jsonschema:"Build labels to query, e.g. //src/core:core."`
}

type outputsArgs struct {
	Targets []string `json:"targets" jsonschema:"Build labels to query, e.g. //src/core:core."`
	JSON    bool     `json:"json,omitempty" jsonschema:"Print the outputs as JSON."`
}

type whatinputsArgs struct {
	Files  []string `json:"files" jsonschema:"File paths relative to the repo root. Files that aren't an input to any target map to an empty list."`
	Hidden bool     `json:"hidden,omitempty" jsonschema:"Report hidden targets rather than their parent."`
	Fields []string `json:"fields,omitempty" jsonschema:"Restrict the returned target definitions to these fields (e.g. srcs, deps, outs). Omit for all fields."`
}

type whatoutputsArgs struct {
	Files  []string `json:"files" jsonschema:"Output file paths relative to the repo root (within plz-out)."`
	Fields []string `json:"fields,omitempty" jsonschema:"Restrict the returned target definitions to these fields (e.g. srcs, deps, outs). Omit for all fields."`
}

// registerTools registers all the query tools on the given MCP server.
func (s *server) registerTools(srv *sdk.Server) {
	s.registerGraphTools(srv)
	s.registerTargetTools(srv)
	s.registerFileTools(srv)
	s.registerAdminTools(srv)
}

// registerGraphTools registers the tools that walk the dependency graph.
func (s *server) registerGraphTools(srv *sdk.Server) {
	addStructuredTool(srv, "deps",
		"Returns the transitive dependencies of a set of build targets, with each target's definition.",
		func(ctx context.Context, in depsArgs) (targetsResult, error) {
			level := in.Level
			if level <= 0 {
				level = -1
			}
			return s.targetsQuery(in.Fields, func(state *core.BuildState) (core.BuildLabels, error) {
				labels, err := resolveLabels(state, in.Targets)
				if err != nil {
					return nil, err
				}
				return query.DepsLabels(state, labels, in.Hidden, level), nil
			})
		})

	addStructuredTool(srv, "revdeps",
		"Returns the targets that depend on a set of build targets (reverse dependencies), with each target's definition.",
		func(ctx context.Context, in revdepsArgs) (targetsResult, error) {
			level := in.Level
			if level == 0 {
				level = 1
			}
			return s.targetsQuery(in.Fields, func(state *core.BuildState) (core.BuildLabels, error) {
				labels, err := resolveLabels(state, in.Targets)
				if err != nil {
					return nil, err
				}
				return query.ReverseDepsLabels(state, labels, level, in.Hidden), nil
			})
		})

	addStructuredTool(srv, "somepath",
		"Finds a dependency path between two build targets, returning the path in order and each target's definition.",
		func(ctx context.Context, in somepathArgs) (pathResult, error) {
			ret := pathResult{}
			if err := validatePrintFields(in.Fields); err != nil {
				return ret, err
			}
			err := s.withState(func(state *core.BuildState) error {
				path, err := s.somePath(state, in)
				if err != nil {
					return err
				}
				ret.Path = labelStrings(path)
				ret.Targets = targetMaps(state, path, in.Fields)
				return nil
			})
			return ret, err
		})
}

// somePath resolves the labels in the given args and finds a path between them.
func (s *server) somePath(state *core.BuildState, in somepathArgs) (core.BuildLabels, error) {
	from, err := resolveLabels(state, []string{in.From})
	if err != nil {
		return nil, err
	}
	to, err := resolveLabels(state, []string{in.To})
	if err != nil {
		return nil, err
	}
	except := []core.BuildLabel{}
	if len(in.Except) > 0 {
		if except, err = resolveLabels(state, in.Except); err != nil {
			return nil, err
		}
	}
	return query.SomePathLabels(state.Graph, from, to, except, in.Hidden)
}

// registerTargetTools registers the tools that inspect individual targets and target sets.
func (s *server) registerTargetTools(srv *sdk.Server) {
	addStructuredTool(srv, "print",
		"Returns the definition of build targets as they exist in the build graph, in plz query print --json format.",
		func(ctx context.Context, in printArgs) (targetsResult, error) {
			return s.targetsQuery(in.Fields, func(state *core.BuildState) (core.BuildLabels, error) {
				return resolveLabels(state, in.Targets)
			})
		})

	addStructuredTool(srv, "alltargets",
		"Returns all the build targets in the graph, optionally filtered to a set of packages, with each target's definition.",
		func(ctx context.Context, in alltargetsArgs) (targetsResult, error) {
			return s.targetsQuery(in.Fields, func(state *core.BuildState) (core.BuildLabels, error) {
				labels, err := resolveWholeGraphLabels(state, in.Targets)
				if err != nil {
					return nil, err
				}
				return visibleLabels(labels, in.Hidden), nil
			})
		})

	addStructuredTool(srv, "filter",
		"Filters a set of targets by include/exclude labels (as passed to plz --include / --exclude), returning each match's definition.",
		func(ctx context.Context, in filterArgs) (targetsResult, error) {
			return s.targetsQuery(in.Fields, func(state *core.BuildState) (core.BuildLabels, error) {
				labels, err := resolveWholeGraphLabels(state, in.Targets)
				if err != nil {
					return nil, err
				}
				state.SetIncludeAndExclude(in.Include, in.Exclude)
				defer state.SetIncludeAndExclude(nil, nil)
				ret := core.BuildLabels{}
				for _, l := range visibleLabels(labels, in.Hidden) {
					if state.ShouldInclude(state.Graph.TargetOrDie(l)) {
						ret = append(ret, l)
					}
				}
				return ret, nil
			})
		})
}

// registerFileTools registers the tools that map between files and targets.
func (s *server) registerFileTools(srv *sdk.Server) {
	addStructuredTool(srv, "whatinputs",
		"Finds the build targets that the given files are inputs (sources) to, with each target's definition.",
		func(ctx context.Context, in whatinputsArgs) (filesResult, error) {
			return s.filesQuery(in.Fields, func(state *core.BuildState) map[string]core.BuildLabels {
				return query.WhatInputsLabels(state.Graph, in.Files, in.Hidden)
			})
		})

	addStructuredTool(srv, "whatoutputs",
		"Finds the build targets that produce the given output files, with each target's definition.",
		func(ctx context.Context, in whatoutputsArgs) (filesResult, error) {
			return s.filesQuery(in.Fields, func(state *core.BuildState) map[string]core.BuildLabels {
				return query.WhatOutputsLabels(state.Graph, in.Files)
			})
		})

	addTool(srv, "inputs",
		"Lists all the input files (sources) of a set of build targets.",
		func(ctx context.Context, in labelsArgs) (string, error) {
			return s.runQuery(func(state *core.BuildState) error {
				labels, err := resolveLabels(state, in.Targets)
				if err != nil {
					return err
				}
				query.TargetInputs(state.Graph, labels)
				return nil
			})
		})

	addTool(srv, "outputs",
		"Lists the output files of a set of build targets.",
		func(ctx context.Context, in outputsArgs) (string, error) {
			return s.runQuery(func(state *core.BuildState) error {
				labels, err := resolveLabels(state, in.Targets)
				if err != nil {
					return err
				}
				query.TargetOutputs(state.Graph, labels, in.JSON)
				return nil
			})
		})
}

// registerAdminTools registers the tools that manage the server itself.
func (s *server) registerAdminTools(srv *sdk.Server) {
	addTool(srv, "reload_graph",
		"Re-parses the build graph from the BUILD files on disk. Use this after BUILD files have changed; other queries are answered from a cached graph. Configuration (.plzconfig) changes require a server restart.",
		func(ctx context.Context, in struct{}) (string, error) {
			s.mu.Lock()
			defer s.mu.Unlock()
			if err := s.parseGraph(); err != nil {
				return "", err
			}
			return fmt.Sprintf("Build graph reloaded: %d targets in %d packages.",
				len(s.state.Graph.AllTargets()), len(s.state.Graph.PackageMap())), nil
		})
}

// targetsQuery runs f to obtain a set of labels and returns their structured definitions.
func (s *server) targetsQuery(
	fields []string,
	f func(state *core.BuildState) (core.BuildLabels, error),
) (targetsResult, error) {
	ret := targetsResult{}
	if err := validatePrintFields(fields); err != nil {
		return ret, err
	}
	err := s.withState(func(state *core.BuildState) error {
		labels, err := f(state)
		if err != nil {
			return err
		}
		ret.Targets = targetMaps(state, labels, fields)
		return nil
	})
	return ret, err
}

// filesQuery runs f to obtain a file-to-labels mapping and returns it along with the
// structured definitions of all matched targets.
func (s *server) filesQuery(
	fields []string,
	f func(state *core.BuildState) map[string]core.BuildLabels,
) (filesResult, error) {
	ret := filesResult{}
	if err := validatePrintFields(fields); err != nil {
		return ret, err
	}
	err := s.withState(func(state *core.BuildState) error {
		ret.Files = map[string][]string{}
		all := core.BuildLabels{}
		for file, labels := range f(state) {
			ret.Files[file] = labelStrings(labels)
			all = append(all, labels...)
		}
		ret.Targets = targetMaps(state, all, fields)
		return nil
	})
	return ret, err
}

// targetMaps builds the print-style JSON representation of each of the given targets.
func targetMaps(
	state *core.BuildState,
	labels core.BuildLabels,
	fields []string,
) map[string]map[string]any {
	order := parse.BuildRuleArgOrder(state)
	ret := make(map[string]map[string]any, len(labels))
	for _, l := range labels {
		if target := state.Graph.Target(l); target != nil {
			ret[l.String()] = query.TargetToMap(order, fields, target)
		}
	}
	return ret
}

// labelStrings converts a set of build labels to their string forms.
func labelStrings(labels core.BuildLabels) []string {
	ret := make([]string, len(labels))
	for i, l := range labels {
		ret[i] = l.String()
	}
	return ret
}

// visibleLabels filters out hidden targets unless hidden is set.
func visibleLabels(labels core.BuildLabels, hidden bool) core.BuildLabels {
	if hidden {
		return labels
	}
	ret := make(core.BuildLabels, 0, len(labels))
	for _, l := range labels {
		if !strings.HasPrefix(l.Name, "_") {
			ret = append(ret, l)
		}
	}
	return ret
}

// resolveWholeGraphLabels is like resolveLabels but treats an empty input as the whole graph.
func resolveWholeGraphLabels(state *core.BuildState, in []string) ([]core.BuildLabel, error) {
	if len(in) == 0 {
		return state.ExpandLabels(core.WholeGraph), nil
	}
	return resolveLabels(state, in)
}

// validatePrintFields checks that the given field names exist on a build target;
// query.Print dies on unknown fields, which would take the server with it.
func validatePrintFields(fields []string) error {
	if len(fields) == 0 {
		return nil
	}
	valid := validPrintFieldNames()
	for _, f := range fields {
		if _, present := valid[f]; !present {
			names := make([]string, 0, len(valid))
			for name := range valid {
				names = append(names, name)
			}
			sort.Strings(names)
			return fmt.Errorf("unknown field %s; known fields are: %s", f, strings.Join(names, ", "))
		}
	}
	return nil
}

// validPrintFieldNames returns the set of field names query.Print understands,
// mirroring its name resolution (the 'name' struct tag, else the lowercased field name).
func validPrintFieldNames() map[string]struct{} {
	valid := map[string]struct{}{}
	add := func(t reflect.Type) {
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if name := f.Tag.Get("name"); name != "" {
				valid[name] = struct{}{}
			} else {
				valid[strings.ToLower(f.Name)] = struct{}{}
			}
		}
	}
	targetType := reflect.TypeOf(core.BuildTarget{})
	add(targetType)
	if testField, ok := targetType.FieldByName("Test"); ok {
		add(testField.Type.Elem())
	}
	return valid
}
