package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HTTPRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "voidwg_http_requests_total",
		Help: "Total HTTP requests",
	}, []string{"method", "route", "status"})

	HTTPDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "voidwg_http_request_duration_seconds",
		Help:    "HTTP request duration",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route"})

	PeersActive = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "voidwg_peers_active",
		Help: "Number of active peers",
	})

	AgentsOnline = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "voidwg_agents_online",
		Help: "Online agents count",
	})
)
