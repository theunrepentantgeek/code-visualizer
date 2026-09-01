package geometry

import (
	"math"
	"testing"
)

const vectorTolerance = 1e-12

func TestVectorValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		vector Vector
		want   bool
	}{
		{name: "finite zero", vector: Vector{}, want: true},
		{name: "finite components", vector: Vector{X: 1.5, Y: -2.5}, want: true},
		{name: "nan x", vector: Vector{X: math.NaN(), Y: 1}, want: false},
		{name: "nan y", vector: Vector{X: 1, Y: math.NaN()}, want: false},
		{name: "positive infinity x", vector: Vector{X: math.Inf(1), Y: 1}, want: false},
		{name: "negative infinity y", vector: Vector{X: 1, Y: math.Inf(-1)}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.vector.Valid(); got != tt.want {
				t.Fatalf("Valid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVectorAdd(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		left  Vector
		right Vector
		want  Vector
	}{
		{
			name:  "adds positive and negative components",
			left:  Vector{X: 1, Y: 2},
			right: Vector{X: -3, Y: 4},
			want:  Vector{X: -2, Y: 6},
		},
		{name: "adds zero vector", left: Vector{X: -5.5, Y: 8.25}, right: Vector{}, want: Vector{X: -5.5, Y: 8.25}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertVectorClose(t, tt.left.Add(tt.right), tt.want)
		})
	}
}

func TestVectorSubtract(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		left  Vector
		right Vector
		want  Vector
	}{
		{
			name:  "subtracts positive and negative components",
			left:  Vector{X: 1, Y: 2},
			right: Vector{X: -3, Y: 4},
			want:  Vector{X: 4, Y: -2},
		},
		{
			name:  "subtracts zero vector",
			left:  Vector{X: -5.5, Y: 8.25},
			right: Vector{},
			want:  Vector{X: -5.5, Y: 8.25},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertVectorClose(t, tt.left.Subtract(tt.right), tt.want)
		})
	}
}

func TestVectorScale(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		vector Vector
		factor float64
		want   Vector
	}{
		{name: "scales by positive factor", vector: Vector{X: 2, Y: -3}, factor: 2.5, want: Vector{X: 5, Y: -7.5}},
		{name: "scales by zero", vector: Vector{X: 2, Y: -3}, factor: 0, want: Vector{}},
		{name: "scales by negative factor", vector: Vector{X: 2, Y: -3}, factor: -1, want: Vector{X: -2, Y: 3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertVectorClose(t, tt.vector.Scale(tt.factor), tt.want)
		})
	}
}

func TestVectorDot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		left  Vector
		right Vector
		want  float64
	}{
		{name: "positive and negative components", left: Vector{X: 1, Y: 2}, right: Vector{X: -3, Y: 4}, want: 5},
		{name: "orthogonal vectors", left: Vector{X: 3, Y: 0}, right: Vector{X: 0, Y: -7}, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertFloatClose(t, tt.left.Dot(tt.right), tt.want)
		})
	}
}

func TestVectorLengthSquared(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		vector Vector
		want   float64
	}{
		{name: "zero vector", vector: Vector{}, want: 0},
		{name: "three four five vector", vector: Vector{X: 3, Y: 4}, want: 25},
		{name: "negative components", vector: Vector{X: -6, Y: 8}, want: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertFloatClose(t, tt.vector.LengthSquared(), tt.want)
		})
	}
}

func TestVectorLength(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		vector Vector
		want   float64
	}{
		{name: "zero vector", vector: Vector{}, want: 0},
		{name: "three four five vector", vector: Vector{X: 3, Y: 4}, want: 5},
		{name: "negative components", vector: Vector{X: -6, Y: 8}, want: 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertFloatClose(t, tt.vector.Length(), tt.want)
		})
	}
}

