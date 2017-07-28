package run

// A GoroutinePool manages a set of worker goroutines, analogous to a traditional threadpool.
// Obviously in classic Go you do not need one, but this can be useful when you have external
// resources being driven by it that can't scale like goroutines (for example processes)
type GoroutinePool struct {
	ch chan func()
}

// NewGoroutinePool allocates a new goroutine pool with the given maximum capacity
// (i.e. number of goroutines).
func NewGoroutinePool(capacity int) *GoroutinePool {
	// Buffer to a reasonably large capacity to try to prevent too much blocking on Submit().
	ch := make(chan func(), capacity*10)
	for i := 0; i < capacity; i++ {
		go runWorker(ch)
	}
	return &GoroutinePool{
		ch: ch,
	}
}

// Submit submits a new work unit to the pool. It will be handled once a worker is free.
// Note that we only accept a niladic function, and do not provide an indication of when it
// completes, so you would typically wrap the call you want in an anonymous function, i.e.
//   var wg sync.WaitGroup
//   wg.Add(1)
//   pool.Submit(func() {
//       callMyRealFunction(someParam)
//       wg.Done()
//   })
//   wg.Wait()
//
// Hint: ensure you are careful about closing over loop variables, Go closes over them by
//       reference not value so you may need to wrap them again (or use SubmitParam instead).
//
// No particular guarantee is made about whether this function will block or not.
func (pool *GoroutinePool) Submit(f func()) {
	pool.ch <- f
}

// SubmitParam is similar to Submit but allows submitting a single parameter with the function.
// This is often convenient to close over loop variables etc.
func (pool *GoroutinePool) SubmitParam(f func(interface{}), p interface{}) {
	pool.ch <- func() {
		f(p)
	}
}

// runWorker is the body of the actual worker.
func runWorker(ch <-chan func()) {
	for {
		f := <-ch
		f()
	}
}
