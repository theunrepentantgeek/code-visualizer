package legend

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
)

// MinReservableSize is the smallest canvas dimension (px) that still
// produces a usable visualization. If reserving legend space would shrink
// either dimension below this, ReserveAndLayout falls back to the full
// canvas (overlay behaviour).
const MinReservableSize = 100

// ReserveAndLayout returns the layout dimensions after reserving space
// for the legend. Falls back to (width, height) when reservation would
// shrink either dimension below MinReservableSize.
func ReserveAndLayout(cfg *Config, width, height int) (layoutW, layoutH int) {
	if cfg == nil {
		return width, height
	}

	reserved := cfg.ReserveSpace()

	w := width - int(reserved.Width)
	h := height - int(reserved.Height)

	if w < MinReservableSize || h < MinReservableSize {
		return width, height
	}

	return w, h
}

// LayoutOffset returns the (dx, dy) offset to apply to layout output
// when space has been reserved for the legend.
func LayoutOffset(cfg *Config, reserved geometry.Size) (dx, dy float64) {
	if cfg == nil {
		return 0, 0
	}

	switch cfg.Position {
	case model.LegendPositionTopCenter:
		return 0, reserved.Height
	case model.LegendPositionCenterLeft:
		return reserved.Width, 0
	default:
		return cornerOffset(cfg, reserved)
	}
}

func cornerOffset(cfg *Config, reserved geometry.Size) (dx, dy float64) {
	isTop := cfg.Position == model.LegendPositionTopLeft || cfg.Position == model.LegendPositionTopRight
	isLeft := cfg.Position == model.LegendPositionTopLeft || cfg.Position == model.LegendPositionBottomLeft

	if cfg.Orientation == model.LegendOrientationVertical {
		if isLeft {
			return reserved.Width, 0
		}

		return 0, 0
	}

	if isTop {
		return 0, reserved.Height
	}

	return 0, 0
}
