package inks

import (
	"slices"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/legendlayout"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// LegendData extracts the legend entry kind and swatch list from an Ink.
// Thin shim over the polymorphic Ink.LegendData method, retained so callers
// can keep the free-function form.
func LegendData(ink Ink) (model.LegendEntryKind, []model.LegendSwatch) {
	return ink.LegendData()
}

// LegendData reports that fixed inks contribute no legend swatches.
func (*fixedInk) LegendData() (model.LegendEntryKind, []model.LegendSwatch) {
	return model.LegendEntryNumeric, nil
}

// LegendData produces the N+1 bucket swatches for a numeric ink.
// N boundaries separate two buckets each, so N boundaries produce N+1 buckets.
// Zero boundaries (e.g. empty dataset) still yields a single labelless swatch
// drawn in the first palette colour.
func (ink *numericInk) LegendData() (model.LegendEntryKind, []model.LegendSwatch) {
	if len(ink.pal.Colours) == 0 {
		return model.LegendEntryNumeric, nil
	}

	boundaries := ink.boundaries.Boundaries
	n := len(boundaries) + 1

	swatches := make([]model.LegendSwatch, n)

	for i := range n {
		colour := palette.MapNumericToColour(i, n, ink.pal)

		var label string
		if i < len(boundaries) {
			label = legendlayout.FormatBreakpoint(boundaries[i])
		}

		swatches[i] = model.LegendSwatch{
			Colour: colour,
			Label:  label,
		}
	}

	return model.LegendEntryNumeric, swatches
}

// LegendData produces one swatch per distinct category, sorted alphabetically.
func (ink *categoricalInk) LegendData() (model.LegendEntryKind, []model.LegendSwatch) {
	if len(ink.categories) == 0 || len(ink.pal.Colours) == 0 {
		return model.LegendEntryCategorical, nil
	}

	sorted := make([]string, len(ink.categories))
	copy(sorted, ink.categories)
	slices.Sort(sorted)

	swatches := make([]model.LegendSwatch, len(sorted))

	for i, cat := range sorted {
		swatches[i] = model.LegendSwatch{
			Colour: ink.catMapper.Map(cat),
			Label:  cat,
		}
	}

	return model.LegendEntryCategorical, swatches
}
