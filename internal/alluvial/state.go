package alluvial

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/viz"
)

// State holds data prepared for a later alluvial layout and render pipeline.
type State struct {
	WidthMetric metric.Name
	Fill        viz.ColourEncoding
	FillLabel   metric.Name
	FillDelta   bool
	FillInk     inks.Ink
	Snapshots   []Snapshot
	Data        Data
	Layout      Layout
	Legend      *legend.Config
}

// Snapshot is the scanned directory tree at one caller-ordered reference.
type Snapshot struct {
	Reference string
	Root      *model.Directory
}

// Options controls the metric and directory detail represented in Data.
type Options struct {
	Metric     metric.Name
	FillMetric metric.Name
	FillDelta  bool
	Expand     []string
}
