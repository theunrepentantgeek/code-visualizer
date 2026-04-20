package render

import (
	"fmt"
	"html"
	"os"

	"github.com/fogleman/gg"

	"github.com/bevan/code-visualizer/internal/metric"
)

// writeSVGLegend writes the SVG legend elements to the file.
// Does nothing if info is nil or position is none.
func writeSVGLegend(f *os.File, info *LegendInfo, canvasW, canvasH int) {
	if info == nil || info.Position == LegendPositionNone || len(info.Entries) == 0 {
		return
	}

	// Use a temporary gg context for text measurement only.
	dc := gg.NewContext(1, 1)
	w, h := measureLegend(dc, info)

	ox, oy := legendOrigin(
		info.Position,
		float64(canvasW), float64(canvasH),
		w, h,
	)

	fmt.Fprintf(f, "<g transform=\"translate(%.2f,%.2f)\">\n", ox, oy)

	writeSVGLegendBackground(f, w, h)
	writeSVGLegendEntries(f, dc, info)

	fmt.Fprint(f, "</g>\n")
}

// writeSVGLegendBackground writes the semi-transparent background rect.
func writeSVGLegendBackground(f *os.File, w, h float64) {
	fmt.Fprintf(f,
		"<rect x=\"0\" y=\"0\" width=\"%.2f\" height=\"%.2f\""+
			" rx=\"4\" ry=\"4\""+
			" fill=\"#ffffff\" fill-opacity=\"0.9\""+
			" stroke=\"#999999\" stroke-opacity=\"0.8\" stroke-width=\"1\"/>\n",
		w, h)
}

// writeSVGLegendEntries renders all entries inside the legend group.
func writeSVGLegendEntries(f *os.File, dc *gg.Context, info *LegendInfo) {
	cy := legendPadding

	for i, entry := range info.Entries {
		if i > 0 {
			cy += entryGap
		}

		cy = writeSVGSingleEntry(f, dc, info.Orientation, entry, legendPadding, cy)
	}
}

// writeSVGSingleEntry writes one legend entry and returns the new Y position.
func writeSVGSingleEntry(
	f *os.File,
	dc *gg.Context,
	orientation LegendOrientation,
	entry LegendEntry,
	x, y float64,
) float64 {
	// Title
	title := html.EscapeString(fmt.Sprintf("%s: %s", entry.Role, entry.MetricName))
	fmt.Fprintf(f,
		"<text x=\"%.2f\" y=\"%.2f\""+
			" font-family=\"sans-serif\" font-size=\"%.1f\""+
			" fill=\"#262626\">%s</text>\n",
		x, y+titleFontSize, titleFontSize, title)

	y += titleFontSize + labelGap

	if entry.Kind == metric.Classification {
		return writeSVGCategorySwatches(f, dc, orientation, entry, x, y)
	}

	return writeSVGNumericSwatches(f, dc, orientation, entry, x, y)
}

// writeSVGNumericSwatches renders colour swatches for numeric metrics.
func writeSVGNumericSwatches(
	f *os.File,
	_ *gg.Context,
	orientation LegendOrientation,
	entry LegendEntry,
	x, y float64,
) float64 {
	if entry.NumBuckets <= 0 || len(entry.Palette.Colours) == 0 {
		return y
	}

	if orientation == LegendOrientationHorizontal {
		return writeSVGNumericH(f, entry, x, y)
	}

	return writeSVGNumericV(f, entry, x, y)
}

// writeSVGNumericV writes vertically stacked numeric swatches.
func writeSVGNumericV(f *os.File, entry LegendEntry, x, y float64) float64 {
	for i := range entry.NumBuckets {
		colour := mapBucketColour(i, entry)
		writeSVGSwatch(f, x, y, colourToHex(colour))

		if entry.Buckets != nil && i < len(entry.Buckets.Boundaries) {
			label := formatBreakpoint(entry.Buckets.Boundaries[i])
			fmt.Fprintf(f,
				"<text x=\"%.2f\" y=\"%.2f\""+
					" font-family=\"sans-serif\" font-size=\"%.1f\""+
					" fill=\"#333333\" dominant-baseline=\"central\">%s</text>\n",
				x+swatchSize+labelGap, y+swatchSize,
				legendFontSize, html.EscapeString(label))
		}

		y += swatchSize
	}

	return y
}

