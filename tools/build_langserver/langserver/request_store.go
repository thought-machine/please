package langserver

import (
	"context"
	"fmt"
	"sync"

	"github.com/sourcegraph/jsonrpc2"
)

type requestStore struct {
	mu       sync.Mutex
	requests map[jsonrpc2.ID]request
}

// Perhaps later we can store more things in the request we might want to use
type request struct {
	id     string
	cancel func()
}

func (rs *requestStore) Store(ctx context.Context, req *jsonrpc2.Request) context.Context {
	ctx, cancel := context.WithCancel(ctx)

	rs.mu.Lock()
	// Cancellation function definition,
	// calling both cancel and delete id from the requests map
	cancelFunc := func() {
		rs.mu.Lock()
		cancel()
		delete(rs.requests, req.ID)
		rs.mu.Unlock()
	}

	rs.requests[req.ID] = request{
		id:     req.ID.String(),
		cancel: cancelFunc,
	}

	defer rs.mu.Unlock()

	// returns the sub-context for the specific request.ID
	return ctx
}

// Cancel method removes the id from the requests map and calls cancel function of the request
func (rs *requestStore) Cancel(id jsonrpc2.ID) {
	if rs.requests != nil {
		req, ok := rs.requests[id]
		if ok {
			req.cancel()
		} else {
			log.Info(fmt.Sprintf("Request id '%s' does not exist in the map, no action", id))
		}
	}
}
