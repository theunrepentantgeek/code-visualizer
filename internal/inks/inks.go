package inks

import (
	"image/color"
	"slices"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// BuildMetricInk creates an Ink for a given metric, using the appropriate
// constructor based on the metric kind (numeric vs categorical). Returns a
// fixed-colour ink when the metric is unknown or when no values are present.
func BuildMetricInk(
	root *model.Directory,
	d provider.BaseMetricDescriptor,
	palName palette.PaletteName,
	fallback color.RGBA,
) Ink {
	if d.Name == "" {
		return FixedInk(fallback)
	}

	pal := palette.GetPalette(palName)

	if d.Kind == metric.Quantity || d.Kind == metric.Measure {
		values := CollectNumericValues(root, d.Name)
		if len(values) == 0 {
			return FixedInk(fallback)
		}

		return NumericInk(d.Name, values, pal)
	}

	types := CollectDistinctTypes(root, d.Name)

	return CategoricalInk(d.Name, types, pal)
}

// MetricValueForFile builds a MetricValue from a file's data for the given
// ink. Returns the zero MetricValue when file is nil, when the ink is fixed,
// or when the file has no value for the ink's metric.
func MetricValueForFile(file *model.File, ink Ink) MetricValue {
	if file == nil {
		return MetricValue{}
	}

	info := ink.Info()

	switch info.Kind {
	case KindNumeric:
		m := info.MetricName
		if v, ok := file.Quantity(m); ok {
			return MetricValue{Kind: metric.Quantity, Quantity: int(v)}
		}

		if v, ok := file.Measure(m); ok {
			return MetricValue{Kind: metric.Measure, Measure: v}
		}

		return MetricValue{}
	case KindCategorical:
		m := info.MetricName
		if v, ok := file.Classification(m); ok {
			return MetricValue{Kind: metric.Classification, Category: v}
		}

		return MetricValue{}
	default:
		return MetricValue{}
	}
}

// CollectNumericValues walks the directory tree and returns every file's
// numeric value for metric m (quantity preferred, then measure).
func CollectNumericValues(root *model.Directory, m metric.Name) []float64 {
	values := make([]float64, 0, root.AllFileCount)

	model.WalkFiles(root, func(f *model.File) {
		values = append(values, extractNumeric(f, m))
	})

	return values
}

// CollectDistinctTypes returns the sorted distinct classification values
// observed for metric m across all files under root.
func CollectDistinctTypes(root *model.Directory, m metric.Name) []string {
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

func extractNumeric(f *model.File, m metric.Name) float64 {
	if v, ok := f.Quantity(m); ok {
		return float64(v)
	}

	if v, ok := f.Measure(m); ok {
		return v
	}

	return 0
}
