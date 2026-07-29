package mcp

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/query"
)

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

type depsArgs struct {
	Targets []string `json:"targets" jsonschema:"Build labels to query, e.g. //src/core:core. Pseudo-targets like //src/... and //src/core:all are supported."`
	Hidden  bool     `json:"hidden,omitempty" jsonschema:"Include hidden targets (names beginning with an underscore)."`
	Level   int      `json:"level,omitempty" jsonschema:"Maximum depth to traverse; omit or 0 for unlimited."`
	DOT     bool     `json:"dot,omitempty" jsonschema:"Output the dependency graph in Graphviz DOT format."`
}

type revdepsArgs struct {
	Targets []string `json:"targets" jsonschema:"Build labels to query, e.g. //src/core:core. Pseudo-targets like //src/... and //src/core:all are supported."`
	Hidden  bool     `json:"hidden,omitempty" jsonschema:"Include hidden targets (names beginning with an underscore)."`
	Level   int      `json:"level,omitempty" jsonschema:"Levels of reverse dependencies to include; -1 for the full transitive set. Omitting it or 0 means 1 level, like plz query revdeps."`
}

type somepathArgs struct {
	From   string   `json:"from" jsonschema:"Build label to start from."`
	To     string   `json:"to" jsonschema:"Build label to find a path to."`
	Except []string `json:"except,omitempty" jsonschema:"Build labels to exclude from the path."`
	Hidden bool     `json:"hidden,omitempty" jsonschema:"Include hidden targets in the path."`
}

type printArgs struct {
	Targets    []string `json:"targets" jsonschema:"Build labels to print, e.g. //src/core:core."`
	Fields     []string `json:"fields,omitempty" jsonschema:"Only print these fields of each target (e.g. srcs, deps, outs)."`
	Labels     []string `json:"labels,omitempty" jsonschema:"Only print labels of each target with these prefixes."`
	OmitHidden bool     `json:"omit_hidden,omitempty" jsonschema:"Omit hidden fields from the output."`
	JSON       bool     `json:"json,omitempty" jsonschema:"Print the targets as JSON."`
}

type alltargetsArgs struct {
	Targets []string `json:"targets,omitempty" jsonschema:"Packages to list targets in, e.g. //src/... . Omit to list the entire graph."`
	Hidden  bool     `json:"hidden,omitempty" jsonschema:"Include hidden targets (names beginning with an underscore)."`
}

type filterArgs struct {
	Targets []string `json:"targets,omitempty" jsonschema:"Build labels to filter. Omit to filter the entire graph."`
	Include []string `json:"include,omitempty" jsonschema:"Only include targets with at least one of these labels/tags."`
	Exclude []string `json:"exclude,omitempty" jsonschema:"Exclude targets with any of these labels/tags."`
	Hidden  bool     `json:"hidden,omitempty" jsonschema:"Include hidden targets (names beginning with an underscore)."`
}

type labelsArgs struct {
	Targets []string `json:"targets" jsonschema:"Build labels to query, e.g. //src/core:core."`
}

type outputsArgs struct {
	Targets []string `json:"targets" jsonschema:"Build labels to query, e.g. //src/core:core."`
	JSON    bool     `json:"json,omitempty" jsonschema:"Print the outputs as JSON."`
}

type whatinputsArgs struct {
	Files     []string `json:"files" jsonschema:"File paths relative to the repo root. Files that aren't an input to any target are silently ignored."`
	Hidden    bool     `json:"hidden,omitempty" jsonschema:"Report hidden targets rather than their parent."`
	EchoFiles bool     `json:"echo_files,omitempty" jsonschema:"Echo the file name alongside each target."`
}

type whatoutputsArgs struct {
	Files     []string `json:"files" jsonschema:"Output file paths relative to the repo root (within plz-out)."`
	EchoFiles bool     `json:"echo_files,omitempty" jsonschema:"Echo the file name alongside each target."`
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
	addTool(srv, "deps", "Lists the transitive dependencies of a set of build targets.", func(ctx context.Context, in depsArgs) (string, error) {
		level := in.Level
		if level <= 0 {
			level = -1
		}
		return s.runQuery(func(state *core.BuildState) error {
			labels, err := resolveLabels(state, in.Targets)
			if err != nil {
				return err
			}
			// os.Stdout is the capture pipe at this point (see runQuery).
			query.Deps(os.Stdout, state, labels, in.Hidden, level, in.DOT)
			return nil
		})
	})

	addTool(srv, "revdeps", "Lists the targets that depend on a set of build targets (reverse dependencies).", func(ctx context.Context, in revdepsArgs) (string, error) {
		level := in.Level
		if level == 0 {
			level = 1
		}
		return s.runQuery(func(state *core.BuildState) error {
			labels, err := resolveLabels(state, in.Targets)
			if err != nil {
				return err
			}
			query.ReverseDeps(state, labels, level, in.Hidden)
			return nil
		})
	})

	addTool(srv, "somepath", "Finds a dependency path between two build targets, if one exists.", func(ctx context.Context, in somepathArgs) (string, error) {
		return s.runQuery(func(state *core.BuildState) error {
			from, err := resolveLabels(state, []string{in.From})
			if err != nil {
				return err
			}
			to, err := resolveLabels(state, []string{in.To})
			if err != nil {
				return err
			}
			except := []core.BuildLabel{}
			if len(in.Except) > 0 {
				if except, err = resolveLabels(state, in.Except); err != nil {
					return err
				}
			}
			return query.SomePath(state.Graph, from, to, except, in.Hidden)
		})
	})

}

