// +build !bootstrap

package follow

import (
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"

	"github.com/thought-machine/please/src/core"
)

// resourceUpdateFrequency is the frequency that we re-check CPU usage etc at.
// We don't want to sample too often; it will actually make CPU usage less accurate, and
// we don't want to spend all our time looking at it anyway.
var resourceUpdateFrequency = 500 * time.Millisecond

// UpdateResources continuously updates the resources that we store on the BuildState object.
// It should probably be run in a goroutine since it never returns.
func UpdateResources(state *core.BuildState) {
	lastTime := time.Now()
	// Assume this doesn't change through the process lifetime.
	count, _ := cpu.Counts(true)
	state.Stats.CPU.Count = count
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
		if thisTotal, thisIO := getCPUTimes(); thisTotal > 0.0 {
			elapsed := timeNow.Sub(lastTime).Seconds()
			state.Stats.CPU.Used = clamp(100.0 * (thisTotal - lastTotal) / elapsed)
			state.Stats.CPU.IOWait = clamp(100.0 * (thisIO - lastIO) / elapsed)
			lastTotal, lastIO = thisTotal, thisIO
		}
		// Thank goodness memory is a simpler beast.
		if vm, err := mem.VirtualMemory(); err != nil {
			log.Error("Error getting memory usage: %s", err)
		} else {
			state.Stats.Memory.Total = vm.Total
			state.Stats.Memory.Used = vm.Used
			state.Stats.Memory.UsedPercent = vm.UsedPercent
		}
		lastTime = timeNow
	}
}

func getCPUTimes() (float64, float64) {
	ts, err := cpu.Times(false) // not per CPU
	if err != nil || len(ts) == 0 {
		log.Error("Error getting CPU info: %s", err)
		return 0.0, 0.0
	}
	t := ts[0]
	return t.Total() - t.Idle - t.Iowait, t.Iowait
}
