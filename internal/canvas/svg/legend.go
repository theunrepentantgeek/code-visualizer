package svg

import (
	"bytes"
	"fmt"
	"html"

	"github.com/fogleman/gg"

	"github.com/bevan/code-visualizer/internal/canvas/legendlayout"
	"github.com/bevan/code-visualizer/internal/canvas/model"
)

const orientationHorizontal = "horizontal"

// writeSVGLegend writes the SVG legend elements into the buffer.
// Does nothing if position is "none" or there are no entries.
func writeSVGLegend(buf *bytes.Buffer, data model.LegendData, canvasW, canvasH int) {
	if data.Position == "none" || len(data.Entries) == 0 {
		return
	}

	w, h := legendlayout.MeasureLegend(&data)
	ox, oy := legendlayout.LegendOrigin(data.Position, float64(canvasW), float64(canvasH), w, h)

	fmt.Fprintf(buf, "<g transform=\"translate(%.2f,%.2f)\">\n", ox, oy)
	writeSVGLegendBackground(buf, w, h)
	writeSVGLegendEntries(buf, &data)
	fmt.Fprint(buf, "</g>\n")
}

// writeSVGLegendBackground writes the semi-transparent background rect.
func writeSVGLegendBackground(buf *bytes.Buffer, w, h float64) {
	fmt.Fprintf(buf,
		"<rect x=\"0\" y=\"0\" width=\"%.2f\" height=\"%.2f\""+
			" rx=\"4\" ry=\"4\""+
			" fill=\"#ffffff\" fill-opacity=\"0.9\""+
			" stroke=\"#999999\" stroke-opacity=\"0.8\" stroke-width=\"1\"/>\n",
		w, h)
}

// writeSVGLegendEntries renders all entries inside the legend group.
func writeSVGLegendEntries(buf *bytes.Buffer, data *model.LegendData) {
	if data.Orientation == orientationHorizontal {
		writeSVGLegendEntriesH(buf, data)

		return
	}

	cy := model.LegendPadding

	for i, entry := range data.Entries {
		if i > 0 {
			cy += model.EntryGap
		}

		cy = writeSVGSingleEntry(buf, data.Orientation, entry, model.LegendPadding, cy)
	}
}

// writeSVGLegendEntriesH renders entries side-by-side for horizontal layout.
func writeSVGLegendEntriesH(buf *bytes.Buffer, data *model.LegendData) {
	dc := gg.NewContext(1, 1)
	cx := model.LegendPadding

	for i, entry := range data.Entries {
		if i > 0 {
			cx += model.EntryGap
		}

		ew := measureSingleEntryHWidth(dc, entry)
		writeSVGSingleEntry(buf, data.Orientation, entry, cx, model.LegendPadding)
		cx += ew
	}
}

// measureSingleEntryHWidth measures just the width of a horizontal entry.
func measureSingleEntryHWidth(dc *gg.Context, entry model.LegendEntryData) float64 {
	tw, _ := dc.MeasureString(entry.Title)

	var entryW float64

	if entry.Kind == model.LegendEntryCategorical {
		entryW = measureCatHWidth(dc, entry)
	} else {
		entryW = float64(len(entry.Swatches)) * model.SwatchSize
	}

	return max(tw, entryW)
}

// measureCatHWidth measures the width of horizontal category swatches.
func measureCatHWidth(dc *gg.Context, entry model.LegendEntryData) float64 {
	w := 0.0

	for _, sw := range entry.Swatches {
		tw, _ := dc.MeasureString(sw.Label)
		w += max(model.SwatchSize, tw) + model.SwatchGap + model.LabelGap
	}

	return w
}

// writeSVGSingleEntry writes one legend entry and returns the new Y position.
func writeSVGSingleEntry(
	buf *bytes.Buffer,
	orientation string,
	entry model.LegendEntryData,
	x, y float64,
) float64 {
	title := html.EscapeString(entry.Title)
	fmt.Fprintf(buf,
		"<text x=\"%.2f\" y=\"%.2f\""+
			" font-family=\"sans-serif\" font-size=\"%.1f\""+
			" fill=\"#262626\">%s</text>\n",
		x, y+model.TitleFontSize, model.TitleFontSize, title)

	y += model.TitleFontSize + model.LabelGap

	if entry.Kind == model.LegendEntryCategorical {
		return writeSVGCategorySwatches(buf, orientation, entry, x, y)
	}

	return writeSVGNumericSwatches(buf, orientation, entry, x, y)
}

// writeSVGNumericSwatches renders colour swatches for numeric metrics.
func writeSVGNumericSwatches(
	buf *bytes.Buffer,
	orientation string,
	entry model.LegendEntryData,
	x, y float64,
) float64 {
	if len(entry.Swatches) == 0 {
		return y
	}

	if orientation == orientationHorizontal {
		return writeSVGNumericH(buf, entry, x, y)
	}

	return writeSVGNumericV(buf, entry, x, y)
}

