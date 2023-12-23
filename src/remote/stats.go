package remote

import (
	"context"
	"sync/atomic"
	"time"

	"google.golang.org/grpc/stats"
)

// updateFrequency is the rate at which we update stats internally (which is independent of display updates)
const updateFrequency = 1 * time.Second

// A statsHandler is an implementation of a grpc stats.Handler that calculates an estimate of
// instantaneous performance.
type statsHandler struct {
	in, out           atomic.Int64 // aggregated for the current second
	rateIn, rateOut   atomic.Int64 // the stats for the previous second (which gets displayed)
	totalIn, totalOut atomic.Int64 // aggregated total for all time
}

func newStatsHandler() *statsHandler {
	h := &statsHandler{}
	go h.update()
	return h
}

func (h *statsHandler) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	return ctx
}

func (h *statsHandler) HandleRPC(ctx context.Context, s stats.RPCStats) {
	switch p := s.(type) {
	case *stats.InHeader:
		h.in.Add(int64(p.WireLength))
	case *stats.OutHeader:
		// The out header seems not to have any size on it that we can use
	case *stats.InPayload:
		h.in.Add(int64(p.WireLength))
	case *stats.OutPayload:
		h.out.Add(int64(p.WireLength))
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

// update runs continually, updating the aggregated stats on the handler.
func (h *statsHandler) update() {
	for range time.NewTicker(updateFrequency).C {
		in := h.in.Swap(0)
		out := h.out.Swap(0)
		h.rateIn.Store(in)
		h.rateOut.Store(out)
		h.totalIn.Add(in)
		h.totalOut.Add(out)
	}
}
