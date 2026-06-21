package inks

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
)

// Info carries introspection data about an Ink.
type Info struct {
	Kind       Kind
	MetricName metric.Name
}

// Info returns introspection data about the ink's kind and metric.
func (*fixedInk) Info() Info {
	return Info{Kind: KindFixed}
}

// Info returns introspection data about the ink's kind and metric.
func (ink *numericInk) Info() Info {
	return Info{Kind: KindNumeric, MetricName: ink.metricName}
}

// Info returns introspection data about the ink's kind and metric.
func (ink *categoricalInk) Info() Info {
	return Info{Kind: KindCategorical, MetricName: ink.metricName}
}
