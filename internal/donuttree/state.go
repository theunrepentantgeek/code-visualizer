package donuttree

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/viz"
)

// State is the viz-specific pipeline state for the donut tree visualization.
type State struct {
	SizeMetric   metric.Name
	Fill         viz.ColourEncoding
	Border       viz.ColourEncoding
	DisplayRoot  *model.Directory
	Inks         Inks
	Layout       LayoutResult
	LegendConfig *legend.Config
}
