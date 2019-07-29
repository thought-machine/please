// Package worker implements functions for communicating with subordinate worker processes.
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"

	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/core"
)

var log = logging.MustGetLogger("worker")

// A workerServer is the structure we use to maintain information about a remote work server.
type workerServer struct {
	requests      chan *Request
	responses     map[string]chan *Response
	responseMutex sync.Mutex
	process       *exec.Cmd
	stderr        *stderrLogger
	state         *core.BuildState
	closing       bool
}

// workerMap contains all the remote workers we've started so far.
var workerMap = map[string]*workerServer{}
var workerMutex sync.Mutex

// BuildRemotely runs a single build request and returns its response.
func BuildRemotely(state *core.BuildState, target *core.BuildTarget, worker string, req *Request) (*Response, error) {
	return buildRemotely(state, target, worker, "building (using "+worker+")", req)
}

func buildRemotely(state *core.BuildState, target *core.BuildTarget, worker, msg string, req *Request) (*Response, error) {
	w, err := getOrStartWorker(state, worker)
	if err != nil {
		return nil, err
	}
	ch := make(chan *Response, 2)
	w.responseMutex.Lock()
	w.responses[req.Rule] = ch
	w.responseMutex.Unlock()

	if target != nil {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go state.ProcessExecutor.LogProgress(ctx, target)
	}

	// Time out this request appropriately
	ctx, cancel := context.WithTimeout(context.Background(), target.BuildTimeout)
	defer cancel()
	w.requests <- req
	select {
	case response := <-ch:
		return response, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// ProvideParse sends a request to a subprocess to derive pseudo-contents of a BUILD file from
// a directory (e.g. they may infer it from file contents).
// If the provider cannot infer anything, they will return an empty string.
func ProvideParse(state *core.BuildState, worker string, dir string) (string, error) {
	w, err := getOrStartWorker(state, worker)
	if err != nil {
		return "", err
	}
	w.requests <- &Request{
		Rule: dir,
	}
	ch := make(chan *Response, 1)
	w.responseMutex.Lock()
	w.responses[dir] = ch
	w.responseMutex.Unlock()
	response := <-ch
	return response.BuildFile, nil
}

// EnsureWorkerStarted ensures that a worker server is started and has responded saying it's ready.
func EnsureWorkerStarted(state *core.BuildState, worker, test string, target *core.BuildTarget) (*Response, error) {
	resp, err := buildRemotely(state, target, worker, "waiting for "+worker+" to start", &Request{
		Rule:    target.Label.String(),
		Test:    true,
		Options: []string{test},
	})
	if err == nil && !resp.Success {
		return nil, fmt.Errorf(strings.Join(resp.Messages, "\n"))
	}
	return resp, err
}

// getOrStartWorker either retrieves an existing worker process or starts a new one.
func getOrStartWorker(state *core.BuildState, worker string) (*workerServer, error) {
	workerMutex.Lock()
	defer workerMutex.Unlock()
	if w, present := workerMap[worker]; present {
		return w, nil
	}
	// Need to create a new process
	if !strings.Contains(worker, "/") {
		path, err := core.LookBuildPath(worker, state.Config)
		if err != nil {
			return nil, err
		}
		worker = path
	}
	cmd := state.ProcessExecutor.ExecCommand(worker)
	cmd.Env = core.GeneralBuildEnvironment(state.Config)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr := &stderrLogger{}
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	w := &workerServer{
		requests:  make(chan *Request),
		responses: map[string]chan *Response{},
		process:   cmd,
		stderr:    stderr,
		state:     state,
	}
	workerMap[worker] = w
	go w.sendRequests(stdin)
	go w.readResponses(stdout)
	go w.wait()
	state.Stats.NumWorkerProcesses = len(workerMap)
	return w, nil
}

// sendRequests sends requests to a running worker server.
func (w *workerServer) sendRequests(stdin io.Writer) {
	e := json.NewEncoder(stdin)
	for request := range w.requests {
		if err := e.Encode(request); err != nil {
			w.dispatchResponse(&Response{
				Rule:     request.Rule,
				Success:  false,
				Messages: []string{err.Error()},
			})
			continue
		}
		stdin.Write([]byte{'\n'}) // Newline delimit them as a nicety.
	}
}

// readResponses reads the responses from a running worker server and dispatches them appropriately.
func (w *workerServer) readResponses(stdout io.Reader) {
	decoder := json.NewDecoder(stdout)
	for {
		response := Response{}
		if err := decoder.Decode(&response); err != nil {
			w.Error("Failed to read response: %s", err)
			break
		}
		w.dispatchResponse(&response)
	}
}

// dispatchResponse sends a single response on the appropriate channel.
func (w *workerServer) dispatchResponse(response *Response) {
	w.responseMutex.Lock()
	ch, present := w.responses[response.Rule]
	delete(w.responses, response.Rule)
	w.responseMutex.Unlock()
	if present {
		log.Debug("Got response from remote worker for %s, success: %v", response.Rule, response.Success)
		ch <- response
	} else {
		w.Error("Couldn't find response channel for %s", response.Rule)
	}
}

// wait waits for the process to terminate. If it dies unexpectedly this handles various failures.
func (w *workerServer) wait() {
	if err := w.process.Wait(); !w.closing {
		if err != nil {
			log.Error("Worker process died unexpectedly: %s", err)
		} else {
			log.Error("Worker process terminated unexpectedly")
		}
		w.responseMutex.Lock()
		defer w.responseMutex.Unlock()
		for label, ch := range w.responses {
			ch <- &Response{
				Rule:     label,
				Messages: []string{fmt.Sprintf("Worker failed: %s\n%s", err, string(w.stderr.History))},
			}
		}
	}
}

func (w *workerServer) Error(msg string, args ...interface{}) {
	if !w.closing {
		log.Error(msg, args...)
	}
}

// stderrLogger is used to log any errors from our worker tools.
type stderrLogger struct {
	buffer  []byte
	History []byte
	// suppress will silence any further logging messages when set.
	Suppress bool
}

// Write implements the io.Writer interface
func (l *stderrLogger) Write(msg []byte) (int, error) {
	l.buffer = append(l.buffer, msg...)
	if len(l.buffer) > 0 && l.buffer[len(l.buffer)-1] == '\n' {
		if !l.Suppress {
			if msg := strings.TrimSpace(string(l.buffer)); strings.HasPrefix(msg, "WARNING") {
				log.Warning("Warning from remote worker: %s", msg)
			} else {
				log.Error("Error from remote worker: %s", msg)
			}
		}
		l.History = append(l.History, l.buffer...)
		l.buffer = nil
	}
	return len(msg), nil
}

// StopAll stops any running worker processes.
// This should be called before the process terminates to ensure they are all correctly cleaned up.
func StopAll() {
	for name, worker := range workerMap {
		log.Debug("Terminating build worker %s", name)
		worker.closing = true         // suppress any error messages from worker
		worker.stderr.Suppress = true // Make sure we don't print anything as they die.
		worker.state.ProcessExecutor.KillProcess(worker.process)
	}
	workerMap = map[string]*workerServer{}
}
