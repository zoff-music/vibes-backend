// Package metrics sets up and handles our promethous collectors.
package metrics

import (
	"math"
	"strconv"
	"sync"

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
	timeToProcessRequest = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration",
		Help:    "Time spent processing requests",
		Buckets: []float64{.005, .01, .025, .05, .075, .1, .25, .5, .75, 1.0, 2.5, 5.0, 7.5, 10.0, math.Inf(1)},
	}, []string{"operation_name"})
	taskStatus = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "task_status_code",
		Help: "Status codes returned by background tasks",
	}, []string{"status_code", "operation_name"})
	taskDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "task_duration",
		Help:    "Time spent processing background tasks",
		Buckets: []float64{.005, .01, .025, .05, .075, .1, .25, .5, .75, 1.0, 2.5, 5.0, 7.5, 10.0, math.Inf(1)},
	}, []string{"operation_name"})
)

var registerOnce sync.Once

// RegisterPrometheusCollectors tells prometheus to set up collectors.
func RegisterPrometheusCollectors() {
	registerOnce.Do(func() {
		prometheus.MustRegister(requestsReceived)
		prometheus.MustRegister(timeToProcessRequest)
		prometheus.MustRegister(taskStatus)
		prometheus.MustRegister(taskDuration)
	})
}

// ObserveTimeToProcess records the time spent processing an operation.
func ObserveTimeToProcess(operation string, t float64) {
	timeToProcessRequest.WithLabelValues(operation).Observe(t)
}

// ReceivedRequest records the status code returned for each request.
func ReceivedRequest(statusCode int, operationName string) {
	requestsReceived.WithLabelValues(strconv.Itoa(statusCode), operationName).Inc()
}

// ObserveTaskDuration records the time spent processing a background task.
func ObserveTaskDuration(operation string, t float64) {
	taskDuration.WithLabelValues(operation).Observe(t)
}

// ProcessedTask records the status returned by a background task.
func ProcessedTask(statusCode int, operationName string) {
	taskStatus.WithLabelValues(strconv.Itoa(statusCode), operationName).Inc()
}
