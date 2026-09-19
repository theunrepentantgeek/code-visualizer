package radialtree

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/viz"
)

// State is the viz-specific pipeline state for the radial tree visualization.
type State struct {
	// Resolved during the pipeline:
	DiscSize          metric.Name
	DirectoryDiscSize metric.Name
	Fill              viz.ColourEncoding
	Border            viz.ColourEncoding
	DirectoryFill     viz.ColourEncoding
	DirectoryBorder   viz.ColourEncoding
	Labels            LabelMode
	Grain             Grain
	DisplayRoot       *model.Directory
	Inks              Inks
	Nodes             RadialNode
	LegendConfig      *legend.Config
}