func TestVectorUnit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		vector     Vector
		want       Vector
		wantOK     bool
		wantLength float64
	}{
		{
			name:       "three four five vector",
			vector:     Vector{X: 3, Y: 4},
			want:       Vector{X: 0.6, Y: 0.8},
			wantOK:     true,
			wantLength: 1,
		},
		{name: "zero vector", vector: Vector{}, want: Vector{}, wantOK: false},
		{name: "nan component", vector: Vector{X: math.NaN(), Y: 1}, want: Vector{}, wantOK: false},
		{name: "infinite component", vector: Vector{X: 1, Y: math.Inf(1)}, want: Vector{}, wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := tt.vector.Unit()
			if ok != tt.wantOK {
				t.Fatalf("Unit() ok = %v, want %v (got %v)", ok, tt.wantOK, got)
			}

			assertVectorClose(t, got, tt.want)

			if tt.wantOK {
				assertFloatClose(t, got.Length(), tt.wantLength)
			}
		})
	}
}

func TestVectorUnitExtremeFiniteValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		vector Vector
	}{
		{name: "max float64 components", vector: Vector{X: math.MaxFloat64, Y: math.MaxFloat64}},
		{
			name:   "smallest nonzero float64 components",
			vector: Vector{X: math.SmallestNonzeroFloat64, Y: math.SmallestNonzeroFloat64},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := tt.vector.Unit()
			if !ok {
				t.Fatalf("Unit() ok = false, want true (got %v)", got)
			}

			if !got.Valid() {
				t.Fatalf("Unit() produced invalid vector %v", got)
			}

			assertFloatClose(t, got.Length(), 1)
		})
	}
}

func TestVectorAlgebraicIdentities(t *testing.T) {
	t.Parallel()

	v := Vector{X: 2, Y: -5}
	w := Vector{X: -7, Y: 11}

	tests := []struct {
		name  string
		check func(t *testing.T)
	}{
		{
			name: "subtracting addend recovers original vector",
			check: func(t *testing.T) {
				t.Helper()
				assertVectorClose(t, v.Add(w).Subtract(w), v)
			},
		},
		{
			name: "subtracting self yields zero vector",
			check: func(t *testing.T) {
				t.Helper()
				assertVectorClose(t, v.Subtract(v), Vector{})
			},
		},
		{
			name: "self dot product matches length squared",
			check: func(t *testing.T) {
				t.Helper()
				assertFloatClose(t, v.Dot(v), v.LengthSquared())
			},
		},
		{
			name: "scaling by one preserves vector",
			check: func(t *testing.T) {
				t.Helper()
				assertVectorClose(t, v.Scale(1), v)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.check(t)
		})
	}
}

func TestVectorMethodsDoNotMutateReceiver(t *testing.T) {
	t.Parallel()

	original := Vector{X: 3, Y: 4}
	other := Vector{X: -1, Y: 2}
	want := original

	tests := []struct {
		name string
		call func()
	}{
		{name: "Add", call: func() { _ = original.Add(other) }},
		{name: "Subtract", call: func() { _ = original.Subtract(other) }},
		{name: "Scale", call: func() { _ = original.Scale(2) }},
		{name: "Dot", call: func() { _ = original.Dot(other) }},
		{name: "Length", call: func() { _ = original.Length() }},
		{name: "LengthSquared", call: func() { _ = original.LengthSquared() }},
		{name: "Unit", call: func() { _, _ = original.Unit() }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.call()
			assertVectorClose(t, original, want)
		})
	}
}

func assertVectorClose(t *testing.T, got, want Vector) {
	t.Helper()

	assertFloatClose(t, got.X, want.X)
	assertFloatClose(t, got.Y, want.Y)
}

func assertFloatClose(t *testing.T, got, want float64) {
	t.Helper()

	if math.IsNaN(got) || math.IsNaN(want) {
		t.Fatalf("expected finite floats, got=%v want=%v", got, want)
	}

	if math.IsInf(got, 0) || math.IsInf(want, 0) {
		t.Fatalf("expected finite floats, got=%v want=%v", got, want)
	}

	if math.Abs(got-want) > vectorTolerance {
		t.Fatalf("got %v, want %v", got, want)
	}
}
