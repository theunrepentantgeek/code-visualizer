package donuttree

import (
	"image/color"

	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

var (
	donutFallbackFill   = color.RGBA{R: 0xCC, G: 0xCC, B: 0xCC, A: 0xFF}
	donutFallbackBorder = color.RGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xFF}
)

// Inks contains the directory fill and border inks for a donut tree render.
type Inks struct {
	inks.ShapeInks
	HasBorderMetric bool
	LabelMetrics    LabelMetrics
}

// BuildInks creates directory metric inks from the effective configuration.
func BuildInks(
	root *model.Directory,
	requested stages.RequestedMetrics,
	fillMetric metric.Name,
	fillPaletteName palette.PaletteName,
	borderMetric metric.Name,
	borderPaletteName palette.PaletteName,
) Inks {
	result := Inks{
		ShapeInks:    inks.ShapeInks{Border: inks.FixedInk(donutFallbackBorder)},
		LabelMetrics: LabelMetrics{Size: fillMetric},
	}

	fillDesc, _ := requested.DescriptorFor(fillMetric)
	result.Fill = inks.BuildDirectoryMetricInk(root, fillDesc, fillPaletteName, donutFallbackFill)

	if borderMetric != "" {
		borderDesc, _ := requested.DescriptorFor(borderMetric)
		result.Border = inks.BuildDirectoryMetricInk(root, borderDesc, borderPaletteName, donutFallbackBorder)
		result.HasBorderMetric = true
	}

	return result
}
