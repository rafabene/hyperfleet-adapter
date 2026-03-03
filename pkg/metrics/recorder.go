// Package metrics provides Prometheus metrics recording for the HyperFleet adapter.
// It follows the HyperFleet Metrics Standard with the hyperfleet_adapter_ prefix.
package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Recorder registers and records adapter-level Prometheus metrics.
// All methods are nil-safe: calling methods on a nil *Recorder is a no-op,
// which allows dry-run mode to skip metrics without nil checks at every call site.
type Recorder struct {
	eventsProcessed    *prometheus.CounterVec
	processingDuration prometheus.Observer
	errorsTotal        *prometheus.CounterVec
}

// NewRecorder creates a new Recorder and registers metrics with the given registerer.
// If reg is nil, prometheus.DefaultRegisterer is used.
func NewRecorder(component, version string, reg prometheus.Registerer) *Recorder {
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}

	eventsProcessed := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "hyperfleet_adapter_events_processed_total",
			Help: "Total number of CloudEvents processed by the adapter",
			ConstLabels: prometheus.Labels{
				"component": component,
				"version":   version,
			},
		},
		[]string{"status"},
	)

	processingDuration := prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "hyperfleet_adapter_event_processing_duration_seconds",
			Help:    "Duration of event processing in seconds",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120},
			ConstLabels: prometheus.Labels{
				"component": component,
				"version":   version,
			},
		},
	)

	errorsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "hyperfleet_adapter_errors_total",
			Help: "Total number of errors encountered by the adapter",
			ConstLabels: prometheus.Labels{
				"component": component,
				"version":   version,
			},
		},
		[]string{"error_type"},
	)

	reg.MustRegister(eventsProcessed)
	reg.MustRegister(processingDuration)
	reg.MustRegister(errorsTotal)

	return &Recorder{
		eventsProcessed:    eventsProcessed,
		processingDuration: processingDuration,
		errorsTotal:        errorsTotal,
	}
}

// RecordEventProcessed increments the events_processed_total counter for the given status.
// Valid status values: "success", "failed", "skipped".
func (r *Recorder) RecordEventProcessed(status string) {
	if r == nil {
		return
	}
	r.eventsProcessed.WithLabelValues(status).Inc()
}

// ObserveProcessingDuration records the event processing duration in seconds.
func (r *Recorder) ObserveProcessingDuration(d time.Duration) {
	if r == nil {
		return
	}
	r.processingDuration.Observe(d.Seconds())
}

// RecordError increments the errors_total counter for the given error type.
// Error types correspond to execution phases: "param_extraction", "preconditions",
// "resources", "post_actions".
func (r *Recorder) RecordError(errorType string) {
	if r == nil {
		return
	}
	r.errorsTotal.WithLabelValues(errorType).Inc()
}
