package legend

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
)

// MinReservableSize is the smallest canvas dimension (px) that still
// produces a usable visualization. If reserving legend space would shrink
// either dimension below this, ReserveLayout falls back to the full canvas
// with no offset (overlay behaviour).
const MinReservableSize = 100

// Reservation describes the content area left after reserving legend space.
type Reservation struct {
	Width  int
	Height int
	Offset geometry.Vector
}

// ReserveLayout returns content dimensions and their matching offset after
// reserving legend space. It returns the full dimensions and zero offset when
// reservation would leave an unusably small content area.
func ReserveLayout(cfg *Config, width, height int) Reservation {
	result := Reservation{Width: width, Height: height}
	if cfg == nil {
		return result
	}

	reserved := cfg.ReserveSpace()
	result.Width -= int(reserved.Width)
	result.Height -= int(reserved.Height)
	if result.Width < MinReservableSize || result.Height < MinReservableSize {
		return Reservation{Width: width, Height: height}
	}

	dx, dy := layoutOffset(cfg, reserved)
	result.Offset = geometry.NewVector(dx, dy)

	return result
}

func layoutOffset(cfg *Config, reserved geometry.Size) (dx, dy float64) {
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
