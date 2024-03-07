package core

import (
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

// resourceUpdateFrequency is the frequency that we re-check CPU usage etc at.
// We don't want to sample too often; it will actually make CPU usage less accurate, and
// we don't want to spend all our time looking at it anyway.
var resourceUpdateFrequency = 500 * time.Millisecond

// UpdateResources continuously updates the resources on this state object.
// It should probably be run in a goroutine since it never returns.
func (state *BuildState) UpdateResources() {
	stats := &state.stats.Stats

	lastTime := time.Now()
	// Assume this doesn't change through the process lifetime.
	count, _ := cpu.Counts(true)
	state.stats.Lock()
	stats.CPU.Count = count
	state.stats.Unlock()

	// Top out max CPU; sometimes we get our maths slightly wrong, probably because of
	// mild uncertainty in times.
	maxCPU := float64(100 * count)
	clamp := func(f float64) float64 {
		if f >= maxCPU {
			return maxCPU
		} else if f <= 0.0 {
			return 0.0
		}
		return f
	}
	// CPU is a bit of a fiddle since the kernel only gives us totals,
	// so we have to sample how busy we think it's been.
	lastTotal, lastIO := getCPUTimes()
	for timeNow := range time.NewTicker(resourceUpdateFrequency).C {
		state.stats.Lock()
		if thisTotal, thisIO := getCPUTimes(); thisTotal > 0.0 {
			elapsed := timeNow.Sub(lastTime).Seconds()
			stats.CPU.Used = clamp(100.0 * (thisTotal - lastTotal) / elapsed)
			stats.CPU.IOWait = clamp(100.0 * (thisIO - lastIO) / elapsed)
			lastTotal, lastIO = thisTotal, thisIO
		}
		// Thank goodness memory is a simpler beast.
		if vm, err := mem.VirtualMemory(); err != nil {
			log.Error("Error getting memory usage: %s", err)
		} else {
			stats.Memory.Total = vm.Total
			stats.Memory.Used = vm.Used
			stats.Memory.UsedPercent = vm.UsedPercent
		}
		state.stats.Unlock()
		lastTime = timeNow
	}
}

// SystemStats returns the current view of system stats.
func (state *BuildState) SystemStats() SystemStats {
	state.stats.Lock()
	defer state.stats.Unlock()
	return state.stats.Stats
}

// SetNumWorkerStat sets the stat for the number of background workers
func (state *BuildState) SetNumWorkerStat(n int) {
	state.stats.Lock()
	defer state.stats.Unlock()
	state.stats.Stats.NumWorkerProcesses = n
}

func getCPUTimes() (float64, float64) {
	ts, err := cpu.Times(false) // not per CPU
	if err != nil || len(ts) == 0 {
		log.Error("Error getting CPU info: %s", err)
		return 0.0, 0.0
	}
	t := ts[0]
	return t.Total() - t.Idle - t.Iowait, t.Iowait // nolint:staticcheck
}
