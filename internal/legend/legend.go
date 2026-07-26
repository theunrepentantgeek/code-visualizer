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

// Builder collects the standard legend inputs and any visualization-specific entries.
type Builder struct {
	Position          model.LegendPosition
	Orientation       model.LegendOrientation
	FillInk           inks.Ink
	FillMetric        metric.Name
	BorderInk         inks.Ink
	BorderMetric      metric.Name
	SizeMetric        metric.Name
	AdditionalEntries []Entry
}

// Build constructs a Config from the builder. Returns nil if the legend is
// disabled ("none") or no entries would be produced.
func (b Builder) Build() *Config {
	if b.Position == model.LegendPositionNone {
		return nil
	}

	orientation := b.Orientation
	if orientation == "" {
		orientation = DefaultOrientation(b.Position)
	}

	entries := make([]Entry, 0, 3+len(b.AdditionalEntries))
	if b.FillMetric != "" {
		entries = append(entries, Entry{
			Role: RoleFill, MetricName: string(b.FillMetric), Ink: b.FillInk,
		})
	}

	if b.BorderMetric != "" {
		entries = append(entries, Entry{
			Role: RoleBorder, MetricName: string(b.BorderMetric), Ink: b.BorderInk,
		})
	}

	if b.SizeMetric != "" && b.SizeMetric != b.FillMetric {
		entries = append(entries, Entry{
			Role: RoleSize, MetricName: string(b.SizeMetric), Ink: inks.FixedInk(palette.White),
		})
	}

	entries = append(entries, b.AdditionalEntries...)
	if len(entries) == 0 {
		return nil
	}

	return &Config{
		Position:    b.Position,
		Orientation: orientation,
		Entries:     entries,
	}
}
