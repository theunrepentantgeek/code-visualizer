package render

import (
	"fmt"
	"image/color"
	"strconv"

	"github.com/fogleman/gg"

	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/palette"
)

// drawLegend draws the legend overlay on the gg context.
// Does nothing if info is nil or position is none.
func drawLegend(dc *gg.Context, info *LegendInfo, canvasW, canvasH int) {
	if info == nil || info.Position == LegendPositionNone || len(info.Entries) == 0 {
		return
	}

	w, h := measureLegend(dc, info)
	ox, oy := legendOrigin(
		info.Position,
		float64(canvasW), float64(canvasH),
		w, h,
	)

	drawLegendBackground(dc, ox, oy, w, h)
	drawLegendEntries(dc, info, ox+legendPadding, oy+legendPadding)
}

// drawLegendBackground draws a semi-transparent white rectangle with border.
func drawLegendBackground(dc *gg.Context, x, y, w, h float64) {
	dc.SetRGBA(1, 1, 1, 0.9)
	dc.DrawRoundedRectangle(x, y, w, h, 4)
	dc.Fill()

	dc.SetRGBA(0.6, 0.6, 0.6, 0.8)
	dc.SetLineWidth(1)
	dc.DrawRoundedRectangle(x, y, w, h, 4)
	dc.Stroke()
}

// drawLegendEntries renders all entries within the legend box.
func drawLegendEntries(dc *gg.Context, info *LegendInfo, x, y float64) {
	if info.Orientation == LegendOrientationHorizontal {
		drawLegendEntriesH(dc, info, x, y)

		return
	}

	cy := y

	for i, entry := range info.Entries {
		if i > 0 {
			cy += entryGap
		}

		cy = drawSingleEntry(dc, info.Orientation, entry, x, cy)
	}
}

// drawLegendEntriesH renders entries side-by-side for horizontal layout.
func drawLegendEntriesH(dc *gg.Context, info *LegendInfo, x, y float64) {
	cx := x

	for i, entry := range info.Entries {
		if i > 0 {
			cx += entryGap
		}

		ew, _ := measureSingleEntryH(dc, entry)
		drawSingleEntry(dc, info.Orientation, entry, cx, y)
		cx += ew
	}
}

// drawSingleEntry draws one legend entry and returns the new Y cursor position.
func drawSingleEntry(
	dc *gg.Context,
	orientation LegendOrientation,
	entry LegendEntry,
	x, y float64,
) float64 {
	// Title
	dc.SetRGB(0.15, 0.15, 0.15)
	dc.DrawString(fmt.Sprintf("%s: %s", entry.Role, entry.MetricName), x, y+titleFontSize)
	y += titleFontSize + labelGap

	if entry.Kind == metric.Classification {
		return drawCategorySwatches(dc, orientation, entry, x, y)
	}

	return drawNumericSwatches(dc, orientation, entry, x, y)
}

// drawNumericSwatches renders colour swatches for Quantity/Measure metrics.
func drawNumericSwatches(
	dc *gg.Context,
	orientation LegendOrientation,
	entry LegendEntry,
	x, y float64,
) float64 {
	if entry.NumBuckets() <= 0 || len(entry.Palette.Colours) == 0 {
		return y
	}

	if orientation == LegendOrientationHorizontal {
		return drawNumericSwatchesH(dc, entry, x, y)
	}

	return drawNumericSwatchesV(dc, entry, x, y)
}

// drawNumericSwatchesV draws vertically stacked swatches with breakpoints.
func drawNumericSwatchesV(
	dc *gg.Context,
	entry LegendEntry,
	x, y float64,
) float64 {
	for i := range entry.NumBuckets() {
		colour := mapBucketColour(i, entry)
		dc.SetColor(colour)
		dc.DrawRectangle(x, y, swatchSize, swatchSize)
		dc.Fill()

		dc.SetRGB(0.4, 0.4, 0.4)
		dc.SetLineWidth(0.5)
		dc.DrawRectangle(x, y, swatchSize, swatchSize)
		dc.Stroke()

		// Breakpoint label at divider between swatches
		if entry.Buckets != nil && i < len(entry.Buckets.Boundaries) {
			label := formatBreakpoint(entry.Buckets.Boundaries[i])

			dc.SetRGB(0.2, 0.2, 0.2)
			dc.DrawStringAnchored(
				label,
				x+swatchSize+labelGap,
				y+swatchSize,
				0, 0.5,
			)
		}

		y += swatchSize
	}

	return y
}

