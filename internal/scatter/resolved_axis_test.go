package scatter_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/scatter"
)

// NumericTicks tests

func TestResolvedAxis_NumericTicks_NilReceiverReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var a *scatter.ResolvedAxis
	g.Expect(a.NumericTicks()).To(BeNil())
}

func TestResolvedAxis_NumericTicks_NilNumericReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	a := &scatter.ResolvedAxis{}
	g.Expect(a.NumericTicks()).To(BeNil())
}

func TestResolvedAxis_NumericTicks_ReturnsTicks(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	a := &scatter.ResolvedAxis{
		Numeric: &scatter.NumericAxis{
			Ticks: []scatter.AxisTick{
				{Value: 10, Label: "10", Position: 0.1},
				{Value: 20, Label: "20", Position: 0.2},
			},
		},
	}

	ticks := a.NumericTicks()
	g.Expect(ticks).To(HaveLen(2))
	if len(ticks) == 2 {
		g.Expect(ticks[0].Value).To(Equal(10.0))
		g.Expect(ticks[1].Value).To(Equal(20.0))
	}
}

// CategoricalBands tests

func TestResolvedAxis_CategoricalBands_NilReceiverReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var a *scatter.ResolvedAxis
	g.Expect(a.CategoricalBands()).To(BeNil())
}

func TestResolvedAxis_CategoricalBands_NilCategoricalReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	a := &scatter.ResolvedAxis{}
	g.Expect(a.CategoricalBands()).To(BeNil())
}

func TestResolvedAxis_CategoricalBands_ReturnsBands(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	a := &scatter.ResolvedAxis{
		Categorical: &scatter.CategoricalAxis{
			Bands: []scatter.AxisBand{
				{Label: "go", Start: 0, End: 50, Center: 25},
				{Label: "ts", Start: 50, End: 100, Center: 75},
			},
		},
	}

	bands := a.CategoricalBands()
	g.Expect(bands).To(HaveLen(2))
	if len(bands) == 2 {
		g.Expect(bands[0].Label).To(Equal("go"))
		g.Expect(bands[1].Label).To(Equal("ts"))
	}
}

// Offset tests

func TestResolvedAxis_Offset_NilReceiverIsNoOp(t *testing.T) {
	t.Parallel()

	// Should not panic.
	var a *scatter.ResolvedAxis
	a.Offset(100)
}

func TestResolvedAxis_Offset_ShiftsNumericTickPositions(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	a := &scatter.ResolvedAxis{
		Numeric: &scatter.NumericAxis{
			Ticks: []scatter.AxisTick{
				{Value: 0, Label: "0", Position: 10},
				{Value: 50, Label: "50", Position: 60},
			},
		},
	}

	a.Offset(20)

	g.Expect(a.Numeric.Ticks[0].Position).To(Equal(30.0))
	g.Expect(a.Numeric.Ticks[1].Position).To(Equal(80.0))
}

func TestResolvedAxis_Offset_ShiftsCategoricalBandPositions(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	a := &scatter.ResolvedAxis{
		Categorical: &scatter.CategoricalAxis{
			Bands: []scatter.AxisBand{
				{Label: "go", Start: 0, End: 50, Center: 25},
			},
		},
	}

	a.Offset(10)

	g.Expect(a.Categorical.Bands[0].Start).To(Equal(10.0))
	g.Expect(a.Categorical.Bands[0].End).To(Equal(60.0))
	g.Expect(a.Categorical.Bands[0].Center).To(Equal(35.0))
}

func TestResolvedAxis_Offset_ShiftsCategoricalCentersMap(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	a := &scatter.ResolvedAxis{
		Categorical: &scatter.CategoricalAxis{
			Bands: []scatter.AxisBand{
				{Label: "go", Start: 0, End: 50, Center: 25},
			},
			Centers: map[string]float64{
				"go": 25,
			},
		},
	}

	a.Offset(15)

	g.Expect(a.Categorical.Centers["go"]).To(Equal(40.0))
}

func TestResolvedAxis_Offset_NilCategoricalIsNoOp(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	a := &scatter.ResolvedAxis{
		Numeric: &scatter.NumericAxis{
			Ticks: []scatter.AxisTick{
				{Value: 0, Position: 5},
			},
		},
	}

	a.Offset(3)

	g.Expect(a.Numeric.Ticks[0].Position).To(Equal(8.0))
}
