package run

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPoolRunsWorker(t *testing.T) {
	ch := make(chan int)
	p := NewGoroutinePool(5)
	p.Submit(func() {
		ch <- 42
	})
	assert.Equal(t, 42, <-ch)
}

func TestPoolHasSufficientWorkers(t *testing.T) {
	var wg sync.WaitGroup
	var count int64
	wg.Add(5)
	p := NewGoroutinePool(5)
	for i := 0; i < 5; i++ {
		p.Submit(func() {
			atomic.AddInt64(&count, 1)
			wg.Done()
			// This blocks forever so if the pool hasn't provisioned enough workers
			// we'll never get to the end of the test.
			select {}
		})
	}
	wg.Wait()
	assert.EqualValues(t, 5, count)
}

func TestSubmitParam(t *testing.T) {
	ch := make(chan interface{})
	p := NewGoroutinePool(5)
	p.SubmitParam(func(p interface{}) {
		ch <- p
	}, 42)
	v := <-ch
	assert.Equal(t, 42, v.(int))
}
