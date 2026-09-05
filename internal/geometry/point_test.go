package geometry

import (
	"math"
	"testing"

	. "github.com/onsi/gomega"
)

func TestOriginPoint(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(OriginPoint).To(Equal(Point{X: 0, Y: 0}))
}

func TestNewPoint(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		x, y float64
		want Point
	}{
		"positive components": {x: 1.5, y: 2.5, want: Point{X: 1.5, Y: 2.5}},
		"negative components": {x: -3, y: -4, want: Point{X: -3, Y: -4}},
		"zero components":     {x: 0, y: 0, want: Point{X: 0, Y: 0}},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			g.Expect(NewPoint(tt.x, tt.y)).To(Equal(tt.want))
		})
	}
}

func TestPointValid(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		point Point
		want  bool
	}{
		"zero point":          {point: Point{}, want: true},
		"finite coordinates":  {point: Point{X: 1.5, Y: -2.5}, want: true},
		"nan x":               {point: Point{X: math.NaN(), Y: 1}, want: false},
		"nan y":               {point: Point{X: 1, Y: math.NaN()}, want: false},
		"positive infinity x": {point: Point{X: math.Inf(1), Y: 1}, want: false},
		"negative infinity x": {point: Point{X: math.Inf(-1), Y: 1}, want: false},
		"positive infinity y": {point: Point{X: 1, Y: math.Inf(1)}, want: false},
		"negative infinity y": {point: Point{X: 1, Y: math.Inf(-1)}, want: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			g.Expect(tt.point.Valid()).To(Equal(tt.want))
		})
	}
}

func TestPointTranslate(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	point := Point{X: 2, Y: -3}
	vector := Vector{X: -5, Y: 7}

	assertPointClose(g, point.Translate(vector), Point{X: -3, Y: 4})
}

func TestPointVectorTo(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	from := Point{X: 2, Y: -3}
	to := Point{X: -5, Y: 7}

	assertVectorClose(g, from.VectorTo(to), Vector{X: -7, Y: 10})
}

func TestPointDistances(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	from := Point{X: -1, Y: 2}
	to := Point{X: 2, Y: 6}

	assertFloatClose(g, from.DistanceSquaredTo(to), 25)
	assertFloatClose(g, from.DistanceTo(to), 5)
}

func TestPointDistancesAreSymmetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	a := Point{X: -3.5, Y: 8}
	b := Point{X: 10, Y: -4.25}

	assertFloatClose(g, a.DistanceSquaredTo(b), b.DistanceSquaredTo(a))
	assertFloatClose(g, a.DistanceTo(b), b.DistanceTo(a))
}

func TestMidpoint(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	a := Point{X: -2, Y: 3}
	b := Point{X: 8, Y: -5}

	assertPointClose(g, Midpoint(a, b), Point{X: 3, Y: -1})
}

func TestLerp(t *testing.T) {
	t.Parallel()

	a := Point{X: -2, Y: 3}
	b := Point{X: 8, Y: -5}

	tests := map[string]struct {
		fraction float64
		want     Point
	}{
		"start endpoint": {fraction: 0, want: a},
		"quarter":        {fraction: 0.25, want: Point{X: 0.5, Y: 1}},
		"end endpoint":   {fraction: 1, want: b},
		"extrapolation":  {fraction: 1.5, want: Point{X: 13, Y: -9}},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			assertPointClose(g, Lerp(a, b, tt.fraction), tt.want)
		})
	}
}

func TestLerpExtremeFiniteEndpoints(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	a := Point{X: -math.MaxFloat64, Y: math.MaxFloat64}
	b := Point{X: math.MaxFloat64, Y: -math.MaxFloat64}

	g.Expect(Lerp(a, b, 0)).To(Equal(a), "Lerp(a, b, 0) must return the exact start endpoint")
	g.Expect(Lerp(a, b, 1)).To(Equal(b), "Lerp(a, b, 1) must return the exact end endpoint")
}

func TestMidpointOppositeExtremeFinitePoints(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	a := Point{X: -math.MaxFloat64, Y: math.MaxFloat64}
	b := Point{X: math.MaxFloat64, Y: -math.MaxFloat64}
	got := Midpoint(a, b)

	g.Expect(got.Valid()).To(BeTrue())
	g.Expect(got).To(Equal(OriginPoint), "Midpoint of opposite extreme finite points must be the finite origin")
}

