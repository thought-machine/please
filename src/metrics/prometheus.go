package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/thought-machine/please/src/core"
	"gopkg.in/op/go-logging.v1"
)

var log = logging.MustGetLogger("metrics")

// Push performs a single push of all registered metrics to the pushgateway (if configured).
func Push(config *core.Configuration) {
	if config.Metrics.PrometheusGatewayURL == "" {
		return
	}
	if err := push.New(config.Metrics.PrometheusGatewayURL, "please").Gatherer(prometheus.DefaultGatherer).Push(); err != nil {
		log.Warning("Error pushing Prometheus metrics: %s", err)
	}
}
