package scatter

import (
	"image/color"
	"slices"

	pkginks "github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
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

// Inks holds the fill and border inks for scatter points.
type Inks struct {
	Fill            pkginks.Ink
	Border          pkginks.Ink
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
	inks := Inks{
		Fill:   buildMetricInk(dataset.Files(), requested, fillMetric, fillPaletteName, scatterDefaultFill),
		Border: pkginks.FixedInk(scatterDefaultBorder),
	}

	if borderMetric != "" {
		inks.Border = buildMetricInk(dataset.Files(), requested, borderMetric, borderPaletteName, scatterDefaultBorder)
		inks.HasBorderMetric = true
	}

	return inks
}

func buildMetricInk(
	files []*model.File,
	requested stages.RequestedMetrics,
	name metric.Name,
	paletteName palette.PaletteName,
	fallback color.RGBA,
) pkginks.Ink {
	if name == "" {
		return pkginks.FixedInk(fallback)
	}

	descriptor, ok := requested.DescriptorFor(name)
	if !ok {
		return pkginks.FixedInk(fallback)
	}

	pal := palette.GetPalette(paletteName)

	if descriptor.Kind == metric.Quantity || descriptor.Kind == metric.Measure {
		return buildNumericInk(files, name, pal, fallback)
	}

	return buildCategoricalInk(files, name, pal, fallback)
}

func buildNumericInk(
	files []*model.File,
	name metric.Name,
	pal palette.ColourPalette,
	fallback color.RGBA,
) pkginks.Ink {
	values := make([]float64, 0, len(files))
	for _, file := range files {
		if value, ok := numericValueForFile(file, name); ok {
			values = append(values, value)
		}
	}

	if len(values) == 0 {
		return pkginks.FixedInk(fallback)
	}

	return pkginks.NumericInk(name, values, pal)
}

func buildCategoricalInk(
	files []*model.File,
	name metric.Name,
	pal palette.ColourPalette,
	fallback color.RGBA,
) pkginks.Ink {
	categories := uniqueCategories(files, name)
	if len(categories) == 0 {
		return pkginks.FixedInk(fallback)
	}

	slices.Sort(categories)

	return pkginks.CategoricalInk(name, categories, pal)
}

func uniqueCategories(files []*model.File, name metric.Name) []string {
	seen := map[string]bool{}
	categories := make([]string, 0, len(files))

	for _, file := range files {
		if value, ok := file.Classification(name); ok && !seen[value] {
			seen[value] = true
			categories = append(categories, value)
		}
	}

	return categories
}
