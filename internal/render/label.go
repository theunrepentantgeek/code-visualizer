package render

import (
	"image/color"

	"github.com/fogleman/gg"

	"github.com/bevan/code-visualizer/internal/palette"
	"github.com/bevan/code-visualizer/internal/treemap"
)

const (
	minLabelWidth          = 40.0 // minimum rect width to consider showing a label
	minLabelHeight         = 16.0 // minimum rect height to consider showing a label
	labelHorizontalPadding = 4.0
	labelVerticalPadding   = 2.0
)

// ShouldShowLabel returns true if the rectangle is large enough to display a label.
// It uses a temporary gg context to measure the label text with the default font.
func ShouldShowLabel(rect treemap.TreemapRectangle) bool {
	if rect.W < minLabelWidth || rect.H < minLabelHeight {
		return false
	}

	if rect.Label == "" {
		return false
	}

	dc := gg.NewContext(1, 1)
	tw, th := dc.MeasureString(rect.Label)

	availW := rect.W - 2*labelHorizontalPadding
	availH := rect.H - 2*labelVerticalPadding

	return availW >= tw && availH >= th
}

// TextColourFor returns black or white text depending on fill luminance.
func TextColourFor(fill color.RGBA) color.RGBA {
	lum := palette.RelativeLuminance(fill)
	if lum > 0.5 {
		return color.RGBA{R: 0, G: 0, B: 0, A: 255}
	}

	return color.RGBA{R: 255, G: 255, B: 255, A: 255}
}
