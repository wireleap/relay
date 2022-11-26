package telemetry

import (
	"reflect"
	"time"

	"github.com/cabify/gotoprom/prometheusvanilla"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// TimeHistogramType is the reflect.Type of the TimeHistogram interface
	TimeHistogramType = reflect.TypeOf((*TimeHistogram)(nil)).Elem()
)

// registerTimeHistogram registers a TimeHistogram after registering the underlying prometheus.Histogram in the prometheus.Registerer provided
// The function it returns returns a TimeHistogram type as an interface{}
func registerTimeHistogram(name, help, namespace string, labelNames []string, tag reflect.StructTag) (func(prometheus.Labels) interface{}, prometheus.Collector, error) {
	f, collector, err := prometheusvanilla.BuildHistogram(name, help, namespace, labelNames, tag)
	if err != nil {
		return nil, nil, err
	}

	return func(labels prometheus.Labels) interface{} {
		return timeHistogramAdapter{Histogram: f(labels).(prometheus.Histogram)}
	}, collector, nil
}

// TimeHistogram offers the basic prometheus.Histogram functionality
// with additional time-observing functions
type TimeHistogram interface {
	prometheus.Histogram
	// Duration observes the duration in seconds
	Duration(duration time.Duration)
	// Since observes the duration in seconds since the time point provided
	Since(time.Time)
}

type timeHistogramAdapter struct {
	prometheus.Histogram
}

// Duration observes the duration in seconds
func (to timeHistogramAdapter) Duration(duration time.Duration) {
	to.Observe(duration.Seconds())
}

// Since observes the duration in seconds since the time point provided
func (to timeHistogramAdapter) Since(duration time.Time) {
	to.Duration(time.Since(duration))
}
