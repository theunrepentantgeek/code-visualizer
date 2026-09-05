package geometry

import (
	"math"
	"testing"

	. "github.com/onsi/gomega"
)

func TestNewCircle(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	center := NewPoint(10, 20)

	g.Expect(NewCircle(center, 5)).To(Equal(Circle{Center: center, Radius: 5}))
}

func TestCircleValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		circle Circle
		want   bool
	}{
		{name: "zero value", circle: Circle{}, want: true},
		{name: "ordinary circle", circle: NewCircle(NewPoint(10, 20), 5), want: true},
		{name: "zero radius", circle: NewCircle(NewPoint(10, 20), 0), want: true},
		{name: "negative radius", circle: NewCircle(NewPoint(10, 20), -5), want: false},
		{name: "nan radius", circle: NewCircle(NewPoint(10, 20), math.NaN()), want: false},
		{
			name:   "positive infinite radius",
			circle: NewCircle(NewPoint(10, 20), math.Inf(1)),
			want:   false,
		},
		{
			name:   "negative infinite radius",
			circle: NewCircle(NewPoint(10, 20), math.Inf(-1)),
			want:   false,
		},
		{name: "invalid center", circle: NewCircle(NewPoint(math.NaN(), 20), 5), want: false},
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

	circle := NewCircle(NewPoint(10, 20), 5)

	tests := []struct {
		name  string
		point Point
		want  bool
	}{
		{name: "center", point: NewPoint(10, 20), want: true},
		{name: "interior point", point: NewPoint(12, 22), want: true},
		{name: "boundary point is inclusive", point: NewPoint(15, 20), want: true},
		{name: "just outside boundary", point: NewPoint(15.1, 20), want: false},
		{name: "far outside", point: NewPoint(100, 100), want: false},
		{name: "invalid point", point: NewPoint(math.NaN(), 20), want: false},
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
	invalid := NewCircle(NewPoint(10, 20), -5)

	g.Expect(invalid.Contains(NewPoint(10, 20))).To(BeFalse())
}

func TestCircleEncloses(t *testing.T) {
	t.Parallel()

	outer := NewCircle(NewPoint(0, 0), 10)

	tests := []struct {
		name  string
		other Circle
		want  bool
	}{
		{name: "identical circle", other: NewCircle(NewPoint(0, 0), 10), want: true},
		{name: "smaller concentric circle", other: NewCircle(NewPoint(0, 0), 5), want: true},
		{name: "smaller circle touching boundary", other: NewCircle(NewPoint(5, 0), 5), want: true},
		{name: "smaller circle exceeding boundary", other: NewCircle(NewPoint(6, 0), 5), want: false},
		{name: "larger circle", other: NewCircle(NewPoint(0, 0), 11), want: false},
		{name: "disjoint circle", other: NewCircle(NewPoint(100, 100), 1), want: false},
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
	valid := NewCircle(NewPoint(0, 0), 10)
	invalid := NewCircle(NewPoint(0, 0), -1)

	g.Expect(invalid.Encloses(valid)).To(BeFalse())
	g.Expect(valid.Encloses(invalid)).To(BeFalse())
}

func TestCircleIntersects(t *testing.T) {
	t.Parallel()

	circle := NewCircle(NewPoint(10, 20), 5)

	tests := []struct {
		name  string
		other Circle
		want  bool
	}{
		{name: "overlapping circle", other: NewCircle(NewPoint(12, 20), 5), want: true},
		{
			name:  "tangent circle counts as intersecting",
			other: NewCircle(NewPoint(20, 20), 5),
			want:  true,
		},
		{name: "disjoint circle", other: NewCircle(NewPoint(30, 20), 5), want: false},
		{name: "identical circle", other: NewCircle(NewPoint(10, 20), 5), want: true},
		{name: "enclosed circle", other: NewCircle(NewPoint(10, 20), 1), want: true},
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
	valid := NewCircle(NewPoint(0, 0), 10)
	invalid := NewCircle(NewPoint(0, 0), -1)

	g.Expect(invalid.Intersects(valid)).To(BeFalse())
	g.Expect(valid.Intersects(invalid)).To(BeFalse())
}

func TestCircleBounds(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	circle := NewCircle(NewPoint(10, 20), 5)

	g.Expect(circle.Bounds()).To(Equal(Rect{
		Min: NewPoint(5, 15),
		Max: NewPoint(15, 25),
	}))
}

func TestCircleTranslate(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	circle := NewCircle(NewPoint(10, 20), 5)

	got := circle.Translate(Vector{X: -3, Y: 7})

	g.Expect(got).To(Equal(NewCircle(NewPoint(7, 27), 5)))
}

func TestCircleTranslateByZeroVectorIsIdentity(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	circle := NewCircle(NewPoint(10, 20), 5)

	g.Expect(circle.Translate(Vector{})).To(Equal(circle))
}

func TestCirclePredicates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	circle := NewCircle(NewPoint(10, 20), 5)

	g.Expect(circle.Contains(NewPoint(15, 20))).To(BeTrue())
	g.Expect(circle.Contains(NewPoint(15.1, 20))).To(BeFalse())
	g.Expect(circle.Intersects(NewCircle(
		NewPoint(20, 20),
		5,
	))).To(BeTrue())
	g.Expect(circle.Bounds()).To(Equal(Rect{
		Min: NewPoint(5, 15),
		Max: NewPoint(15, 25),
	}))
}

func TestCircleMethodsDoNotMutateReceiver(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	original := NewCircle(NewPoint(10, 20), 5)
	want := original

	_ = original.Valid()
	_ = original.Contains(NewPoint(12, 20))
	_ = original.Encloses(NewCircle(NewPoint(10, 20), 2))
	_ = original.Intersects(NewCircle(NewPoint(20, 20), 5))
	_ = original.Bounds()
	_ = original.Translate(Vector{X: 1, Y: 1})

	g.Expect(original).To(Equal(want))
}
