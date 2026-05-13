// Package legendlayout provides shared legend measurement and positioning
// used by the Canvas layer and both backends.
package legendlayout

import (
	"fmt"
	"strconv"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
)

// FormatBreakpoint formats a numeric breakpoint for display.
func FormatBreakpoint(v float64) string {
	if v == float64(int(v)) {
		return strconv.Itoa(int(v))
	}

	return fmt.Sprintf("%.1f", v)
}

// MeasureLegend computes the total width and height of the legend box
// including padding. Returns (0, 0) if data is nil or has no entries.
func MeasureLegend(data *model.LegendData, measurer StringMeasurer) (width, height float64) {
	if data == nil || len(data.Entries) == 0 {
		return 0, 0
	}

	if data.Orientation == "horizontal" {
		return measureLegendH(measurer, data)
	}

	return measureLegendV(measurer, data)
}

// LegendOrigin computes the top-left (x, y) for the legend box.
func LegendOrigin(
	position string,
	canvasW, canvasH float64,
	legendW, legendH float64,
) (ox, oy float64) {
	m := model.LegendMargin

	switch position {
	case "top-left":
		return m, m
	case "top-center":
		return (canvasW - legendW) / 2, m
	case "top-right":
		return canvasW - legendW - m, m
	case "center-right":
		return canvasW - legendW - m, (canvasH - legendH) / 2
	case "bottom-right":
		return canvasW - legendW - m, canvasH - legendH - m
	case "bottom-center":
		return (canvasW - legendW) / 2, canvasH - legendH - m
	case "center-left":
		return m, (canvasH - legendH) / 2
	default:
		return m, canvasH - legendH - m
	}
}

// ReserveSpace computes the width and height reductions needed to reserve
// space for the legend. Returns zeros if data is nil, position is "none",
// or there are no entries.
func ReserveSpace(data *model.LegendData, measurer StringMeasurer) (widthReduction, heightReduction float64) {
	if data == nil || data.Position == "none" || len(data.Entries) == 0 {
		return 0, 0
	}

	w, h := MeasureLegend(data, measurer)
	m := model.LegendMargin

	switch data.Position {
	case "center-left", "center-right":
		return w + 2*m, 0
	case "top-center", "bottom-center":
		return 0, h + 2*m
	default:
		if data.Orientation == "horizontal" {
			return 0, h + 2*m
		}

		return w + 2*m, 0
	}
}

func measureLegendV(measurer StringMeasurer, data *model.LegendData) (width, height float64) {
	var totalH float64

	maxW := 0.0

	for i, entry := range data.Entries {
		if i > 0 {
			totalH += model.EntryGap
		}

		tw, _ := measurer.MeasureString(entry.Title)
		totalH += model.TitleFontSize + model.LabelGap

		if tw > maxW {
			maxW = tw
		}

		entryW, entryH := measureEntryV(measurer, entry)
		totalH += entryH

		if entryW > maxW {
			maxW = entryW
		}
	}

	return maxW + 2*model.LegendPadding, totalH + 2*model.LegendPadding
}

func measureLegendH(measurer StringMeasurer, data *model.LegendData) (width, height float64) {
	var totalW float64

	maxH := 0.0

	for i, entry := range data.Entries {
		if i > 0 {
			totalW += model.EntryGap
		}

		entryW, entryH := measureSingleEntryH(measurer, entry)
		totalW += entryW

		if entryH > maxH {
			maxH = entryH
		}
	}

	return totalW + 2*model.LegendPadding, maxH + 2*model.LegendPadding
}

func measureSingleEntryH(measurer StringMeasurer, entry model.LegendEntryData) (width, height float64) {
	tw, _ := measurer.MeasureString(entry.Title)
	titleH := model.TitleFontSize + model.LabelGap

	entryW, entryH := measureEntryH(measurer, entry)

	w := max(tw, entryW)
	h := titleH + entryH

	return w, h
}

func measureEntryV(measurer StringMeasurer, entry model.LegendEntryData) (width, height float64) {
	if entry.Kind == model.LegendEntryCategorical {
		return measureCategoryV(measurer, entry)
	}

	return measureNumericV(measurer, entry)
}

func measureEntryH(measurer StringMeasurer, entry model.LegendEntryData) (width, height float64) {
	if entry.Kind == model.LegendEntryCategorical {
		return measureCategoryH(measurer, entry)
	}

	return measureNumericH(entry)
}

func measureNumericV(measurer StringMeasurer, entry model.LegendEntryData) (width, height float64) {
	n := len(entry.Swatches)
	h := float64(n) * model.SwatchSize
	w := model.SwatchSize

	for _, sw := range entry.Swatches {
		if sw.Label != "" {
			tw, _ := measurer.MeasureString(sw.Label)

			if bw := model.SwatchSize + model.LabelGap + tw; bw > w {
				w = bw
			}
		}
	}

	return w, h
}

func measureNumericH(entry model.LegendEntryData) (width, height float64) {
	n := len(entry.Swatches)
	w := float64(n) * model.SwatchSize
	h := model.SwatchSize + model.LegendLineHeight + model.LabelGap

	return w, h
}

func measureCategoryV(measurer StringMeasurer, entry model.LegendEntryData) (width, height float64) {
	n := len(entry.Swatches)

	w := model.SwatchSize
	h := float64(n) * (model.SwatchSize + model.SwatchGap)

	for _, sw := range entry.Swatches {
		tw, _ := measurer.MeasureString(sw.Label)

		if cw := model.SwatchSize + model.LabelGap + tw; cw > w {
			w = cw
		}
	}

	return w, h
}

func measureCategoryH(measurer StringMeasurer, entry model.LegendEntryData) (width, height float64) {
	w := 0.0

	for _, sw := range entry.Swatches {
		tw, _ := measurer.MeasureString(sw.Label)
		w += max(model.SwatchSize, tw) + model.SwatchGap + model.LabelGap
	}

	h := model.SwatchSize + model.LegendLineHeight + model.LabelGap

	return w, h
}

// MeasureEntryHWidth returns the total width of one legend entry in
// horizontal layout mode. Used by the Canvas layer to position entries
// when decomposing the legend into primitives.
func MeasureEntryHWidth(entry model.LegendEntryData) float64 {
	dc := gg.NewContext(1, 1)
	w, _ := measureSingleEntryH(dc, entry)

	return w
}

// MeasureCatSwatchColumnWidth returns the width of a single categorical
// swatch column (swatch plus label gap) for horizontal layout.
func MeasureCatSwatchColumnWidth(label string) float64 {
	dc := gg.NewContext(1, 1)
	tw, _ := dc.MeasureString(label)

	return max(model.SwatchSize, tw) + model.SwatchGap + model.LabelGap
}
