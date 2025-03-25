package metrics

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/prometheus/common/expfmt"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/cli/logging"
	"github.com/thought-machine/please/src/version"
)

var log = logging.Log

var registerer = prometheus.WrapRegistererWith(prometheus.Labels{
	"version": version.PleaseVersion,
}, prometheus.DefaultRegisterer)

type Config struct {
	PrometheusGatewayURL string       `help:"The aggregation gateway URL to push prometheus updates to."`
	Timeout              cli.Duration `help:"timeout for pushing to the gateway. Defaults to 2 seconds." `
	PushHostInfo         bool         `help:"Whether to push host info"`
}

// Push performs a single push of all registered metrics to the pushgateway (if configured).
func Push(config Config, isRemoteExecution bool) {
	if family, err := prometheus.DefaultGatherer.Gather(); err == nil {
		var buf strings.Builder
		for _, fam := range family {
			buf.Reset()
			if _, err := expfmt.MetricFamilyToText(&buf, fam); err == nil {
				for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
					if !strings.HasPrefix(line, "#") {
						log.Debug("Metric recorded: %s", line)
					}
				}
			}
		}
	}

	if config.PrometheusGatewayURL == "" {
		return
	}

	if config.PushHostInfo {
		name, _ := os.Hostname()
		counter := prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "plz",
			Subsystem: "metrics",
			Name:      "hostinfo",
			Help:      "Please host running info",
			ConstLabels: prometheus.Labels{
				"remote":   strconv.FormatBool(isRemoteExecution),
				"hostname": name,
			},
		})
		MustRegister(counter)
		counter.Inc()
	}

	if err := push.New(config.PrometheusGatewayURL, "please").
		Client(&http.Client{Timeout: time.Duration(config.Timeout)}).
		Gatherer(prometheus.DefaultGatherer).Format(expfmt.NewFormat(expfmt.TypeTextPlain)).
		Push(); err != nil {
		log.Warning("Error pushing Prometheus metrics: %s", err)
	}
}

// MustRegister registers the given metric with Prometheus, applying some standard labels.
// This should typically be called from an init() function to ensure it happens exactly once.
func MustRegister(cs ...prometheus.Collector) {
	registerer.MustRegister(cs...)
}

// NewCounterVec creates & registers a new counter.
func NewCounterVec(subsystem, name, help string, labelNames []string) *prometheus.CounterVec {
	counter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "plz",
		Subsystem: subsystem,
		Name:      name,
		Help:      help,
	}, labelNames)
	MustRegister(counter)
	return counter
}

// NewHistogramVec creates & registers a new histogram.
func NewHistogramVec(subsystem, name, help string, buckets []float64, labelNames []string) *prometheus.HistogramVec {
	histogram := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "plz",
		Subsystem: subsystem,
		Name:      name,
		Buckets:   buckets,
		Help:      help,
	}, labelNames)
	MustRegister(histogram)
	return histogram
}

func ExponentialBuckets(start, factor float64, numBuckets int) []float64 {
	return prometheus.ExponentialBuckets(start, factor, numBuckets)
}

// CILabel is the value to set for `ci` labels on metrics based on the presence of the `CI` environment variable.
var CILabel = func() string {
	if val := strings.ToLower(os.Getenv("CI")); val == "true" {
		return "true"
	}
	return "false"
}()
