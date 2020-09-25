package benchmark

import (
	"testing"
	"time"
)

func BenchmarkOneSecWait(b *testing.B) {
	time.Sleep(time.Second)
}

func Benchmark100msWait(b *testing.B) {
	time.Sleep(100 * time.Millisecond)
}

func TestSomething(t *testing.T) {
	panic("shouldn't run")
}
