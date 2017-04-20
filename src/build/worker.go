// +build proto
// Contains functions related to dispatching work to remote processes.
// Right now those processes must be on the same box because they use
// the local temporary directories, but in the future this might form
// a foundation for doing real distributed work.

package build

import (
	"encoding/binary"
	"fmt"
	"io"
	"os/exec"
	"path"
	"strings"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/google/shlex"

	pb "build/proto/worker"
	"core"
)

// A workerServer is the structure we use to maintain information about a remote work server.
type workerServer struct {
	requests      chan *pb.BuildRequest
	responses     map[string]chan *pb.BuildResponse
	responseMutex sync.Mutex
	process       *exec.Cmd
	closing       bool
}

// workerMap contains all the remote workers we've started so far.
var workerMap = map[string]*workerServer{}
var workerMutex sync.Mutex

// buildMaybeRemotely builds a target, either sending it to a remote worker if needed,
// or locally if not.
func buildMaybeRemotely(state *core.BuildState, target *core.BuildTarget, inputHash []byte) ([]byte, error) {
	worker, workerArgs, localCmd := workerCommandAndArgs(target)
	if worker == "" {
		return runBuildCommand(state, target, localCmd, inputHash)
	}
	// The scheme here is pretty minimal; remote workers currently have quite a bit less info than
	// local ones get. Over time we'll probably evolve it to add more information.
	opts, err := shlex.Split(workerArgs)
	if err != nil {
		return nil, err
	}
	log.Debug("Sending remote build request to %s; opts %s", worker, workerArgs)
	resp, err := buildRemotely(state.Config, worker, &pb.BuildRequest{
		Rule:    target.Label.String(),
		Labels:  target.Labels,
		TempDir: path.Join(core.RepoRoot, target.TmpDir()),
		Srcs:    target.AllSourcePaths(state.Graph),
		Opts:    opts,
	})
	if err != nil {
		return nil, err
	}
	out := strings.Join(resp.Messages, "\n")
	if !resp.Success {
		return nil, fmt.Errorf("Error building target %s: %s", target.Label, out)
	}
	// Okay, now we might need to do something locally too...
	if localCmd != "" {
		out2, err := runBuildCommand(state, target, localCmd, inputHash)
		return append([]byte(out+"\n"), out2...), err
	}
	return []byte(out), nil
}

// buildRemotely runs a single build request and returns its response.
func buildRemotely(config *core.Configuration, worker string, req *pb.BuildRequest) (*pb.BuildResponse, error) {
	w, err := getOrStartWorker(config, worker)
	if err != nil {
		return nil, err
	}
	w.requests <- req
	ch := make(chan *pb.BuildResponse, 1)
	w.responseMutex.Lock()
	w.responses[req.Rule] = ch
	w.responseMutex.Unlock()
	response := <-ch
	return response, nil
}

// getOrStartWorker either retrieves an existing worker process or starts a new one.
func getOrStartWorker(config *core.Configuration, worker string) (*workerServer, error) {
	workerMutex.Lock()
	defer workerMutex.Unlock()
	if w, present := workerMap[worker]; present {
		return w, nil
	}
	// Need to create a new process
	cmd := exec.Command(worker)
	cmd.Env = core.GeneralBuildEnvironment(config)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = &stderrLogger{}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	w := &workerServer{
		requests:  make(chan *pb.BuildRequest),
		responses: map[string]chan *pb.BuildResponse{},
		process:   cmd,
	}
	go w.sendRequests(stdin)
	go w.readResponses(stdout)
	workerMap[worker] = w
	return w, nil
}

// sendRequests sends requests to a running worker server.
func (w *workerServer) sendRequests(stdin io.Writer) {
	for request := range w.requests {
		b, err := proto.Marshal(request)
		if err != nil { // This shouldn't really happen
			log.Error("Failed to serialise request: %s", err)
			continue
		}
		// Protos can't be streamed so we have to do our own framing.
		binary.Write(stdin, binary.LittleEndian, int32(len(b)))
		stdin.Write(b)
	}
}

// readResponses reads the responses from a running worker server and dispatches them appropriately.
func (w *workerServer) readResponses(stdout io.Reader) {
	var size int32
	for {
		if err := binary.Read(stdout, binary.LittleEndian, &size); err != nil {
			w.Error("Failed to read response: %s", err)
			break
		}
		buf := make([]byte, size)
		if _, err := stdout.Read(buf); err != nil {
			w.Error("Failed to read response: %s", err)
			break
		}
		response := pb.BuildResponse{}
		if err := proto.Unmarshal(buf, &response); err != nil {
			w.Error("Error unmarshaling response: %s", err)
			continue
		}
		w.responseMutex.Lock()
		ch, present := w.responses[response.Rule]
		delete(w.responses, response.Rule)
		w.responseMutex.Unlock()
		if present {
			log.Debug("Got response from remote worker for %s, success: %v", response.Rule, response.Success)
			ch <- &response
		} else {
			w.Error("Couldn't find response channel for %s", response.Rule)
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
	buffer []byte
	// suppress will silence any further logging messages when set.
	suppress bool
}

// Write implements the io.Writer interface
func (l *stderrLogger) Write(msg []byte) (int, error) {
	l.buffer = append(l.buffer, msg...)
	if len(l.buffer) > 0 && l.buffer[len(l.buffer)-1] == '\n' {
		if !l.suppress {
			log.Error("Error from remote worker: %s", strings.TrimSpace(string(l.buffer)))
		}
		l.buffer = nil
	}
	return len(msg), nil
}

// StopWorkers stops any running worker processes.
func StopWorkers() {
	for name, worker := range workerMap {
		log.Debug("Killing build worker %s", name)
		worker.closing = true // suppress any error messages from worker
		if l, ok := worker.process.Stderr.(*stderrLogger); ok {
			l.suppress = true // Make sure we don't print anything as they die.
		}
		worker.process.Process.Kill()
	}
}
