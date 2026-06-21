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
	LegendData() (model.LegendEntryKind, []model.LegendSwatch)
}

// fixedInk always produces the same colour regardless of input.
type fixedInk struct {
	color   color.RGBA
	opacity float64
}

// numericInk maps numeric metric values to palette colours via bucketing.
type numericInk struct {
	metricName metric.Name
	boundaries metric.BucketBoundaries
	pal        palette.ColourPalette
	opacity    float64
}

// categoricalInk maps string categories to palette colours.
type categoricalInk struct {
	metricName metric.Name
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

	return &fixedInk{
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

	return &numericInk{
		metricName: name,
		boundaries: metric.ComputeBuckets(values, len(pal.Colours)),
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

	return &categoricalInk{
		metricName: name,
		catMapper:  palette.NewCategoricalMapper(categories, pal),
		pal:        pal,
		categories: categories,
		opacity:    cfg.opacity,
	}
}

func (ink *fixedInk) Dip(MetricValue) color.RGBA {
	return applyOpacity(ink.color, ink.opacity)
}

func (ink *fixedInk) Fill(value MetricValue, _ model.Point) model.Fill {
	return model.SolidFill{Color: ink.Dip(value)}
}

func (ink *numericInk) Dip(value MetricValue) color.RGBA {
	var numericVal float64

	switch value.Kind {
	case metric.Quantity:
		numericVal = float64(value.Quantity)
	default:
		numericVal = value.Measure
	}

	idx := ink.boundaries.BucketIndex(numericVal)
	c := palette.MapNumericToColour(idx, ink.boundaries.NumBuckets(), ink.pal)

	return applyOpacity(c, ink.opacity)
}

func (ink *numericInk) Fill(value MetricValue, _ model.Point) model.Fill {
	return model.SolidFill{Color: ink.Dip(value)}
}

func (ink *categoricalInk) Dip(value MetricValue) color.RGBA {
	c := ink.catMapper.Map(value.Category)

	return applyOpacity(c, ink.opacity)
}

func (ink *categoricalInk) Fill(value MetricValue, _ model.Point) model.Fill {
	return model.SolidFill{Color: ink.Dip(value)}
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
