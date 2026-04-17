package render

import (
	"fmt"
	"os"
)

const svgFontAttrs = `font-size="12" font-family="sans-serif"`

// writeSVGText writes an SVG <text> element.
// anchor is the text-anchor value (e.g. "middle", "start", "end");
// pass "" to omit text-anchor. baseline is the dominant-baseline value.
func writeSVGText(
	f *os.File,
	x, y float64,
	fill string,
	anchor string,
	baseline string,
	content string,
) {
	anchorAttr := ""
	if anchor != "" {
		anchorAttr = fmt.Sprintf(` text-anchor="%s"`, anchor)
	}

	fmt.Fprintf(f,
		"<text x=\"%.2f\" y=\"%.2f\" fill=\"%s\" %s%s"+
			" dominant-baseline=\"%s\">%s</text>\n",
		x, y, fill, svgFontAttrs, anchorAttr, baseline, content)
}

// writeSVGTextRotated writes a rotated SVG <text> element.
func writeSVGTextRotated(
	f *os.File,
	x, y float64,
	fill string,
	anchor string,
	baseline string,
	rotDeg float64,
	content string,
) {
	fmt.Fprintf(f,
		"<text x=\"%.2f\" y=\"%.2f\" fill=\"%s\" %s"+
			" text-anchor=\"%s\" dominant-baseline=\"%s\""+
			" transform=\"rotate(%.2f %.2f %.2f)\">%s</text>\n",
		x, y, fill, svgFontAttrs,
		anchor, baseline,
		rotDeg, x, y, content)
}
