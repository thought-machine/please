package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"core"
)

const url = "http://localhost:9999"
const verySlow = 10000000 // Long duration so it never actually reports anything.

var label = core.BuildLabel{PackageName: "src/metrics", Name: "prometheus"}

func TestNoMetrics(t *testing.T) {
	m := initMetrics(url, verySlow, nil)
	assert.Equal(t, 0, m.errors)
	assert.Equal(t, 0, m.pushes)
	m.stop()
	assert.Equal(t, 0, m.errors, "Stop should not push when there aren't metrics")
}

func TestSomeMetrics(t *testing.T) {
	m := initMetrics(url, verySlow, nil)
	assert.Equal(t, 0, m.errors)
	assert.Equal(t, 0, m.pushes)
	m.record(core.NewBuildTarget(label), time.Millisecond)
	m.stop()
	assert.Equal(t, 1, m.errors, "Stop should push once more when there are metrics")
}

func TestTargetStates(t *testing.T) {
	m := initMetrics(url, verySlow, nil)
	assert.Equal(t, 0, m.errors)
	assert.Equal(t, 0, m.pushes)
	target := core.NewBuildTarget(label)
	m.record(target, time.Millisecond)
	target.SetState(core.Cached)
	m.record(target, time.Millisecond)
	target.SetState(core.Built)
	m.record(target, time.Millisecond)
	target.Results.NumTests = 3
	m.record(target, time.Millisecond)
	target.Results.Failed = 1
	m.record(target, time.Millisecond)
	target.Results.Cached = true
	m.record(target, time.Millisecond)
	m.stop()
	assert.Equal(t, 1, m.errors)
}

func TestPushAttempts(t *testing.T) {
	m := initMetrics(url, 1, nil) // Fast push attempts
	assert.Equal(t, 0, m.errors)
	assert.Equal(t, 0, m.pushes)
	m.record(core.NewBuildTarget(label), time.Millisecond)
	time.Sleep(50 * time.Millisecond) // Not ideal but should be heaps of time for it to attempt pushes.
	assert.Equal(t, maxErrors, m.errors)
	assert.True(t, m.cancelled)
	m.stop()
	assert.Equal(t, maxErrors, m.errors, "Should not push again if it's hit the max errors")
}

func TestCustomLabels(t *testing.T) {
	m := initMetrics(url, verySlow, map[string]string{
		"mylabel": "echo hello",
	})
	// It's a little bit fiddly to observe that the const label has been set as expected.
	c := m.cacheCounter.WithLabelValues("//src/metrics:metrics_test", "false")
	assert.Contains(t, c.Desc().String(), `mylabel="hello"`)
}

func TestCustomLabelsShlex(t *testing.T) {
	// Naive splitting will not produce good results here.
	m := initMetrics(url, verySlow, map[string]string{
		"mylabel": "bash -c 'echo hello'",
	})
	c := m.cacheCounter.WithLabelValues("//src/metrics:metrics_test", "false")
	assert.Contains(t, c.Desc().String(), `mylabel="hello"`)
}

func TestCustomLabelsShlexInvalid(t *testing.T) {
	assert.Panics(t, func() {
		initMetrics(url, verySlow, map[string]string{
			"mylabel": "bash -c 'echo hello", // missing trailing quote
		})
	})
}

func TestCustomLabelsCommandFails(t *testing.T) {
	assert.Panics(t, func() {
		initMetrics(url, verySlow, map[string]string{
			"mylabel": "wibble",
		})
	})
}

func TestCustomLabelsCommandNewlines(t *testing.T) {
	assert.Panics(t, func() {
		initMetrics(url, verySlow, map[string]string{
			"mylabel": "echo 'hello\nworld\n'",
		})
	})
}

func TestExportedFunctions(t *testing.T) {
	// For various reasons it's important that this is the only test that uses the global singleton.
	config := core.DefaultConfiguration()
	config.Metrics.PushGatewayURL = url
	config.Metrics.PushFrequency = verySlow
	InitFromConfig(config)
	Record(core.NewBuildTarget(label), time.Millisecond)
	Stop()
	assert.Equal(t, 1, m.errors)
}
