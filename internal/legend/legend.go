// Package legend constructs canvas.LegendConfig values from resolved
// visualization options and reserves canvas space for legend rendering.
// It is reusable across all visualization types.
package legend

import (
	"image/color"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
)

// white is the colour used for FixedInk in size-only entries.
var white = color.RGBA{R: 255, G: 255, B: 255, A: 255} //nolint:gochecknoglobals // shared colour constant

// ResolveOptions resolves legend position and orientation from raw strings.
// Empty position defaults to "bottom-right"; empty orientation is derived
// from the resolved position.
func ResolveOptions(posStr, orientStr string) (canvas.LegendPosition, canvas.LegendOrientation) {
	pos := canvas.LegendPosition(posStr)
	if pos == "" {
		pos = canvas.LegendPositionBottomRight
	}

	orient := canvas.LegendOrientation(orientStr)
	if orient == "" {
		orient = canvas.DefaultOrientation(pos)
	}

	return pos, orient
}

// Build constructs a LegendConfig from resolved options and the pre-built
// Ink objects used for rendering. Returns nil if the legend is disabled
// ("none") or no entries would be produced.
func Build(
	position canvas.LegendPosition,
	orientation canvas.LegendOrientation,
	fillInk canvas.Ink,
	fillMetric metric.Name,
	borderInk canvas.Ink,
	borderMetric metric.Name,
	sizeMetric metric.Name,
) *canvas.LegendConfig {
	if position == canvas.LegendPositionNone {
		return nil
	}

	if orientation == "" {
		orientation = canvas.DefaultOrientation(position)
	}

	var entries []canvas.LegendEntry

	if fillMetric != "" {
		entries = append(entries, canvas.LegendEntry{
			Role:       canvas.LegendRoleFill,
			MetricName: string(fillMetric),
			Ink:        fillInk,
		})
	}

	if borderMetric != "" {
		entries = append(entries, canvas.LegendEntry{
			Role:       canvas.LegendRoleBorder,
			MetricName: string(borderMetric),
			Ink:        borderInk,
		})
	}

	if sizeMetric != "" && sizeMetric != fillMetric {
		entries = append(entries, canvas.LegendEntry{
			Role:       canvas.LegendRoleSize,
			MetricName: string(sizeMetric),
			Ink:        canvas.FixedInk(white),
		})
	}

	if len(entries) == 0 {
		return nil
	}

	return &canvas.LegendConfig{
		Position:    position,
		Orientation: orientation,
		Entries:     entries,
	}
}
