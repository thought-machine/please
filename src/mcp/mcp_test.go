package mcp_test

import (
	"context"
	"encoding/json"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/mcp"
)

// targetsResult mirrors the structured result the target-returning tools produce.
type targetsResult struct {
	Targets map[string]map[string]any `json:"targets"`
}

// filesResult mirrors the structured result the file-mapping tools produce.
type filesResult struct {
	Files   map[string][]string       `json:"files"`
	Targets map[string]map[string]any `json:"targets"`
}

func TestListTools(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	session := newTestSession(t, testState())
	res, err := session.ListTools(t.Context(), nil)
	r.NoError(err)

	names := make([]string, len(res.Tools))
	for i, tool := range res.Tools {
		names[i] = tool.Name
	}
	a.ElementsMatch([]string{
		"deps", "revdeps", "somepath", "print", "alltargets", "filter",
		"whatinputs", "whatoutputs", "inputs", "outputs", "reload_graph",
	}, names)
}

func TestTargetQueries(t *testing.T) {
	tests := []struct {
		name            string
		tool            string
		args            map[string]any
		expectedTargets []string
	}{
		{
			name:            "print",
			tool:            "print",
			args:            map[string]any{"targets": []string{"//package1:target1"}},
			expectedTargets: []string{"//package1:target1"},
		},
		{
			// deps reports the dependencies of the queried targets, not the targets themselves.
			name:            "deps",
			tool:            "deps",
			args:            map[string]any{"targets": []string{"//package1:target1"}},
			expectedTargets: []string{"//package2:target2"},
		},
		{
			name:            "revdeps",
			tool:            "revdeps",
			args:            map[string]any{"targets": []string{"//package2:target2"}},
			expectedTargets: []string{"//package1:target1"},
		},
		{
			name:            "alltargets",
			tool:            "alltargets",
			args:            map[string]any{},
			expectedTargets: []string{"//package1:target1", "//package2:target2"},
		},
		{
			name:            "pseudo-target expansion",
			tool:            "print",
			args:            map[string]any{"targets": []string{"//package1:all"}},
			expectedTargets: []string{"//package1:target1"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a := assert.New(t)
			r := require.New(t)

			session := newTestSession(t, testState())
			res, err := session.CallTool(t.Context(), &sdk.CallToolParams{
				Name:      test.tool,
				Arguments: test.args,
			})
			r.NoError(err)
			r.False(res.IsError, "tool returned an error: %s", contentText(res))

			var out targetsResult
			decodeStructured(t, res, &out)
			a.ElementsMatch(test.expectedTargets, keys(out.Targets))
		})
	}
}

func TestFieldsAreRestricted(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	session := newTestSession(t, testState())
	res, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: "print",
		Arguments: map[string]any{
			"targets": []string{"//package1:target1"},
			"fields":  []string{"deps"},
		},
	})
	r.NoError(err)
	r.False(res.IsError, "tool returned an error: %s", contentText(res))

	var out targetsResult
	decodeStructured(t, res, &out)
	r.Contains(out.Targets, "//package1:target1")
	a.Equal([]string{"deps"}, keys(out.Targets["//package1:target1"]))
}

func TestWhatInputs(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	session := newTestSession(t, testState())
	res, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name:      "whatinputs",
		Arguments: map[string]any{"files": []string{"package1/file1.txt"}},
	})
	r.NoError(err)
	r.False(res.IsError, "tool returned an error: %s", contentText(res))

	var out filesResult
	decodeStructured(t, res, &out)
	a.Equal(map[string][]string{
		"package1/file1.txt": {"//package1:target1"},
	}, out.Files)
	a.ElementsMatch([]string{"//package1:target1"}, keys(out.Targets))
}

