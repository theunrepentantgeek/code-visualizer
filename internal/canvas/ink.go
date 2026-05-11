package canvas

import (
	"image/color"
	"slices"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/legendlayout"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
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
	metricName metric.Name
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
func NumericInk(name metric.Name, values []float64, pal palette.ColourPalette, opts ...InkOption) Ink {
	cfg := defaultInkConfig()
	for _, o := range opts {
		o(&cfg)
	}

	// Strategy selection (quantile/linear/logarithmic) not yet implemented;
	// currently always uses quantile via metric.ComputeBuckets.
	buckets := metric.ComputeBuckets(values, len(pal.Colours))

	return Ink{
		kind:       inkNumeric,
		metricName: name,
		boundaries: &buckets,
		pal:        pal,
		opacity:    cfg.opacity,
	}
}

// CategoricalInk maps string categories to palette colours.
func CategoricalInk(name metric.Name, categories []string, pal palette.ColourPalette, opts ...InkOption) Ink {
	cfg := defaultInkConfig()
	for _, o := range opts {
		o(&cfg)
	}

	return Ink{
		kind:       inkCategorical,
		metricName: name,
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

// legendEntryKind returns the LegendEntryKind for this ink.
func (ink Ink) legendEntryKind() model.LegendEntryKind {
	if ink.kind == inkCategorical {
		return model.LegendEntryCategorical
	}

	return model.LegendEntryNumeric
}

// legendSwatches extracts resolved swatch data for legend rendering.
// Returns nil for fixed inks (no meaningful swatch data).
func (ink Ink) legendSwatches() []model.LegendSwatch {
	switch ink.kind {
	case inkNumeric:
		return ink.numericLegendSwatches()
	case inkCategorical:
		return ink.categoricalLegendSwatches()
	default:
		return nil
	}
}

func (ink Ink) numericLegendSwatches() []model.LegendSwatch {
	if ink.boundaries == nil {
		return nil
	}

	n := ink.boundaries.NumBuckets()
	if n <= 0 || len(ink.pal.Colours) == 0 {
		return nil
	}

	swatches := make([]model.LegendSwatch, n)

	for i := range n {
		colour := palette.MapNumericToColour(i, n, ink.pal)

		var label string
		if i < len(ink.boundaries.Boundaries) {
			label = legendlayout.FormatBreakpoint(ink.boundaries.Boundaries[i])
		}

		swatches[i] = model.LegendSwatch{
			Colour: colour,
			Label:  label,
		}
	}

	return swatches
}

func (ink Ink) categoricalLegendSwatches() []model.LegendSwatch {
	if ink.catMapper == nil || len(ink.categories) == 0 {
		return nil
	}

	sorted := make([]string, len(ink.categories))
	copy(sorted, ink.categories)
	slices.Sort(sorted)

	swatches := make([]model.LegendSwatch, len(sorted))

	for i, cat := range sorted {
		swatches[i] = model.LegendSwatch{
			Colour: ink.catMapper.Map(cat),
			Label:  cat,
		}
	}

	return swatches
}
