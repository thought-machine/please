// Package mcp implements a Model Context Protocol server that exposes plz query
// functionality over stdio. The build graph is parsed once at startup and kept
// in memory between queries, so clients don't pay the graph construction cost
// on every query as they would invoking plz directly.
package mcp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sync"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/thought-machine/please/src/cli/logging"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/parse"
	"github.com/thought-machine/please/src/plz"
	"github.com/thought-machine/please/src/version"
)

var log = logging.Log

// A server holds the cached build state that queries are served from.
type server struct {
	// mu serialises queries and reloads; queries capture os.Stdout which is process-global.
	mu     sync.Mutex
	state  *core.BuildState
	config *core.Configuration
}

// Serve parses the build graph, then runs an MCP server over stdio until the
// client disconnects or ctx is cancelled.
func Serve(ctx context.Context, config *core.Configuration) error {
	// The MCP protocol runs over stdout, so anything else that writes there would
	// corrupt the framing. Point os.Stdout at stderr for the life of the process
	// and hand the real stdout to the transport; queries capture os.Stdout per-call.
	protocolOut := os.Stdout
	os.Stdout = os.Stderr

	s := &server{config: config}
	log.Notice("Parsing build graph...")
	if err := s.parseGraph(); err != nil {
		return err
	}
	log.Notice("Parsed %d targets; serving MCP over stdio", len(s.state.Graph.AllTargets()))

	srv := sdk.NewServer(&sdk.Implementation{
		Name:    "please",
		Title:   "Please build system",
		Version: version.PleaseVersion,
	}, nil)
	s.registerTools(srv)
	return srv.Run(ctx, &sdk.IOTransport{Reader: os.Stdin, Writer: protocolOut})
}

// parseGraph parses the entire build graph into a fresh build state.
// On success the new state replaces the current one; on failure the old state is kept.
// Callers must hold s.mu (except before the server has started).
func (s *server) parseGraph() error {
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

// withState runs f against the current build state under the server lock,
// converting panics into errors so a misbehaving query can't kill the server.
func (s *server) withState(f func(state *core.BuildState) error) (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("query failed: %v", p)
		}
	}()
	return f(s.state)
}

// runQuery runs f against the current build state under the server lock,
// capturing anything it writes to os.Stdout and returning it.
func (s *server) runQuery(f func(state *core.BuildState) error) (out string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, w, pipeErr := os.Pipe()
	if pipeErr != nil {
		return "", pipeErr
	}
	oldStdout := os.Stdout
	os.Stdout = w
	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		io.Copy(&buf, r)
		close(done)
	}()
	defer func() {
		os.Stdout = oldStdout
		w.Close()
		<-done
		r.Close()
		if p := recover(); p != nil {
			err = fmt.Errorf("query failed: %v", p)
		}
		if err == nil {
			out = buf.String()
		}
	}()
	err = f(s.state)
	return out, err
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
