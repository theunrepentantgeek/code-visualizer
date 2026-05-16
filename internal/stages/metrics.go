package stages

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// SpecMetric returns the metric name from a *MetricSpec, or "" if nil.
func SpecMetric(s *config.MetricSpec) metric.Name {
	if s == nil {
		return ""
	}

	return s.Metric
}

// SpecPalette returns the palette name from a *MetricSpec, or "" if nil.
func SpecPalette(s *config.MetricSpec) palette.PaletteName {
	if s == nil {
		return ""
	}

	return s.Palette
}

// CollectRequestedMetrics returns the unique ordered list of metric names
// implied by size + optional fill + optional border specs.
func CollectRequestedMetrics(size metric.Name, fill, border *config.MetricSpec) []metric.Name {
	seen := map[metric.Name]bool{size: true}
	names := []metric.Name{size}

	for _, spec := range []*config.MetricSpec{fill, border} {
		if spec != nil && spec.Metric != "" {
			if !seen[spec.Metric] {
				seen[spec.Metric] = true
				names = append(names, spec.Metric)
			}
		}
	}

	return names
}

// ResolveFillPalette returns the fill palette to use, consulting (in order)
// the explicit fill spec, the provider's default palette, and palette.Neutral.
func ResolveFillPalette(fill *config.MetricSpec, fillMetric metric.Name) palette.PaletteName {
	if fp := SpecPalette(fill); fp != "" {
		return fp
	}

	if d, ok := provider.GetDescriptor(fillMetric); ok {
		return d.DefaultPalette
	}

	return palette.Neutral
}

// ResolveBorderMetricAndPalette returns the effective border metric and
// palette name, or ("", "") when no border is configured.
func ResolveBorderMetricAndPalette(border *config.MetricSpec) (metric.Name, palette.PaletteName) {
	borderMetric := SpecMetric(border)
	if borderMetric == "" {
		return "", ""
	}

	borderPaletteName := SpecPalette(border)
	if borderPaletteName == "" {
		if d, ok := provider.GetDescriptor(borderMetric); ok {
			borderPaletteName = d.DefaultPalette
		} else {
			borderPaletteName = palette.Neutral
		}
	}

	return borderMetric, borderPaletteName
}
