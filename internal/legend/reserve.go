package legend

import "github.com/theunrepentantgeek/code-visualizer/internal/canvas"

// MinReservableSize is the smallest canvas dimension (px) that still
// produces a usable visualization. If reserving legend space would shrink
// either dimension below this, ReserveAndLayout falls back to the full
// canvas (overlay behaviour).
const MinReservableSize = 100

// ReserveAndLayout returns the layout dimensions after reserving space
// for the legend. Falls back to (width, height) when reservation would
// shrink either dimension below MinReservableSize.
func ReserveAndLayout(cfg *canvas.LegendConfig, width, height int) (layoutW, layoutH int) {
	if cfg == nil {
		return width, height
	}

	wReduce, hReduce := cfg.ReserveSpace()

	w := width - int(wReduce)
	h := height - int(hReduce)

	if w < MinReservableSize || h < MinReservableSize {
		return width, height
	}

	return w, h
}

// LayoutOffset returns the (dx, dy) offset to apply to layout output
// when space has been reserved for the legend.
func LayoutOffset(cfg *canvas.LegendConfig, wReduce, hReduce float64) (dx, dy float64) {
	if cfg == nil {
		return 0, 0
	}

	switch cfg.Position {
	case canvas.LegendPositionTopCenter:
		return 0, hReduce
	case canvas.LegendPositionCenterLeft:
		return wReduce, 0
	default:
		return cornerOffset(cfg, wReduce, hReduce)
	}
}

func cornerOffset(cfg *canvas.LegendConfig, wReduce, hReduce float64) (dx, dy float64) {
	isTop := cfg.Position == canvas.LegendPositionTopLeft || cfg.Position == canvas.LegendPositionTopRight
	isLeft := cfg.Position == canvas.LegendPositionTopLeft || cfg.Position == canvas.LegendPositionBottomLeft

	if cfg.Orientation == canvas.LegendOrientationVertical {
		if isLeft {
			return wReduce, 0
		}

		return 0, 0
	}

	if isTop {
		return 0, hReduce
	}

	return 0, 0
}
