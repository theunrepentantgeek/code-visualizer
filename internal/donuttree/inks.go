package donuttree

import (
	"image/color"

	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
	"github.com/theunrepentantgeek/code-visualizer/internal/viz"
)

var (
	donutFallbackFill   = color.RGBA{R: 0xCC, G: 0xCC, B: 0xCC, A: 0xFF}
	donutFallbackBorder = color.RGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xFF}
)

// Inks contains the directory fill and border inks for a donut tree render.
type Inks struct {
	inks.ShapeInks
	HasBorderMetric bool
}

// BuildInks creates directory metric inks from the effective configuration.
func BuildInks(
	root *model.Directory,
	requested stages.RequestedMetrics,
	fill viz.ColourEncoding,
	border viz.ColourEncoding,
) Inks {
	result := Inks{
		ShapeInks: inks.ShapeInks{Border: inks.FixedInk(donutFallbackBorder)},
	}

	fillDesc, _ := requested.DescriptorFor(fill.Metric)
	result.Fill = inks.BuildDirectoryMetricInk(root, fillDesc, fill.Palette, donutFallbackFill)

	if border.IsSet() {
		borderDesc, _ := requested.DescriptorFor(border.Metric)
		result.Border = inks.BuildDirectoryMetricInk(root, borderDesc, border.Palette, donutFallbackBorder)
		result.HasBorderMetric = true
	}

	return result
}
