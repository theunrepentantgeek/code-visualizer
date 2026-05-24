package scatter

import (
	"image/color"
	"slices"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
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
	Fill            canvas.Ink
	Border          canvas.Ink
	HasBorderMetric bool
}

// BuildInks creates point inks from the plotted dataset.
func BuildInks(
	dataset Dataset,
	fillMetric metric.Name,
	fillPaletteName palette.PaletteName,
	borderMetric metric.Name,
	borderPaletteName palette.PaletteName,
) Inks {
	inks := Inks{
		Fill:   buildMetricInk(dataset.Files(), fillMetric, fillPaletteName, scatterDefaultFill),
		Border: canvas.FixedInk(scatterDefaultBorder),
	}

	if borderMetric != "" {
		inks.Border = buildMetricInk(dataset.Files(), borderMetric, borderPaletteName, scatterDefaultBorder)
		inks.HasBorderMetric = true
	}

	return inks
}

func buildMetricInk(
	files []*model.File,
	name metric.Name,
	paletteName palette.PaletteName,
	fallback color.RGBA,
) canvas.Ink {
	if name == "" {
		return canvas.FixedInk(fallback)
	}

	descriptor, ok := provider.GetDescriptor(name)
	if !ok {
		return canvas.FixedInk(fallback)
	}

	pal := palette.GetPalette(paletteName)
	if descriptor.Kind == metric.Quantity || descriptor.Kind == metric.Measure {
		values := make([]float64, 0, len(files))
		for _, file := range files {
			if value, ok := numericValueForFile(file, name); ok {
				values = append(values, value)
			}
		}
		if len(values) == 0 {
			return canvas.FixedInk(fallback)
		}

		return canvas.NumericInk(name, values, pal)
	}

	seen := map[string]bool{}
	categories := make([]string, 0, len(files))
	for _, file := range files {
		if value, ok := file.Classification(name); ok && !seen[value] {
			seen[value] = true
			categories = append(categories, value)
		}
	}
	if len(categories) == 0 {
		return canvas.FixedInk(fallback)
	}

	slices.Sort(categories)

	return canvas.CategoricalInk(name, categories, pal)
}