// drawNumericSwatchesH draws horizontally arranged swatches with breakpoints.
func drawNumericSwatchesH(
	dc *gg.Context,
	entry LegendEntry,
	x, y float64,
) float64 {
	cx := x

	for i := range entry.NumBuckets() {
		colour := mapBucketColour(i, entry)
		dc.SetColor(colour)
		dc.DrawRectangle(cx, y, swatchSize, swatchSize)
		dc.Fill()

		dc.SetRGB(0.4, 0.4, 0.4)
		dc.SetLineWidth(0.5)
		dc.DrawRectangle(cx, y, swatchSize, swatchSize)
		dc.Stroke()

		// Breakpoint at divider
		if entry.Buckets != nil && i < len(entry.Buckets.Boundaries) {
			label := formatBreakpoint(entry.Buckets.Boundaries[i])

			dc.SetRGB(0.2, 0.2, 0.2)
			dc.DrawStringAnchored(
				label,
				cx+swatchSize,
				y+swatchSize+legendLineHeight,
				0.5, 0.5,
			)
		}

		cx += swatchSize
	}

	return y + swatchSize + legendLineHeight + labelGap
}

// drawCategorySwatches renders swatches for Classification metrics.
func drawCategorySwatches(
	dc *gg.Context,
	orientation LegendOrientation,
	entry LegendEntry,
	x, y float64,
) float64 {
	cats := entry.Categories

	if orientation == LegendOrientationHorizontal {
		y = drawCategorySwatchesH(dc, cats, x, y)
	} else {
		y = drawCategorySwatchesV(dc, cats, x, y)
	}

	return y
}

// drawCategorySwatchesV draws vertically stacked category swatches.
func drawCategorySwatchesV(
	dc *gg.Context,
	cats []CategorySwatch,
	x, y float64,
) float64 {
	for _, cat := range cats {
		dc.SetColor(cat.Colour)
		dc.DrawRectangle(x, y, swatchSize, swatchSize)
		dc.Fill()

		dc.SetRGB(0.4, 0.4, 0.4)
		dc.SetLineWidth(0.5)
		dc.DrawRectangle(x, y, swatchSize, swatchSize)
		dc.Stroke()

		dc.SetRGB(0.2, 0.2, 0.2)
		dc.DrawStringAnchored(
			cat.Label,
			x+swatchSize+labelGap,
			y+swatchSize/2,
			0, 0.5,
		)

		y += swatchSize + swatchGap
	}

	return y
}

// drawCategorySwatchesH draws horizontally arranged category swatches.
func drawCategorySwatchesH(
	dc *gg.Context,
	cats []CategorySwatch,
	x, y float64,
) float64 {
	cx := x

	for _, cat := range cats {
		dc.SetColor(cat.Colour)
		dc.DrawRectangle(cx, y, swatchSize, swatchSize)
		dc.Fill()

		dc.SetRGB(0.4, 0.4, 0.4)
		dc.SetLineWidth(0.5)
		dc.DrawRectangle(cx, y, swatchSize, swatchSize)
		dc.Stroke()

		dc.SetRGB(0.2, 0.2, 0.2)
		tw, _ := dc.MeasureString(cat.Label)
		dc.DrawStringAnchored(
			cat.Label,
			cx+swatchSize/2,
			y+swatchSize+legendLineHeight,
			0.5, 0.5,
		)

		cx += max(swatchSize, tw) + swatchGap + labelGap
	}

	return y + swatchSize + legendLineHeight + labelGap
}

// measureLegend computes the total width and height of the legend box.
func measureLegend(dc *gg.Context, info *LegendInfo) (width, height float64) {
	if info.Orientation == LegendOrientationHorizontal {
		return measureLegendH(dc, info)
	}

	return measureLegendV(dc, info)
}

