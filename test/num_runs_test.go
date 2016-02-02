package test

import "testing"
import "time"

func TestNumRuns(t *testing.T) {
	// Just take up a little time here so it's obvious how a number of runs
	// multiplies up the duration (Go's so fast that this is essentially instant otherwise).
	time.Sleep(100 * time.Millisecond)
}
