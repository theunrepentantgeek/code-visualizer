package canvas

import (
	"image/color"

	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
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
