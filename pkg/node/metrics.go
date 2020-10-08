package node

import "github.com/prometheus/client_golang/prometheus"

var metricInternalChannelSize = prometheus.NewGauge(
	prometheus.GaugeOpts{
		//Namespace: "facade",
		Name: "messages_channel_size",
		Help: "The number of incoming message still in queue.",
	},
)

func init() {
	prometheus.MustRegister(metricInternalChannelSize)
}
