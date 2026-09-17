package stages

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/viz"
)

// CollectRequestedMetrics returns the classified set of metrics
// implied by size + optional fill + optional border specs.
func CollectRequestedMetrics(size metric.Name, specs ...*config.MetricSpec) RequestedMetrics {
	seen := map[metric.Name]bool{size: true}
	names := []metric.Name{size}

	for _, spec := range specs {
		if name := spec.MetricName(); name != "" && !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}

	return ClassifyRequestedMetrics(names, metric.LevelDirectory)
}

// CollectRequestedMetricNames returns the directory-level requested metrics
// implied by already-resolved metric names.
func CollectRequestedMetricNames(names ...metric.Name) RequestedMetrics {
	seen := make(map[metric.Name]bool, len(names))
	distinct := make([]metric.Name, 0, len(names))

	for _, name := range names {
		if name != "" && !seen[name] {
			seen[name] = true
			distinct = append(distinct, name)
		}
	}

	return ClassifyRequestedMetrics(distinct, metric.LevelDirectory)
}

// ResolveColourEncoding returns the effective metric and palette for a colour
// channel. The fallback metric is used when the config does not select one.
func ResolveColourEncoding(spec *config.MetricSpec, fallback metric.Name) viz.ColourEncoding {
	name := spec.MetricName()
	if name == "" {
		name = fallback
	}

	if name == "" {
		return viz.ColourEncoding{}
	}

	return viz.ColourEncoding{
		Metric:  name,
		Palette: ResolveFillPalette(spec, name),
	}
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
