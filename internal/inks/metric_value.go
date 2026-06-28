package inks

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
)

// MetricValue carries the metric data needed to resolve a colour.
// The Kind field determines which of the remaining fields is used.
type MetricValue struct {
	Kind     metric.Kind
	Measure  float64
	Quantity int
	Category string
}

// MeasureValue creates a MetricValue for a float64 measure.
func MeasureValue(v float64) MetricValue {
	return MetricValue{Kind: metric.Measure, Measure: v}
}

// QuantityValue creates a MetricValue for an integer quantity.
func QuantityValue(v int) MetricValue {
	return MetricValue{Kind: metric.Quantity, Quantity: v}
}

// CategoryValue creates a MetricValue for a string classification.
func CategoryValue(v string) MetricValue {
	return MetricValue{Kind: metric.Classification, Category: v}
}
