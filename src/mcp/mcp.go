// Package mcp implements a Model Context Protocol server that exposes plz query
// functionality over stdio. The build graph is parsed once at startup and kept
// in memory between queries, so clients don't pay the graph construction cost
// on every query as they would invoking plz directly.
package mcp

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/thought-machine/please/src/cli/logging"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/parse"
	"github.com/thought-machine/please/src/plz"
	"github.com/thought-machine/please/src/version"
)

var log = logging.Log

// A Server holds the cached build state that queries are served from.
type Server struct {
	// mu serialises queries and reloads; reloads swap out the state, and some
	// queries (filter) temporarily mutate it.
	mu     sync.Mutex
	state  *core.BuildState
	graph  *core.BuildGraph
	config *core.Configuration

	transport sdk.Transport
}

// Option provides a mechanism to set options on the MCP server.
type Option func(*Server)

// WithTransport overrides the transport that the server receives requests on and
// sends responses over. The default is stdio. Tests can pass one half of
// sdk.NewInMemoryTransports() and drive the server with a real MCP client.
func WithTransport(t sdk.Transport) Option {
	return func(s *Server) {
		s.transport = t
	}
}

// WithState supplies a pre-parsed build state to serve queries from, in place of
// parsing the build graph at startup.
func WithState(state *core.BuildState) Option {
	return func(s *Server) {
		s.state = state
	}
}

// NewServer instantiates a new MCP server.
func NewServer(config *core.Configuration, opts ...Option) *Server {
	s := &Server{
		config: config,
		graph:  core.NewGraph(),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Serve runs an MCP server until the client disconnects or ctx is cancelled.
// By default, the server uses lazy loading to parse the build graph on demand.
func (s *Server) Serve(ctx context.Context) error {
	if s.transport == nil {
		s.transport = stdioTransport()
	}
	if s.state == nil {
		log.Notice("Serving MCP with lazy-loaded build graph...")
	} else {
		log.Notice("Serving MCP for %d targets", len(s.state.Graph.AllTargets()))
	}

	srv := sdk.NewServer(&sdk.Implementation{
		Name:    "please",
		Title:   "Please build system",
		Version: version.PleaseVersion,
	}, nil)
	s.registerTools(srv)
	return srv.Run(ctx, s.transport)
}

// stdioTransport returns a transport communicating over stdin and stdout.
// The MCP protocol runs over stdout, so anything else that writes there would
// corrupt the framing. Point os.Stdout at stderr for the life of the process
// and hand the real stdout to the transport; queries capture os.Stdout per-call.
// Stdout is wrapped so that the transport doesn't close it when the session ends.
func stdioTransport() sdk.Transport {
	out := os.Stdout
	os.Stdout = os.Stderr
	return &sdk.IOTransport{Reader: os.Stdin, Writer: nopCloserWriter{out}}
}

// nopCloserWriter is an io.WriteCloser with a trivial Close method.
type nopCloserWriter struct {
	io.Writer
}

func (nopCloserWriter) Close() error { return nil }

// parseGraph parses the entire build graph into a fresh build state.
// On success the new state replaces the current one; on failure the old state is kept.
// Callers must hold s.mu (except before the server has started).
func (s *Server) parseGraph() error {
	state := core.NewBuildState(s.config)
	state.NeedBuild = false
	parse.InitParser(state)
	plz.RunHost(core.WholeGraph, state)
	if failed, _, _ := state.Failures(); failed {
		return fmt.Errorf("failed to parse the build graph; see server logs for details")
	}
	s.state = state
	return nil
}

// withState runs f against a build state under the server lock, converting panics
// into errors so a misbehaving query can't kill the server.
// Under lazy-loading, it parses the given targets on demand into the persistent graph.
func (s *Server) withState(targets []string, f func(state *core.BuildState) error) (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("query failed: %v", p)
		}
	}()

	if s.state != nil {
		return f(s.state)
	}

	state := core.NewBuildState(s.config)
	state.NeedBuild = false
	state.Graph = s.graph

	if len(targets) == 0 {
		return f(state)
	}

	labels := make([]core.BuildLabel, 0, len(targets))
	for _, t := range targets {
		l, err := core.TryParseBuildLabel(t, "", "")
		if err != nil {
			return fmt.Errorf("invalid build label %q: %w", t, err)
		}
		labels = append(labels, l)
	}
	plz.RunHost(labels, state)
	if failed, _, _ := state.Failures(); failed {
		var errs []string
		for r := range state.Results() {
			if r.Status.IsFailure() {
				errs = append(errs, fmt.Sprintf("%s (%s): %s", r.Label, r.Status, r.Err))
			}
		}
		return fmt.Errorf("failed to parse the build graph: %s", strings.Join(errs, "; "))
	}

	return f(state)
}

// resolveLabels parses a set of label strings, expands pseudo-targets (:all and /...)
// against the graph and verifies that every resulting target exists.
// Verification matters: the query functions call TargetOrDie on the labels they're
// given, which would kill the server on an unknown target.
func resolveLabels(state *core.BuildState, in []string) ([]core.BuildLabel, error) {
	labels := make([]core.BuildLabel, 0, len(in))
	for _, l := range in {
		label, err := core.TryParseBuildLabel(l, "", "")
		if err != nil {
			return nil, fmt.Errorf("invalid build label %q: %w", l, err)
		}
		labels = append(labels, label)
	}
	expanded := state.ExpandLabels(labels)
	for _, l := range expanded {
		if state.Graph.Target(l) == nil {
			return nil, fmt.Errorf("target %s not found in the build graph", l)
		}
	}
	if len(expanded) == 0 {
		return nil, fmt.Errorf("no targets matched the given labels")
	}
	return expanded, nil
}

// textResult wraps a string as an MCP tool result.
func textResult(text string) *sdk.CallToolResult {
	if text == "" {
		text = "(no output)"
	}
	return &sdk.CallToolResult{
		Content: []sdk.Content{&sdk.TextContent{Text: text}},
	}
}
