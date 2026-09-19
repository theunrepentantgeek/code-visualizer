package spiral

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/viz"
)

// State is the viz-specific pipeline state for the spiral visualization.
type State struct {
	// Resolved during the pipeline:
	Size           metric.Name
	Fill           viz.ColourEncoding
	Border         viz.ColourEncoding
	SurfaceEnabled bool
	Surface        viz.ColourEncoding
	Resolution     Resolution
	SpotsPerLap    int

	Buckets      []TimeBucket
	Inks         Inks
	SurfaceInk   inks.Ink
	Layout       SpiralLayout
	DiscLabels   []canvas.BlockLabel
	LegendConfig *legend.Config
}
