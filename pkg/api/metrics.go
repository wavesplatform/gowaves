package api

import (
	"github.com/prometheus/client_golang/prometheus"
)

const httpAPIMetricsNamespace = "http_api"

var (
	metricApiTotalRequests = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: httpAPIMetricsNamespace,
			Name:      "total_hits",
			Help:      "Node HTTP API Requests count",
		},
	)

	metricApiHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: httpAPIMetricsNamespace,
			Name:      "path_hits",
			Help:      "Node HTTP API paths hits",
		},
		[]string{"status", "path"},
	)

	metricApiRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: httpAPIMetricsNamespace,
			Name:      "path_duration",
			// TODO(nickeskov): add custom buckets
		},
		[]string{"method", "path"},
	)
)

func init() {
	prometheus.MustRegister(
		metricApiTotalRequests,
		metricApiHits,
		metricApiRequestDuration,
	)
}
