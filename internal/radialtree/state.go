package radialtree

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// State is the viz-specific pipeline state for the radial tree visualization.
type State struct {
	// Resolved during the pipeline:
	DiscSize               metric.Name
	DirectoryDiscSize      metric.Name
	FillMetric             metric.Name
	FillPalette            palette.PaletteName
	BorderMetric           metric.Name
	BorderPalette          palette.PaletteName
	DirectoryFillMetric    metric.Name
	DirectoryFillPalette   palette.PaletteName
	DirectoryBorderMetric  metric.Name
	DirectoryBorderPalette palette.PaletteName
	Labels                 LabelMode
	Grain                  Grain
	DisplayRoot            *model.Directory
	Inks                   Inks
	Nodes                  RadialNode
	LegendConfig           *legend.Config
}
