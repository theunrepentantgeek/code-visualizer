package radialtree

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// State is the pipeline state for the radial tree visualization.
type State struct {
	stages.CommonState

	Config             *config.Radial
	IncludeBinaryFiles bool

	// Resolved during the pipeline:
	DiscSize      metric.Name
	FillMetric    metric.Name
	FillPalette   palette.PaletteName
	BorderMetric  metric.Name
	BorderPalette palette.PaletteName
	Labels        LabelMode
	Inks          Inks
	Nodes         RadialNode
	LegendConfig  *canvas.LegendConfig
}

// Common exposes the embedded CommonState so shared stages can mutate it.
func (s *State) Common() *stages.CommonState { return &s.CommonState }

// IncludeBinary lets State satisfy stages.BinaryFilterToggler.
func (s *State) IncludeBinary() bool { return s.IncludeBinaryFiles }