// writeSVGNumericV writes vertically stacked numeric swatches.
func writeSVGNumericV(buf *bytes.Buffer, entry model.LegendEntryData, x, y float64) float64 {
	for _, sw := range entry.Swatches {
		writeSVGSwatch(buf, x, y, rgbaToCSS(sw.Colour))

		if sw.Label != "" {
			fmt.Fprintf(buf,
				"<text x=\"%.2f\" y=\"%.2f\""+
					" font-family=\"sans-serif\" font-size=\"%.1f\""+
					" fill=\"#333333\" dominant-baseline=\"central\">%s</text>\n",
				x+model.SwatchSize+model.LabelGap, y+model.SwatchSize,
				model.LegendFontSize, html.EscapeString(sw.Label))
		}

		y += model.SwatchSize
	}

	return y
}

// writeSVGNumericH writes horizontally arranged numeric swatches.
func writeSVGNumericH(buf *bytes.Buffer, entry model.LegendEntryData, x, y float64) float64 {
	cx := x

	for _, sw := range entry.Swatches {
		writeSVGSwatch(buf, cx, y, rgbaToCSS(sw.Colour))

		if sw.Label != "" {
			fmt.Fprintf(buf,
				"<text x=\"%.2f\" y=\"%.2f\""+
					" font-family=\"sans-serif\" font-size=\"%.1f\""+
					" fill=\"#333333\" text-anchor=\"middle\""+
					" dominant-baseline=\"central\">%s</text>\n",
				cx+model.SwatchSize, y+model.SwatchSize+model.LegendLineHeight,
				model.LegendFontSize, html.EscapeString(sw.Label))
		}

		cx += model.SwatchSize
	}

	return y + model.SwatchSize + model.LegendLineHeight + model.LabelGap
}

// writeSVGCategorySwatches renders category swatches.
func writeSVGCategorySwatches(
	buf *bytes.Buffer,
	orientation string,
	entry model.LegendEntryData,
	x, y float64,
) float64 {
	if orientation == orientationHorizontal {
		return writeSVGCategoryH(buf, entry, x, y)
	}

	return writeSVGCategoryV(buf, entry, x, y)
}

// writeSVGCategoryV writes vertically stacked category swatches.
func writeSVGCategoryV(buf *bytes.Buffer, entry model.LegendEntryData, x, y float64) float64 {
	for _, sw := range entry.Swatches {
		writeSVGSwatch(buf, x, y, rgbaToCSS(sw.Colour))

		fmt.Fprintf(buf,
			"<text x=\"%.2f\" y=\"%.2f\""+
				" font-family=\"sans-serif\" font-size=\"%.1f\""+
				" fill=\"#333333\" dominant-baseline=\"central\">%s</text>\n",
			x+model.SwatchSize+model.LabelGap, y+model.SwatchSize/2,
			model.LegendFontSize, html.EscapeString(sw.Label))

		y += model.SwatchSize + model.SwatchGap
	}

	return y
}

// writeSVGCategoryH writes horizontally arranged category swatches.
func writeSVGCategoryH(buf *bytes.Buffer, entry model.LegendEntryData, x, y float64) float64 {
	dc := gg.NewContext(1, 1)
	cx := x

	for _, sw := range entry.Swatches {
		writeSVGSwatch(buf, cx, y, rgbaToCSS(sw.Colour))

		fmt.Fprintf(buf,
			"<text x=\"%.2f\" y=\"%.2f\""+
				" font-family=\"sans-serif\" font-size=\"%.1f\""+
				" fill=\"#333333\" text-anchor=\"middle\""+
				" dominant-baseline=\"central\">%s</text>\n",
			cx+model.SwatchSize/2, y+model.SwatchSize+model.LegendLineHeight,
			model.LegendFontSize, html.EscapeString(sw.Label))

		tw, _ := dc.MeasureString(sw.Label)
		cx += max(model.SwatchSize, tw) + model.SwatchGap + model.LabelGap
	}

	return y + model.SwatchSize + model.LegendLineHeight + model.LabelGap
}

// writeSVGSwatch writes a single coloured rectangle.
func writeSVGSwatch(buf *bytes.Buffer, x, y float64, fillCSS string) {
	fmt.Fprintf(buf,
		"<rect x=\"%.2f\" y=\"%.2f\" width=\"%.2f\" height=\"%.2f\""+
			" fill=\"%s\" stroke=\"#666666\" stroke-width=\"0.5\"/>\n",
		x, y, model.SwatchSize, model.SwatchSize, fillCSS)
}
