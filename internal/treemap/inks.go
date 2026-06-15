package treemap

import (
	"image/color"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	pkginks "github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

const (
	headerHeight = HeaderHeight
	minBorderDim = 20.0
	midBorderDim = 100.0
)

var (
	structuralBorder = color.RGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xFF}
	headerFill       = color.RGBA{R: 0x44, G: 0x44, B: 0x44, A: 0xFF}
	defaultFill      = color.RGBA{R: 0xCC, G: 0xCC, B: 0xCC, A: 0xFF}
	bgColour         = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	whiteText        = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
)

// Inks holds the fill and border Ink instances for a treemap render pass.
type Inks struct {
	Fill   canvas.Ink
	Border canvas.Ink
}

// BuildInks creates fill and border inks from metric configuration.
func BuildInks(
	root *model.Directory,
	requested stages.RequestedMetrics,
	fillMetric metric.Name,
	fillPaletteName palette.PaletteName,
	borderMetric metric.Name,
	borderPaletteName palette.PaletteName,
) Inks {
	inks := Inks{
		Border: canvas.FixedInk(structuralBorder),
	}

	fillDesc, _ := requested.DescriptorFor(fillMetric)
	inks.Fill = pkginks.BuildMetricInk(root, fillDesc, fillPaletteName, defaultFill)

	if borderMetric != "" {
		borderDesc, _ := requested.DescriptorFor(borderMetric)
		inks.Border = pkginks.BuildMetricInk(root, borderDesc, borderPaletteName, structuralBorder)
	}

	return inks
}
