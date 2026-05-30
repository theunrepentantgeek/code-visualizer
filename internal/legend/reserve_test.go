package legend_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	model0 "github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
)

// zeroReductionCfg returns a LegendConfig whose ReserveSpace() returns (0,0).
// Position=None causes toLegendData to return nil, so legendlayout.ReserveSpace
// gets nil and produces no reduction.
func zeroReductionCfg(pos model0.LegendPosition, orient model0.LegendOrientation) *canvas.LegendConfig {
	return &canvas.LegendConfig{Position: pos, Orientation: orient}
}

// --- ReserveAndLayout ---

func TestReserveAndLayout_NilConfig_Passthrough(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	w, h := legend.ReserveAndLayout(nil, 1000, 800)
	g.Expect(w).To(Equal(1000))
	g.Expect(h).To(Equal(800))
}

func TestReserveAndLayout_LargeCanvas_ZeroReduction_Passthrough(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// A config with no entries produces zero reduction; the full dims are returned.
	cfg := zeroReductionCfg(model0.LegendPositionNone, model0.LegendOrientationVertical)
	w, h := legend.ReserveAndLayout(cfg, 1000, 800)
	g.Expect(w).To(Equal(1000))
	g.Expect(h).To(Equal(800))
}

func TestReserveAndLayout_SmallCanvas_FallsBack(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Canvas smaller than MinReservableSize triggers the fallback even with zero reduction.
	cfg := zeroReductionCfg(model0.LegendPositionNone, model0.LegendOrientationVertical)
	w, h := legend.ReserveAndLayout(cfg, 50, 50)
	g.Expect(w).To(Equal(50))
	g.Expect(h).To(Equal(50))
}

func TestReserveAndLayout_ExactlyMinSize_Passthrough(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Exactly at MinReservableSize should NOT fall back (boundary: >= 100 is fine).
	cfg := zeroReductionCfg(model0.LegendPositionNone, model0.LegendOrientationVertical)
	w, h := legend.ReserveAndLayout(cfg, legend.MinReservableSize, legend.MinReservableSize)
	g.Expect(w).To(Equal(legend.MinReservableSize))
	g.Expect(h).To(Equal(legend.MinReservableSize))
}

// --- LayoutOffset ---

func TestLayoutOffset_NilConfig_ZeroOffset(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dx, dy := legend.LayoutOffset(nil, 200, 150)
	g.Expect(dx).To(Equal(0.0))
	g.Expect(dy).To(Equal(0.0))
}

func TestLayoutOffset_TopCenter_YOffset(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := &canvas.LegendConfig{Position: model0.LegendPositionTopCenter, Orientation: model0.LegendOrientationHorizontal}
	dx, dy := legend.LayoutOffset(cfg, 0, 150)
	g.Expect(dx).To(Equal(0.0))
	g.Expect(dy).To(Equal(150.0))
}

func TestLayoutOffset_CenterLeft_XOffset(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := &canvas.LegendConfig{Position: model0.LegendPositionCenterLeft, Orientation: model0.LegendOrientationVertical}
	dx, dy := legend.LayoutOffset(cfg, 200, 0)
	g.Expect(dx).To(Equal(200.0))
	g.Expect(dy).To(Equal(0.0))
}

// cornerOffset: vertical orientation, left-side positions shift X.

func TestLayoutOffset_TopLeft_Vertical_XOffset(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := &canvas.LegendConfig{Position: model0.LegendPositionTopLeft, Orientation: model0.LegendOrientationVertical}
	dx, dy := legend.LayoutOffset(cfg, 200, 150)
	g.Expect(dx).To(Equal(200.0))
	g.Expect(dy).To(Equal(0.0))
}

func TestLayoutOffset_BottomLeft_Vertical_XOffset(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := &canvas.LegendConfig{Position: model0.LegendPositionBottomLeft, Orientation: model0.LegendOrientationVertical}
	dx, dy := legend.LayoutOffset(cfg, 200, 150)
	g.Expect(dx).To(Equal(200.0))
	g.Expect(dy).To(Equal(0.0))
}

func TestLayoutOffset_TopRight_Vertical_ZeroOffset(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := &canvas.LegendConfig{Position: model0.LegendPositionTopRight, Orientation: model0.LegendOrientationVertical}
	dx, dy := legend.LayoutOffset(cfg, 200, 150)
	g.Expect(dx).To(Equal(0.0))
	g.Expect(dy).To(Equal(0.0))
}

func TestLayoutOffset_BottomRight_Vertical_ZeroOffset(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := &canvas.LegendConfig{Position: model0.LegendPositionBottomRight, Orientation: model0.LegendOrientationVertical}
	dx, dy := legend.LayoutOffset(cfg, 200, 150)
	g.Expect(dx).To(Equal(0.0))
	g.Expect(dy).To(Equal(0.0))
}

// cornerOffset: horizontal orientation, top positions shift Y.

func TestLayoutOffset_TopLeft_Horizontal_YOffset(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := &canvas.LegendConfig{Position: model0.LegendPositionTopLeft, Orientation: model0.LegendOrientationHorizontal}
	dx, dy := legend.LayoutOffset(cfg, 200, 150)
	g.Expect(dx).To(Equal(0.0))
	g.Expect(dy).To(Equal(150.0))
}

func TestLayoutOffset_TopRight_Horizontal_YOffset(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := &canvas.LegendConfig{Position: model0.LegendPositionTopRight, Orientation: model0.LegendOrientationHorizontal}
	dx, dy := legend.LayoutOffset(cfg, 200, 150)
	g.Expect(dx).To(Equal(0.0))
	g.Expect(dy).To(Equal(150.0))
}

func TestLayoutOffset_BottomLeft_Horizontal_ZeroOffset(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := &canvas.LegendConfig{Position: model0.LegendPositionBottomLeft, Orientation: model0.LegendOrientationHorizontal}
	dx, dy := legend.LayoutOffset(cfg, 200, 150)
	g.Expect(dx).To(Equal(0.0))
	g.Expect(dy).To(Equal(0.0))
}

func TestLayoutOffset_BottomRight_Horizontal_ZeroOffset(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := &canvas.LegendConfig{Position: model0.LegendPositionBottomRight, Orientation: model0.LegendOrientationHorizontal}
	dx, dy := legend.LayoutOffset(cfg, 200, 150)
	g.Expect(dx).To(Equal(0.0))
	g.Expect(dy).To(Equal(0.0))
}
