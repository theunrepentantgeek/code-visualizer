package geometry

import (
	"math"
	"testing"

	. "github.com/onsi/gomega"
)

func TestPointValid(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	tests := []struct {
		name  string
		point Point
		want  bool
	}{
		{name: "finite zero", point: Point{}, want: true},
		{name: "finite components", point: Point{X: 1.5, Y: -2.5}, want: true},
		{name: "nan x", point: Point{X: math.NaN(), Y: 1}, want: false},
		{name: "nan y", point: Point{X: 1, Y: math.NaN()}, want: false},
		{name: "positive infinity x", point: Point{X: math.Inf(1), Y: 1}, want: false},
		{name: "negative infinity y", point: Point{X: 1, Y: math.Inf(-1)}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g.Expect(tt.point.Valid()).To(Equal(tt.want))
		})
	}
}

func TestPointVectorSemantics(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	start := Point{X: 2, Y: 3}
	end := Point{X: 5, Y: 7}

	g.Expect(start.VectorTo(end)).To(Equal(Vector{X: 3, Y: 4}))
	g.Expect(start.Translate(start.VectorTo(end))).To(Equal(end))
	g.Expect(start.DistanceTo(end)).To(Equal(5.0))
	g.Expect(start.DistanceSquaredTo(end)).To(Equal(25.0))
	g.Expect(Midpoint(start, end)).To(Equal(Point{X: 3.5, Y: 5}))
	g.Expect(Lerp(start, end, 0)).To(Equal(start))
	g.Expect(Lerp(start, end, 1)).To(Equal(end))
}

func TestPointInterpolationAndPropagation(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	start := Point{X: -2, Y: 4}
	end := Point{X: 8, Y: 14}

	g.Expect(Lerp(start, end, 0.25)).To(Equal(Point{X: 0.5, Y: 6.5}))
	g.Expect(Lerp(start, end, 0.5)).To(Equal(Midpoint(start, end)))
	g.Expect(Lerp(start, end, -1)).To(Equal(Point{X: -12, Y: -6}))
	g.Expect(Lerp(start, end, 1.5)).To(Equal(Point{X: 13, Y: 19}))

	nanPoint := Point{X: math.NaN(), Y: 1}
	infPoint := Point{X: 1, Y: math.Inf(1)}

	gotVector := start.VectorTo(nanPoint)
	gotPoint := start.Translate(Vector{X: math.NaN(), Y: 1})

	assertNaN(g, gotVector.X)
	g.Expect(gotVector.Y).To(Equal(-3.0))
	assertNaN(g, gotPoint.X)
	g.Expect(gotPoint.Y).To(Equal(5.0))
	g.Expect(math.IsNaN(nanPoint.DistanceTo(end))).To(BeTrue())
	g.Expect(math.IsNaN(end.DistanceTo(infPoint))).To(BeTrue())
	g.Expect(math.IsNaN(nanPoint.DistanceSquaredTo(end))).To(BeTrue())
	g.Expect(math.IsNaN(end.DistanceSquaredTo(infPoint))).To(BeTrue())
}

func TestPointDistanceSymmetry(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	a := Point{X: -3.5, Y: 10}
	b := Point{X: 6.25, Y: -2.75}

	g.Expect(a.DistanceTo(b)).To(Equal(b.DistanceTo(a)))
	g.Expect(a.DistanceSquaredTo(b)).To(Equal(b.DistanceSquaredTo(a)))
}

func TestPointMethodsDoNotMutateReceiver(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	original := Point{X: 3, Y: 4}
	other := Point{X: -1, Y: 2}
	want := original

	_ = original.Translate(Vector{X: 5, Y: -6})
	_ = original.VectorTo(other)
	_ = original.DistanceTo(other)
	_ = original.DistanceSquaredTo(other)
	_ = Midpoint(original, other)
	_ = Lerp(original, other, 0.25)

	g.Expect(original).To(Equal(want))
}

func assertNaN(g Gomega, got float64) {
	g.Expect(math.IsNaN(got)).To(BeTrue())
}
