package canvas

import (
	"image/color"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

type inkKind int

const (
	inkFixed inkKind = iota
	inkNumeric
	inkCategorical
)

// Ink resolves metric values to colours.
// Fixed inks ignore the metric value; metric inks resolve via palette + mapping strategy.
//
// Ink is safe to copy; internal state is shared via pointers.
type Ink struct {
	kind       inkKind
	color      color.RGBA
	boundaries *metric.BucketBoundaries
	catMapper  *palette.CategoricalMapper
	pal        palette.ColourPalette
	categories []string
	opacity    float64
}

// FixedInk always produces the same colour regardless of input.
func FixedInk(c color.RGBA, opts ...InkOption) Ink {
	cfg := defaultInkConfig()
	for _, o := range opts {
		o(&cfg)
	}

	return Ink{
		kind:    inkFixed,
		color:   c,
		opacity: cfg.opacity,
	}
}

// NumericInk maps numeric metric values to palette colours.
// Takes the full dataset of values (for bucketing), the palette,
// and optional configuration options.
func NumericInk(values []float64, pal palette.ColourPalette, opts ...InkOption) Ink {
	cfg := defaultInkConfig()
	for _, o := range opts {
		o(&cfg)
	}

	// Strategy selection (quantile/linear/logarithmic) not yet implemented;
	// currently always uses quantile via metric.ComputeBuckets.
	buckets := metric.ComputeBuckets(values, len(pal.Colours))

	return Ink{
		kind:       inkNumeric,
		boundaries: &buckets,
		pal:        pal,
		opacity:    cfg.opacity,
	}
}

// CategoricalInk maps string categories to palette colours.
func CategoricalInk(categories []string, pal palette.ColourPalette, opts ...InkOption) Ink {
	cfg := defaultInkConfig()
	for _, o := range opts {
		o(&cfg)
	}

	return Ink{
		kind:       inkCategorical,
		catMapper:  palette.NewCategoricalMapper(categories, pal),
		pal:        pal,
		categories: categories,
		opacity:    cfg.opacity,
	}
}

// Dip resolves a MetricValue to an RGBA colour.
func (ink Ink) Dip(value MetricValue) color.RGBA {
	var c color.RGBA

	switch ink.kind {
	case inkFixed:
		c = ink.color
	case inkNumeric:
		c = ink.dipNumeric(value)
	case inkCategorical:
		c = ink.catMapper.Map(value.Category)
	default:
		c = color.RGBA{A: 255} // fallback to opaque black
	}

	return applyOpacity(c, ink.opacity)
}

func (ink Ink) dipNumeric(value MetricValue) color.RGBA {
	var numericVal float64

	switch value.Kind {
	case metric.Quantity:
		numericVal = float64(value.Quantity)
	default:
		numericVal = value.Measure
	}

	idx := ink.boundaries.BucketIndex(numericVal)

	return palette.MapNumericToColour(idx, ink.boundaries.NumBuckets(), ink.pal)
}

func applyOpacity(c color.RGBA, opacity float64) color.RGBA {
	if opacity >= 1.0 {
		return c
	}

	c.A = uint8(float64(c.A) * clamp01(opacity))

	return c
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}

	if v > 1 {
		return 1
	}

	return v
}
