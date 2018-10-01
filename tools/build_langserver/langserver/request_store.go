package langserver

import (
	"context"
	"github.com/sourcegraph/jsonrpc2"
	"sync"
)

type requestStore struct {
	mu sync.Mutex
	requests map[jsonrpc2.ID]request
}

// Perhaps later we can store more things in the request we might want to use
type request struct {
	id 	   string
	cancel func()
}

func (rs *requestStore) Store(ctx context.Context, req *jsonrpc2.Request) context.Context {
	ctx, cancel := context.WithCancel(ctx)
	rs.mu.Lock()
	if rs.requests == nil {
		rs.requests = make(map[jsonrpc2.ID]request)
	}
	rs.mu.Unlock()

	rs.mu.Lock()

	// Cancellation function definition,
	// calling both cancel and delete id from the requests map
	cancelFunc := func () {
		rs.mu.Lock()
		cancel()
		delete(rs.requests, req.ID)
		rs.mu.Unlock()
	}

	rs.requests[req.ID] = request{
		id: req.ID.String(),
		cancel: cancelFunc,
	}

	rs.mu.Unlock()

	// returns the sub-context for the specific request.ID
	return ctx
}


func (rs *requestStore) Cancel(id jsonrpc2.ID) {
	rs.mu.Lock()
	if rs.requests != nil {
		req := rs.requests[id]
		req.cancel()
	}
	rs.mu.Unlock()
}