// Package legendlayout provides shared legend measurement and positioning
// used by the Canvas layer and both backends.
package legendlayout

import (
	"fmt"
	"strconv"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
)

// EntryArrangement controls how metric entries are grouped within a legend.
type EntryArrangement int

const (
	// EntriesStacked places each metric entry below the previous one.
	EntriesStacked EntryArrangement = iota
	// EntriesSideBySide places each metric entry beside the previous one.
	EntriesSideBySide
)

// FormatBreakpoint formats a numeric breakpoint for display.
func FormatBreakpoint(v float64) string {
	if v == float64(int(v)) {
		return strconv.Itoa(int(v))
	}

	return fmt.Sprintf("%.1f", v)
}

// MeasureLegend computes the total width and height of the legend box
// including padding. Returns a zero size if data is nil or has no entries.
func MeasureLegend(data *model.LegendData, measurer StringMeasurer) geometry.Size {
	if data == nil || len(data.Entries) == 0 {
		return geometry.Size{}
	}

	return MeasureLegendWithArrangement(data, measurer, preferredEntryArrangement(data.Orientation))
}

// MeasureLegendWithArrangement computes the legend size while laying out
// metric entries according to arrangement. Each entry retains data's
// configured swatch orientation.
func MeasureLegendWithArrangement(
	data *model.LegendData, measurer StringMeasurer, arrangement EntryArrangement,
) geometry.Size {
	if data == nil || len(data.Entries) == 0 {
		return geometry.Size{}
	}

	entriesSize := measureEntries(data, measurer, arrangement)
	if data.LabelSample != nil {
		entriesSize = includeLabelSample(data, measurer, entriesSize)
	}

	return geometry.NewSize(
		entriesSize.Width+2*model.LegendPadding,
		entriesSize.Height+2*model.LegendPadding,
	)
}

func measureEntries(
	data *model.LegendData, measurer StringMeasurer, arrangement EntryArrangement,
) geometry.Size {
	var size geometry.Size

	for index, entry := range data.Entries {
		if index > 0 {
			if arrangement == EntriesStacked {
				size.Height += model.EntryGap
			} else {
				size.Width += model.EntryGap
			}
		}

		entrySize := measureSingleEntry(data.Orientation, measurer, entry)

		if arrangement == EntriesStacked {
			size.Width = max(size.Width, entrySize.Width)
			size.Height += entrySize.Height
		} else {
			size.Width += entrySize.Width
			size.Height = max(size.Height, entrySize.Height)
		}
	}

	return size
}

func measureSingleEntry(
	orientation model.LegendOrientation,
	measurer StringMeasurer,
	entry model.LegendEntryData,
) geometry.Size {
	if orientation == model.LegendOrientationHorizontal {
		width, height := measureSingleEntryH(measurer, entry)

		return geometry.NewSize(width, height)
	}

	width, height := measureSingleEntryV(measurer, entry)

	return geometry.NewSize(width, height)
}

func includeLabelSample(
	data *model.LegendData, measurer StringMeasurer, entriesSize geometry.Size,
) geometry.Size {
	sampleSize := measureLabelSampleImpl(measurer, data.LabelSample)
	if data.Orientation == model.LegendOrientationVertical {
		entriesSize.Width = max(entriesSize.Width, sampleSize.Width)
		entriesSize.Height += sampleSize.Height + model.EntryGap
	} else {
		entriesSize.Width += sampleSize.Width + model.EntryGap
		entriesSize.Height = max(entriesSize.Height, sampleSize.Height)
	}

	return entriesSize
}

// ResolveEntryArrangement keeps the configured grouping when it fits and
// otherwise reflows entries onto the other axis.
func ResolveEntryArrangement(
	data *model.LegendData, available geometry.Size, measurer StringMeasurer,
) EntryArrangement {
	if data == nil {
		return EntriesStacked
	}

	preferred := preferredEntryArrangement(data.Orientation)
	if len(data.Entries) < 2 {
		return preferred
	}

	preferredSize := MeasureLegendWithArrangement(data, measurer, preferred)
	if fitsWithin(preferredSize, available) {
		return preferred
	}

	if preferred == EntriesStacked {
		return EntriesSideBySide
	}

	return EntriesStacked
}

