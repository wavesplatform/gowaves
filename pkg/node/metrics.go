package node

import "github.com/prometheus/client_golang/prometheus"

var metricInternalChannelSize = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "messages_channel_size",
		Help: "The number of incoming message still in queue.",
	},
)

var metricPeersMessage = prometheus.NewCounter(
	prometheus.CounterOpts{
		Namespace: "messages",
		Name:      "peers",
		Help:      "Counter of Peers message.",
	},
)

var metricGetPeersMessage = prometheus.NewCounter(
	prometheus.CounterOpts{
		Namespace: "messages",
		Name:      "get_peers",
		Help:      "Counter of GetPeers message.",
	},
)

var metricBlockMessage = prometheus.NewCounter(
	prometheus.CounterOpts{
		Namespace: "messages",
		Name:      "block",
		Help:      "Counter of Block message.",
	},
)

var metricGetBlockMessage = prometheus.NewCounter(
	prometheus.CounterOpts{
		Namespace: "messages",
		Name:      "get_block",
		Help:      "Counter of GetBlock message.",
	},
)

func init() {
	prometheus.MustRegister(metricInternalChannelSize)
	prometheus.MustRegister(metricPeersMessage)
	prometheus.MustRegister(metricGetPeersMessage)
	prometheus.MustRegister(metricBlockMessage)
	prometheus.MustRegister(metricGetBlockMessage)
}
