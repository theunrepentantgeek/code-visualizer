package geometry

import (
	"math"
	"testing"
)

func TestPointValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		point Point
		want  bool
	}{
		{name: "zero point", point: Point{}, want: true},
		{name: "finite coordinates", point: Point{X: 1.5, Y: -2.5}, want: true},
		{name: "nan x", point: Point{X: math.NaN(), Y: 1}, want: false},
		{name: "nan y", point: Point{X: 1, Y: math.NaN()}, want: false},
		{name: "positive infinity x", point: Point{X: math.Inf(1), Y: 1}, want: false},
		{name: "negative infinity x", point: Point{X: math.Inf(-1), Y: 1}, want: false},
		{name: "positive infinity y", point: Point{X: 1, Y: math.Inf(1)}, want: false},
		{name: "negative infinity y", point: Point{X: 1, Y: math.Inf(-1)}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.point.Valid(); got != tt.want {
				t.Fatalf("Valid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPointTranslate(t *testing.T) {
	t.Parallel()

	point := Point{X: 2, Y: -3}
	vector := Vector{X: -5, Y: 7}

	assertPointClose(t, point.Translate(vector), Point{X: -3, Y: 4})
}

func TestPointVectorTo(t *testing.T) {
	t.Parallel()

	from := Point{X: 2, Y: -3}
	to := Point{X: -5, Y: 7}

	assertVectorClose(t, from.VectorTo(to), Vector{X: -7, Y: 10})
}

func TestPointDistances(t *testing.T) {
	t.Parallel()

	from := Point{X: -1, Y: 2}
	to := Point{X: 2, Y: 6}

	assertFloatClose(t, from.DistanceSquaredTo(to), 25)
	assertFloatClose(t, from.DistanceTo(to), 5)
}

func TestPointDistancesAreSymmetric(t *testing.T) {
	t.Parallel()

	a := Point{X: -3.5, Y: 8}
	b := Point{X: 10, Y: -4.25}

	assertFloatClose(t, a.DistanceSquaredTo(b), b.DistanceSquaredTo(a))
	assertFloatClose(t, a.DistanceTo(b), b.DistanceTo(a))
}

func TestMidpoint(t *testing.T) {
	t.Parallel()

	a := Point{X: -2, Y: 3}
	b := Point{X: 8, Y: -5}

	assertPointClose(t, Midpoint(a, b), Point{X: 3, Y: -1})
}

func TestLerp(t *testing.T) {
	t.Parallel()

	a := Point{X: -2, Y: 3}
	b := Point{X: 8, Y: -5}

	tests := []struct {
		name     string
		fraction float64
		want     Point
	}{
		{name: "start endpoint", fraction: 0, want: a},
		{name: "quarter", fraction: 0.25, want: Point{X: 0.5, Y: 1}},
		{name: "end endpoint", fraction: 1, want: b},
		{name: "extrapolation", fraction: 1.5, want: Point{X: 13, Y: -9}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertPointClose(t, Lerp(a, b, tt.fraction), tt.want)
		})
	}
}

func TestLerpExtremeFiniteEndpoints(t *testing.T) {
	t.Parallel()

	a := Point{X: -math.MaxFloat64, Y: math.MaxFloat64}
	b := Point{X: math.MaxFloat64, Y: -math.MaxFloat64}

	if got := Lerp(a, b, 0); got != a {
		t.Fatalf("Lerp(a, b, 0) = %v, want exact endpoint %v", got, a)
	}
	if got := Lerp(a, b, 1); got != b {
		t.Fatalf("Lerp(a, b, 1) = %v, want exact endpoint %v", got, b)
	}
}

func TestMidpointOppositeExtremeFinitePoints(t *testing.T) {
	t.Parallel()

	a := Point{X: -math.MaxFloat64, Y: math.MaxFloat64}
	b := Point{X: math.MaxFloat64, Y: -math.MaxFloat64}
	got := Midpoint(a, b)

	if !got.Valid() || got != (Point{}) {
		t.Fatalf("Midpoint(a, b) = %v, want finite origin", got)
	}
}

func TestPointInvalidValuesPropagateThroughCoordinateOperations(t *testing.T) {
	t.Parallel()

	nanPoint := Point{X: math.NaN(), Y: 2}
	infinitePoint := Point{X: 1, Y: math.Inf(1)}
	finitePoint := Point{X: 3, Y: 4}

	if got := nanPoint.Translate(Vector{}); !math.IsNaN(got.X) {
		t.Fatalf("Translate() X = %v, want NaN", got.X)
	}

	if got := finitePoint.VectorTo(infinitePoint); !math.IsInf(got.Y, 1) {
		t.Fatalf("VectorTo() Y = %v, want +Inf", got.Y)
	}

	if got := Midpoint(finitePoint, nanPoint); !math.IsNaN(got.X) {
		t.Fatalf("Midpoint() X = %v, want NaN", got.X)
	}

	if got := Lerp(finitePoint, infinitePoint, 0.5); !math.IsInf(got.Y, 1) {
		t.Fatalf("Lerp() Y = %v, want +Inf", got.Y)
	}
}

func TestPointInvalidValuesPropagateThroughDistanceOperations(t *testing.T) {
	t.Parallel()

	nanPoint := Point{X: math.NaN(), Y: 2}
	infinitePoint := Point{X: 1, Y: math.Inf(1)}
	finitePoint := Point{X: 3, Y: 4}

	invalidPoints := []Point{nanPoint, infinitePoint}
	for _, invalid := range invalidPoints {
		assertInvalidPointDistances(t, finitePoint, invalid)
	}
}

func assertInvalidPointDistances(t *testing.T, finite, invalid Point) {
	t.Helper()

	if got := finite.DistanceSquaredTo(invalid); !math.IsNaN(got) {
		t.Fatalf("DistanceSquaredTo(%v) = %v, want NaN", invalid, got)
	}

	if got := invalid.DistanceSquaredTo(finite); !math.IsNaN(got) {
		t.Fatalf("%v.DistanceSquaredTo() = %v, want NaN", invalid, got)
	}

	if got := finite.DistanceTo(invalid); !math.IsNaN(got) {
		t.Fatalf("DistanceTo(%v) = %v, want NaN", invalid, got)
	}

	if got := invalid.DistanceTo(finite); !math.IsNaN(got) {
		t.Fatalf("%v.DistanceTo() = %v, want NaN", invalid, got)
	}
}

func TestPointOperationsDoNotMutateInputs(t *testing.T) {
	t.Parallel()

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

	assertPointClose(t, a, wantA)
	assertPointClose(t, b, wantB)
	assertVectorClose(t, vector, wantVector)
}

func assertPointClose(t *testing.T, got, want Point) {
	t.Helper()

	assertFloatClose(t, got.X, want.X)
	assertFloatClose(t, got.Y, want.Y)
}