func preferredEntryArrangement(orientation model.LegendOrientation) EntryArrangement {
	if orientation == model.LegendOrientationHorizontal {
		return EntriesSideBySide
	}

	return EntriesStacked
}

func fitsWithin(size, available geometry.Size) bool {
	return size.Width <= available.Width && size.Height <= available.Height
}

// ReserveSpace computes the width and height reductions needed to reserve
// space for the legend. Returns a zero size if data is nil, position is
// "none", or there are no entries.
func ReserveSpace(data *model.LegendData, measurer StringMeasurer) geometry.Size {
	if data == nil {
		return geometry.Size{}
	}

	return ReserveSpaceWithArrangement(data, measurer, preferredEntryArrangement(data.Orientation))
}

// ReserveSpaceWithArrangement computes the space needed for a legend using
// the supplied metric-entry arrangement.
func ReserveSpaceWithArrangement(
	data *model.LegendData, measurer StringMeasurer, arrangement EntryArrangement,
) geometry.Size {
	if data == nil || data.Position == model.LegendPositionNone || len(data.Entries) == 0 {
		return geometry.Size{}
	}

	size := MeasureLegendWithArrangement(data, measurer, arrangement)

	return reserveSpaceForSize(data, size)
}

func reserveSpaceForSize(data *model.LegendData, size geometry.Size) geometry.Size {
	m := model.LegendMargin

	switch data.Position {
	case model.LegendPositionCenterLeft, model.LegendPositionCenterRight:
		return geometry.Size{Width: size.Width + 2*m}
	case model.LegendPositionTopCenter, model.LegendPositionBottomCenter:
		return geometry.Size{Height: size.Height + 2*m}
	default:
		if data.Orientation == model.LegendOrientationHorizontal {
			return geometry.Size{Height: size.Height + 2*m}
		}

		return geometry.Size{Width: size.Width + 2*m}
	}
}

// LegendOrigin computes the top-left (x, y) for the legend box.
func LegendOrigin(
	position model.LegendPosition,
	available geometry.Size,
	legend geometry.Size,
) (ox, oy float64) {
	m := model.LegendMargin

	switch position {
	case model.LegendPositionTopLeft:
		return m, m
	case model.LegendPositionTopCenter:
		return (available.Width - legend.Width) / 2, m
	case model.LegendPositionTopRight:
		return available.Width - legend.Width - m, m
	case model.LegendPositionCenterRight:
		return available.Width - legend.Width - m, (available.Height - legend.Height) / 2
	case model.LegendPositionBottomRight:
		return available.Width - legend.Width - m, available.Height - legend.Height - m
	case model.LegendPositionBottomCenter:
		return (available.Width - legend.Width) / 2, available.Height - legend.Height - m
	case model.LegendPositionCenterLeft:
		return m, (available.Height - legend.Height) / 2
	default:
		return m, available.Height - legend.Height - m
	}
}

func measureSingleEntryH(measurer StringMeasurer, entry model.LegendEntryData) (width, height float64) {
	lw, _ := measurer.MeasureString(entry.Label)
	mw, _ := measurer.MeasureString(entry.Metric)
	titleW := max(lw, mw)

	entryW, entryH := measureEntryH(measurer, entry)

	w := max(titleW, entryW)
	h := titleHeight() + entryH

	return w, h
}

func measureSingleEntryV(measurer StringMeasurer, entry model.LegendEntryData) (width, height float64) {
	lw, _ := measurer.MeasureString(entry.Label)
	mw, _ := measurer.MeasureString(entry.Metric)
	entryW, entryH := measureEntryV(measurer, entry)

	return max(max(lw, mw), entryW), titleHeight() + entryH
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

	step := model.SwatchSize
	if entry.IsBorder {
		step += model.BorderSwatchOutlineWidth
	}

	h := float64(n) * step
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

	step := model.SwatchSize
	if entry.IsBorder {
		step += model.BorderSwatchOutlineWidth
	}

	w := float64(n) * step
	h := model.SwatchSize + model.LegendLineHeight + model.LabelGap

	return w, h
}