// measureLegendV computes legend size for vertical layout.
func measureLegendV(dc *gg.Context, info *LegendInfo) (width, height float64) {
	var totalH float64

	maxW := 0.0

	for i, entry := range info.Entries {
		if i > 0 {
			totalH += entryGap
		}

		tw, _ := dc.MeasureString(fmt.Sprintf("%s: %s", entry.Role, entry.MetricName))
		totalH += titleFontSize + labelGap

		if tw > maxW {
			maxW = tw
		}

		entryW, entryH := measureEntryV(dc, entry)
		totalH += entryH

		if entryW > maxW {
			maxW = entryW
		}
	}

	return maxW + 2*legendPadding, totalH + 2*legendPadding
}

// measureLegendH computes legend size for horizontal layout.
// Entries are arranged side-by-side, so total width is the sum of all entry
// widths and total height is the tallest single entry.
func measureLegendH(dc *gg.Context, info *LegendInfo) (width, height float64) {
	var totalW float64

	maxH := 0.0

	for i, entry := range info.Entries {
		if i > 0 {
			totalW += entryGap
		}

		entryW, entryH := measureSingleEntryH(dc, entry)
		totalW += entryW

		if entryH > maxH {
			maxH = entryH
		}
	}

	return totalW + 2*legendPadding, maxH + 2*legendPadding
}

// measureSingleEntryH measures one entry including its title for horizontal layout.
func measureSingleEntryH(dc *gg.Context, entry LegendEntry) (width, height float64) {
	tw, _ := dc.MeasureString(fmt.Sprintf("%s: %s", entry.Role, entry.MetricName))
	titleH := titleFontSize + labelGap

	entryW, entryH := measureEntryH(dc, entry)

	w := max(tw, entryW)
	h := titleH + entryH

	return w, h
}

// measureEntryV measures a single entry in vertical layout.
func measureEntryV(dc *gg.Context, entry LegendEntry) (width, height float64) {
	if entry.Kind == metric.Classification {
		return measureCategoryV(dc, entry)
	}

	return measureNumericV(dc, entry)
}

// measureEntryH measures a single entry in horizontal layout.
func measureEntryH(dc *gg.Context, entry LegendEntry) (width, height float64) {
	if entry.Kind == metric.Classification {
		return measureCategoryH(dc, entry)
	}

	return measureNumericH(dc, entry)
}

// measureNumericV measures numeric entry in vertical layout.
func measureNumericV(dc *gg.Context, entry LegendEntry) (width, height float64) {
	h := float64(entry.NumBuckets()) * swatchSize
	w := swatchSize

	if entry.Buckets != nil {
		for _, b := range entry.Buckets.Boundaries {
			tw, _ := dc.MeasureString(formatBreakpoint(b))

			if bw := swatchSize + labelGap + tw; bw > w {
				w = bw
			}
		}
	}

	return w, h
}

// measureNumericH measures numeric entry in horizontal layout.
func measureNumericH(_ *gg.Context, entry LegendEntry) (width, height float64) {
	w := float64(entry.NumBuckets()) * swatchSize
	h := swatchSize + legendLineHeight + labelGap

	return w, h
}

// measureCategoryV measures category entry in vertical layout.
func measureCategoryV(dc *gg.Context, entry LegendEntry) (width, height float64) {
	cats := entry.Categories

	w := swatchSize
	h := float64(len(cats)) * (swatchSize + swatchGap)

	for _, cat := range cats {
		tw, _ := dc.MeasureString(cat.Label)

		if cw := swatchSize + labelGap + tw; cw > w {
			w = cw
		}
	}

	return w, h
}

// measureCategoryH measures category entry in horizontal layout.
func measureCategoryH(dc *gg.Context, entry LegendEntry) (width, height float64) {
	cats := entry.Categories

	w := 0.0

	for _, cat := range cats {
		tw, _ := dc.MeasureString(cat.Label)
		w += max(swatchSize, tw) + swatchGap + labelGap
	}

	h := swatchSize + legendLineHeight + labelGap

	return w, h
}

// mapBucketColour returns the colour for a given bucket index.
func mapBucketColour(bucketIdx int, entry LegendEntry) color.RGBA {
	if len(entry.Palette.Colours) == 0 {
		return color.RGBA{R: 128, G: 128, B: 128, A: 255}
	}

	return palette.MapNumericToColour(bucketIdx, entry.NumBuckets(), entry.Palette)
}

// formatBreakpoint formats a numeric breakpoint for display.
func formatBreakpoint(v float64) string {
	if v == float64(int(v)) {
		return strconv.Itoa(int(v))
	}

	return fmt.Sprintf("%.1f", v)
}
