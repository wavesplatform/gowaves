package network

import "github.com/prometheus/client_golang/prometheus"

var metricGetPeersMessage = prometheus.NewCounter(
	prometheus.CounterOpts{
		Namespace: "messages",
		Name:      "get_peers",
		Help:      "Counter of GetPeers message.",
	},
)

var metricPeersMessage = prometheus.NewCounter(
	prometheus.CounterOpts{
		Namespace: "messages",
		Name:      "peers",
		Help:      "Counter of Peers message.",
	},
)

func init() {
	prometheus.MustRegister(metricPeersMessage)
	prometheus.MustRegister(metricGetPeersMessage)
}
