package legend

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// ResolveOptions resolves legend position and orientation from raw strings.
// Empty position defaults to "bottom-right"; empty orientation is derived
// from the resolved position.
func ResolveOptions(posStr, orientStr string) (model.LegendPosition, model.LegendOrientation) {
	pos := model.LegendPosition(posStr)
	if pos == "" {
		pos = model.LegendPositionBottomRight
	}

	orient := model.LegendOrientation(orientStr)
	if orient == "" {
		orient = DefaultOrientation(pos)
	}

	return pos, orient
}

// Build constructs a Config from resolved options and the pre-built Ink
// objects used for rendering. Returns nil if the legend is disabled
// ("none") or no entries would be produced.
func Build(
	position model.LegendPosition,
	orientation model.LegendOrientation,
	fillInk inks.Ink,
	fillMetric metric.Name,
	borderInk inks.Ink,
	borderMetric metric.Name,
	sizeMetric metric.Name,
) *Config {
	if position == model.LegendPositionNone {
		return nil
	}

	if orientation == "" {
		orientation = DefaultOrientation(position)
	}

	var entries []Entry

	if fillMetric != "" {
		entries = append(entries, Entry{
			Role:       RoleFill,
			MetricName: string(fillMetric),
			Ink:        fillInk,
		})
	}

	if borderMetric != "" {
		entries = append(entries, Entry{
			Role:       RoleBorder,
			MetricName: string(borderMetric),
			Ink:        borderInk,
		})
	}

	if sizeMetric != "" && sizeMetric != fillMetric {
		entries = append(entries, Entry{
			Role:       RoleSize,
			MetricName: string(sizeMetric),
			Ink:        inks.FixedInk(palette.White),
		})
	}

	if len(entries) == 0 {
		return nil
	}

	return &Config{
		Position:    position,
		Orientation: orientation,
		Entries:     entries,
	}
}
