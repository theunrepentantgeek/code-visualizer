package inks

import (
	"slices"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/legendlayout"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// LegendData extracts the legend entry kind and swatch list from an Ink.
// Returns LegendEntryNumeric / nil for fixed inks (no meaningful swatch data).
func LegendData(ink Ink) (model.LegendEntryKind, []model.LegendSwatch) {
	switch ink.Info().Kind {
	case KindNumeric:
		return model.LegendEntryNumeric, numericSwatches(ink)
	case KindCategorical:
		return model.LegendEntryCategorical, categoricalSwatches(ink)
	default:
		return model.LegendEntryNumeric, nil
	}
}

func numericSwatches(ink Ink) []model.LegendSwatch {
	boundaries := ink.Boundaries()
	pal := ink.Palette()

	if len(boundaries) == 0 || len(pal.Colours) == 0 {
		return nil
	}

	// NumBuckets is one more than the number of boundaries: each boundary
	// separates two buckets, so N boundaries produce N+1 buckets. The final
	// (above-last-boundary) bucket has no boundary label.
	n := len(boundaries) + 1

	swatches := make([]model.LegendSwatch, n)

	for i := range n {
		colour := palette.MapNumericToColour(i, n, pal)

		var label string
		if i < len(boundaries) {
			label = legendlayout.FormatBreakpoint(boundaries[i])
		}

		swatches[i] = model.LegendSwatch{
			Colour: colour,
			Label:  label,
		}
	}

	return swatches
}

func categoricalSwatches(ink Ink) []model.LegendSwatch {
	categories := ink.Categories()
	pal := ink.Palette()

	if len(categories) == 0 || len(pal.Colours) == 0 {
		return nil
	}

	mapper := palette.NewCategoricalMapper(categories, pal)

	sorted := make([]string, len(categories))
	copy(sorted, categories)
	slices.Sort(sorted)

	swatches := make([]model.LegendSwatch, len(sorted))

	for i, cat := range sorted {
		swatches[i] = model.LegendSwatch{
			Colour: mapper.Map(cat),
			Label:  cat,
		}
	}

	return swatches
}
