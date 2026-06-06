package treemap

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// State is the viz-specific pipeline state for the treemap visualization.
// Shared state lives in *stages.CommonState; treemap config in *config.Treemap.
type State struct {
	IncludeBinaryFiles bool
	Flat               bool

	Size          metric.Name
	FillMetric    metric.Name
	FillPalette   palette.PaletteName
	BorderMetric  metric.Name
	BorderPalette palette.PaletteName
	Inks          Inks
	Root          TreemapRectangle
	LegendConfig  *canvas.LegendConfig
	BlockLabels   []canvas.BlockLabel
}
