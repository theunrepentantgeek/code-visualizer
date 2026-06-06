package spiral

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// State is the viz-specific pipeline state for the spiral visualization.
type State struct {
	// Resolved during the pipeline:
	Size          metric.Name
	FillMetric    metric.Name
	FillPalette   palette.PaletteName
	BorderMetric  metric.Name
	BorderPalette palette.PaletteName
	Resolution    Resolution
	Labels        LabelMode

	Buckets      []TimeBucket
	Inks         Inks
	Layout       SpiralLayout
	LegendConfig *canvas.LegendConfig
}
