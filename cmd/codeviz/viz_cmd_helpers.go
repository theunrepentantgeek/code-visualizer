package main

import (
	"log/slog"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/scan"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// filterBinaryFiles removes binary files from the tree unless includeBinary is true.
// Binary files are excluded by default because this is a code visualization tool;
// use --include-binary-files to include them.
func filterBinaryFiles(root *model.Directory) error {
	beforeCount, _ := countAll(root)
	filtered := scan.FilterBinaryFiles(root)
	afterCount, _ := countAll(filtered)
	excluded := beforeCount - afterCount
	slog.Debug("binary file filter", "excluded", excluded, "remaining", afterCount)

	if afterCount == 0 {
		return &stages.NoFilesAfterFilterError{
			Msg: stages.NoFilesAfterFilterMsg,
		}
	}

	// Update root in place — avoid struct copy which would copy the mutex.
	root.Files = filtered.Files
	root.Dirs = filtered.Dirs

	return nil
}

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
