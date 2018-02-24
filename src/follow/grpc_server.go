// +build !bootstrap

// Package follow implements remote connections to other plz processes.
// Specifically it implements a gRPC server and client that can stream
// build events.
package follow

import (
	"fmt"
	"net"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"gopkg.in/op/go-logging.v1"

	"core"
	pb "follow/proto/build_event"
)

var log = logging.MustGetLogger("remote")

// disconnectTimeout is the grace period we give clients to disconnect before they are ditched.
var disconnectTimeout = 1 * time.Second

// buffering is the size of buffer we allocate in the server channels.
// Larger values consume more memory but protect better against slow clients.
const buffering = 1000

// InitialiseServer sets up the gRPC server on the given port.
// It dies on any errors.
// The returned function should be called to shut down once the server is no longer required.
func InitialiseServer(state *core.BuildState, port int) func() {
	_, f := initialiseServer(state, port)
	return f
}

// initialiseServer sets up the gRPC server on the given port.
// It's split out from the above for testing purposes.
func initialiseServer(state *core.BuildState, port int) (string, func()) {
	// Set up the channel that we'll get messages off
	state.RemoteResults = make(chan *core.BuildResult, buffering)
	// TODO(peterebden): TLS support
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("%s", err)
	}
	addr := lis.Addr().String()
	s := grpc.NewServer()
	server := &eventServer{State: state}
	go server.MultiplexEvents(state.RemoteResults)
	pb.RegisterPlzEventsServer(s, server)
	go s.Serve(lis)
	log.Notice("Serving events over gRPC on :%s", addr)
	return addr, func() {
		close(state.RemoteResults)
		stopServer(s)
	}
}

// An eventServer handles the RPC requests to connected clients.
type eventServer struct {
	State   *core.BuildState
	Clients []chan *pb.BuildEventResponse
}

// ServerConfig implements the RPC interface.
func (e *eventServer) ServerConfig(ctx context.Context, r *pb.ServerConfigRequest) (*pb.ServerConfigResponse, error) {
	targets := make([]*pb.BuildLabel, len(e.State.OriginalTargets))
	for i, t := range e.State.OriginalTargets {
		targets[i] = toProtoBuildLabel(t)
	}
	return &pb.ServerConfigResponse{
		NumThreads:      int32(e.State.Config.Please.NumThreads),
		OriginalTargets: targets,
		Tests:           e.State.NeedTests,
		Coverage:        e.State.NeedCoverage,
		LastEvents:      toProtos(e.State.LastResults, e.State.NumActive(), e.State.NumDone()),
		StartTime:       e.State.StartTime.UnixNano(),
	}, nil
}

// BuildEvents implements the RPC interface.
func (e *eventServer) BuildEvents(r *pb.BuildEventRequest, s pb.PlzEvents_BuildEventsServer) error {
	if p, ok := peer.FromContext(s.Context()); ok {
		log.Notice("Remote client connected from %s to receive events", p.Addr)
	}
	c := make(chan *pb.BuildEventResponse, buffering)
	e.Clients = append(e.Clients, c)
	// Client is now connected to the stream and will receive all events from here on.
	for event := range c {
		if err := s.Send(event); err != nil {
			// Something's stuffed, disconnect the client from our event streams
			log.Notice("Remote client disconnected (%s)", err)
			for i, client := range e.Clients {
				if client == c {
					copy(e.Clients[i:], e.Clients[i+1:])
					last := len(e.Clients) - 1
					e.Clients[last] = nil
					e.Clients = e.Clients[:last]
				}
			}
			return err
		}
	}
	log.Notice("Events finished, terminating remote session")
	return nil
}

// ResourceUsage implements the RPC interface.
func (e *eventServer) ResourceUsage(r *pb.ResourceUsageRequest, s pb.PlzEvents_ResourceUsageServer) error {
	// This doesn't necessarily have to match the update frequency, but it seems sensible to do so
	// since the clients won't get any benefit of anything more frequent.
	ticker := time.NewTicker(resourceUpdateFrequency)
	defer ticker.Stop()
	for range ticker.C {
		if err := s.Send(resourceToProto(e.State.Stats)); err != nil {
			return err
		}
	}
	return nil
}

// MultiplexEvents receives events from core and distributes them to receiving clients
func (e *eventServer) MultiplexEvents(ch chan *core.BuildResult) {
	for r := range ch {
		p := toProto(r)
		// Target labels don't exist on the internal build events, retrieve them here.
		if t := e.State.Graph.Target(r.Label); t != nil {
			p.Labels = t.Labels
		}
		// Similarly these fields come off the state, they're not stored historically for each event.
		p.NumActive = int64(e.State.NumActive())
		p.NumDone = int64(e.State.NumDone())
		for _, c := range e.Clients {
			c <- p
		}
	}
	log.Info("Reached end of event stream, shutting down connected clients")
	for _, c := range e.Clients {
		close(c) // This terminates communication with whichever client is on the end of it.
	}
	log.Info("Closed channels to all connected clients")
}

// stopServer implements a graceful server stop with a timeout, followed by a non-graceful (ungainly?) shutdown.
// Essentially GracefulStop can block forever and we don't want to allow clients to do that to us.
func stopServer(s *grpc.Server) {
	ch := make(chan bool, 1)
	go func() {
		s.GracefulStop()
		ch <- true
	}()
	select {
	case <-ch:
	case <-time.After(disconnectTimeout):
		log.Warning("Remote client hasn't disconnected in alloted time, rapid shutdown initiated")
		s.Stop()
	}
}
