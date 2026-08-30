package scatter

import (
	"image/color"
	"maps"
	"slices"

	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

var (
	scatterDefaultFill   = color.RGBA{R: 0xCC, G: 0xCC, B: 0xCC, A: 0xFF}
	scatterDefaultBorder = color.RGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xFF}
	scatterAxisColour    = color.RGBA{R: 0x77, G: 0x77, B: 0x77, A: 0xFF}
	scatterGridColour    = color.RGBA{R: 0xDD, G: 0xDD, B: 0xDD, A: 0xFF}
	scatterLabelColour   = color.RGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xFF}
	scatterBgColour      = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
)

// Inks pairs the fill and border inks for scatter points (via embedded
// inks.ShapeInks) plus a flag recording whether the border encodes a metric.
type Inks struct {
	inks.ShapeInks
	HasBorderMetric bool
}

// BuildInks creates point inks from the plotted dataset.
func BuildInks(
	dataset Dataset,
	requested stages.RequestedMetrics,
	fillMetric metric.Name,
	fillPaletteName palette.PaletteName,
	borderMetric metric.Name,
	borderPaletteName palette.PaletteName,
) Inks {
	is := Inks{
		ShapeInks: inks.ShapeInks{
			Fill:   buildMetricInk(dataset.metricSources(), requested, fillMetric, fillPaletteName, scatterDefaultFill),
			Border: inks.FixedInk(scatterDefaultBorder),
		},
	}

	if borderMetric != "" {
		is.Border = buildMetricInk(
			dataset.metricSources(),
			requested,
			borderMetric,
			borderPaletteName,
			scatterDefaultBorder,
		)
		is.HasBorderMetric = true
	}

	return is
}

func buildMetricInk[T metricSource](
	sources []T,
	requested stages.RequestedMetrics,
	name metric.Name,
	paletteName palette.PaletteName,
	fallback color.RGBA,
) inks.Ink {
	if name == "" {
		return inks.FixedInk(fallback)
	}

	descriptor, ok := requested.DescriptorFor(name)
	if !ok {
		return inks.FixedInk(fallback)
	}

	pal := palette.GetPalette(paletteName)

	if descriptor.Kind == metric.Quantity || descriptor.Kind == metric.Measure {
		return buildNumericInk(sources, name, pal, fallback)
	}

	return buildCategoricalInk(sources, name, pal, fallback)
}

func buildNumericInk[T metricSource](
	sources []T,
	name metric.Name,
	pal palette.ColourPalette,
	fallback color.RGBA,
) inks.Ink {
	values := make([]float64, 0, len(sources))
	for _, source := range sources {
		if value, ok := numericValueForSource(source, name); ok {
			values = append(values, value)
		}
	}

	if len(values) == 0 {
		return inks.FixedInk(fallback)
	}

	return inks.NumericInk(name, values, pal)
}

func buildCategoricalInk[T metricSource](
	sources []T,
	name metric.Name,
	pal palette.ColourPalette,
	fallback color.RGBA,
) inks.Ink {
	categories := uniqueCategories(sources, name)
	if len(categories) == 0 {
		return inks.FixedInk(fallback)
	}

	return inks.CategoricalInk(name, categories, pal)
}

func uniqueCategories[T metricSource](sources []T, name metric.Name) []string {
	seen := map[string]struct{}{}

	for _, source := range sources {
		if value, ok := source.Classification(name); ok {
			seen[value] = struct{}{}
		}
	}

	return slices.Sorted(maps.Keys(seen))
}

func numericValueForSource(source metricSource, name metric.Name) (float64, bool) {
	if value, ok := source.Quantity(name); ok {
		return float64(value), true
	}

	if value, ok := source.Measure(name); ok {
		return value, true
	}

	return 0, false
}
