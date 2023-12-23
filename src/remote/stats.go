package remote

import (
	"context"
	"sync/atomic"
	"time"

	"google.golang.org/grpc/stats"
)

// The number of seconds worth of stats that we average over
const windowSize = 3

// updateFrequency is the rate at which we update stats internally (which is independent of display updates)
const updateFrequency = 1 * time.Second

// A statsHandler is an implementation of a grpc stats.Handler that calculates an estimate of
// instantaneous performance.
type statsHandler struct {
	client            *Client
	in, out           [windowSize]atomic.Int64  // most recent first
	rateIn, rateOut   atomic.Int64
	totalIn, totalOut atomic.Int64
}

func newStatsHandler(c *Client) *statsHandler {
	h := &statsHandler{client: c}
	go h.update()
	return h
}

func (h *statsHandler) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	return ctx
}

func (h *statsHandler) HandleRPC(ctx context.Context, s stats.RPCStats) {
	switch p := s.(type) {
	case *stats.InHeader:
		h.in[0].Add(int64(p.WireLength))
	case *stats.OutHeader:
		// The out header seems not to have any size on it that we can use
	case *stats.InPayload:
		h.in[0].Add(int64(p.WireLength))
	case *stats.OutPayload:
		h.out[0].Add(int64(p.WireLength))
	}
}

func (h *statsHandler) TagConn(ctx context.Context, info *stats.ConnTagInfo) context.Context {
	return ctx
}

func (h *statsHandler) HandleConn(ctx context.Context, s stats.ConnStats) {
}

// DataRate returns the current snapshot of the data rate stats.
func (h *statsHandler) DataRate() (int, int, int, int) {
	return int(h.rateIn.Load()), int(h.rateOut.Load()), int(h.totalIn.Load()), int(h.totalOut.Load())
}

// update runs continually, updating the aggregated stats on the Client instance.
func (h *statsHandler) update() {
	for range time.NewTicker(updateFrequency).C {
		// Aggregate the total on the handler
		var in, out int64
		lastIn := h.in[0].Swap(0)
		lastOut := h.out[0].Swap(0)
		h.totalIn.Add(lastIn)
		h.totalOut.Add(lastOut)
		for i := 1; i < windowSize; i++ {
			in += lastIn
			out += lastOut
			h.in[i].Store(lastIn)
			h.out[i].Store(lastOut)
		}
		h.rateIn.Store(in / windowSize)
		h.rateOut.Store(out / windowSize)
	}
}
