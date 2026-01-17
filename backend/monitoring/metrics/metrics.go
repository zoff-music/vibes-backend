// Package metrics sets up and handles our promethous collectors.
package metrics

import (
	"math"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

// Global vars suck but here we are, this is how prom metrics work sadly
var (
	requestsReceived = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_request_status_code",
		Help: "Status codes returned by the API",
	},
		[]string{"status_code", "operation_name"},
	)
	timeToProcessRequest = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "http_request_duration",
		Help:    "Time spent processing requests",
		Buckets: []float64{.005, .01, .025, .05, .075, .1, .25, .5, .75, 1.0, 2.5, 5.0, 7.5, 10.0, math.Inf(1)},
	})
)

// RegisterPrometheusCollectors tells prometheus to set up collectors.
func RegisterPrometheusCollectors() {
	prometheus.MustRegister(requestsReceived)
	prometheus.MustRegister(timeToProcessRequest)
}

// ObserveTimeToProcess records the time spent processing an operation.
func ObserveTimeToProcess(operation string, t float64) {
	timeToProcessRequest.Observe(t)
}

// ReceivedRequest records the status code returned for each request.
func ReceivedRequest(statusCode int, operationName string) {
	requestsReceived.WithLabelValues(strconv.Itoa(statusCode), operationName).Inc()
}
