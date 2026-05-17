package radialtree

import (
	"image/color"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	pkginks "github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

var (
	radialDefaultFileFill = color.RGBA{R: 0xCC, G: 0xCC, B: 0xCC, A: 0xFF}
	radialDefaultDirFill  = color.RGBA{R: 0x44, G: 0x44, B: 0x44, A: 0xFF}
	radialDefaultBorder   = color.RGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xFF}
	radialEdgeColour      = color.RGBA{R: 0x99, G: 0x99, B: 0x99, A: 0xFF}
	radialLabelColour     = color.RGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xFF}
	radialBgColour        = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
)

// Inks holds the fill and border Ink instances for a radial render pass.
type Inks struct {
	Fill   canvas.Ink
	Border canvas.Ink
}

// BuildInks creates fill and border inks from metric configuration.
// A zero borderMetric yields a fixed default border ink.
func BuildInks(
	root *model.Directory,
	fillMetric metric.Name,
	fillPaletteName palette.PaletteName,
	borderMetric metric.Name,
	borderPaletteName palette.PaletteName,
) Inks {
	inks := Inks{
		Border: canvas.FixedInk(radialDefaultBorder),
	}

	inks.Fill = pkginks.BuildMetricInk(root, fillMetric, fillPaletteName, radialDefaultFileFill)
	if borderMetric != "" {
		inks.Border = pkginks.BuildMetricInk(root, borderMetric, borderPaletteName, radialDefaultBorder)
	}

	return inks
}