// writeSVGNumericH writes horizontally arranged numeric swatches.
func writeSVGNumericH(f *os.File, entry LegendEntry, x, y float64) float64 {
	cx := x

	for i := range entry.NumBuckets {
		colour := mapBucketColour(i, entry)
		writeSVGSwatch(f, cx, y, colourToHex(colour))

		if entry.Buckets != nil && i < len(entry.Buckets.Boundaries) {
			label := formatBreakpoint(entry.Buckets.Boundaries[i])
			fmt.Fprintf(f,
				"<text x=\"%.2f\" y=\"%.2f\""+
					" font-family=\"sans-serif\" font-size=\"%.1f\""+
					" fill=\"#333333\" text-anchor=\"middle\""+
					" dominant-baseline=\"central\">%s</text>\n",
				cx+swatchSize, y+swatchSize+legendLineHeight,
				legendFontSize, html.EscapeString(label))
		}

		cx += swatchSize
	}

	return y + swatchSize + legendLineHeight + labelGap
}

// writeSVGCategorySwatches renders category swatches.
func writeSVGCategorySwatches(
	f *os.File,
	dc *gg.Context,
	orientation LegendOrientation,
	entry LegendEntry,
	x, y float64,
) float64 {
	cats := entry.Categories

	if orientation == LegendOrientationHorizontal {
		y = writeSVGCategoryH(f, dc, cats, x, y)
	} else {
		y = writeSVGCategoryV(f, cats, x, y)
	}

	return y
}

// writeSVGCategoryV writes vertically stacked category swatches.
func writeSVGCategoryV(f *os.File, cats []CategorySwatch, x, y float64) float64 {
	for _, cat := range cats {
		writeSVGSwatch(f, x, y, colourToHex(cat.Colour))

		fmt.Fprintf(f,
			"<text x=\"%.2f\" y=\"%.2f\""+
				" font-family=\"sans-serif\" font-size=\"%.1f\""+
				" fill=\"#333333\" dominant-baseline=\"central\">%s</text>\n",
			x+swatchSize+labelGap, y+swatchSize/2,
			legendFontSize, html.EscapeString(cat.Label))

		y += swatchSize + swatchGap
	}

	return y
}

// writeSVGCategoryH writes horizontally arranged category swatches.
func writeSVGCategoryH(
	f *os.File,
	dc *gg.Context,
	cats []CategorySwatch,
	x, y float64,
) float64 {
	cx := x

	for _, cat := range cats {
		writeSVGSwatch(f, cx, y, colourToHex(cat.Colour))

		fmt.Fprintf(f,
			"<text x=\"%.2f\" y=\"%.2f\""+
				" font-family=\"sans-serif\" font-size=\"%.1f\""+
				" fill=\"#333333\" text-anchor=\"middle\""+
				" dominant-baseline=\"central\">%s</text>\n",
			cx+swatchSize/2, y+swatchSize+legendLineHeight,
			legendFontSize, html.EscapeString(cat.Label))

		tw, _ := dc.MeasureString(cat.Label)
		cx += max(swatchSize, tw) + swatchGap + labelGap
	}

	return y + swatchSize + legendLineHeight + labelGap
}

// writeSVGSwatch writes a single coloured rectangle.
func writeSVGSwatch(f *os.File, x, y float64, fillHex string) {
	fmt.Fprintf(f,
		"<rect x=\"%.2f\" y=\"%.2f\" width=\"%.2f\" height=\"%.2f\""+
			" fill=\"%s\" stroke=\"#666666\" stroke-width=\"0.5\"/>\n",
		x, y, swatchSize, swatchSize, fillHex)
}
