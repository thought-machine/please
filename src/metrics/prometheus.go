// +build prometheus
// Package metrics contains support for reporting metrics to an external server,
// currently a Prometheus pushgateway. Because plz runs as a transient process
// we can't wait around for Prometheus to call us, we've got to push to them.
package metrics

import (
	"os/user"
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/op/go-logging.v1"

	"core"
)

var log = logging.MustGetLogger("metrics")

const (
	sleepDuration = 100 * time.Millisecond
	maxErrors     = 3
)

type metrics struct {
	url                                           string
	newMetrics                                    bool
	stop                                          bool
	errors                                        int
	buildCounter, cacheCounter, testCounter       prometheus.CounterVec
	buildHistogram, cacheHistogram, testHistogram prometheus.HistogramVec
}

// m is the singleton metrics instance. Nothing else deals with this.
var m *metrics

// InitFromConfig sets up the initial metrics from the configuration.
func InitFromConfig(config *core.Configuration) {
	if config.Metrics.PushGatewayURL == "" {
		return // Metrics not enabled
	}

	user, err := user.Current()
	if err != nil {
		log.Warning("Can't determine current user name for metrics")
		user = &os.User{Username: "unknown"}
	}
	constLabels := prometheus.Labels{
		"user": user.Username,
		"arch": runtime.GOOS + "_" + runtime.GOARCH,
	}

	m = &metrics{url: config.Metrics.PushGatewayURL}

	// Count of builds for each target.
	m.buildCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:        "build_counts",
		Help:        "Count of number of times each target is built",
		ConstLabels: constLabels,
	}, []string{"target", "success", "incremental"})

	// Count of cache hits for each target
	m.cacheCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:        "cache_hits",
		Help:        "Count of number of times we successfully retrieve from the cache",
		ConstLabels: constLabels,
	}, []string{"target", "hit"})

	// Count of test runs for each target
	m.testCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:        "test_runs",
		Help:        "Count of number of times we run each test",
		ConstLabels: constLabels,
	}, []string{"target", "pass"})

	// Build durations for each target
	m.buildHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:        "build_durations_histogram",
		Help:        "Durations of individual build targets",
		Buckets:     prometheus.LinearBuckets(0, 0.1, 100),
		ConstLabels: constLabels,
	}, []string{"target"})

	// Cache retrieval durations for each target
	m.cacheHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:        "cache_durations_histogram",
		Help:        "Durations to retrieve artifacts from the cache",
		Buckets:     prometheus.LinearBuckets(0, 0.1, 100),
		ConstLabels: constLabels,
	}, []string{"target"})

	// Test durations for each target
	m.testHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:        "test_durations_histogram",
		Help:        "Durations to run tests",
		Buckets:     prometheus.LinearBuckets(0, 1, 100),
		ConstLabels: constLabels,
	}, []string{"target"})

	go m.keepPushing()
}

// Stop shuts down the metrics and ensures the final ones are sent before returning.
func Stop() {
	m.stop = true
	if m.newMetrics {
		m.pushMetrics()
	}
}

// Record records metrics for the given target.
func Record(target *core.BuildTarget, duration time.Duration) {
	label := target.Label.String()
	if target.Results.NumTests > 0 {
		// Tests have run
		m.cacheCounter.WithLabelValues(label, b(target.Results.Cached)).Inc()
		m.testCounter.WithLabelValues(label, b(target.Results.NumFailures == 0)).Inc()
		if target.Results.Cached {
			m.cacheHistogram.GetMetricWithLabelValues(label).Observe(duration.Seconds())
		} else {
			m.testHistogram.GetMetricWithLabelValues(label).Observe(duration.Seconds())
		}
	} else {
		// Build has run
		m.cacheCounter.WithLabelValues(label, b(target.State == core.Cached)).Inc()
		m.buildCounter.WithLabelValues(label, b(target.State != core.Failed), b(target.State != core.Reused)).Inc()
		if target.Results.Cached {
			m.cacheHistogram.GetMetricWithLabelValues(label).Observe(duration.Seconds())
		} else {
			m.buildHistogram.GetMetricWithLabelValues(label).Observe(duration.Seconds())
		}
	}
}

func b(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func (m *metrics) keepPushing() {
	for !m.stop {
		if !m.newMetrics {
			time.Sleep(m.sleepDuration)
		}
		m.newMetrics = false
		m.errors = m.pushMetrics()
		if m.errors >= maxErrors {
			log.Warning("Metrics don't seem to be working, giving up")
			return
		}
	}
}

func (m *metrics) pushMetrics() int {
	if err := push.AddFromGatherer("please", push.HostnameGroupingKey(), m.url, prometheus.DefaultGatherer); err != nil {
		log.Warning("Could not push metrics to the repository: %s", err)
		return m.errors + 1
	}
	return 0
}
