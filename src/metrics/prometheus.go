package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/core"
)

var log = logging.MustGetLogger("metrics")

var registerer = prometheus.WrapRegistererWith(prometheus.Labels{
	"version": core.PleaseVersion,
}, prometheus.DefaultRegisterer)

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
