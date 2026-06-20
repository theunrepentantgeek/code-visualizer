package inks

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// Info carries introspection data about an Ink.
type Info struct {
	Kind       Kind
	MetricName metric.Name
}

// Info returns introspection data about the ink's kind and metric.
func (ink *baseInk) Info() Info {
	return Info{
		Kind:       ink.kind,
		MetricName: ink.metricName,
	}
}

// Boundaries returns the bucket boundary values for numeric inks.
// Returns nil for fixed or categorical inks.
func (ink *baseInk) Boundaries() []float64 {
	if ink.kind != KindNumeric || ink.boundaries == nil {
		return nil
	}

	return ink.boundaries.Boundaries
}

// Palette returns the colour palette used by this ink.
// Returns an empty palette for fixed inks.
func (ink *baseInk) Palette() palette.ColourPalette {
	if ink.kind == KindFixed {
		return palette.ColourPalette{}
	}

	return ink.pal
}

// Categories returns the category labels for categorical inks.
// Returns nil for fixed or numeric inks.
func (ink *baseInk) Categories() []string {
	if ink.kind != KindCategorical {
		return nil
	}

	return ink.categories
}
