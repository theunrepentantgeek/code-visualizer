package treemap

import (
	"image/color"

	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

const (
	minBorderDim = 20.0
	midBorderDim = 100.0
)

var (
	structuralBorder = color.RGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xFF}
	defaultFill      = color.RGBA{R: 0xCC, G: 0xCC, B: 0xCC, A: 0xFF}
)

// headerFills is the private depth palette for directory rails, darkest to
// lightest. Directories select a fill by VisibleDepth % len(headerFills), so
// root-immediate children (depth 0) get index 0 and nested directories cycle
// back to the darkest shade every 5 levels. Every entry meets the WCAG 4.5:1
// minimum contrast ratio against palette.White for the white label text
// painted over it.
//
//nolint:gochecknoglobals // read-only palette table, private to treemap headers
var headerFills = [5]color.RGBA{
	{R: 0x20, G: 0x26, B: 0x31, A: 0xFF},
	{R: 0x2F, G: 0x3B, B: 0x4D, A: 0xFF},
	{R: 0x3D, G: 0x52, B: 0x68, A: 0xFF},
	{R: 0x51, G: 0x6A, B: 0x7D, A: 0xFF},
	{R: 0x5F, G: 0x78, B: 0x88, A: 0xFF},
}

// Inks pairs the fill and border Ink instances for a treemap render pass.
// Alias for inks.ShapeInks so other viz packages share the same struct.
type Inks = inks.ShapeInks

// BuildInks creates fill and border inks from metric configuration.
func BuildInks(
	root *model.Directory,
	requested stages.RequestedMetrics,
	fillMetric metric.Name,
	fillPaletteName palette.PaletteName,
	borderMetric metric.Name,
	borderPaletteName palette.PaletteName,
) Inks {
	is := Inks{
		Border: inks.FixedInk(structuralBorder),
	}

	fillDesc, _ := requested.DescriptorFor(fillMetric)
	is.Fill = inks.BuildMetricInk(root, fillDesc, fillPaletteName, defaultFill)

	if borderMetric != "" {
		borderDesc, _ := requested.DescriptorFor(borderMetric)
		is.Border = inks.BuildMetricInk(root, borderDesc, borderPaletteName, structuralBorder)
	}

	return is
}