func measureCategoryV(measurer StringMeasurer, entry model.LegendEntryData) (width, height float64) {
	n := len(entry.Swatches)

	gap := model.SwatchGap
	if entry.IsBorder {
		gap = model.BorderSwatchOutlineWidth
	}

	w := model.SwatchSize
	h := float64(n) * (model.SwatchSize + gap)

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
	measurer := NewBasicMeasurer()
	w, _ := measureSingleEntryH(measurer, entry)

	return w
}

// MeasureEntryVContentWidth returns the swatch-column content width of
// one legend entry in vertical layout mode. Used by the Canvas layer to
// compute the center x-position for the two-line entry title.
func MeasureEntryVContentWidth(entry model.LegendEntryData) float64 {
	measurer := NewBasicMeasurer()
	w, _ := measureEntryV(measurer, entry)

	return w
}

// MeasureEntryVWidth returns the full width of one vertically oriented entry.
func MeasureEntryVWidth(entry model.LegendEntryData) float64 {
	measurer := NewBasicMeasurer()
	w, _ := measureSingleEntryV(measurer, entry)

	return w
}

// titleHeight returns the vertical space reserved for a two-line entry
// title (label + metric name), including the gap below the title block.
func titleHeight() float64 {
	return 2*model.LegendLineHeight + model.LabelGap
}

// MeasureCatSwatchColumnWidth returns the width of a single categorical
// swatch column (swatch plus label gap) for horizontal layout.
func MeasureCatSwatchColumnWidth(label string) float64 {
	measurer := NewBasicMeasurer()
	tw, _ := measurer.MeasureString(label)

	return max(model.SwatchSize, tw) + model.SwatchGap + model.LabelGap
}

// ContentOffsetV computes the horizontal offset needed to centre the
// entry content block within the legend content area for vertical layout.
// The content block includes swatches (left-aligned) and titles (centred
// over swatch columns); when titles are wider they extend beyond the swatch
// column. This function returns the shift to apply so the overall visual
// centre aligns with the legend centre.
// Note: the label sample is centred independently and excluded from this
// calculation.
func ContentOffsetV(data *model.LegendData) float64 {
	if data == nil || len(data.Entries) == 0 {
		return 0
	}

	measurer := NewBasicMeasurer()

	// Compute visual extent of entries relative to x=0.
	leftExtent := 0.0
	rightExtent := 0.0

	for _, entry := range data.Entries {
		swColW, _ := measureEntryV(measurer, entry)
		rightExtent = max(rightExtent, swColW)

		lw, _ := measurer.MeasureString(entry.Label)
		mw, _ := measurer.MeasureString(entry.Metric)
		titleW := max(lw, mw)

		titleLeft := swColW/2 - titleW/2
		titleRight := swColW/2 + titleW/2

		leftExtent = min(leftExtent, titleLeft)
		rightExtent = max(rightExtent, titleRight)
	}

	contentBoundingWidth := rightExtent - leftExtent
	legendContentSize := MeasureLegendWithArrangement(data, measurer, EntriesStacked)
	legendContentW := legendContentSize.Width - 2*model.LegendPadding

	offset := (legendContentW-contentBoundingWidth)/2 - leftExtent

	return max(0, offset)
}

// MeasureLabelSample returns equal width and height for a square or circular
// label sample.
func MeasureLabelSample(sample *model.LegendLabelSample) geometry.Size {
	return measureLabelSampleImpl(NewBasicMeasurer(), sample)
}

func measureLabelSampleImpl(measurer StringMeasurer, sample *model.LegendLabelSample) geometry.Size {
	if sample == nil || len(sample.Lines) == 0 {
		return geometry.Size{}
	}

	maxTextW := 0.0

	for _, line := range sample.Lines {
		tw, _ := measurer.MeasureString(line)
		maxTextW = max(maxTextW, tw)
	}

	textH := float64(len(sample.Lines)) * model.LegendLineHeight
	side := max(
		model.SwatchSize*2.0,
		maxTextW+2*model.LabelGap,
		textH+2*model.LabelGap,
	)

	return geometry.NewSize(side, side)
}
