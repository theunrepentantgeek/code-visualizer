package legendlayout

import (
	"image/color"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/bevan/code-visualizer/internal/canvas/model"
)

func TestFormatBreakpoint_IntegerValue(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(FormatBreakpoint(42)).To(Equal("42"))
	g.Expect(FormatBreakpoint(0)).To(Equal("0"))
	g.Expect(FormatBreakpoint(1000)).To(Equal("1000"))
}

func TestFormatBreakpoint_FloatValue(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(FormatBreakpoint(3.14)).To(Equal("3.1"))
	g.Expect(FormatBreakpoint(0.5)).To(Equal("0.5"))
}

func TestLegendOrigin_AllPositions_InBounds(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	canvasW, canvasH := 800.0, 600.0
	legendW, legendH := 100.0, 50.0

	positions := []string{
		"top-left", "top-center", "top-right",
		"center-right", "bottom-right", "bottom-center",
		"bottom-left", "center-left",
	}

	for _, pos := range positions {
		ox, oy := LegendOrigin(pos, canvasW, canvasH, legendW, legendH)
		g.Expect(ox).To(BeNumerically(">=", 0), "x out of bounds for %s", pos)
		g.Expect(oy).To(BeNumerically(">=", 0), "y out of bounds for %s", pos)
		g.Expect(ox+legendW).To(BeNumerically("<=", canvasW),
			"right edge out of bounds for %s", pos)
		g.Expect(oy+legendH).To(BeNumerically("<=", canvasH),
			"bottom edge out of bounds for %s", pos)
	}
}

func TestLegendOrigin_TopLeft_IsNearOrigin(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ox, oy := LegendOrigin("top-left", 800, 600, 100, 50)
	g.Expect(ox).To(Equal(model.LegendMargin))
	g.Expect(oy).To(Equal(model.LegendMargin))
}

func TestLegendOrigin_BottomRight_IsNearCorner(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ox, oy := LegendOrigin("bottom-right", 800, 600, 100, 50)
	g.Expect(ox).To(Equal(800.0 - 100.0 - model.LegendMargin))
	g.Expect(oy).To(Equal(600.0 - 50.0 - model.LegendMargin))
}

func TestMeasureLegend_EmptyEntries_ReturnsZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	data := &model.LegendData{Orientation: "vertical"}
	w, h := MeasureLegend(data)
	g.Expect(w).To(BeZero())
	g.Expect(h).To(BeZero())
}

func TestMeasureLegend_Nil_ReturnsZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	w, h := MeasureLegend(nil)
	g.Expect(w).To(BeZero())
	g.Expect(h).To(BeZero())
}

func TestMeasureLegend_Vertical_NonZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	data := makeSampleLegendData("vertical")
	w, h := MeasureLegend(data)
	g.Expect(w).To(BeNumerically(">", 0))
	g.Expect(h).To(BeNumerically(">", 0))
}

func TestMeasureLegend_Horizontal_WiderThanVertical(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dataH := makeSampleLegendData("horizontal")
	dataV := makeSampleLegendData("vertical")
	wH, _ := MeasureLegend(dataH)
	wV, _ := MeasureLegend(dataV)
	g.Expect(wH).To(BeNumerically(">", wV),
		"horizontal legend should be wider than vertical")
}

func TestMeasureLegend_Horizontal_ShorterThanVertical(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dataH := makeSampleLegendData("horizontal")
	dataV := makeSampleLegendData("vertical")
	_, hH := MeasureLegend(dataH)
	_, hV := MeasureLegend(dataV)
	g.Expect(hH).To(BeNumerically("<", hV),
		"horizontal legend should be shorter than vertical")
}

func TestReserveSpace_NilData_ReturnsZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	wReduce, hReduce := ReserveSpace(nil)
	g.Expect(wReduce).To(BeZero())
	g.Expect(hReduce).To(BeZero())
}

func TestReserveSpace_NonePosition_ReturnsZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	data := &model.LegendData{Position: "none"}
	wReduce, hReduce := ReserveSpace(data)
	g.Expect(wReduce).To(BeZero())
	g.Expect(hReduce).To(BeZero())
}

func TestReserveSpace_CenterRight_ReducesWidth(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	data := makeSampleLegendData("vertical")
	data.Position = "center-right"
	wReduce, hReduce := ReserveSpace(data)
	g.Expect(wReduce).To(BeNumerically(">", 0))
	g.Expect(hReduce).To(BeZero())
}

func TestReserveSpace_BottomCenter_ReducesHeight(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	data := makeSampleLegendData("vertical")
	data.Position = "bottom-center"
	wReduce, hReduce := ReserveSpace(data)
	g.Expect(wReduce).To(BeZero())
	g.Expect(hReduce).To(BeNumerically(">", 0))
}

func TestReserveSpace_CornerVertical_ReducesWidth(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	data := makeSampleLegendData("vertical")
	data.Position = "bottom-right"
	wReduce, hReduce := ReserveSpace(data)
	g.Expect(wReduce).To(BeNumerically(">", 0))
	g.Expect(hReduce).To(BeZero())
}

func TestReserveSpace_CornerHorizontal_ReducesHeight(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	data := makeSampleLegendData("horizontal")
	data.Position = "bottom-right"
	wReduce, hReduce := ReserveSpace(data)
	g.Expect(wReduce).To(BeZero())
	g.Expect(hReduce).To(BeNumerically(">", 0))
}

// makeSampleLegendData creates test legend data with both numeric and
// categorical entries.
func makeSampleLegendData(orientation string) *model.LegendData {
	return &model.LegendData{
		Position:    "bottom-right",
		Orientation: orientation,
		Entries: []model.LegendEntryData{
			{
				Title: "Fill: file-size",
				Kind:  model.LegendEntryNumeric,
				Swatches: []model.LegendSwatch{
					{Colour: color.RGBA{R: 50, G: 50, B: 200, A: 255}, Label: "100"},
					{Colour: color.RGBA{R: 100, G: 100, B: 200, A: 255}, Label: "500"},
					{Colour: color.RGBA{R: 150, G: 150, B: 200, A: 255}, Label: "1000"},
					{Colour: color.RGBA{R: 200, G: 200, B: 200, A: 255}, Label: "5000"},
					{Colour: color.RGBA{R: 250, G: 250, B: 200, A: 255}, Label: ""},
				},
			},
			{
				Title: "Border: file-type",
				Kind:  model.LegendEntryCategorical,
				Swatches: []model.LegendSwatch{
					{Colour: color.RGBA{R: 0, G: 173, B: 216, A: 255}, Label: "go"},
					{Colour: color.RGBA{R: 222, G: 165, B: 132, A: 255}, Label: "rs"},
					{Colour: color.RGBA{R: 53, G: 114, B: 165, A: 255}, Label: "py"},
				},
			},
		},
	}
}
