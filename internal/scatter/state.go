package scatter

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/viz"
)

// State is the viz-specific pipeline state for the scatter visualization.
type State struct {
	Grain         viz.Grain
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
	LegendConfig *legend.Config
}
