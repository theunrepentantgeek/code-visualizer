package raster

import (
	"github.com/fogleman/gg"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/legendlayout"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
)

const orientationHorizontal = "horizontal"

// drawLegend draws the legend overlay on the gg context.
func drawLegend(dc *gg.Context, data model.LegendData, canvasW, canvasH int) {
	if data.Position == "none" || len(data.Entries) == 0 {
		return
	}

	w, h := legendlayout.MeasureLegend(&data, legendlayout.NewBasicMeasurer())
	ox, oy := legendlayout.LegendOrigin(data.Position, float64(canvasW), float64(canvasH), w, h)

	drawLegendBackground(dc, ox, oy, w, h)
	drawLegendEntries(dc, &data, ox+model.LegendPadding, oy+model.LegendPadding)
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
func drawLegendEntries(dc *gg.Context, data *model.LegendData, x, y float64) {
	if data.Orientation == orientationHorizontal {
		drawLegendEntriesH(dc, data, x, y)

		return
	}

	cy := y

	for i, entry := range data.Entries {
		if i > 0 {
			cy += model.EntryGap
		}

		cy = drawSingleEntry(dc, data.Orientation, entry, x, cy)
	}
}

// drawLegendEntriesH renders entries side-by-side for horizontal layout.
func drawLegendEntriesH(dc *gg.Context, data *model.LegendData, x, y float64) {
	cx := x

	for i, entry := range data.Entries {
		if i > 0 {
			cx += model.EntryGap
		}

		ew := measureEntryWidth(dc, entry)
		drawSingleEntry(dc, data.Orientation, entry, cx, y)
		cx += ew
	}
}

// measureEntryWidth measures one entry's width for horizontal layout.
func measureEntryWidth(dc *gg.Context, entry model.LegendEntryData) float64 {
	tw, _ := dc.MeasureString(entry.Title)
	entryW := swatchBlockWidth(dc, entry)

	return max(tw, entryW)
}

// swatchBlockWidth returns the width of the swatch block for an entry.
func swatchBlockWidth(dc *gg.Context, entry model.LegendEntryData) float64 {
	if entry.Kind == model.LegendEntryCategorical {
		w := 0.0

		for _, sw := range entry.Swatches {
			tw, _ := dc.MeasureString(sw.Label)
			w += max(model.SwatchSize, tw) + model.SwatchGap + model.LabelGap
		}

		return w
	}

	return float64(len(entry.Swatches)) * model.SwatchSize
}

// drawSingleEntry draws one legend entry and returns the new Y cursor.
func drawSingleEntry(
	dc *gg.Context,
	orientation string,
	entry model.LegendEntryData,
	x, y float64,
) float64 {
	dc.SetRGB(0.15, 0.15, 0.15)
	dc.DrawString(entry.Title, x, y+model.TitleFontSize)

	y += model.TitleFontSize + model.LabelGap

	if entry.Kind == model.LegendEntryCategorical {
		return drawCategorySwatches(dc, orientation, entry, x, y)
	}

	return drawNumericSwatches(dc, orientation, entry, x, y)
}

// drawNumericSwatches renders colour swatches for numeric metrics.
func drawNumericSwatches(
	dc *gg.Context,
	orientation string,
	entry model.LegendEntryData,
	x, y float64,
) float64 {
	if len(entry.Swatches) == 0 {
		return y
	}

	if orientation == orientationHorizontal {
		return drawNumericSwatchesH(dc, entry, x, y)
	}

	return drawNumericSwatchesV(dc, entry, x, y)
}

// drawNumericSwatchesV draws vertically stacked numeric swatches.
func drawNumericSwatchesV(
	dc *gg.Context,
	entry model.LegendEntryData,
	x, y float64,
) float64 {
	for _, sw := range entry.Swatches {
		drawSwatchRect(dc, sw, x, y)

		if sw.Label != "" {
			dc.SetRGB(0.2, 0.2, 0.2)
			dc.DrawStringAnchored(
				sw.Label,
				x+model.SwatchSize+model.LabelGap,
				y+model.SwatchSize,
				0, 0.5,
			)
		}

		y += model.SwatchSize
	}

	return y
}

// drawNumericSwatchesH draws horizontally arranged numeric swatches.
func drawNumericSwatchesH(
	dc *gg.Context,
	entry model.LegendEntryData,
	x, y float64,
) float64 {
	cx := x

	for _, sw := range entry.Swatches {
		drawSwatchRect(dc, sw, cx, y)

		if sw.Label != "" {
			dc.SetRGB(0.2, 0.2, 0.2)
			dc.DrawStringAnchored(
				sw.Label,
				cx+model.SwatchSize,
				y+model.SwatchSize+model.LegendLineHeight,
				0.5, 0.5,
			)
		}

		cx += model.SwatchSize
	}

	return y + model.SwatchSize + model.LegendLineHeight + model.LabelGap
}

// drawCategorySwatches renders swatches for categorical metrics.
func drawCategorySwatches(
	dc *gg.Context,
	orientation string,
	entry model.LegendEntryData,
	x, y float64,
) float64 {
	if orientation == orientationHorizontal {
		return drawCategorySwatchesH(dc, entry, x, y)
	}

	return drawCategorySwatchesV(dc, entry, x, y)
}

// drawCategorySwatchesV draws vertically stacked category swatches.
func drawCategorySwatchesV(
	dc *gg.Context,
	entry model.LegendEntryData,
	x, y float64,
) float64 {
	for _, sw := range entry.Swatches {
		drawSwatchRect(dc, sw, x, y)

		dc.SetRGB(0.2, 0.2, 0.2)
		dc.DrawStringAnchored(
			sw.Label,
			x+model.SwatchSize+model.LabelGap,
			y+model.SwatchSize/2,
			0, 0.5,
		)

		y += model.SwatchSize + model.SwatchGap
	}

	return y
}

// drawCategorySwatchesH draws horizontally arranged category swatches.
func drawCategorySwatchesH(
	dc *gg.Context,
	entry model.LegendEntryData,
	x, y float64,
) float64 {
	cx := x

	for _, sw := range entry.Swatches {
		drawSwatchRect(dc, sw, cx, y)

		dc.SetRGB(0.2, 0.2, 0.2)
		tw, _ := dc.MeasureString(sw.Label)
		dc.DrawStringAnchored(
			sw.Label,
			cx+model.SwatchSize/2,
			y+model.SwatchSize+model.LegendLineHeight,
			0.5, 0.5,
		)

		cx += max(model.SwatchSize, tw) + model.SwatchGap + model.LabelGap
	}

	return y + model.SwatchSize + model.LegendLineHeight + model.LabelGap
}

// drawSwatchRect draws a single colour swatch rectangle with border.
func drawSwatchRect(dc *gg.Context, sw model.LegendSwatch, x, y float64) {
	dc.SetColor(sw.Colour)
	dc.DrawRectangle(x, y, model.SwatchSize, model.SwatchSize)
	dc.Fill()

	dc.SetRGB(0.4, 0.4, 0.4)
	dc.SetLineWidth(0.5)
	dc.DrawRectangle(x, y, model.SwatchSize, model.SwatchSize)
	dc.Stroke()
}
