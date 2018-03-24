package core

// A Pool implements an adjustable worker pool.
// We use it to handle parse tasks where we need to be able to deal with tasks blocking.
// Push onto the channel to submit new tasks.
type Pool chan func()

// NewPool constructs a new pool of the given size & starts its workers.
func NewPool(size int) Pool {
	p := make(Pool)
	for i := 0; i < size; i++ {
		go p.Run()
	}
	return p
}

// Run is a worker function that consumes off the queue.
func (p Pool) Run() {
	for f := range p {
		if f == nil {
			return // Poison message, indicates to stop
		}
		f()
	}
}

// AddWorker adds a new worker to the pool.
func (p Pool) AddWorker() {
	go p.Run()
}

// StopWorker indicates to a worker that it should stop.
func (p Pool) StopWorker() {
	// Invoke in parallel so it doesn't block.
	go func() {
		p <- nil
	}()
}
