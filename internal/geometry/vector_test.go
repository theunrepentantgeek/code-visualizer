package geometry

import (
	"math"
	"testing"

	. "github.com/onsi/gomega"
)

const vectorTolerance = 1e-12

func TestZeroVector(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(ZeroVector).To(Equal(Vector{X: 0, Y: 0}))
}

func TestNewVector(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		x, y float64
		want Vector
	}{
		"positive components": {x: 1.5, y: 2.5, want: Vector{X: 1.5, Y: 2.5}},
		"negative components": {x: -3, y: -4, want: Vector{X: -3, Y: -4}},
		"zero components":     {x: 0, y: 0, want: Vector{X: 0, Y: 0}},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			g.Expect(NewVector(tt.x, tt.y)).To(Equal(tt.want))
		})
	}
}

func TestNewRadialVector(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		angle, length float64
		want          Vector
	}{
		"angle zero": {angle: 0, length: 5, want: Vector{X: 5, Y: 0}},
		"quarter turn": {
			angle: math.Pi / 2, length: 5,
			want: Vector{X: 0, Y: 5},
		},
		"half turn": {angle: math.Pi, length: 5, want: Vector{X: -5, Y: 0}},
		"three quarter turn": {
			angle: -math.Pi / 2, length: 5,
			want: Vector{X: 0, Y: -5},
		},
		"representative angle and length": {
			angle: math.Pi / 4, length: math.Sqrt2,
			want: Vector{X: 1, Y: 1},
		},
		"negative length reverses direction": {
			angle: 0, length: -5,
			want: Vector{X: -5, Y: 0},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			assertVectorClose(g, NewRadialVector(tt.angle, tt.length), tt.want)
		})
	}
}

func TestVectorValid(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		vector Vector
		want   bool
	}{
		"finite zero":         {vector: Vector{}, want: true},
		"finite components":   {vector: Vector{X: 1.5, Y: -2.5}, want: true},
		"nan x":               {vector: Vector{X: math.NaN(), Y: 1}, want: false},
		"nan y":               {vector: Vector{X: 1, Y: math.NaN()}, want: false},
		"positive infinity x": {vector: Vector{X: math.Inf(1), Y: 1}, want: false},
		"negative infinity y": {vector: Vector{X: 1, Y: math.Inf(-1)}, want: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			g.Expect(tt.vector.Valid()).To(Equal(tt.want))
		})
	}
}

func TestVectorAdd(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		left, right, want Vector
	}{
		"adds positive and negative components": {
			left: Vector{X: 1, Y: 2}, right: Vector{X: -3, Y: 4}, want: Vector{X: -2, Y: 6},
		},
		"adds zero vector": {
			left: Vector{X: -5.5, Y: 8.25}, right: Vector{}, want: Vector{X: -5.5, Y: 8.25},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			assertVectorClose(g, tt.left.Add(tt.right), tt.want)
		})
	}
}

func TestVectorSubtract(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		left, right, want Vector
	}{
		"subtracts positive and negative components": {
			left: Vector{X: 1, Y: 2}, right: Vector{X: -3, Y: 4}, want: Vector{X: 4, Y: -2},
		},
		"subtracts zero vector": {
			left: Vector{X: -5.5, Y: 8.25}, right: Vector{}, want: Vector{X: -5.5, Y: 8.25},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			assertVectorClose(g, tt.left.Subtract(tt.right), tt.want)
		})
	}
}

func TestVectorScale(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		vector Vector
		factor float64
		want   Vector
	}{
		"scales by positive factor": {vector: Vector{X: 2, Y: -3}, factor: 2.5, want: Vector{X: 5, Y: -7.5}},
		"scales by zero":            {vector: Vector{X: 2, Y: -3}, factor: 0, want: Vector{}},
		"scales by negative factor": {vector: Vector{X: 2, Y: -3}, factor: -1, want: Vector{X: -2, Y: 3}},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			assertVectorClose(g, tt.vector.Scale(tt.factor), tt.want)
		})
	}
}

func TestVectorDot(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		left, right Vector
		want        float64
	}{
		"positive and negative components": {left: Vector{X: 1, Y: 2}, right: Vector{X: -3, Y: 4}, want: 5},
		"orthogonal vectors":               {left: Vector{X: 3, Y: 0}, right: Vector{X: 0, Y: -7}, want: 0},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			assertFloatClose(g, tt.left.Dot(tt.right), tt.want)
		})
	}
}

func TestVectorLengthSquared(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		vector Vector
		want   float64
	}{
		"zero vector":            {vector: Vector{}, want: 0},
		"three four five vector": {vector: Vector{X: 3, Y: 4}, want: 25},
		"negative components":    {vector: Vector{X: -6, Y: 8}, want: 100},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			assertFloatClose(g, tt.vector.LengthSquared(), tt.want)
		})
	}
}

