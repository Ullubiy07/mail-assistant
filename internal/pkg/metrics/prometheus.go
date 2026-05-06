package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	HttpRequestByPath = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_by_path",
			Help: "Number of HTTP requests by path.",
		},
		[]string{"handle"},
	)
	HttpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration",
			Help: "Response time of HTTP request.",
		},
		[]string{"handle"},
	)
)

func init() {
	prometheus.MustRegister(HttpRequestByPath)
}
