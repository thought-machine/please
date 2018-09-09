// Package worker implements functions for communicating with subordinate worker processes.
package worker

import (
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"

	"gopkg.in/op/go-logging.v1"

	"core"
)

var log = logging.MustGetLogger("worker")

// A workerServer is the structure we use to maintain information about a remote work server.
type workerServer struct {
	requests      chan *BuildRequest
	responses     map[string]chan *BuildResponse
	responseMutex sync.Mutex
	process       *exec.Cmd
	stderr        *stderrLogger
	closing       bool
}

// workerMap contains all the remote workers we've started so far.
var workerMap = map[string]*workerServer{}
var workerMutex sync.Mutex

// BuildRemotely runs a single build request and returns its response.
func BuildRemotely(state *core.BuildState, worker string, req *BuildRequest) (*BuildResponse, error) {
	w, err := getOrStartWorker(state, worker)
	if err != nil {
		return nil, err
	}
	w.requests <- req
	ch := make(chan *BuildResponse, 1)
	w.responseMutex.Lock()
	w.responses[req.Rule] = ch
	w.responseMutex.Unlock()
	response := <-ch
	return response, nil
}

// EnsureWorkerStarted ensures that a worker server is started and has responded saying it's ready.
func EnsureWorkerStarted(state *core.BuildState, worker string, label core.BuildLabel) error {
	resp, err := BuildRemotely(state, worker, &BuildRequest{
		Rule: label.String(),
		Test: true,
	})
	if err == nil && !resp.Success {
		return fmt.Errorf(strings.Join(resp.Messages, "\n"))
	}
	return err
}

// getOrStartWorker either retrieves an existing worker process or starts a new one.
func getOrStartWorker(state *core.BuildState, worker string) (*workerServer, error) {
	workerMutex.Lock()
	defer workerMutex.Unlock()
	if w, present := workerMap[worker]; present {
		return w, nil
	}
	// Need to create a new process
	cmd := core.ExecCommand(worker)
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
		requests:  make(chan *BuildRequest),
		responses: map[string]chan *BuildResponse{},
		process:   cmd,
		stderr:    stderr,
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
			log.Error("Failed to write request: %s", err)
			w.dispatchResponse(&BuildResponse{
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
		response := BuildResponse{}
		if err := decoder.Decode(&response); err != nil {
			w.Error("Failed to read response: %s", err)
			break
		}
		w.dispatchResponse(&response)
	}
}

// dispatchResponse sends a single response on the appropriate channel.
func (w *workerServer) dispatchResponse(response *BuildResponse) {
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
	if err := w.process.Wait(); err != nil && !w.closing {
		log.Error("Worker process died unexpectedly: %s", err)
		w.responseMutex.Lock()
		defer w.responseMutex.Unlock()
		for label, ch := range w.responses {
			ch <- &BuildResponse{
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
			log.Error("Error from remote worker: %s", strings.TrimSpace(string(l.buffer)))
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
		core.KillProcess(worker.process)
	}
	workerMap = map[string]*workerServer{}
}
