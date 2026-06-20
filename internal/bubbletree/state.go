package bubbletree

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// State is the viz-specific pipeline state for the bubbletree visualization.
type State struct {
	Flat bool

	// Resolved during the pipeline:
	Size          metric.Name
	FillMetric    metric.Name
	FillPalette   palette.PaletteName
	BorderMetric  metric.Name
	BorderPalette palette.PaletteName
	Labels        LabelMode
	Inks          Inks
	Nodes         BubbleNode
	LegendConfig  *legend.Config
}
