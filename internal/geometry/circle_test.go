package geometry

import (
	"math"
	"testing"

	. "github.com/onsi/gomega"
)

func TestCircleValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		circle Circle
		want   bool
	}{
		{name: "zero value", circle: Circle{}, want: true},
		{name: "ordinary circle", circle: Circle{Center: Point{X: 10, Y: 20}, Radius: 5}, want: true},
		{name: "zero radius", circle: Circle{Center: Point{X: 10, Y: 20}, Radius: 0}, want: true},
		{name: "negative radius", circle: Circle{Center: Point{X: 10, Y: 20}, Radius: -5}, want: false},
		{name: "nan radius", circle: Circle{Center: Point{X: 10, Y: 20}, Radius: math.NaN()}, want: false},
		{
			name:   "positive infinite radius",
			circle: Circle{Center: Point{X: 10, Y: 20}, Radius: math.Inf(1)},
			want:   false,
		},
		{
			name:   "negative infinite radius",
			circle: Circle{Center: Point{X: 10, Y: 20}, Radius: math.Inf(-1)},
			want:   false,
		},
		{name: "invalid center", circle: Circle{Center: Point{X: math.NaN(), Y: 20}, Radius: 5}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			g.Expect(tt.circle.Valid()).To(Equal(tt.want))
		})
	}
}

func TestCircleContains(t *testing.T) {
	t.Parallel()

	circle := Circle{Center: Point{X: 10, Y: 20}, Radius: 5}

	tests := []struct {
		name  string
		point Point
		want  bool
	}{
		{name: "center", point: Point{X: 10, Y: 20}, want: true},
		{name: "interior point", point: Point{X: 12, Y: 22}, want: true},
		{name: "boundary point is inclusive", point: Point{X: 15, Y: 20}, want: true},
		{name: "just outside boundary", point: Point{X: 15.1, Y: 20}, want: false},
		{name: "far outside", point: Point{X: 100, Y: 100}, want: false},
		{name: "invalid point", point: Point{X: math.NaN(), Y: 20}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			g.Expect(circle.Contains(tt.point)).To(Equal(tt.want))
		})
	}
}

func TestCircleContainsInvalidCircle(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	invalid := Circle{Center: Point{X: 10, Y: 20}, Radius: -5}

	g.Expect(invalid.Contains(Point{X: 10, Y: 20})).To(BeFalse())
}

func TestCircleEncloses(t *testing.T) {
	t.Parallel()

	outer := Circle{Center: Point{X: 0, Y: 0}, Radius: 10}

	tests := []struct {
		name  string
		other Circle
		want  bool
	}{
		{name: "identical circle", other: Circle{Center: Point{X: 0, Y: 0}, Radius: 10}, want: true},
		{name: "smaller concentric circle", other: Circle{Center: Point{X: 0, Y: 0}, Radius: 5}, want: true},
		{name: "smaller circle touching boundary", other: Circle{Center: Point{X: 5, Y: 0}, Radius: 5}, want: true},
		{name: "smaller circle exceeding boundary", other: Circle{Center: Point{X: 6, Y: 0}, Radius: 5}, want: false},
		{name: "larger circle", other: Circle{Center: Point{X: 0, Y: 0}, Radius: 11}, want: false},
		{name: "disjoint circle", other: Circle{Center: Point{X: 100, Y: 100}, Radius: 1}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			g.Expect(outer.Encloses(tt.other)).To(Equal(tt.want))
		})
	}
}

func TestCircleEnclosesInvalidOperands(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	valid := Circle{Center: Point{X: 0, Y: 0}, Radius: 10}
	invalid := Circle{Center: Point{X: 0, Y: 0}, Radius: -1}

	g.Expect(invalid.Encloses(valid)).To(BeFalse())
	g.Expect(valid.Encloses(invalid)).To(BeFalse())
}

func TestCircleIntersects(t *testing.T) {
	t.Parallel()

	circle := Circle{Center: Point{X: 10, Y: 20}, Radius: 5}

	tests := []struct {
		name  string
		other Circle
		want  bool
	}{
		{name: "overlapping circle", other: Circle{Center: Point{X: 12, Y: 20}, Radius: 5}, want: true},
		{
			name:  "tangent circle counts as intersecting",
			other: Circle{Center: Point{X: 20, Y: 20}, Radius: 5},
			want:  true,
		},
		{name: "disjoint circle", other: Circle{Center: Point{X: 30, Y: 20}, Radius: 5}, want: false},
		{name: "identical circle", other: Circle{Center: Point{X: 10, Y: 20}, Radius: 5}, want: true},
		{name: "enclosed circle", other: Circle{Center: Point{X: 10, Y: 20}, Radius: 1}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			g.Expect(circle.Intersects(tt.other)).To(Equal(tt.want))
		})
	}
}

func TestCircleIntersectsInvalidOperands(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	valid := Circle{Center: Point{X: 0, Y: 0}, Radius: 10}
	invalid := Circle{Center: Point{X: 0, Y: 0}, Radius: -1}

	g.Expect(invalid.Intersects(valid)).To(BeFalse())
	g.Expect(valid.Intersects(invalid)).To(BeFalse())
}

func TestCircleBounds(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	circle := Circle{Center: Point{X: 10, Y: 20}, Radius: 5}

	g.Expect(circle.Bounds()).To(Equal(Rect{
		Min: Point{X: 5, Y: 15},
		Max: Point{X: 15, Y: 25},
	}))
}

func TestCircleTranslate(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	circle := Circle{Center: Point{X: 10, Y: 20}, Radius: 5}

	got := circle.Translate(Vector{X: -3, Y: 7})

	g.Expect(got).To(Equal(Circle{Center: Point{X: 7, Y: 27}, Radius: 5}))
}

func TestCircleTranslateByZeroVectorIsIdentity(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	circle := Circle{Center: Point{X: 10, Y: 20}, Radius: 5}

	g.Expect(circle.Translate(Vector{})).To(Equal(circle))
}

func TestCirclePredicates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	circle := Circle{Center: Point{X: 10, Y: 20}, Radius: 5}

	g.Expect(circle.Contains(Point{X: 15, Y: 20})).To(BeTrue())
	g.Expect(circle.Contains(Point{X: 15.1, Y: 20})).To(BeFalse())
	g.Expect(circle.Intersects(Circle{
		Center: Point{X: 20, Y: 20},
		Radius: 5,
	})).To(BeTrue())
	g.Expect(circle.Bounds()).To(Equal(Rect{
		Min: Point{X: 5, Y: 15},
		Max: Point{X: 15, Y: 25},
	}))
}

func TestCircleMethodsDoNotMutateReceiver(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	original := Circle{Center: Point{X: 10, Y: 20}, Radius: 5}
	want := original

	_ = original.Valid()
	_ = original.Contains(Point{X: 12, Y: 20})
	_ = original.Encloses(Circle{Center: Point{X: 10, Y: 20}, Radius: 2})
	_ = original.Intersects(Circle{Center: Point{X: 20, Y: 20}, Radius: 5})
	_ = original.Bounds()
	_ = original.Translate(Vector{X: 1, Y: 1})

	g.Expect(original).To(Equal(want))
}
