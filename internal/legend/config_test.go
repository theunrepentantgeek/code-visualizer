package legend

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

func TestDefaultOrientation_CenterPositions_ReturnsHorizontal(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(DefaultOrientation(model.LegendPositionTopCenter)).To(Equal(model.LegendOrientationHorizontal))
	g.Expect(DefaultOrientation(model.LegendPositionBottomCenter)).To(Equal(model.LegendOrientationHorizontal))
}

func TestDefaultOrientation_SidePositions_ReturnsVertical(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sides := []model.LegendPosition{
		model.LegendPositionTopLeft,
		model.LegendPositionTopRight,
		model.LegendPositionCenterRight,
		model.LegendPositionBottomRight,
		model.LegendPositionBottomLeft,
		model.LegendPositionCenterLeft,
	}

	for _, pos := range sides {
		g.Expect(DefaultOrientation(pos)).To(Equal(model.LegendOrientationVertical),
			"expected vertical for %s", pos)
	}
}

func TestToLegendData_NilEntries_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := &Config{
		Position:    model.LegendPositionNone,
		Orientation: model.LegendOrientationVertical,
	}

	g.Expect(cfg.toLegendData()).To(BeNil())
}

func TestToLegendData_NumericEntry_ProducesSwatches(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pal := palette.GetPalette(palette.Temperature)
	fillInk := inks.NumericInk("file-size", []float64{10, 50, 100, 500, 1000}, pal)

	cfg := &Config{
		Position:    model.LegendPositionBottomRight,
		Orientation: model.LegendOrientationVertical,
		Entries: []Entry{
			{Role: RoleFill, MetricName: "file-size", Ink: fillInk},
		},
	}

	data := cfg.toLegendData()
	g.Expect(data).NotTo(BeNil())

	if data == nil {
		return // unreachable; satisfies nilaway
	}

	g.Expect(data.Position).To(Equal(model.LegendPositionBottomRight))
	g.Expect(data.Orientation).To(Equal(model.LegendOrientationVertical))
	g.Expect(data.Entries).To(HaveLen(1))
	g.Expect(data.Entries[0].Label).To(Equal("Fill"))
	g.Expect(data.Entries[0].Metric).To(Equal("file-size"))
	g.Expect(data.Entries[0].Kind).To(Equal(model.LegendEntryNumeric))
	g.Expect(data.Entries[0].Swatches).NotTo(BeEmpty())
}

func TestToLegendData_CategoricalEntry_ProducesSwatches(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pal := palette.GetPalette(palette.Categorization)
	borderInk := inks.CategoricalInk("file-type", []string{"go", "py", "rs"}, pal)

	cfg := &Config{
		Position:    model.LegendPositionTopLeft,
		Orientation: model.LegendOrientationHorizontal,
		Entries: []Entry{
			{Role: RoleBorder, MetricName: "file-type", Ink: borderInk},
		},
	}

	data := cfg.toLegendData()
	g.Expect(data).NotTo(BeNil())

	if data == nil {
		return // unreachable; satisfies nilaway
	}

	g.Expect(data.Entries[0].Kind).To(Equal(model.LegendEntryCategorical))
	g.Expect(data.Entries[0].Swatches).To(HaveLen(3))
}

func TestToLegendData_FixedInkEntry_EmptySwatches(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := &Config{
		Position:    model.LegendPositionBottomRight,
		Orientation: model.LegendOrientationVertical,
		Entries: []Entry{
			{Role: RoleSize, MetricName: "file-lines", Ink: inks.FixedInk(white)},
		},
	}

	data := cfg.toLegendData()
	g.Expect(data).NotTo(BeNil())

	if data == nil {
		return // unreachable; satisfies nilaway
	}

	g.Expect(data.Entries[0].Kind).To(Equal(model.LegendEntryNumeric))
	g.Expect(data.Entries[0].Swatches).To(BeNil())
}

func TestToLegendData_RoundTrip_PositionConstants(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pal := palette.GetPalette(palette.Temperature)
	fillInk := inks.NumericInk("file-size", []float64{10, 100}, pal)

	positions := []model.LegendPosition{
		model.LegendPositionTopLeft, model.LegendPositionTopCenter, model.LegendPositionTopRight,
		model.LegendPositionCenterRight, model.LegendPositionBottomRight, model.LegendPositionBottomCenter,
		model.LegendPositionBottomLeft, model.LegendPositionCenterLeft,
	}

	for _, pos := range positions {
		cfg := &Config{
			Position: pos,
			Entries:  []Entry{{Role: RoleFill, MetricName: "file-size", Ink: fillInk}},
		}
		data := cfg.toLegendData()
		g.Expect(data).NotTo(BeNil(), "position %q produced nil data", pos)

		if data != nil {
			g.Expect(data.Position).To(Equal(pos),
				"position %q did not round-trip", pos)
		}
	}
}

func TestToLegendData_RoundTrip_OrientationConstants(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pal := palette.GetPalette(palette.Temperature)
	fillInk := inks.NumericInk("file-size", []float64{10, 100}, pal)

	orientations := []model.LegendOrientation{
		model.LegendOrientationVertical,
		model.LegendOrientationHorizontal,
	}

	for _, orient := range orientations {
		cfg := &Config{
			Position:    model.LegendPositionBottomRight,
			Orientation: orient,
			Entries:     []Entry{{Role: RoleFill, MetricName: "file-size", Ink: fillInk}},
		}
		data := cfg.toLegendData()
		g.Expect(data).NotTo(BeNil(), "orientation %q produced nil data", orient)

		if data != nil {
			g.Expect(data.Orientation).To(Equal(orient),
				"orientation %q did not round-trip", orient)
		}
	}
}

func TestReserveSpace_NonePosition_ReturnsZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := &Config{Position: model.LegendPositionNone}
	wReduce, hReduce := cfg.ReserveSpace()
	g.Expect(wReduce).To(BeZero())
	g.Expect(hReduce).To(BeZero())
}

func TestReserveSpace_WithEntries_NonZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pal := palette.GetPalette(palette.Temperature)
	fillInk := inks.NumericInk("file-size", []float64{10, 50, 100}, pal)

	cfg := &Config{
		Position:    model.LegendPositionCenterRight,
		Orientation: model.LegendOrientationVertical,
		Entries: []Entry{
			{Role: RoleFill, MetricName: "file-size", Ink: fillInk},
		},
	}

	wReduce, hReduce := cfg.ReserveSpace()
	g.Expect(wReduce).To(BeNumerically(">", 0))
	g.Expect(hReduce).To(BeZero())
}
