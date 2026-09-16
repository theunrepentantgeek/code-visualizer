package legend_test

import (
	"testing"

	. "github.com/onsi/gomega"

	canvasmodel "github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

func reservingCfg(
	position canvasmodel.LegendPosition,
	orientation canvasmodel.LegendOrientation,
) *legend.Config {
	return &legend.Config{
		Position:    position,
		Orientation: orientation,
		Entries: []legend.Entry{{
			Role:       legend.RoleFill,
			MetricName: "file-size",
			Ink:        inks.FixedInk(palette.White),
		}},
	}
}

func TestReserveLayoutPassthrough(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  *legend.Config
	}{
		{name: "nil config"},
		{
			name: "config without entries",
			cfg: &legend.Config{
				Position:    canvasmodel.LegendPositionTopCenter,
				Orientation: canvasmodel.LegendOrientationHorizontal,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			got := legend.ReserveLayout(tt.cfg, 1000, 800)

			g.Expect(got).To(Equal(legend.Reservation{Width: 1000, Height: 800}))
		})
	}
}

func TestReserveLayoutReturnsMatchingDimensionsAndOffset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		position    canvasmodel.LegendPosition
		orientation canvasmodel.LegendOrientation
		offset      func(geometry.Size) geometry.Vector
	}{
		{
			name:        "top centre",
			position:    canvasmodel.LegendPositionTopCenter,
			orientation: canvasmodel.LegendOrientationHorizontal,
			offset:      func(size geometry.Size) geometry.Vector { return geometry.NewVector(0, size.Height) },
		},
		{
			name:        "centre left",
			position:    canvasmodel.LegendPositionCenterLeft,
			orientation: canvasmodel.LegendOrientationVertical,
			offset:      func(size geometry.Size) geometry.Vector { return geometry.NewVector(size.Width, 0) },
		},
		{
			name:        "top left vertical",
			position:    canvasmodel.LegendPositionTopLeft,
			orientation: canvasmodel.LegendOrientationVertical,
			offset:      func(size geometry.Size) geometry.Vector { return geometry.NewVector(size.Width, 0) },
		},
		{
			name:        "top right horizontal",
			position:    canvasmodel.LegendPositionTopRight,
			orientation: canvasmodel.LegendOrientationHorizontal,
			offset:      func(size geometry.Size) geometry.Vector { return geometry.NewVector(0, size.Height) },
		},
		{
			name:        "bottom right",
			position:    canvasmodel.LegendPositionBottomRight,
			orientation: canvasmodel.LegendOrientationVertical,
			offset:      func(geometry.Size) geometry.Vector { return geometry.ZeroVector },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)
			cfg := reservingCfg(tt.position, tt.orientation)
			reserved := cfg.ReserveSpace()

			got := legend.ReserveLayout(cfg, 1000, 800)

			g.Expect(got).To(Equal(legend.Reservation{
				Width:  1000 - int(reserved.Width),
				Height: 800 - int(reserved.Height),
				Offset: tt.offset(reserved),
			}))
		})
	}
}

func TestReserveLayoutFallbackReturnsFullDimensionsAndZeroOffset(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	cfg := reservingCfg(canvasmodel.LegendPositionTopCenter, canvasmodel.LegendOrientationHorizontal)

	got := legend.ReserveLayout(cfg, 50, 50)

	g.Expect(got).To(Equal(legend.Reservation{Width: 50, Height: 50}))
}

func TestReserveLayoutAcceptsExactlyMinimumContentSize(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	cfg := reservingCfg(canvasmodel.LegendPositionTopCenter, canvasmodel.LegendOrientationHorizontal)
	reserved := cfg.ReserveSpace()

	got := legend.ReserveLayout(
		cfg,
		legend.MinReservableSize+int(reserved.Width),
		legend.MinReservableSize+int(reserved.Height),
	)

	g.Expect(got.Width).To(Equal(legend.MinReservableSize))
	g.Expect(got.Height).To(Equal(legend.MinReservableSize))
	g.Expect(got.Offset).To(Equal(geometry.NewVector(0, reserved.Height)))
}
