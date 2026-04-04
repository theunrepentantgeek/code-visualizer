package render

import (
	"image/color"
	"math"

	"github.com/fogleman/gg"

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
	lum := relativeLuminance(fill)
	if lum > 0.5 {
		return color.RGBA{R: 0, G: 0, B: 0, A: 255}
	}
	return color.RGBA{R: 255, G: 255, B: 255, A: 255}
}

func relativeLuminance(c color.RGBA) float64 {
	r := linearize(float64(c.R) / 255.0)
	g := linearize(float64(c.G) / 255.0)
	b := linearize(float64(c.B) / 255.0)
	return 0.2126*r + 0.7152*g + 0.0722*b
}

func linearize(v float64) float64 {
	if v <= 0.03928 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}
