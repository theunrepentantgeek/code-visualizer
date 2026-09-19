package bubbletree

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/viz"
)

// State is the viz-specific pipeline state for the bubbletree visualization.
type State struct {
	Flat bool

	// Resolved during the pipeline:
	Size         metric.Name
	Fill         viz.ColourEncoding
	Border       viz.ColourEncoding
	Labels       LabelMode
	Inks         Inks
	Nodes        BubbleNode
	LegendConfig *legend.Config
}
