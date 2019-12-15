package remote

import (
	"context"
	"sync"
	"time"

	"google.golang.org/grpc/stats"
)

// The length of time we keep stats around for.
const windowDurationSeconds = 5
const windowDuration = -windowDurationSeconds * time.Second

// updateFrequency is the rate at which we update stats internally (which is independent of display updates)
const updateFrequency = 1 * time.Second

// A stat is a single statistic comprising an observation time and size value (in bytes)
type stat struct {
	Time time.Time
	Val  int
}

// A statsHandler is an implementation of a grpc stats.Handler that calculates an estimate of
// instantaneous performance.
type statsHandler struct {
	client        *Client
	in, out       []stat
	inmtx, outmtx sync.Mutex
}

func newStatsHandler(c *Client) *statsHandler {
	h := statsHandler{client: c}
	go h.update()
	return h
}

func (h *statsHandler) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	return ctx
}

func (h *statsHandler) HandleRPC(ctx context.Context, s stats.RPCStats) {
	switch s.(type) {
	case *stats.InPayload:
		h.inmtx.Lock()
		defer h.inmtx.Unlock()
		h.in = append(h.in, stat{Time: s.RecvTime, Val: s.Length})
	case *stats.OutPayload:
		h.outmtx.Lock()
		defer h.outmtx.Unlock()
		h.out = append(h.out, stat{Time: s.SentTime, Val: s.Length})
	}
}

func (h *statsHandler) TagConn(ctx context.Context, info *stats.ConnTagInfo) context.Context {
	return ctx
}

func (h *statsHandler) HandleConn(ctx context.Context, s stats.ConnStats) {
}

// update runs continually, updating the aggregated stats on the Client instance.
func (h *statsHandler) update() {
	for range time.NewTicker(updateFrequency) {
		h.client.byteRateIn = h.updateStat(&h.in, &h.inmtx)
		h.client.byteRateOut = h.updateStat(&h.out, &h.outmtx)
	}
}

func (h *statsHandler) updateStat(stats *[]stat, mtx *sync.Mutex) float32 {
	mtx.Lock()
	defer mtx.Unlock()
	s := *stats
	idx := h.survivingStats(s, time.Now().Add(windowDuration))
	if idx == 0 {
		return // Nothing's changed
	}
	// Shuffle them all back by this much. We *could* just slice here but that has rather nasty
	// allocation behaviour (in that we would be continually reallocating the underlying buffer)
	copy(s, s[idx:])
	*stats = s[:idx]
	// Now recalculate the observed value
	total := 0
	for _, stat := range *stats {
		total += stat.Val
	}
	return total / windowDurationSeconds
}

func (h *statsHandler) survivingStats(stats []stat, deadline time.Time) int {
	// This assumes that we receive stats in time order, which seems reasonable; if it turns out not
	// to be the case we could record the current time ourselves when we get them.
	for i, stat := range stats {
		if stat.Time.After(deadline) {
			return i
		}
	}
	return len(stats)
}
