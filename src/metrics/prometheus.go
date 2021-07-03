package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/core"
)

var log = logging.MustGetLogger("metrics")

var registerer = prometheus.WrapRegistererWith(prometheus.Labels{
	"version": core.PleaseVersion,
}, prometheus.DefaultRegisterer)

// defBuckets are the default histogram buckets (which are a bit longer than Prometheus'
// default since we have a bunch of operations that can run longer than theirs)
var defBuckets = []float64{.05, .1, .5, 1.0, 5, 10, 50, 100, 500}

// Push performs a single push of all registered metrics to the pushgateway (if configured).
func Push(config *core.Configuration) {
	if config.Metrics.PrometheusGatewayURL == "" {
		return
	}
	if err := push.New(config.Metrics.PrometheusGatewayURL, "please").Gatherer(prometheus.DefaultGatherer).Push(); err != nil {
		log.Warning("Error pushing Prometheus metrics: %s", err)
	}
}

// MustRegister registers the given metric with Prometheus, applying some standard labels.
// This should typically be called from an init() function to ensure it happens exactly once.
func MustRegister(cs ...prometheus.Collector) {
	registerer.MustRegister(cs...)
}

// NewCounter creates & registers a new counter.
func NewCounter(subsystem, name, help string) prometheus.Counter {
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "plz",
		Subsystem: subsystem,
		Name:      name,
		Help:      help,
	})
	MustRegister(counter)
	return counter
}

// NewHistogram creates & registers a new histogram.
func NewHistogram(subsystem, name, help string, labels ...string) prometheus.Histogram {
	histogram := prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "plz",
		Subsystem: subsystem,
		Name:      name,
		Help:      help,
		Buckets:   defBuckets,
	})
	MustRegister(histogram)
	return histogram
}

// NewLabelledHistogram creates & registers a new histogram with labels.
func NewLabelledHistogram(subsystem, name, help string, labels []string) *prometheus.HistogramVec {
	histogram := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "plz",
		Subsystem: subsystem,
		Name:      name,
		Help:      help,
		Buckets:   defBuckets,
	}, labels)
	MustRegister(histogram)
	return histogram
}

// Duration provides a convenience wrapper for observing histogram durations.
// Use it like so:
// defer metrics.Duration(histogram).Observe()
func Duration(histogram prometheus.Observer) Observer {
	return Observer{
		hist:  histogram,
		start: time.Now(),
	}
}

type Observer struct {
	hist  prometheus.Observer
	start time.Time
}

func (obs Observer) Observe() {
	obs.hist.Observe(time.Since(obs.start).Seconds())
}
