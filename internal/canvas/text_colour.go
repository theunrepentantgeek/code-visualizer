package canvas

import (
	"image/color"

	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// Colours used by canvas-internal rendering and tests.
//
//nolint:gochecknoglobals // package-level colour constants used across canvas
var (
	white = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	black = color.RGBA{R: 0, G: 0, B: 0, A: 255}
)

// TextColourFor returns black or white text depending on fill luminance.
// Uses WCAG 2.0 relative luminance with a 0.5 threshold.
func TextColourFor(fill color.RGBA) color.RGBA {
	lum := palette.RelativeLuminance(fill)
	if lum > 0.5 {
		return black
	}

	return white
}
