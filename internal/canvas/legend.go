package canvas

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/legendlayout"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
)

// LegendPosition specifies where the legend is placed on the canvas.
type LegendPosition string

const (
	LegendPositionNone         LegendPosition = "none"
	LegendPositionTopLeft      LegendPosition = "top-left"
	LegendPositionTopCenter    LegendPosition = "top-center"
	LegendPositionTopRight     LegendPosition = "top-right"
	LegendPositionCenterRight  LegendPosition = "center-right"
	LegendPositionBottomRight  LegendPosition = "bottom-right"
	LegendPositionBottomCenter LegendPosition = "bottom-center"
	LegendPositionBottomLeft   LegendPosition = "bottom-left"
	LegendPositionCenterLeft   LegendPosition = "center-left"
)

// LegendOrientation controls whether swatches are stacked vertically
// or laid out horizontally.
type LegendOrientation string

const (
	LegendOrientationVertical   LegendOrientation = "vertical"
	LegendOrientationHorizontal LegendOrientation = "horizontal"
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
	Position    LegendPosition
	Orientation LegendOrientation
	Entries     []LegendEntry
}

// DefaultOrientation returns the default orientation for a given position.
// Top-center and bottom-center default to horizontal; all others to vertical.
func DefaultOrientation(pos LegendPosition) LegendOrientation {
	switch pos {
	case LegendPositionTopCenter, LegendPositionBottomCenter:
		return LegendOrientationHorizontal
	default:
		return LegendOrientationVertical
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
	if lc == nil || lc.Position == LegendPositionNone || len(lc.Entries) == 0 {
		return nil
	}

	entries := make([]model.LegendEntryData, len(lc.Entries))

	for i, e := range lc.Entries {
		entries[i] = model.LegendEntryData{
			Title:    string(e.Role) + ": " + e.MetricName,
			Kind:     e.Ink.legendEntryKind(),
			Swatches: e.Ink.legendSwatches(),
		}
	}

	orient := lc.Orientation
	if orient == "" {
		orient = DefaultOrientation(lc.Position)
	}

	return &model.LegendData{
		Position:    string(lc.Position),
		Orientation: string(orient),
		Entries:     entries,
	}
}
