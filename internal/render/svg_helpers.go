package render

import (
	"fmt"
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

// writeSVGTextRotated writes a rotated SVG <text> element.
func writeSVGTextRotated(
	f *os.File,
	x, y float64,
	fill string,
	anchor string,
	rotDeg float64,
	content string,
) {
	fmt.Fprintf(f,
		"<text x=\"%.2f\" y=\"%.2f\" fill=\"%s\" %s"+
			" text-anchor=\"%s\" dominant-baseline=\"central\""+
			" transform=\"rotate(%.2f %.2f %.2f)\">%s</text>\n",
		x, y, fill, svgFontAttrs,
		anchor,
		rotDeg, x, y, content)
}
