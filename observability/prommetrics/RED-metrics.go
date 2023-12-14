package prommetrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"time"
)

/*
go get github.com/prometheus/client_golang/prometheus
go get github.com/prometheus/client_golang/prometheus/promauto
*/

type HandlerMetrics struct {
	failed    prometheus.Counter
	requests  prometheus.Counter
	durations *prometheus.HistogramVec
}

type RequestMetrics struct {
	start          time.Time
	handlerMetrics *HandlerMetrics
}

func NewHandlerMetrics(namespace, subsystem, name string) *HandlerMetrics {
	return &HandlerMetrics{
		failed: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      name + "_errors",
			Help:      "Total number of errors",
		}),
		requests: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      name + "_requests",
			Help:      "Total number of requests",
		}),
		durations: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      name + "_durations",
			Help:      "Total request time duration",
		}, []string{"method", "status", "message"}),
	}
}

func (r *HandlerMetrics) StartRequest() *RequestMetrics {
	r.requests.Inc()
	return &RequestMetrics{
		start:          time.Now(),
		handlerMetrics: r,
	}
}

func (r *RequestMetrics) Success(method string) {
	r.handlerMetrics.durations.WithLabelValues(method, "success", "no_error").Observe(time.Since(r.start).Seconds())
}

func (r *RequestMetrics) Failure(method, message string) {
	r.handlerMetrics.failed.Inc()
	r.handlerMetrics.durations.WithLabelValues(method, "failed", message).Observe(time.Since(r.start).Seconds())
}
