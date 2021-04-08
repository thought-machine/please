package remote

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/prometheus/common/expfmt"
)

// remoteMetrics is a type for maintaining remote execution metrics
type remoteMetrics struct {
	downloadErrorCounter prometheus.Counter
}

func newRemoteMetrics() *remoteMetrics {
	// Note: this will be called for each new remote Client, but won't affect the counter on the
	// aggregation gateway.
	downloadErrorCounter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "tree_digest_download_eof_error",
		Help: "Number of times the Unexpected EOF error has been seen during a tree digest download",
	})

	return &remoteMetrics{
		downloadErrorCounter: downloadErrorCounter,
	}
}

func (c *Client) downloadErrorCounterInc() {
	if c.state.Config.Metrics.PrometheusGatewayURL == "" {
		log.Debug("No Prometheus pushgateway URL to push Digest Download error to")
		return
	}
	c.metrics.downloadErrorCounter.Inc()
	if err := push.New(
		c.state.Config.Metrics.PrometheusGatewayURL, "tree_digest_download_eof_error",
	).Collector(c.metrics.downloadErrorCounter).Format(expfmt.FmtText).Push(); err != nil {
		log.Warningf("Error pushing to Prometheus pushgateway: %s", err)
	}
}