// registerTargetTools registers the tools that inspect individual targets and target sets.
func (s *server) registerTargetTools(srv *sdk.Server) {
	addTool(srv, "print", "Prints the definition of build targets as they exist in the build graph, optionally restricted to specific fields.", func(ctx context.Context, in printArgs) (string, error) {
		if err := validatePrintFields(in.Fields); err != nil {
			return "", err
		}
		return s.runQuery(func(state *core.BuildState) error {
			labels, err := resolveLabels(state, in.Targets)
			if err != nil {
				return err
			}
			query.Print(state, labels, in.Fields, in.Labels, in.OmitHidden, in.JSON)
			return nil
		})
	})

	addTool(srv, "alltargets", "Lists all the build targets in the graph, optionally filtered to a set of packages.", func(ctx context.Context, in alltargetsArgs) (string, error) {
		return s.runQuery(func(state *core.BuildState) error {
			labels, err := resolveWholeGraphLabels(state, in.Targets)
			if err != nil {
				return err
			}
			query.AllTargets(state.Graph, labels, in.Hidden)
			return nil
		})
	})

	addTool(srv, "filter", "Filters a set of targets by include/exclude labels (as passed to plz --include / --exclude).", func(ctx context.Context, in filterArgs) (string, error) {
		return s.runQuery(func(state *core.BuildState) error {
			labels, err := resolveWholeGraphLabels(state, in.Targets)
			if err != nil {
				return err
			}
			state.SetIncludeAndExclude(in.Include, in.Exclude)
			defer state.SetIncludeAndExclude(nil, nil)
			query.Filter(state, labels, in.Hidden)
			return nil
		})
	})

}

// registerFileTools registers the tools that map between files and targets.
func (s *server) registerFileTools(srv *sdk.Server) {
	addTool(srv, "inputs", "Lists all the input files (sources) of a set of build targets.", func(ctx context.Context, in labelsArgs) (string, error) {
		return s.runQuery(func(state *core.BuildState) error {
			labels, err := resolveLabels(state, in.Targets)
			if err != nil {
				return err
			}
			query.TargetInputs(state.Graph, labels)
			return nil
		})
	})

	addTool(srv, "outputs", "Lists the output files of a set of build targets.", func(ctx context.Context, in outputsArgs) (string, error) {
		return s.runQuery(func(state *core.BuildState) error {
			labels, err := resolveLabels(state, in.Targets)
			if err != nil {
				return err
			}
			query.TargetOutputs(state.Graph, labels, in.JSON)
			return nil
		})
	})

	addTool(srv, "whatinputs", "Finds the build targets that the given files are inputs (sources) to.", func(ctx context.Context, in whatinputsArgs) (string, error) {
		return s.runQuery(func(state *core.BuildState) error {
			// ignoreUnknown is always true; the alternative kills the process on unknown files.
			query.WhatInputs(state.Graph, in.Files, in.Hidden, in.EchoFiles, true)
			return nil
		})
	})

	addTool(srv, "whatoutputs", "Finds the build targets that produce the given output files.", func(ctx context.Context, in whatoutputsArgs) (string, error) {
		return s.runQuery(func(state *core.BuildState) error {
			query.WhatOutputs(state.Graph, in.Files, in.EchoFiles)
			return nil
		})
	})

}

// registerAdminTools registers the tools that manage the server itself.
func (s *server) registerAdminTools(srv *sdk.Server) {
	addTool(srv, "reload_graph", "Re-parses the build graph from the BUILD files on disk. Use this after BUILD files have changed; other queries are answered from a cached graph. Configuration (.plzconfig) changes require a server restart.", func(ctx context.Context, in struct{}) (string, error) {
		s.mu.Lock()
		defer s.mu.Unlock()
		if err := s.parseGraph(); err != nil {
			return "", err
		}
		return fmt.Sprintf("Build graph reloaded: %d targets in %d packages.",
			len(s.state.Graph.AllTargets()), len(s.state.Graph.PackageMap())), nil
	})
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
