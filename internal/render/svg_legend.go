package render

import (
	"fmt"
	"html"
	"image/color"
	"os"
	"strconv"

	"github.com/bevan/code-visualizer/internal/metric"
)

// svgLegendTextColour is the text colour used for legend labels in SVG output.
var svgLegendTextColour = color.RGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xFF}

// writeSVGLegend writes a legend band as SVG elements at position (x, y)
// within the given width. Each LegendRow becomes a horizontal swatch bar
// with a metric label and breakpoint/category labels.
// This is the SVG equivalent of DrawLegendBand (which operates on a gg.Context).
func writeSVGLegend(f *os.File, info *LegendInfo, y, width float64) {
	if info == nil || len(info.Rows) == 0 {
		return
	}

	fmt.Fprintf(f, "<g transform=\"translate(0,%.2f)\">\n", y)

	rowY := legendPaddingTop

	for i := range info.Rows {
		row := &info.Rows[i]
		writeSVGLegendRow(f, row, 0, rowY, width)

		rowY += legendRowHeight + legendRowGap
	}

	fmt.Fprint(f, "</g>\n")
}

// writeSVGLegendRow renders one metric row: label on the left, colour swatches
// and breakpoint/category labels on the right.
func writeSVGLegendRow(f *os.File, row *LegendRow, x, y, width float64) {
	if len(row.Colours) == 0 {
		return
	}

	// Metric name label on the left, vertically centred on the swatch bar.
	labelY := y + legendSwatchHeight/2.0

	writeSVGText(f,
		x+4, labelY,
		colourToHex(svgLegendTextColour),
		"",
		html.EscapeString(row.MetricName))

	// Swatch area starts after the label region.
	swatchX := x + legendLabelWidth
	swatchW := width - legendLabelWidth

	if swatchW <= 0 {
		return
	}

	writeSVGSwatches(f, row.Colours, swatchX, y, swatchW)

	switch row.Kind {
	case metric.Quantity, metric.Measure:
		writeSVGNumericLabels(f, row, swatchX, y, swatchW)
	case metric.Classification:
		writeSVGCategoryLabels(f, row, swatchX, y, swatchW)
	default:
		// Unknown metric kind — skip labels.
	}
}

// writeSVGSwatches draws a horizontal row of coloured rectangles with thin borders.
func writeSVGSwatches(f *os.File, colours []color.RGBA, x, y, totalWidth float64) {
	n := float64(len(colours))
	gaps := (n - 1) * legendSwatchGap
	cellW := (totalWidth - gaps) / n

	for i, c := range colours {
		cx := x + float64(i)*(cellW+legendSwatchGap)

		fmt.Fprintf(f,
			"<rect x=\"%.2f\" y=\"%.2f\" width=\"%.2f\" height=\"%.2f\""+
				" fill=\"%s\" stroke=\"#808080\" stroke-width=\"0.5\"/>\n",
			cx, y, cellW, legendSwatchHeight,
			colourToHex(c))
	}
}

// writeSVGNumericLabels draws breakpoint values at swatch dividers.
func writeSVGNumericLabels(f *os.File, row *LegendRow, swatchX, swatchY, swatchW float64) {
	n := float64(len(row.Colours))
	gaps := (n - 1) * legendSwatchGap
	cellW := (swatchW - gaps) / n
	labelY := swatchY + legendSwatchHeight + legendLabelFontSize + 1

	for i, bp := range row.Breakpoints {
		divX := swatchX + float64(i+1)*(cellW+legendSwatchGap) - legendSwatchGap/2
		label := svgFormatBreakpoint(bp, row.Kind)

		writeSVGText(f,
			divX, labelY,
			colourToHex(svgLegendTextColour),
			"middle",
			html.EscapeString(label))
	}
}

// writeSVGCategoryLabels draws category names centred under each swatch.
func writeSVGCategoryLabels(f *os.File, row *LegendRow, swatchX, swatchY, swatchW float64) {
	n := float64(len(row.Colours))
	gaps := (n - 1) * legendSwatchGap
	cellW := (swatchW - gaps) / n
	labelY := swatchY + legendSwatchHeight + legendLabelFontSize + 1

	for i, cat := range row.Categories {
		cx := swatchX + float64(i)*(cellW+legendSwatchGap) + cellW/2

		writeSVGText(f,
			cx, labelY,
			colourToHex(svgLegendTextColour),
			"middle",
			html.EscapeString(cat))
	}
}

// svgFormatBreakpoint formats a numeric breakpoint for SVG display.
// Mirrors formatBreakpoint from legend.go.
func svgFormatBreakpoint(v float64, kind metric.Kind) string {
	if kind == metric.Quantity {
		return strconv.FormatInt(int64(v), 10)
	}

	if v == float64(int64(v)) {
		return strconv.FormatFloat(v, 'f', 0, 64)
	}

	return strconv.FormatFloat(v, 'g', 2, 64)
}
