package main

import (
	"sort"

	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/palette"
	"github.com/bevan/code-visualizer/internal/provider"
	"github.com/bevan/code-visualizer/internal/render"
)

// resolveLegendOptions resolves the legend position and orientation from config.
// Empty position defaults to "bottom-right"; empty orientation is resolved from position.
func resolveLegendOptions(posStr, orientStr string) (render.LegendPosition, render.LegendOrientation) {
	pos := render.LegendPosition(posStr)
	if pos == "" {
		pos = render.LegendPositionBottomRight
	}

	orient := render.LegendOrientation(orientStr)
	if orient == "" {
		orient = render.DefaultOrientation(pos)
	}

	return pos, orient
}

// buildLegendInfo constructs a LegendInfo from the resolved CLI/config options
// and the scanned model root. Returns nil if legend is disabled.
func buildLegendInfo(
	position render.LegendPosition,
	orientation render.LegendOrientation,
	fillMetric metric.Name,
	fillPaletteName palette.PaletteName,
	borderMetric metric.Name,
	borderPaletteName palette.PaletteName,
	sizeMetric metric.Name,
	root *model.Directory,
) *render.LegendInfo {
	if position == render.LegendPositionNone {
		return nil
	}

	if orientation == "" {
		orientation = render.DefaultOrientation(position)
	}

	entries := collectLegendEntries(
		fillMetric, fillPaletteName,
		borderMetric, borderPaletteName,
		sizeMetric, root,
	)

	if len(entries) == 0 {
		return nil
	}

	return &render.LegendInfo{
		Position:    position,
		Orientation: orientation,
		Entries:     entries,
	}
}

// collectLegendEntries gathers legend entries for fill, border, and size metrics.
func collectLegendEntries(
	fillMetric metric.Name,
	fillPaletteName palette.PaletteName,
	borderMetric metric.Name,
	borderPaletteName palette.PaletteName,
	sizeMetric metric.Name,
	root *model.Directory,
) []render.LegendEntry {
	var entries []render.LegendEntry

	if fillMetric != "" {
		entry := buildLegendEntry("Fill", fillMetric, fillPaletteName, root)
		if entry != nil {
			entries = append(entries, *entry)
		}
	}

	if borderMetric != "" {
		entry := buildLegendEntry("Border", borderMetric, borderPaletteName, root)
		if entry != nil {
			entries = append(entries, *entry)
		}
	}

	if sizeMetric != "" && sizeMetric != fillMetric {
		entries = append(entries, render.LegendEntry{
			Role:       "Size",
			MetricName: string(sizeMetric),
			Kind:       metric.Quantity,
		})
	}

	return entries
}

// buildLegendEntry creates a legend entry for a colour-mapped metric.
func buildLegendEntry(
	role string,
	metricName metric.Name,
	paletteName palette.PaletteName,
	root *model.Directory,
) *render.LegendEntry {
	p, ok := provider.Get(metricName)
	if !ok {
		return nil
	}

	pal := palette.GetPalette(paletteName)

	entry := render.LegendEntry{
		Role:       role,
		MetricName: string(metricName),
		Kind:       p.Kind(),
		Palette:    pal,
	}

	if p.Kind() == metric.Quantity || p.Kind() == metric.Measure {
		values := collectNumericValues(root, metricName)
		if len(values) > 0 {
			buckets := metric.ComputeBuckets(values, len(pal.Colours))
			entry.Buckets = &buckets
		}
	} else {
		types := collectDistinctTypes(root, metricName)
		mapper := palette.NewCategoricalMapper(types, pal)
		entry.Categories = buildCategorySwatches(types, mapper)
	}

	return &entry
}

// buildCategorySwatches pairs category labels with their mapped colours.
func buildCategorySwatches(
	types []string,
	mapper *palette.CategoricalMapper,
) []render.CategorySwatch {
	sort.Strings(types)

	swatches := make([]render.CategorySwatch, len(types))
	for i, t := range types {
		swatches[i] = render.CategorySwatch{
			Label:  t,
			Colour: mapper.Map(t),
		}
	}

	return swatches
}