// The text-returning tools write through an io.Writer rather than stdout, so their
// output arrives as text content.
func TestTextQueries(t *testing.T) {
	tests := []struct {
		name         string
		tool         string
		args         map[string]any
		expectedText string
	}{
		{
			name:         "inputs",
			tool:         "inputs",
			args:         map[string]any{"targets": []string{"//package1:target1"}},
			expectedText: "package1/file1.txt\n",
		},
		{
			name:         "outputs",
			tool:         "outputs",
			args:         map[string]any{"targets": []string{"//package1:target1"}},
			expectedText: "plz-out/gen/package1/out1.txt\n",
		},
		{
			name: "outputs as JSON",
			tool: "outputs",
			args: map[string]any{
				"targets": []string{"//package1:target1"},
				"json":    true,
			},
			expectedText: "{\n  \"//package1:target1\": [\n    \"plz-out/gen/package1/out1.txt\"\n  ]\n}\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a := assert.New(t)
			r := require.New(t)

			session := newTestSession(t, testState())
			res, err := session.CallTool(t.Context(), &sdk.CallToolParams{
				Name:      test.tool,
				Arguments: test.args,
			})
			r.NoError(err)
			r.False(res.IsError, "tool returned an error: %s", contentText(res))
			a.Equal(test.expectedText, contentText(res))
		})
	}
}

// Unknown targets must come back as tool errors; the underlying query functions
// call TargetOrDie, which would otherwise take the server down with it.
func TestErrorsAreReportedNotFatal(t *testing.T) {
	tests := []struct {
		name            string
		tool            string
		args            map[string]any
		expectedMessage string
	}{
		{
			name:            "unknown target",
			tool:            "print",
			args:            map[string]any{"targets": []string{"//package1:nope"}},
			expectedMessage: "not found in the build graph",
		},
		{
			name:            "invalid label",
			tool:            "print",
			args:            map[string]any{"targets": []string{"not a label"}},
			expectedMessage: "invalid build label",
		},
		{
			name: "unknown field",
			tool: "print",
			args: map[string]any{
				"targets": []string{"//package1:target1"},
				"fields":  []string{"nonsense"},
			},
			expectedMessage: "unknown field nonsense",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a := assert.New(t)
			r := require.New(t)

			session := newTestSession(t, testState())
			res, err := session.CallTool(t.Context(), &sdk.CallToolParams{
				Name:      test.tool,
				Arguments: test.args,
			})
			r.NoError(err)
			a.True(res.IsError)
			a.Contains(contentText(res), test.expectedMessage)

			// The session is still usable afterwards.
			_, err = session.ListTools(t.Context(), nil)
			a.NoError(err)
		})
	}
}

// newTestSession starts a server serving the given state over an in-memory
// transport and returns a client session connected to it.
func newTestSession(t *testing.T, state *core.BuildState) *sdk.ClientSession {
	t.Helper()
	r := require.New(t)

	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)

	serverTransport, clientTransport := sdk.NewInMemoryTransports()
	srv := mcp.NewServer(
		state.Config,
		mcp.WithTransport(serverTransport),
		mcp.WithState(state),
	)
	done := make(chan struct{})
	go func() {
		defer close(done)
		srv.Serve(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})

	client := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "1"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	r.NoError(err)
	t.Cleanup(func() { session.Close() })
	return session
}

// testState returns a build state with a small hand-built graph in it.
func testState() *core.BuildState {
	state := core.NewDefaultBuildState()
	pkg1 := core.NewPackage("package1")
	pkg2 := core.NewPackage("package2")

	target2 := core.NewBuildTarget(core.NewBuildLabel(pkg2.Name, "target2"))
	target1 := core.NewBuildTarget(core.NewBuildLabel(pkg1.Name, "target1"))
	target1.AddSource(core.FileLabel{File: "file1.txt", Package: pkg1.Name})
	target1.AddDependency(target2.Label)
	target1.AddOutput("out1.txt")

	for _, target := range []*core.BuildTarget{target1, target2} {
		state.Graph.AddTarget(target)
	}
	pkg1.AddTarget(target1)
	pkg2.AddTarget(target2)
	state.Graph.AddPackage(pkg1)
	state.Graph.AddPackage(pkg2)
	return state
}

// decodeStructured decodes the structured content of a tool result into out.
func decodeStructured(t *testing.T, res *sdk.CallToolResult, out any) {
	t.Helper()
	r := require.New(t)

	b, err := json.Marshal(res.StructuredContent)
	r.NoError(err)
	r.NoError(json.Unmarshal(b, out))
}

// contentText returns the concatenated text content of a tool result.
func contentText(res *sdk.CallToolResult) string {
	text := ""
	for _, content := range res.Content {
		if tc, ok := content.(*sdk.TextContent); ok {
			text += tc.Text
		}
	}
	return text
}

func keys[V any](m map[string]V) []string {
	ret := make([]string, 0, len(m))
	for k := range m {
		ret = append(ret, k)
	}
	return ret
}
