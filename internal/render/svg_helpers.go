package render

import (
	"fmt"
	"image/color"
	"os"
)

const svgFontAttrs = `font-size="12" font-family="sans-serif"`

// writeSVGText writes an SVG <text> element.
// anchor is the text-anchor value (e.g. "middle", "start", "end");
// pass "" to omit text-anchor.
func writeSVGText(
	f *os.File,
	x, y float64,
	fill string,
	anchor string,
	content string,
) {
	anchorAttr := ""
	if anchor != "" {
		anchorAttr = fmt.Sprintf(` text-anchor="%s"`, anchor)
	}

	fmt.Fprintf(f,
		"<text x=\"%.2f\" y=\"%.2f\" fill=\"%s\" %s%s"+
			" dominant-baseline=\"central\">%s</text>\n",
		x, y, fill, svgFontAttrs, anchorAttr, content)
}

// colourToHex converts a colour to a CSS hex string.
func colourToHex(c color.RGBA) string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}
