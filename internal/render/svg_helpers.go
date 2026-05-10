package render

import (
	"fmt"
	"image/color"
)

// colourToHex converts a colour to a CSS hex string.
func colourToHex(c color.RGBA) string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}
