package scatter

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// State is the viz-specific pipeline state for the scatter visualization.
type State struct {
	IncludeBinaryFiles bool

	XAxis         AxisSpec
	YAxis         AxisSpec
	Size          metric.Name
	FillMetric    metric.Name
	FillPalette   palette.PaletteName
	BorderMetric  metric.Name
	BorderPalette palette.PaletteName

	Dataset      Dataset
	Inks         Inks
	Layout       ScatterLayout
	LegendConfig *canvas.LegendConfig
}
