package render

import (
	"fmt"
	"html"
	"image/color"
	"os"

	"github.com/rotisserie/eris"

	"github.com/bevan/code-visualizer/internal/treemap"
)

// renderTreemapSVG generates an SVG file from the treemap layout.
func renderTreemapSVG(
	root treemap.TreemapRectangle, width, height int, legend *LegendInfo, outputPath string,
) (err error) {
	legendH := ComputeLegendHeight(legend)
	totalHeight := height + legendH

	f, err := os.Create(outputPath)
	if err != nil {
		return eris.Wrap(err, "failed to create SVG file")
	}

	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = eris.Wrap(closeErr, "failed to close SVG file")
		}
	}()

	fmt.Fprintf(f, `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">
`, width, totalHeight, width, totalHeight)

	// Background
	fmt.Fprintf(f, `<rect x="0" y="0" width="%d" height="%d" fill="%s"/>
`, width, totalHeight, colourToHex(bgColour))

	writeSVGRect(f, root)

	writeSVGLegend(f, legend, 0, float64(height), float64(width))

	fmt.Fprint(f, "</svg>\n")

	return nil
}

func writeSVGRect(f *os.File, rect treemap.TreemapRectangle) {
	if rect.IsDirectory {
		writeSVGDirectoryHeader(f, rect)
	} else {
		writeSVGFileRect(f, rect)
	}

	for _, child := range rect.Children {
		writeSVGRect(f, child)
	}
}

func writeSVGDirectoryHeader(f *os.File, rect treemap.TreemapRectangle) {
	// Header bar
	fmt.Fprintf(f, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="%s"/>
`,
		rect.X, rect.Y, rect.W, treemap.HeaderHeight,
		colourToHex(headerFill))

	// Header label
	if rect.Label != "" {
		writeSVGText(f,
			rect.X+4, rect.Y+treemap.HeaderHeight/2,
			colourToHex(color.RGBA{R: 255, G: 255, B: 255, A: 255}),
			"",
			html.EscapeString(rect.Label))
	}

	// Border
	fmt.Fprintf(f, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="none" stroke="%s" stroke-width="1"/>
`,
		rect.X, rect.Y, rect.W, rect.H,
		colourToHex(structuralBorder))
}

func writeSVGFileRect(f *os.File, rect treemap.TreemapRectangle) {
	if rect.W <= 0 || rect.H <= 0 {
		return
	}

	fill := defaultFill
	if rect.FillColour.A > 0 {
		fill = rect.FillColour
	}

	border := structuralBorder
	if rect.BorderColour != nil {
		border = *rect.BorderColour
	}

	fmt.Fprintf(f, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="%s" stroke="%s" stroke-width="1"/>
`,
		rect.X, rect.Y, rect.W, rect.H,
		colourToHex(fill), colourToHex(border))

	// Label
	if ShouldShowLabel(rect) {
		textCol := TextColourFor(fill)

		writeSVGText(f,
			rect.X+rect.W/2, rect.Y+rect.H/2,
			colourToHex(textCol),
			"middle",
			html.EscapeString(rect.Label))
	}
}

// colourToHex converts a colour to a CSS hex string.
func colourToHex(c color.RGBA) string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}
