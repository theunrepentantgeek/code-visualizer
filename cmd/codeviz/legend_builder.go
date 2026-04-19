package main

import (
	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/palette"
	"github.com/bevan/code-visualizer/internal/provider"
	"github.com/bevan/code-visualizer/internal/render"
)

// buildLegendRow constructs a LegendRow for a metric/palette combination
// by inspecting the model tree. Returns nil if the metric is empty or unknown.
func buildLegendRow(
	root *model.Directory,
	metricName metric.Name,
	paletteName palette.PaletteName,
) *render.LegendRow {
	if metricName == "" {
		return nil
	}

	p, ok := provider.Get(metricName)
	if !ok {
		return nil
	}

	pal := palette.GetPalette(paletteName)

	if p.Kind() == metric.Quantity || p.Kind() == metric.Measure {
		values := collectNumericValues(root, metricName)
		if len(values) == 0 {
			return nil
		}

		buckets := metric.ComputeBuckets(values, len(pal.Colours))
		numBuckets := len(buckets.Boundaries) + 1
		row := render.BuildNumericLegendRow(string(metricName), p.Kind(), buckets, numBuckets, pal)

		return &row
	}

	// Classification
	types := collectDistinctTypes(root, metricName)
	if len(types) == 0 {
		return nil
	}

	row := render.BuildCategoricalLegendRow(string(metricName), types, pal)

	return &row
}

// buildLegendInfo assembles a LegendInfo from fill and border legend rows,
// respecting the noLegend flag. Returns nil if the legend should be suppressed.
func buildLegendInfo(noLegend *bool, rows ...*render.LegendRow) *render.LegendInfo {
	if noLegend != nil && *noLegend {
		return nil
	}

	var legendRows []render.LegendRow

	for _, r := range rows {
		if r != nil {
			legendRows = append(legendRows, *r)
		}
	}

	if len(legendRows) == 0 {
		return nil
	}

	return &render.LegendInfo{Rows: legendRows}
}
