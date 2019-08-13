// +build !bootstrap

package follow

import (
	"context"
	"fmt"
	"io"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/thought-machine/please/src/core"
	pb "github.com/thought-machine/please/src/follow/proto/build_event"
	"github.com/thought-machine/please/src/output"
)

// Used to track the state of the remote connection.
var remoteClosed, remoteDisconnected bool

// ConnectClient connects a gRPC client to the given URL.
// It returns once the client has received that the remote build finished,
// returning true if that build was successful.
// It dies on any errors.
func ConnectClient(state *core.BuildState, url string, retries int, delay time.Duration) bool {
	connectClient(state, url, retries, delay)
	// Context must be terminated once all results are consumed.
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for range state.Results() {
		}
		cancel()
	}()
	// Now run output, this will exit when the goroutine in connectClient() hits its end.
	return runOutput(ctx, state)
}

// connectClient connects a gRPC client to the given URL.
// It is split out of the above for testing purposes.
func connectClient(state *core.BuildState, url string, retries int, delay time.Duration) {
	state.TaskQueues() // This is important to start consuming events in the background
	var err error
	for i := 0; i <= retries; i++ {
		if err = connectSingleTry(state, url); err == nil {
			return
		} else if retries > 0 && i < retries {
			log.Warning("Failed to connect to remote server, will retry in %s: %s", delay, err)
			time.Sleep(delay)
		}
	}
	log.Fatalf("%s", err)
}

// connectSingleTry performs one try at connecting the gRPC client.
func connectSingleTry(state *core.BuildState, url string) error {
	// TODO(peterebden): TLS
	conn, err := grpc.Dial(url, grpc.WithInsecure())
	if err != nil {
		return err
	}
	client := pb.NewPlzEventsClient(conn)
	// Get the deets of what the server is doing.
	resp, err := client.ServerConfig(context.Background(), &pb.ServerConfigRequest{})
	if err != nil {
		if s, ok := status.FromError(err); ok && s.Code() == codes.Unavailable {
			// Slightly nicer version for an obvious failure which gets a bit technical by default
			return fmt.Errorf("Failed to set up communication with remote server; check the address is correct and it's running")
		}
		return fmt.Errorf("Failed to set up communication with remote server: %s", err)
	}
	// Let the user know we're connected now and what it's up to.
	output.PrintConnectionMessage(url, fromProtoBuildLabels(resp.OriginalTargets), resp.Tests, resp.Coverage)
	// Update the config appropriately; the output code reads some of its fields.
	state.Config.Please.NumThreads = int(resp.NumThreads)
	state.NeedBuild = false // We're not actually building ourselves
	state.NeedTests = resp.Tests
	state.NeedCoverage = resp.Coverage
	state.StartTime = time.Unix(0, resp.StartTime)
	state.Config.Display.SystemStats = true
	// Catch up on the last result of each thread
	log.Info("Got %d initial build events, dispatching...", len(resp.LastEvents))
	for _, r := range resp.LastEvents {
		streamEvent(state, r)
	}

	// Now start streaming events into it
	stream, err := client.BuildEvents(context.Background(), &pb.BuildEventRequest{})
	if err != nil {
		return fmt.Errorf("Error receiving build events: %s", err)
	}
	go func() {
		for {
			event, err := stream.Recv()
			if err == io.EOF {
				remoteClosed = true
				break
			} else if err != nil {
				remoteDisconnected = true
				log.Error("Error receiving build events: %s", err)
				break
			}
			streamEvent(state, event)
		}
		log.Info("Reached end of server event stream, shutting down internal queue")
		state.KillAll()
	}()
	log.Info("Established connection to remote server for build events")
	// Stream back resource usage as well
	go streamResources(state, client)
	return nil
}

// streamEvent adds an event to our internal stream.
func streamEvent(state *core.BuildState, event *pb.BuildEventResponse) {
	e := fromProto(event)
	// Put a version of this target into our graph, it will help a lot of things later.
	if t := state.Graph.Target(e.Label); t == nil {
		t = state.Graph.AddTarget(core.NewBuildTarget(e.Label))
		t.Labels = event.Labels
	}
	state.LogResult(e)
	state.SetTaskNumbers(event.NumActive, event.NumDone)
}

// streamResources receives system resource usage information from the server and copies
// them into the build state.
func streamResources(state *core.BuildState, client pb.PlzEventsClient) {
	stream, err := client.ResourceUsage(context.Background(), &pb.ResourceUsageRequest{})
	if err != nil {
		log.Error("Error receiving resource usage: %s", err)
		return
	}
	for {
		resources, err := stream.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Error("Error receiving resource usage: %s", err)
			break
		}
		state.Stats = resourceFromProto(resources)
	}
}

// runOutput is just a wrapper around output.MonitorState for convenience in testing.
func runOutput(ctx context.Context, state *core.BuildState) bool {
	output.MonitorState(ctx, state, true, false, false, "")
	output.PrintDisconnectionMessage(state.Success, remoteClosed, remoteDisconnected)
	return state.Success
}
