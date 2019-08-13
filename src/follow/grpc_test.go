// Integration tests between the gRPC event server & client.
//
// It's a little limited in how much it can test due to extensive synchronisation
// issues (discussed ina  little more detail below); essentially the scheme is designed
// for clients following a series of events in "human" time (i.e. a stream that runs
// for many seconds), without a hard requirement to observe all initial events
// correctly (there's an expectation that we'll catch up soon enough).
// That doesn't work so well for a test where everything happens on the scale of
// microseconds and we want to assert precise events, but we do the best we can.

package follow

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
)

const retries = 3
const delay = 10 * time.Millisecond

func init() {
	cli.InitLogging(cli.MaxVerbosity)
	// The usual 1 second is pretty annoying in this test.
	disconnectTimeout = 1 * time.Millisecond
	// As is half a second wait for this.
	resourceUpdateFrequency = 1 * time.Millisecond
}

var (
	l1 = core.ParseBuildLabel("//src/remote:target1", "")
	l2 = core.ParseBuildLabel("//src/remote:target2", "")
	l3 = core.ParseBuildLabel("//src/remote:target3", "")
)

func TestClientToServerCommunication(t *testing.T) {
	// Note that the ordering of things here is pretty important; we need to get the
	// client connected & ready to receive events before we push them all into the
	// server and shut it down again.
	config := core.DefaultConfiguration()
	config.Please.NumThreads = 5
	serverState := core.NewBuildState(config)
	addr, shutdown := initialiseServer(serverState, 0)

	// This is a bit awkward. We want to assert that we receive a matching set of
	// build events, but it's difficult to build strong synchronisation into this
	// scheme which is really designed for builds taking a significant amount of
	// real time in which remote clients have a chance to sync up.
	// This test does the best it can to assert a reliable set of observable events.

	// Dispatch the first round of build events now
	serverState.LogBuildResult(0, l1, core.PackageParsed, fmt.Sprintf("Parsed %s", l1))
	serverState.LogBuildResult(0, l1, core.TargetBuilding, fmt.Sprintf("Building %s", l1))
	serverState.LogBuildResult(2, l2, core.TargetBuilding, fmt.Sprintf("Building %s", l2))
	serverState.LogBuildResult(0, l1, core.TargetBuilt, fmt.Sprintf("Built %s", l1))
	serverState.LogBuildResult(1, l3, core.TargetBuilding, fmt.Sprintf("Building %s", l3))

	clientState := core.NewDefaultBuildState()
	results := clientState.Results()
	connectClient(clientState, addr, retries, delay)
	// The client state should have synced up with the server's number of threads
	assert.Equal(t, 5, clientState.Config.Please.NumThreads)

	// We should be able to receive the latest build events for each thread.
	// Note that they come out in thread order, not time order.
	r := <-results
	log.Info("Received first build event")
	assert.Equal(t, "Built //src/remote:target1", r.Description)
	r = <-results
	assert.Equal(t, "Building //src/remote:target3", r.Description)
	r = <-results
	assert.Equal(t, "Building //src/remote:target2", r.Description)

	// Here we hit a bit of a synchronisation problem, whereby we can't guarantee that
	// the client is actually going to be ready to receive the events in time, which
	// manifests by blocking when we try to receive below. Conversely, we also race between
	// the client connecting and these results going in; we can miss them if it's still
	// not really receiving. Finally, the server can block on shutdown() if the client
	// isn't trying to read any pending events.
	go func() {
		defer func() {
			recover() // Send on closed channel, can happen because shutdown() is out of sync.
		}()
		serverState.LogBuildResult(1, l3, core.TargetBuilt, fmt.Sprintf("Built %s", l3))
		serverState.LogBuildResult(2, l2, core.TargetBuilt, fmt.Sprintf("Built %s", l2))
	}()
	go func() {
		for r := range results {
			log.Info("Received result from thread %d", r.ThreadID)
		}
	}()
	log.Info("Shutting down server")
	shutdown()
	log.Info("Server shutdown")
}

func TestWithOutput(t *testing.T) {
	serverState := core.NewDefaultBuildState()
	addr, shutdown := initialiseServer(serverState, 0)
	clientState := core.NewDefaultBuildState()
	connectClient(clientState, addr, retries, delay)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		serverState.LogBuildResult(0, l1, core.PackageParsed, fmt.Sprintf("Parsed %s", l1))
		serverState.LogBuildResult(0, l1, core.TargetBuilding, fmt.Sprintf("Building %s", l1))
		serverState.LogBuildResult(2, l2, core.TargetBuilding, fmt.Sprintf("Building %s", l2))
		serverState.LogBuildResult(0, l1, core.TargetBuilt, fmt.Sprintf("Built %s", l1))
		serverState.LogBuildResult(1, l3, core.TargetBuilding, fmt.Sprintf("Building %s", l3))
		serverState.LogBuildResult(1, l3, core.TargetBuilt, fmt.Sprintf("Built %s", l3))
		serverState.LogBuildResult(2, l2, core.TargetBuilt, fmt.Sprintf("Built %s", l2))
		log.Info("Shutting down server")
		shutdown()
		cancel()
	}()
	assert.True(t, runOutput(ctx, clientState))
}

func TestResources(t *testing.T) {
	serverState := core.NewDefaultBuildState()
	go UpdateResources(serverState)
	addr, shutdown := initialiseServer(serverState, 0)
	defer shutdown()
	clientState := core.NewDefaultBuildState()
	connectClient(clientState, addr, retries, delay)
	// Fortunately this is a lot less fiddly than the others, because we always
	// receive updates eventually. On the downside it's hard to know when it'll
	// be done since we can't observe the actual goroutines that are doing it.
	for i := 0; i < 20; i++ {
		time.Sleep(resourceUpdateFrequency)
		if clientState.Stats.Memory.Used > 0.0 {
			break
		}
	}
	// Hard to know what any of the values should be, but of course we must be using
	// *some* memory.
	assert.True(t, clientState.Stats.Memory.Used > 0.0)
}
