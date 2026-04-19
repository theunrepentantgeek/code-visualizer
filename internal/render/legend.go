package render

import (
	"image/color"
	"strconv"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"github.com/rotisserie/eris"
	"golang.org/x/image/font"

	"github.com/bevan/code-visualizer/internal/metric"
)

// Legend layout constants.
const (
	legendRowHeight     = 30.0  // height of one colour-bar row
	legendSwatchHeight  = 18.0  // height of colour swatches within a row
	legendLabelFontSize = 11.0  // font size for metric name and breakpoint labels
	legendPaddingTop    = 8.0   // space above the first row
	legendPaddingBottom = 6.0   // space below the last row
	legendRowGap        = 6.0   // vertical gap between rows
	legendLabelWidth    = 120.0 // reserved width for the metric name on the left
	legendSwatchGap     = 1.0   // gap between adjacent swatches
)

// LegendRow describes a single metric row in the legend.
type LegendRow struct {
	// MetricName is the human-readable metric label shown on the left.
	MetricName string

	// Kind is the metric kind (Quantity, Measure, or Classification).
	Kind metric.Kind

	// Colours are the palette colours for the row, one per bucket or category.
	Colours []color.RGBA

	// Breakpoints are the numeric bucket boundaries (len = len(Colours)-1 for numeric metrics).
	// Empty for Classification metrics.
	Breakpoints []float64

	// Categories are the label strings for Classification metrics (len = len(Colours)).
	// Empty for Quantity/Measure metrics.
	Categories []string
}

// LegendInfo captures everything needed to render the legend band.
type LegendInfo struct {
	Rows []LegendRow
}

// ComputeLegendHeight returns the total pixel height needed for the legend band.
// Returns 0 if there are no rows to display.
func ComputeLegendHeight(info *LegendInfo) int {
	if info == nil || len(info.Rows) == 0 {
		return 0
	}

	n := float64(len(info.Rows))
	h := legendPaddingTop + n*legendRowHeight + (n-1)*legendRowGap + legendPaddingBottom

	return int(h + 0.5) // round up
}

// DrawLegendBand draws the legend onto dc at position (x, y) with the given width.
// Each row renders a horizontal colour bar with labels.
func DrawLegendBand(dc *gg.Context, info *LegendInfo, x, y, width float64) error {
	if info == nil || len(info.Rows) == 0 {
		return nil
	}

	face := legendFontFace(legendLabelFontSize)

	rowY := y + legendPaddingTop

	for i := range info.Rows {
		row := &info.Rows[i]
		if err := drawLegendRow(dc, row, face, x, rowY, width); err != nil {
			return eris.Wrapf(err, "drawing legend row %q", row.MetricName)
		}

		rowY += legendRowHeight + legendRowGap
	}

	return nil
}

// legendFontFace returns a font.Face for the legend labels using goregular.
func legendFontFace(size float64) font.Face {
	return truetype.NewFace(parsedFont, &truetype.Options{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
}

// drawLegendRow renders one metric row: label on the left, colour swatches + labels on the right.
func drawLegendRow(
	dc *gg.Context,
	row *LegendRow,
	face font.Face,
	x, y, width float64,
) error {
	if len(row.Colours) == 0 {
		return nil
	}

	dc.SetFontFace(face)

	// Draw metric name label on the left, vertically centred on the swatch bar.
	labelColour := color.RGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xFF}
	dc.SetColor(labelColour)

	labelY := y + legendSwatchHeight/2.0
	dc.DrawStringAnchored(row.MetricName, x+4, labelY, 0, 0.5)

	// Swatch area starts after the label region.
	swatchX := x + legendLabelWidth
	swatchW := width - legendLabelWidth

	if swatchW <= 0 {
		return nil
	}

	drawSwatches(dc, row.Colours, swatchX, y, swatchW)

	switch row.Kind {
	case metric.Quantity, metric.Measure:
		drawNumericLabels(dc, row, face, swatchX, y, swatchW)
	case metric.Classification:
		drawCategoryLabels(dc, row, face, swatchX, y, swatchW)
	default:
		return eris.Errorf("unknown metric kind: %d", row.Kind)
	}

	return nil
}

// drawSwatches draws a horizontal row of coloured rectangles.
func drawSwatches(dc *gg.Context, colours []color.RGBA, x, y, totalWidth float64) {
	n := float64(len(colours))
	gaps := (n - 1) * legendSwatchGap
	cellW := (totalWidth - gaps) / n

	for i, c := range colours {
		cx := x + float64(i)*(cellW+legendSwatchGap)

		dc.SetColor(c)
		dc.DrawRectangle(cx, y, cellW, legendSwatchHeight)
		dc.Fill()

		// Thin border around each swatch.
		dc.SetColor(color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xFF})
		dc.SetLineWidth(0.5)
		dc.DrawRectangle(cx, y, cellW, legendSwatchHeight)
		dc.Stroke()
	}
}

// drawNumericLabels draws breakpoint values aligned with the dividers between swatches.
func drawNumericLabels(
	dc *gg.Context,
	row *LegendRow,
	face font.Face,
	swatchX, swatchY, swatchW float64,
) {
	dc.SetFontFace(face)
	dc.SetColor(color.RGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xFF})

	n := float64(len(row.Colours))
	gaps := (n - 1) * legendSwatchGap
	cellW := (swatchW - gaps) / n
	labelY := swatchY + legendSwatchHeight + legendLabelFontSize + 1

	for i, bp := range row.Breakpoints {
		// Divider between swatch i and swatch i+1.
		divX := swatchX + float64(i+1)*(cellW+legendSwatchGap) - legendSwatchGap/2
		label := formatBreakpoint(bp, row.Kind)
		dc.DrawStringAnchored(label, divX, labelY, 0.5, 0.5)
	}
}

// drawCategoryLabels draws category names centred under each swatch.
func drawCategoryLabels(
	dc *gg.Context,
	row *LegendRow,
	face font.Face,
	swatchX, swatchY, swatchW float64,
) {
	dc.SetFontFace(face)
	dc.SetColor(color.RGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xFF})

	n := float64(len(row.Colours))
	gaps := (n - 1) * legendSwatchGap
	cellW := (swatchW - gaps) / n
	labelY := swatchY + legendSwatchHeight + legendLabelFontSize + 1

	for i, cat := range row.Categories {
		cx := swatchX + float64(i)*(cellW+legendSwatchGap) + cellW/2
		dc.DrawStringAnchored(cat, cx, labelY, 0.5, 0.5)
	}
}

// formatBreakpoint formats a numeric breakpoint value for display.
func formatBreakpoint(v float64, kind metric.Kind) string {
	if kind == metric.Quantity {
		return strconv.FormatInt(int64(v), 10)
	}

	// Measure: use compact float formatting.
	if v == float64(int64(v)) {
		return strconv.FormatFloat(v, 'f', 0, 64)
	}

	return strconv.FormatFloat(v, 'g', 2, 64)
}
