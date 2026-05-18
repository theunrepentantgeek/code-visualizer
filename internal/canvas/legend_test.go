package canvas

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

func TestDefaultOrientation_CenterPositions_ReturnsHorizontal(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(DefaultOrientation(LegendPositionTopCenter)).To(Equal(LegendOrientationHorizontal))
	g.Expect(DefaultOrientation(LegendPositionBottomCenter)).To(Equal(LegendOrientationHorizontal))
}

func TestDefaultOrientation_SidePositions_ReturnsVertical(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sides := []LegendPosition{
		LegendPositionTopLeft,
		LegendPositionTopRight,
		LegendPositionCenterRight,
		LegendPositionBottomRight,
		LegendPositionBottomLeft,
		LegendPositionCenterLeft,
	}

	for _, pos := range sides {
		g.Expect(DefaultOrientation(pos)).To(Equal(LegendOrientationVertical),
			"expected vertical for %s", pos)
	}
}

func TestToLegendData_NilEntries_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	lc := &LegendConfig{
		Position:    LegendPositionNone,
		Orientation: LegendOrientationVertical,
	}

	g.Expect(lc.toLegendData()).To(BeNil())
}

func TestToLegendData_NumericEntry_ProducesSwatches(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pal := palette.GetPalette(palette.Temperature)
	fillInk := NumericInk("file-size", []float64{10, 50, 100, 500, 1000}, pal)

	lc := &LegendConfig{
		Position:    LegendPositionBottomRight,
		Orientation: LegendOrientationVertical,
		Entries: []LegendEntry{
			{Role: LegendRoleFill, MetricName: "file-size", Ink: fillInk},
		},
	}

	data := lc.toLegendData()
	g.Expect(data).NotTo(BeNil())

	if data == nil {
		return // unreachable; satisfies nilaway
	}

	g.Expect(data.Position).To(Equal(model.LegendPositionBottomRight))
	g.Expect(data.Orientation).To(Equal(model.LegendOrientationVertical))
	g.Expect(data.Entries).To(HaveLen(1))
	g.Expect(data.Entries[0].Title).To(Equal("Fill: file-size"))
	g.Expect(data.Entries[0].Kind).To(Equal(model.LegendEntryNumeric))
	g.Expect(data.Entries[0].Swatches).NotTo(BeEmpty())
}

func TestToLegendData_CategoricalEntry_ProducesSwatches(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pal := palette.GetPalette(palette.Categorization)
	borderInk := CategoricalInk("file-type", []string{"go", "py", "rs"}, pal)

	lc := &LegendConfig{
		Position:    LegendPositionTopLeft,
		Orientation: LegendOrientationHorizontal,
		Entries: []LegendEntry{
			{Role: LegendRoleBorder, MetricName: "file-type", Ink: borderInk},
		},
	}

	data := lc.toLegendData()
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

	lc := &LegendConfig{
		Position:    LegendPositionBottomRight,
		Orientation: LegendOrientationVertical,
		Entries: []LegendEntry{
			{Role: LegendRoleSize, MetricName: "file-lines", Ink: FixedInk(white)},
		},
	}

	data := lc.toLegendData()
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
	fillInk := NumericInk("file-size", []float64{10, 100}, pal)

	positions := []LegendPosition{
		LegendPositionTopLeft, LegendPositionTopCenter, LegendPositionTopRight,
		LegendPositionCenterRight, LegendPositionBottomRight, LegendPositionBottomCenter,
		LegendPositionBottomLeft, LegendPositionCenterLeft,
	}

	for _, pos := range positions {
		lc := &LegendConfig{
			Position: pos,
			Entries:  []LegendEntry{{Role: LegendRoleFill, MetricName: "file-size", Ink: fillInk}},
		}
		data := lc.toLegendData()
		g.Expect(data).NotTo(BeNil(), "position %q produced nil data", pos)

		if data != nil {
			g.Expect(data.Position).To(Equal(model.LegendPosition(pos)),
				"position %q did not round-trip", pos)
		}
	}
}

func TestToLegendData_RoundTrip_OrientationConstants(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pal := palette.GetPalette(palette.Temperature)
	fillInk := NumericInk("file-size", []float64{10, 100}, pal)

	orientations := []LegendOrientation{
		LegendOrientationVertical,
		LegendOrientationHorizontal,
	}

	for _, orient := range orientations {
		lc := &LegendConfig{
			Position:    LegendPositionBottomRight,
			Orientation: orient,
			Entries:     []LegendEntry{{Role: LegendRoleFill, MetricName: "file-size", Ink: fillInk}},
		}
		data := lc.toLegendData()
		g.Expect(data).NotTo(BeNil(), "orientation %q produced nil data", orient)

		if data != nil {
			g.Expect(data.Orientation).To(Equal(model.LegendOrientation(orient)),
				"orientation %q did not round-trip", orient)
		}
	}
}


func TestReserveSpace_NonePosition_ReturnsZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	lc := &LegendConfig{Position: LegendPositionNone}
	wReduce, hReduce := lc.ReserveSpace()
	g.Expect(wReduce).To(BeZero())
	g.Expect(hReduce).To(BeZero())
}

func TestReserveSpace_WithEntries_NonZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pal := palette.GetPalette(palette.Temperature)
	fillInk := NumericInk("file-size", []float64{10, 50, 100}, pal)

	lc := &LegendConfig{
		Position:    LegendPositionCenterRight,
		Orientation: LegendOrientationVertical,
		Entries: []LegendEntry{
			{Role: LegendRoleFill, MetricName: "file-size", Ink: fillInk},
		},
	}

	wReduce, hReduce := lc.ReserveSpace()
	g.Expect(wReduce).To(BeNumerically(">", 0))
	g.Expect(hReduce).To(BeZero())
}
