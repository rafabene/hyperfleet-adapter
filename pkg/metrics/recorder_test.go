package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRecorder(t *testing.T) {
	registry := prometheus.NewRegistry()
	recorder := NewRecorder("test-adapter", "v0.1.0", registry)
	require.NotNil(t, recorder)

	// Trigger all metrics so they appear in Gather()
	recorder.RecordEventProcessed("success")
	recorder.ObserveProcessingDuration(1 * time.Millisecond)
	recorder.RecordError("test")

	families, err := registry.Gather()
	require.NoError(t, err)

	names := make(map[string]bool)
	for _, f := range families {
		names[f.GetName()] = true
	}

	assert.True(t, names["hyperfleet_adapter_events_processed_total"],
		"events_processed_total should be registered")
	assert.True(t, names["hyperfleet_adapter_event_processing_duration_seconds"],
		"event_processing_duration_seconds should be registered")
	assert.True(t, names["hyperfleet_adapter_errors_total"],
		"errors_total should be registered")
}

func TestRecordEventProcessed(t *testing.T) {
	registry := prometheus.NewRegistry()
	recorder := NewRecorder("test-adapter", "v0.1.0", registry)

	recorder.RecordEventProcessed("success")
	recorder.RecordEventProcessed("success")
	recorder.RecordEventProcessed("failed")
	recorder.RecordEventProcessed("skipped")
	recorder.RecordEventProcessed("skipped")
	recorder.RecordEventProcessed("skipped")

	families, err := registry.Gather()
	require.NoError(t, err)

	var eventsFamily *dto.MetricFamily
	for _, f := range families {
		if f.GetName() == "hyperfleet_adapter_events_processed_total" {
			eventsFamily = f
			break
		}
	}
	require.NotNil(t, eventsFamily, "events_processed_total metric family should exist")

	counts := make(map[string]float64)
	for _, m := range eventsFamily.GetMetric() {
		for _, l := range m.GetLabel() {
			if l.GetName() == "status" {
				counts[l.GetValue()] = m.GetCounter().GetValue()
			}
		}
	}

	assert.Equal(t, float64(2), counts["success"], "success count")
	assert.Equal(t, float64(1), counts["failed"], "failed count")
	assert.Equal(t, float64(3), counts["skipped"], "skipped count")
}

func TestRecordEventProcessed_ConstLabels(t *testing.T) {
	registry := prometheus.NewRegistry()
	recorder := NewRecorder("my-adapter", "v1.2.3", registry)

	recorder.RecordEventProcessed("success")

	families, err := registry.Gather()
	require.NoError(t, err)

	var eventsFamily *dto.MetricFamily
	for _, f := range families {
		if f.GetName() == "hyperfleet_adapter_events_processed_total" {
			eventsFamily = f
			break
		}
	}
	require.NotNil(t, eventsFamily)

	// Verify component and version ConstLabels are present
	m := eventsFamily.GetMetric()[0]
	labels := make(map[string]string)
	for _, l := range m.GetLabel() {
		labels[l.GetName()] = l.GetValue()
	}

	assert.Equal(t, "my-adapter", labels["component"], "component label")
	assert.Equal(t, "v1.2.3", labels["version"], "version label")
}

func TestObserveProcessingDuration(t *testing.T) {
	registry := prometheus.NewRegistry()
	recorder := NewRecorder("test-adapter", "v0.1.0", registry)

	recorder.ObserveProcessingDuration(500 * time.Millisecond)
	recorder.ObserveProcessingDuration(2 * time.Second)

	families, err := registry.Gather()
	require.NoError(t, err)

	var durationFamily *dto.MetricFamily
	for _, f := range families {
		if f.GetName() == "hyperfleet_adapter_event_processing_duration_seconds" {
			durationFamily = f
			break
		}
	}
	require.NotNil(t, durationFamily, "event_processing_duration_seconds metric family should exist")

	m := durationFamily.GetMetric()[0]
	histogram := m.GetHistogram()
	require.NotNil(t, histogram)

	assert.Equal(t, uint64(2), histogram.GetSampleCount(), "sample count")
	assert.InDelta(t, 2.5, histogram.GetSampleSum(), 0.01, "sample sum")

	// Verify bucket boundaries match expected values
	expectedBuckets := []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120}
	buckets := histogram.GetBucket()
	require.Len(t, buckets, len(expectedBuckets), "number of buckets")
	for i, b := range buckets {
		assert.Equal(t, expectedBuckets[i], b.GetUpperBound(), "bucket %d upper bound", i)
	}
}

func TestRecordError(t *testing.T) {
	registry := prometheus.NewRegistry()
	recorder := NewRecorder("test-adapter", "v0.1.0", registry)

	recorder.RecordError("param_extraction")
	recorder.RecordError("preconditions")
	recorder.RecordError("preconditions")
	recorder.RecordError("resources")

	families, err := registry.Gather()
	require.NoError(t, err)

	var errorsFamily *dto.MetricFamily
	for _, f := range families {
		if f.GetName() == "hyperfleet_adapter_errors_total" {
			errorsFamily = f
			break
		}
	}
	require.NotNil(t, errorsFamily, "errors_total metric family should exist")

	counts := make(map[string]float64)
	for _, m := range errorsFamily.GetMetric() {
		for _, l := range m.GetLabel() {
			if l.GetName() == "error_type" {
				counts[l.GetValue()] = m.GetCounter().GetValue()
			}
		}
	}

	assert.Equal(t, float64(1), counts["param_extraction"], "param_extraction error count")
	assert.Equal(t, float64(2), counts["preconditions"], "preconditions error count")
	assert.Equal(t, float64(1), counts["resources"], "resources error count")
}

func TestNilRecorderNoPanic(t *testing.T) {
	var recorder *Recorder

	// All methods should be no-ops and not panic
	assert.NotPanics(t, func() {
		recorder.RecordEventProcessed("success")
	}, "RecordEventProcessed on nil recorder")

	assert.NotPanics(t, func() {
		recorder.ObserveProcessingDuration(1 * time.Second)
	}, "ObserveProcessingDuration on nil recorder")

	assert.NotPanics(t, func() {
		recorder.RecordError("test_error")
	}, "RecordError on nil recorder")
}
