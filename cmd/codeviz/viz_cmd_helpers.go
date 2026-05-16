package main

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// resolveFillPalette determines the fill palette from config or provider
// defaults.
func resolveFillPalette(fill *config.MetricSpec, fillMetric metric.Name) palette.PaletteName {
	if fp := specPalette(fill); fp != "" {
		return fp
	}

	if d, ok := provider.GetDescriptor(fillMetric); ok {
		return d.DefaultPalette
	}

	return palette.Neutral
}

// resolveBorderMetricAndPalette determines the effective border metric and
// palette from config or provider defaults.
func resolveBorderMetricAndPalette(
	border *config.MetricSpec,
) (metric.Name, palette.PaletteName) {
	borderMetric := specMetric(border)
	if borderMetric == "" {
		return "", ""
	}

	borderPaletteName := specPalette(border)
	if borderPaletteName == "" {
		if d, ok := provider.GetDescriptor(borderMetric); ok {
			borderPaletteName = d.DefaultPalette
		} else {
			borderPaletteName = palette.Neutral
		}
	}

	return borderMetric, borderPaletteName
}
