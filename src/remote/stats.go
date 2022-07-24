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
	client            *Client
	in, out           []stat
	mutex             sync.Mutex
	rateIn, rateOut   int
	totalIn, totalOut int
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
	h.mutex.Lock()
	defer h.mutex.Unlock()
	switch p := s.(type) {
	case *stats.InPayload:
		h.in = append(h.in, stat{Time: p.RecvTime, Val: p.Length})
		h.totalIn += p.Length
	case *stats.OutPayload:
		h.out = append(h.out, stat{Time: p.SentTime, Val: p.Length})
		h.totalOut += p.Length
	}
}

func (h *statsHandler) TagConn(ctx context.Context, info *stats.ConnTagInfo) context.Context {
	return ctx
}

func (h *statsHandler) HandleConn(ctx context.Context, s stats.ConnStats) {
}

// DataRate returns the current snapshot of the data rate stats.
func (h *statsHandler) DataRate() (int, int, int, int) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	return h.rateIn, h.rateOut, h.totalIn, h.totalOut
}

// update runs continually, updating the aggregated stats on the Client instance.
func (h *statsHandler) update() {
	for range time.NewTicker(updateFrequency).C {
		h.mutex.Lock()
		h.rateIn = h.updateStat(&h.in)
		h.rateOut = h.updateStat(&h.out)
		h.mutex.Unlock()
	}
}

func (h *statsHandler) updateStat(stats *[]stat) int {
	s := *stats
	idx := h.survivingStats(s, time.Now().Add(windowDuration))
	if idx != 0 {
		// Shuffle them all back by this much. We *could* just slice here but that has rather nasty
		// allocation behaviour (in that we would be continually reallocating the underlying buffer)
		copy(s, s[idx:])
		*stats = s[:idx]
	}
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
