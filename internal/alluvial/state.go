package alluvial

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/viz"
)

// State holds data prepared for a later alluvial layout and render pipeline.
type State struct {
	WidthMetric metric.Name
	Fill        BandFill
	Snapshots   []Snapshot
	Data        Data
	Layout      Layout
	Legend      *legend.Config
}

// BandFill is the resolved colour encoding used by all alluvial stages.
type BandFill struct {
	Encoding viz.ColourEncoding
	Label    metric.Name
	Explicit bool
	Temporal metric.TemporalName
	Ink      inks.Ink
}

// LabelMetric returns the fill label only when the user explicitly selected
// a fill metric.
func (f *BandFill) LabelMetric() metric.Name {
	if !f.Explicit {
		return ""
	}

	return f.Label
}

// ResolveInk builds the numeric ink, centering delta values around zero.
func (f *BandFill) ResolveInk(values []float64) {
	if f.Temporal == metric.TemporalDelta || f.Temporal == metric.TemporalStepDelta {
		for _, value := range values {
			values = append(values, -value)
		}

		values = append(values, 0)
	}

	f.Ink = inks.NumericInk(f.Label, values, palette.GetPalette(f.Encoding.Palette))
}

// Snapshot is the scanned directory tree at one caller-ordered reference.
type Snapshot struct {
	Reference string
	Root      *model.Directory
}

// Options controls the metric and directory detail represented in Data.
type Options struct {
	Metric       metric.Name
	FillMetric   metric.Name
	FillTemporal metric.TemporalName
	Expand       []string
}
