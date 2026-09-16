package treemap

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/viz"
)

// State is the viz-specific pipeline state for the treemap visualization.
// Shared state lives in *stages.CommonState; treemap config in *config.Treemap.
type State struct {
	Flat bool

	// Resolved during the pipeline:
	Size         metric.Name
	Fill         viz.ColourEncoding
	Border       viz.ColourEncoding
	Inks         Inks
	Root         TreemapRectangle
	LegendConfig *legend.Config
	BlockLabels  []canvas.BlockLabel
}
