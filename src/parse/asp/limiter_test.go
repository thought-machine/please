package asp

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// waitForQueued blocks until the limiter has the given number of waiters in each queue.
func waitForQueued(t *testing.T, l *limiter, resume, start int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		l.mu.Lock()
		r, s := l.resume.len(), l.start.len()
		l.mu.Unlock()
		if r == resume && s == start {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d resume / %d start waiters", resume, start)
}

func TestLimiterResumeBeforeStart(t *testing.T) {
	l := newLimiter(1)
	l.Start(context.Background()) // Take the only slot

	order := make(chan string, 4)
	acquire := func(name string, f func()) {
		go func() {
			f()
			order <- name
		}()
	}
	acquire("start1", func() { l.Start(context.Background()) })
	waitForQueued(t, l, 0, 1)
	acquire("start2", func() { l.Start(context.Background()) })
	waitForQueued(t, l, 0, 2)
	acquire("resume1", l.Acquire)
	waitForQueued(t, l, 1, 2)
	acquire("resume2", l.Acquire)
	waitForQueued(t, l, 2, 2)

	// Each release should hand the slot to the next waiter: resumes first, then starts, each in FIFO order.
	for _, expected := range []string{"resume1", "resume2", "start1", "start2"} {
		l.Release()
		assert.Equal(t, expected, <-order)
	}
	l.Release()
	assert.Equal(t, 1, l.free)
}

func TestLimiterCapacity(t *testing.T) {
	l := newLimiter(3)
	l.Start(context.Background())
	l.Acquire()
	l.Start(context.Background())
	assert.Equal(t, 0, l.free)
	l.Release()
	l.Release()
	l.Release()
	assert.Equal(t, 3, l.free)
	assert.Equal(t, 0, l.resume.len())
	assert.Equal(t, 0, l.start.len())
}

func TestLimiterStartYieldsToWaiters(t *testing.T) {
	// A new Start mustn't jump ahead of anyone already waiting, even if it arrives just as a slot frees up.
	l := newLimiter(1)
	l.Start(context.Background())
	done := make(chan struct{})
	go func() {
		l.Acquire()
		close(done)
	}()
	waitForQueued(t, l, 1, 0)
	l.Release() // Handed straight to the waiting Acquire, so free stays at zero
	<-done
	assert.Equal(t, 0, l.free)
	l.Release()
	assert.Equal(t, 1, l.free)
}

func TestLimiterStartWithinParse(t *testing.T) {
	l := newLimiter(1)
	l.Start(context.Background())
	order := make(chan string, 2)
	go func() {
		l.Start(context.Background())
		order <- "toplevel"
	}()
	waitForQueued(t, l, 0, 1)
	go func() {
		l.Start(withinParse(context.Background()))
		order <- "nested"
	}()
	waitForQueued(t, l, 1, 1)
	l.Release()
	assert.Equal(t, "nested", <-order)
	l.Release()
	assert.Equal(t, "toplevel", <-order)
	l.Release()
}

func TestWithinParse(t *testing.T) {
	ctx := context.Background()
	assert.False(t, isWithinParse(ctx))
	ctx2 := withinParse(ctx)
	assert.True(t, isWithinParse(ctx2))
	assert.Equal(t, ctx2, withinParse(ctx2)) // Doesn't keep wrapping
	assert.True(t, isWithinParse(context.WithoutCancel(ctx2)))
}
