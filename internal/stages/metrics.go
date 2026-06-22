package stages

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// CollectRequestedMetrics returns the classified set of metrics
// implied by size + optional fill + optional border specs.
func CollectRequestedMetrics(size metric.Name, fill *config.MetricSpec, border *config.MetricSpec) RequestedMetrics {
	seen := map[metric.Name]bool{size: true}
	names := []metric.Name{size}

	for _, spec := range []*config.MetricSpec{fill, border} {
		if name := spec.MetricName(); name != "" && !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}

	return ClassifyRequestedMetrics(names, metric.LevelDirectory)
}

// ResolveFillPalette returns the fill palette to use, consulting (in order)
// the explicit fill spec, the provider's default palette, and palette.Neutral.
// For expression metrics (e.g. "commit-count.mean"), the base metric's default
// palette is used so aggregations inherit meaningful colour schemes.
func ResolveFillPalette(fill *config.MetricSpec, fillMetric metric.Name) palette.PaletteName {
	if fp := fill.PaletteName(); fp != "" {
		return fp
	}

	if d, ok := provider.GetBase(fillMetric); ok {
		return d.DefaultPalette
	}

	// For expression metrics like "commit-count.mean", inherit the base metric's
	// default palette so aggregations get meaningful colour schemes automatically.
	if expr, err := metric.ParseExpression(string(fillMetric)); err == nil {
		if d, ok := provider.GetBase(expr.Base); ok {
			return d.DefaultPalette
		}
	}

	return palette.Neutral
}

// ResolveBorderMetricAndPalette returns the effective border metric and
// palette name, or ("", "") when no border is configured. For expression
// metrics (e.g. "commit-count.mean"), the base metric's default palette is
// used so aggregations inherit meaningful colour schemes.
func ResolveBorderMetricAndPalette(border *config.MetricSpec) (metric.Name, palette.PaletteName) {
	borderMetric := border.MetricName()
	if borderMetric == "" {
		return "", ""
	}

	borderPaletteName := border.PaletteName()
	if borderPaletteName == "" {
		if d, ok := provider.GetBase(borderMetric); ok {
			borderPaletteName = d.DefaultPalette
		} else {
			// For expression metrics like "commit-count.mean", inherit the base
			// metric's default palette so aggregations get meaningful colour schemes.
			if expr, err := metric.ParseExpression(string(borderMetric)); err == nil {
				if d, ok := provider.GetBase(expr.Base); ok {
					borderPaletteName = d.DefaultPalette
				} else {
					borderPaletteName = palette.Neutral
				}
			} else {
				borderPaletteName = palette.Neutral
			}
		}
	}

	return borderMetric, borderPaletteName
}
