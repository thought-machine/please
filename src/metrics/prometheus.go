package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/prometheus/common/expfmt"
	"github.com/thought-machine/please/src/core"
	"gopkg.in/op/go-logging.v1"
)

var log = logging.MustGetLogger("metrics")

type metrics struct {
	gatewayURL           string
	downloadErrorCounter prometheus.Counter
}

var m *metrics

// InitFromConfig sets up the initial metrics from the configuration.
func InitFromConfig(config *core.Configuration) {
	m = &metrics{
		gatewayURL: config.Metrics.PrometheusGatewayURL,
	}

	m.downloadErrorCounter = prometheus.NewCounter(prometheus.CounterOpts{
		// Note: this can be called multiple times and won't affect the gateway's counter value.
		Name: "tree_digest_download_eof_error",
		Help: "Number of times the Unexpected EOF error has been seen during a tree digest download",
	})
}

// DownloadErrorCounterInc increments the tree_digest_download_eof_error counter
func DownloadErrorCounterInc() {
	if m == nil {
		log.Debug("Metrics have not been initialised")
		return
	}
	if m.gatewayURL == "" {
		log.Debug("No Prometheus pushgateway URL to push Digest Download error to")
		return
	}
	m.downloadErrorCounter.Inc()
	if err := push.New(
		m.gatewayURL, "tree_digest_download_eof_error",
	).Collector(m.downloadErrorCounter).Format(expfmt.FmtText).Push(); err != nil {
		log.Warningf("Error pushing to Prometheus pushgateway: %s", err)
	}
}
