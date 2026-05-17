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
	defaultFileFill = color.RGBA{R: 0xCC, G: 0xCC, B: 0xCC, A: 0xFF}
	defaultDirFill  = color.RGBA{R: 0x44, G: 0x44, B: 0x44, A: 0xFF}
	defaultBorder   = color.RGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xFF}
	edgeColour      = color.RGBA{R: 0x99, G: 0x99, B: 0x99, A: 0xFF}
	labelColour     = color.RGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xFF}
	bgColour        = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
)

// Inks holds the fill and border Ink instances for a radial tree render pass.
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
		Border: canvas.FixedInk(defaultBorder),
	}

	inks.Fill = pkginks.BuildMetricInk(root, fillMetric, fillPaletteName, defaultFileFill)
	if borderMetric != "" {
		inks.Border = pkginks.BuildMetricInk(root, borderMetric, borderPaletteName, defaultBorder)
	}

	return inks
}
