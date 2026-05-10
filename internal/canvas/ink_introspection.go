package canvas

import (
	"image/color"

	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/palette"
)

// InkKind identifies the type of ink for introspection.
type InkKind int

const (
	InkFixed       InkKind = InkKind(inkFixed)
	InkNumeric     InkKind = InkKind(inkNumeric)
	InkCategorical InkKind = InkKind(inkCategorical)
)

// InkInfo carries introspection data about an Ink.
type InkInfo struct {
	Kind       InkKind
	MetricName metric.Name
}

// Info returns introspection data about the ink's kind and metric.
func (ink Ink) Info() InkInfo {
	return InkInfo{
		Kind:       InkKind(ink.kind),
		MetricName: ink.metricName,
	}
}

// Colours used by introspection tests and internal defaults.
var (
	white = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	black = color.RGBA{R: 0, G: 0, B: 0, A: 255}
)

// Boundaries returns the bucket boundary values for numeric inks.
// Returns nil for fixed or categorical inks.
func (ink Ink) Boundaries() []float64 {
	if ink.kind != inkNumeric || ink.boundaries == nil {
		return nil
	}

	return ink.boundaries.Boundaries
}

// Palette returns the colour palette used by this ink.
// Returns an empty palette for fixed inks.
func (ink Ink) Palette() palette.ColourPalette {
	if ink.kind == inkFixed {
		return palette.ColourPalette{}
	}

	return ink.pal
}

// Categories returns the category labels for categorical inks.
// Returns nil for fixed or numeric inks.
func (ink Ink) Categories() []string {
	if ink.kind != inkCategorical {
		return nil
	}

	return ink.categories
}
