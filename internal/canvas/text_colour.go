package canvas

import (
	"image/color"

	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// black is the canvas-internal text colour for light fills.
//
//nolint:gochecknoglobals // package-level colour constant
var black = color.RGBA{R: 0, G: 0, B: 0, A: 255}

// TextColourFor returns black or white text depending on fill luminance.
// Uses WCAG 2.0 relative luminance with a 0.5 threshold.
func TextColourFor(fill color.RGBA) color.RGBA {
	lum := palette.RelativeLuminance(fill)
	if lum > 0.5 {
		return black
	}

	return palette.White
}
