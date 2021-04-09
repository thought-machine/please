package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/prometheus/common/expfmt"
	"gopkg.in/op/go-logging.v1"
)

var log = logging.MustGetLogger("metrics")

// RemoteMetrics is a type for maintaining please remote metrics
type RemoteMetrics struct {
	downloadErrorCounter prometheus.Counter
}

// NewRemoteMetrics initialises the remote metrics
func NewRemoteMetrics() *RemoteMetrics {
	downloadErrorCounter := prometheus.NewCounter(prometheus.CounterOpts{
		// Note: this can be called multiple times and won't affect the gateway's counter value.
		Name: "tree_digest_download_eof_error",
		Help: "Number of times the Unexpected EOF error has been seen during a tree digest download",
	})

	return &RemoteMetrics{
		downloadErrorCounter: downloadErrorCounter,
	}
}

// DownloadErrorCounterInc increments the 'tree digest download error' counter
func (r *RemoteMetrics) DownloadErrorCounterInc(gatewayURL string) {
	if gatewayURL == "" {
		log.Debug("No Prometheus pushgateway URL to push Digest Download error to")
		return
	}
	r.downloadErrorCounter.Inc()
	if err := push.New(
		gatewayURL, "tree_digest_download_eof_error",
	).Collector(r.downloadErrorCounter).Format(expfmt.FmtText).Push(); err != nil {
		log.Warningf("Error pushing to Prometheus pushgateway: %s", err)
	}
}
