package canvas

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/legendlayout"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
)

// LegendRole identifies what visual property a legend entry describes.
type LegendRole string

const (
	LegendRoleFill   LegendRole = "Fill"
	LegendRoleBorder LegendRole = "Border"
	LegendRoleSize   LegendRole = "Size"
)

// LegendEntry describes one metric shown in the legend.
type LegendEntry struct {
	Role       LegendRole
	MetricName string
	Ink        Ink
}

// LegendConfig holds everything needed to render a legend.
type LegendConfig struct {
	Position    model.LegendPosition
	Orientation model.LegendOrientation
	Entries     []LegendEntry
}

// DefaultOrientation returns the default orientation for a given position.
// Top-center and bottom-center default to horizontal; all others to vertical.
func DefaultOrientation(pos model.LegendPosition) model.LegendOrientation {
	switch pos {
	case model.LegendPositionTopCenter, model.LegendPositionBottomCenter:
		return model.LegendOrientationHorizontal
	default:
		return model.LegendOrientationVertical
	}
}

// ReserveSpace computes the width and height reductions needed to reserve
// space for the legend within the canvas. Returns zeros if the legend is
// disabled or has no entries.
func (lc *LegendConfig) ReserveSpace() (widthReduction, heightReduction float64) {
	data := lc.toLegendData()

	return legendlayout.ReserveSpace(data, legendlayout.NewBasicMeasurer())
}

// toLegendData converts the canvas-facing LegendConfig to the backend-facing
// LegendData. Returns nil if the legend is disabled or has no entries.
func (lc *LegendConfig) toLegendData() *model.LegendData {
	if lc == nil || lc.Position == model.LegendPositionNone || len(lc.Entries) == 0 {
		return nil
	}

	entries := make([]model.LegendEntryData, len(lc.Entries))

	for i, e := range lc.Entries {
		entries[i] = model.LegendEntryData{
			Title:    string(e.Role) + ": " + e.MetricName,
			Kind:     e.Ink.legendEntryKind(),
			Swatches: e.Ink.legendSwatches(),
			IsBorder: e.Role == LegendRoleBorder,
		}
	}

	orient := lc.Orientation
	if orient == "" {
		orient = DefaultOrientation(lc.Position)
	}

	return &model.LegendData{
		Position:    lc.Position,
		Orientation: orient,
		Entries:     entries,
	}
}
