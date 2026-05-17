package treemap

import (
	"image/color"
	"slices"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

const (
	headerHeight = HeaderHeight
	minBorderDim = 20.0
	midBorderDim = 100.0
)

var (
	structuralBorder = color.RGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xFF}
	headerFill       = color.RGBA{R: 0x44, G: 0x44, B: 0x44, A: 0xFF}
	defaultFill      = color.RGBA{R: 0xCC, G: 0xCC, B: 0xCC, A: 0xFF}
	bgColour         = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	whiteText        = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
)

// Inks holds the fill and border Ink instances for a treemap render pass.
type Inks struct {
	Fill   canvas.Ink
	Border canvas.Ink
}

// BuildInks creates fill and border inks from metric configuration.
func BuildInks(
	root *model.Directory,
	fillMetric metric.Name,
	fillPaletteName palette.PaletteName,
	borderMetric metric.Name,
	borderPaletteName palette.PaletteName,
) Inks {
	inks := Inks{
		Border: canvas.FixedInk(structuralBorder),
	}

	inks.Fill = buildMetricInk(root, fillMetric, fillPaletteName, defaultFill)
	if borderMetric != "" {
		inks.Border = buildMetricInk(root, borderMetric, borderPaletteName, structuralBorder)
	}

	return inks
}

// buildMetricInk creates an Ink for a given metric, using the appropriate
// constructor based on the metric kind (numeric vs categorical).
func buildMetricInk(
	root *model.Directory,
	m metric.Name,
	palName palette.PaletteName,
	fallback color.RGBA,
) canvas.Ink {
	d, ok := provider.GetDescriptor(m)
	if !ok {
		return canvas.FixedInk(fallback)
	}

	pal := palette.GetPalette(palName)

	if d.Kind == metric.Quantity || d.Kind == metric.Measure {
		values := collectNumericValues(root, m)
		if len(values) == 0 {
			return canvas.FixedInk(fallback)
		}

		return canvas.NumericInk(m, values, pal)
	}

	types := collectDistinctTypes(root, m)

	return canvas.CategoricalInk(m, types, pal)
}

// metricValueForFile builds a MetricValue from a file's data for the given ink.
func metricValueForFile(file *model.File, ink canvas.Ink) canvas.MetricValue {
	if file == nil {
		return canvas.MetricValue{}
	}

	info := ink.Info()

	switch info.Kind {
	case canvas.InkNumeric:
		m := info.MetricName
		if v, ok := file.Quantity(m); ok {
			return canvas.MetricValue{Kind: metric.Quantity, Quantity: int(v)}
		}

		if v, ok := file.Measure(m); ok {
			return canvas.MetricValue{Kind: metric.Measure, Measure: v}
		}

		return canvas.MetricValue{}
	case canvas.InkCategorical:
		m := info.MetricName
		if v, ok := file.Classification(m); ok {
			return canvas.MetricValue{Kind: metric.Classification, Category: v}
		}

		return canvas.MetricValue{}
	default:
		return canvas.MetricValue{}
	}
}

func extractNumeric(f *model.File, m metric.Name) float64 {
	if v, ok := f.Quantity(m); ok {
		return float64(v)
	}

	if v, ok := f.Measure(m); ok {
		return v
	}

	return 0
}

func collectNumericValues(root *model.Directory, m metric.Name) []float64 {
	var values []float64

	model.WalkFiles(root, func(f *model.File) {
		values = append(values, extractNumeric(f, m))
	})

	return values
}

func collectDistinctTypes(root *model.Directory, m metric.Name) []string {
	seen := map[string]bool{}

	model.WalkFiles(root, func(f *model.File) {
		if v, ok := f.Classification(m); ok {
			seen[v] = true
		}
	})

	types := make([]string, 0, len(seen))
	for t := range seen {
		types = append(types, t)
	}

	slices.Sort(types)

	return types
}