func TestVectorLength(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		vector Vector
		want   float64
	}{
		"zero vector":            {vector: Vector{}, want: 0},
		"three four five vector": {vector: Vector{X: 3, Y: 4}, want: 5},
		"negative components":    {vector: Vector{X: -6, Y: 8}, want: 10},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			assertFloatClose(g, tt.vector.Length(), tt.want)
		})
	}
}

func TestVectorUnit(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		vector     Vector
		want       Vector
		wantOK     bool
		wantLength float64
	}{
		"three four five vector": {
			vector:     Vector{X: 3, Y: 4},
			want:       Vector{X: 0.6, Y: 0.8},
			wantOK:     true,
			wantLength: 1,
		},
		"zero vector":   {vector: Vector{}, want: Vector{}, wantOK: false},
		"nan component": {vector: Vector{X: math.NaN(), Y: 1}, want: Vector{}, wantOK: false},
		"infinite component": {
			vector: Vector{X: 1, Y: math.Inf(1)}, want: Vector{}, wantOK: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			got, ok := tt.vector.Unit()
			g.Expect(ok).To(Equal(tt.wantOK))
			assertVectorClose(g, got, tt.want)

			if tt.wantOK {
				assertFloatClose(g, got.Length(), tt.wantLength)
			}
		})
	}
}

func TestVectorUnitExtremeFiniteValues(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		vector Vector
	}{
		"max float64 components": {vector: Vector{X: math.MaxFloat64, Y: math.MaxFloat64}},
		"smallest nonzero float64 components": {
			vector: Vector{X: math.SmallestNonzeroFloat64, Y: math.SmallestNonzeroFloat64},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			got, ok := tt.vector.Unit()
			g.Expect(ok).To(BeTrue())
			g.Expect(got.Valid()).To(BeTrue())
			assertFloatClose(g, got.Length(), 1)
		})
	}
}

func TestVectorAlgebraicIdentities(t *testing.T) {
	t.Parallel()

	v := Vector{X: 2, Y: -5}
	w := Vector{X: -7, Y: 11}

	tests := map[string]struct {
		check func(g Gomega)
	}{
		"subtracting addend recovers original vector": {
			check: func(g Gomega) {
				assertVectorClose(g, v.Add(w).Subtract(w), v)
			},
		},
		"subtracting self yields zero vector": {
			check: func(g Gomega) {
				assertVectorClose(g, v.Subtract(v), ZeroVector)
			},
		},
		"self dot product matches length squared": {
			check: func(g Gomega) {
				assertFloatClose(g, v.Dot(v), v.LengthSquared())
			},
		},
		"scaling by one preserves vector": {
			check: func(g Gomega) {
				assertVectorClose(g, v.Scale(1), v)
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

func TestVectorMethodsDoNotMutateReceiver(t *testing.T) {
	t.Parallel()

	original := Vector{X: 3, Y: 4}
	other := Vector{X: -1, Y: 2}
	want := original

	tests := map[string]struct {
		call func()
	}{
		"Add":           {call: func() { _ = original.Add(other) }},
		"Subtract":      {call: func() { _ = original.Subtract(other) }},
		"Scale":         {call: func() { _ = original.Scale(2) }},
		"Dot":           {call: func() { _ = original.Dot(other) }},
		"Length":        {call: func() { _ = original.Length() }},
		"LengthSquared": {call: func() { _ = original.LengthSquared() }},
		"Unit":          {call: func() { _, _ = original.Unit() }},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			tt.call()
			assertVectorClose(g, original, want)
		})
	}
}

// assertVectorClose asserts that got and want are equal within vectorTolerance,
// component-wise.
func assertVectorClose(g Gomega, got, want Vector) {
	assertFloatClose(g, got.X, want.X)
	assertFloatClose(g, got.Y, want.Y)
}

// assertFloatClose asserts that got and want are both finite and equal within
// vectorTolerance.
func assertFloatClose(g Gomega, got, want float64) {
	g.Expect(math.IsNaN(got)).To(BeFalse(), "got must be finite, was %v", got)
	g.Expect(math.IsNaN(want)).To(BeFalse(), "want must be finite, was %v", want)
	g.Expect(math.IsInf(got, 0)).To(BeFalse(), "got must be finite, was %v", got)
	g.Expect(math.IsInf(want, 0)).To(BeFalse(), "want must be finite, was %v", want)
	g.Expect(got).To(BeNumerically("~", want, vectorTolerance))
}
