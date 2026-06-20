// Package inks owns the Ink interface and concrete ink implementations.
// Inks resolve metric values to colours and fill specifications; they are
// consumed by canvas shape specs (canvas.RectangleSpec, canvas.TextSpec, ...).
package inks

import (
	"image/color"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// Kind identifies the type of an Ink for introspection purposes.
type Kind int

const (
	// KindFixed is a fixed-colour ink that ignores its input.
	KindFixed Kind = iota
	// KindNumeric is a numeric-bucket ink mapping float values to palette colours.
	KindNumeric
	// KindCategorical is a categorical ink mapping strings to palette colours.
	KindCategorical
)

// Ink resolves metric values to colours and fill specifications.
type Ink interface {
	Dip(value MetricValue) color.RGBA
	Fill(value MetricValue, focus model.Point) model.Fill
	Info() Info

	// Introspection accessors used by legend extraction and tests.
	// FixedInk values return nil/empty for all three.
	Boundaries() []float64
	Palette() palette.ColourPalette
	Categories() []string
}

type baseInk struct {
	kind       Kind
	metricName metric.Name
	color      color.RGBA
	boundaries *metric.BucketBoundaries
	catMapper  *palette.CategoricalMapper
	pal        palette.ColourPalette
	categories []string
	opacity    float64
}

// FixedInk always produces the same colour regardless of input.
func FixedInk(c color.RGBA, opts ...Option) Ink {
	cfg := defaultConfig()
	for _, o := range opts {
		o(&cfg)
	}

	return &baseInk{
		kind:    KindFixed,
		color:   c,
		opacity: cfg.opacity,
	}
}

// NumericInk maps numeric metric values to palette colours.
// Takes the full dataset of values (for bucketing), the palette,
// and optional configuration options.
func NumericInk(name metric.Name, values []float64, pal palette.ColourPalette, opts ...Option) Ink {
	cfg := defaultConfig()
	for _, o := range opts {
		o(&cfg)
	}

	buckets := metric.ComputeBuckets(values, len(pal.Colours))

	return &baseInk{
		kind:       KindNumeric,
		metricName: name,
		boundaries: &buckets,
		pal:        pal,
		opacity:    cfg.opacity,
	}
}

// CategoricalInk maps string categories to palette colours.
func CategoricalInk(name metric.Name, categories []string, pal palette.ColourPalette, opts ...Option) Ink {
	cfg := defaultConfig()
	for _, o := range opts {
		o(&cfg)
	}

	return &baseInk{
		kind:       KindCategorical,
		metricName: name,
		catMapper:  palette.NewCategoricalMapper(categories, pal),
		pal:        pal,
		categories: categories,
		opacity:    cfg.opacity,
	}
}

// Dip resolves a MetricValue to an RGBA colour.
func (ink *baseInk) Dip(value MetricValue) color.RGBA {
	var c color.RGBA

	switch ink.kind {
	case KindFixed:
		c = ink.color
	case KindNumeric:
		c = ink.dipNumeric(value)
	case KindCategorical:
		c = ink.catMapper.Map(value.Category)
	default:
		c = color.RGBA{A: 255}
	}

	return applyOpacity(c, ink.opacity)
}

func (ink *baseInk) Fill(value MetricValue, _ model.Point) model.Fill {
	return model.SolidFill{Color: ink.Dip(value)}
}

func (ink *baseInk) dipNumeric(value MetricValue) color.RGBA {
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
