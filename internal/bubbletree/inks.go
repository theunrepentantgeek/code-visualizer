package bubbletree

import (
	"image/color"

	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
	"github.com/theunrepentantgeek/code-visualizer/internal/viz"
)

var (
	bubbleDefaultFileFill = color.RGBA{R: 0xCC, G: 0xCC, B: 0xCC, A: 0xFF}
	bubbleDefaultDirFill  = color.RGBA{R: 0x66, G: 0x99, B: 0xCC, A: 0xFF}
	bubbleDefaultBorder   = color.RGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xFF}
	bubbleLabelColour     = color.RGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xFF}
	bubbleBgColour        = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
)

// Inks pairs the fill and border Ink instances for a bubble render pass
// (via embedded inks.ShapeInks) and records whether the border encodes a
// metric so the renderer can choose a thicker stroke.
type Inks struct {
	inks.ShapeInks
	HasBorderMetric bool // true when the border ink encodes a metric (use thicker stroke)
}

// BuildInks creates fill and border inks from metric configuration.
// A zero borderMetric yields a fixed default border ink.
func BuildInks(
	root *model.Directory,
	requested stages.RequestedMetrics,
	fill viz.ColourEncoding,
	border viz.ColourEncoding,
) Inks {
	is := Inks{
		ShapeInks: inks.ShapeInks{Border: inks.FixedInk(bubbleDefaultBorder)},
	}

	fillDesc, _ := requested.DescriptorFor(fill.Metric)
	is.Fill = inks.BuildMetricInk(root, fillDesc, fill.Palette, bubbleDefaultFileFill)

	if border.IsSet() {
		borderDesc, _ := requested.DescriptorFor(border.Metric)
		is.Border = inks.BuildMetricInk(root, borderDesc, border.Palette, bubbleDefaultBorder)
		is.HasBorderMetric = true
	}

	return is
}
