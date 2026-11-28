package asp

import (
	"context"
	"sync"
)

// A limiter is used to bound the number of concurrent parses.
//
// It optimises for resuming existing work over starting new work; anything waiting to resume (or parsing on
// behalf of something that is) is served before anything waiting to start a new parse.
type limiter struct {
	mu     sync.Mutex
	free   int
	resume waitQueue // in-progress work waiting to continue; always served first
	start  waitQueue // new parses waiting to begin
}

func newLimiter(n int) *limiter {
	return &limiter{free: n}
}

// Start acquires a slot to begin a new parse. It yields to anything waiting to resume, unless the context
// indicates that we are parsing on behalf of an in-progress parse (which is therefore waiting on us), in
// which case we acquire with the same priority as resuming it.
func (l *limiter) Start(ctx context.Context) {
	if isWithinParse(ctx) {
		l.acquire(&l.resume)
	} else {
		l.acquire(&l.start)
	}
}

// Acquire reacquires a slot to resume in-progress work. It takes priority over Start.
func (l *limiter) Acquire() {
	l.acquire(&l.resume)
}

// acquire takes a free slot if there is one, otherwise waits on the given queue to be handed one.
func (l *limiter) acquire(q *waitQueue) {
	l.mu.Lock()
	if l.free > 0 {
		l.free--
		l.mu.Unlock()
		return
	}
	ch := q.push()
	l.mu.Unlock()
	<-ch // The slot is handed to us directly by Release
}

// Release gives up a slot, handing it directly to the highest-priority waiter if there is one.
// Handing it over directly (rather than incrementing free and letting waiters race for it) ensures
// a newly arriving Start can't take it ahead of an existing waiter.
func (l *limiter) Release() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if ch := l.resume.pop(); ch != nil {
		close(ch)
	} else if ch := l.start.pop(); ch != nil {
		close(ch)
	} else {
		l.free++
	}
}

// A waitQueue is a FIFO queue of waiters. It's only accessed under the limiter's mutex.
type waitQueue []chan struct{}

func (w *waitQueue) len() int {
	return len(*w)
}

func (w *waitQueue) push() chan struct{} {
	ch := make(chan struct{})
	*w = append(*w, ch)
	return ch
}

func (w *waitQueue) pop() chan struct{} {
	if len(*w) == 0 {
		return nil
	}
	ch := (*w)[0]
	(*w)[0] = nil
	*w = (*w)[1:]
	return ch
}

type withinParseKey struct{}

// withinParse returns a context indicating that anything done with it is on behalf of an in-progress parse.
func withinParse(ctx context.Context) context.Context {
	if isWithinParse(ctx) {
		return ctx
	}
	return context.WithValue(ctx, withinParseKey{}, struct{}{})
}

// isWithinParse returns true if the context was created by withinParse.
func isWithinParse(ctx context.Context) bool {
	return ctx.Value(withinParseKey{}) != nil
}