func TestPointInvalidValuesPropagateThroughCoordinateOperations(t *testing.T) {
	t.Parallel()

	nanPoint := Point{X: math.NaN(), Y: 2}
	infinitePoint := Point{X: 1, Y: math.Inf(1)}
	finitePoint := Point{X: 3, Y: 4}

	tests := map[string]struct {
		check func(g Gomega)
	}{
		"Translate propagates NaN": {
			check: func(g Gomega) {
				got := nanPoint.Translate(ZeroVector)
				g.Expect(math.IsNaN(got.X)).To(BeTrue(), "Translate() X = %v, want NaN", got.X)
			},
		},
		"VectorTo propagates positive infinity": {
			check: func(g Gomega) {
				got := finitePoint.VectorTo(infinitePoint)
				g.Expect(math.IsInf(got.Y, 1)).To(BeTrue(), "VectorTo() Y = %v, want +Inf", got.Y)
			},
		},
		"Midpoint propagates NaN": {
			check: func(g Gomega) {
				got := Midpoint(finitePoint, nanPoint)
				g.Expect(math.IsNaN(got.X)).To(BeTrue(), "Midpoint() X = %v, want NaN", got.X)
			},
		},
		"Lerp propagates positive infinity": {
			check: func(g Gomega) {
				got := Lerp(finitePoint, infinitePoint, 0.5)
				g.Expect(math.IsInf(got.Y, 1)).To(BeTrue(), "Lerp() Y = %v, want +Inf", got.Y)
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)
			tt.check(g)
		})
	}
}

func TestPointInvalidValuesPropagateThroughDistanceOperations(t *testing.T) {
	t.Parallel()

	nanPoint := Point{X: math.NaN(), Y: 2}
	infinitePoint := Point{X: 1, Y: math.Inf(1)}
	finitePoint := Point{X: 3, Y: 4}

	tests := map[string]struct {
		invalid Point
	}{
		"nan point":      {invalid: nanPoint},
		"infinite point": {invalid: infinitePoint},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			assertInvalidPointDistances(g, finitePoint, tt.invalid)
		})
	}
}

func assertInvalidPointDistances(g Gomega, finite, invalid Point) {
	g.Expect(math.IsNaN(finite.DistanceSquaredTo(invalid))).To(BeTrue(),
		"DistanceSquaredTo(%v) = %v, want NaN", invalid, finite.DistanceSquaredTo(invalid))
	g.Expect(math.IsNaN(invalid.DistanceSquaredTo(finite))).To(BeTrue(),
		"%v.DistanceSquaredTo() = %v, want NaN", invalid, invalid.DistanceSquaredTo(finite))
	g.Expect(math.IsNaN(finite.DistanceTo(invalid))).To(BeTrue(),
		"DistanceTo(%v) = %v, want NaN", invalid, finite.DistanceTo(invalid))
	g.Expect(math.IsNaN(invalid.DistanceTo(finite))).To(BeTrue(),
		"%v.DistanceTo() = %v, want NaN", invalid, invalid.DistanceTo(finite))
}

func TestPointOperationsDoNotMutateInputs(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	a := Point{X: 2, Y: -3}
	b := Point{X: -5, Y: 7}
	vector := Vector{X: 11, Y: 13}
	wantA, wantB, wantVector := a, b, vector

	_ = a.Valid()
	_ = a.Translate(vector)
	_ = a.VectorTo(b)
	_ = a.DistanceSquaredTo(b)
	_ = a.DistanceTo(b)
	_ = Midpoint(a, b)
	_ = Lerp(a, b, 0.25)

	assertPointClose(g, a, wantA)
	assertPointClose(g, b, wantB)
	assertVectorClose(g, vector, wantVector)
}

// assertPointClose asserts that got and want are equal within vectorTolerance,
// component-wise.
func assertPointClose(g Gomega, got, want Point) {
	assertFloatClose(g, got.X, want.X)
	assertFloatClose(g, got.Y, want.Y)
}
