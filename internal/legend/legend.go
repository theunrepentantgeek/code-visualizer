package legend

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
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

// Build constructs a Config from resolved options and explicitly ordered
// pre-built legend entries. Returns nil if the legend is disabled ("none") or
// no entries would be produced.
func Build(
	position model.LegendPosition,
	orientation model.LegendOrientation,
	entries []Entry,
) *Config {
	if position == model.LegendPositionNone {
		return nil
	}

	if orientation == "" {
		orientation = DefaultOrientation(position)
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
