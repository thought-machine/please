package metrics

import (
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/prometheus/common/expfmt"

	"github.com/thought-machine/please/src/cli/logging"
	"github.com/thought-machine/please/src/core"
)

var log = logging.Log

var registerer = prometheus.WrapRegistererWith(prometheus.Labels{
	"version": core.PleaseVersion,
}, prometheus.DefaultRegisterer)

// Push performs a single push of all registered metrics to the pushgateway (if configured).
func Push(config *core.Configuration) {
	if config.Metrics.PrometheusGatewayURL == "" {
		return
	}

	if config.Metrics.PushHostinfo {
		name, _ := os.Hostname()
		gauge := prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "plz",
			Subsystem: "metrics",
			Name:      "hostinfo",
			Help:      "Please host running info",
			ConstLabels: prometheus.Labels{
				"remote":   strconv.FormatBool(config.Remote.URL != ""),
				"version":  config.Please.Version.String(),
				"hostname": name,
			},
		})
		MustRegister(gauge)
		gauge.Set(1)
	}

	if err := push.New(config.Metrics.PrometheusGatewayURL, "please").
		Client(&http.Client{Timeout: time.Duration(config.Metrics.Timeout)}).
		Gatherer(prometheus.DefaultGatherer).Format(expfmt.FmtText).
		Push(); err != nil {
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
func NewHistogram(subsystem, name, help string, buckets []float64) prometheus.Histogram {
	histogram := prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "plz",
		Subsystem: subsystem,
		Name:      name,
		Buckets:   buckets,
		Help:      help,
	})
	MustRegister(histogram)
	return histogram
}

func ExponentialBuckets(start, factor float64, numBuckets int) []float64 {
	return prometheus.ExponentialBuckets(start, factor, numBuckets)
}
