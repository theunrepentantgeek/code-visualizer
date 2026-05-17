package stages

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// CollectRequestedMetrics returns the unique ordered list of metric names
// implied by size + optional fill + optional border specs.
func CollectRequestedMetrics(size metric.Name, fill *config.MetricSpec, border *config.MetricSpec) []metric.Name {
	seen := map[metric.Name]bool{size: true}
	names := []metric.Name{size}

	for _, spec := range []*config.MetricSpec{fill, border} {
		if name := spec.MetricName(); name != "" && !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}

	return names
}

// ResolveFillPalette returns the fill palette to use, consulting (in order)
// the explicit fill spec, the provider's default palette, and palette.Neutral.
func ResolveFillPalette(fill *config.MetricSpec, fillMetric metric.Name) palette.PaletteName {
	if fp := fill.PaletteName(); fp != "" {
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
	borderMetric := border.MetricName()
	if borderMetric == "" {
		return "", ""
	}

	borderPaletteName := border.PaletteName()
	if borderPaletteName == "" {
		if d, ok := provider.GetDescriptor(borderMetric); ok {
			borderPaletteName = d.DefaultPalette
		} else {
			borderPaletteName = palette.Neutral
		}
	}

	return borderMetric, borderPaletteName
}
