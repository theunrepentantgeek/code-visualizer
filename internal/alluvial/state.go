package alluvial

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

// State holds data prepared for a later alluvial layout and render pipeline.
type State struct {
	WidthMetric metric.Name
	Snapshots   []Snapshot
	Data        Data
}

// Snapshot is the scanned directory tree at one caller-ordered reference.
type Snapshot struct {
	Reference string
	Root      *model.Directory
}

// Options controls the metric and directory detail represented in Data.
type Options struct {
	Metric metric.Name
	Expand []string
}

// Data is the deterministic, renderer-independent alluvial input model.
type Data struct {
	Columns     []Column
	Transitions []Transition
}

// Column contains metric widths for a single reference snapshot.
type Column struct {
	Reference string
	Values    []Value
}

// Value identifies a directory and its metric width in one snapshot.
type Value struct {
	Path  string
	Width float64
}

// Transition joins one path in adjacent reference columns. A missing endpoint
// has zero width, allowing a renderer to taper introduced and removed paths.
type Transition struct {
	FromReference string
	ToReference   string
	Path          string
	FromWidth     float64
	ToWidth       float64
}
