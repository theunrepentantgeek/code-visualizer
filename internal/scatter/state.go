package scatter

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/viz"
)

// State is the viz-specific pipeline state for the scatter visualization.
type State struct {
	Grain  viz.Grain
	XAxis  AxisSpec
	YAxis  AxisSpec
	Size   metric.Name
	Fill   viz.ColourEncoding
	Border viz.ColourEncoding

	Dataset      Dataset
	Inks         Inks
	Layout       ScatterLayout
	LegendConfig *legend.Config
}
