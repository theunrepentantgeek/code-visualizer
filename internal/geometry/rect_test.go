package geometry

import (
	"math"
	"testing"

	. "github.com/onsi/gomega"
)

func TestRectFromPositionSize(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	got := RectFromPositionSize(Point{X: 10, Y: 20}, Size{Width: 30, Height: 40})

	g.Expect(got).To(Equal(Rect{
		Min: Point{X: 10, Y: 20},
		Max: Point{X: 40, Y: 60},
	}))
}

func TestRectFromPositionSizeDoesNotRepairInvalidInput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// A negative size produces an unordered, invalid rectangle rather than
	// being silently repaired.
	got := RectFromPositionSize(Point{X: 10, Y: 20}, Size{Width: -5, Height: -5})

	g.Expect(got).To(Equal(Rect{
		Min: Point{X: 10, Y: 20},
		Max: Point{X: 5, Y: 15},
	}))
	g.Expect(got.Valid()).To(BeFalse())
}

func TestRectValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		rect Rect
		want bool
	}{
		{name: "zero value", rect: Rect{}, want: true},
		{name: "ordinary rectangle", rect: Rect{Min: Point{X: 10, Y: 20}, Max: Point{X: 30, Y: 50}}, want: true},
		{name: "degenerate zero width", rect: Rect{Min: Point{X: 10, Y: 20}, Max: Point{X: 10, Y: 50}}, want: true},
		{name: "degenerate zero height", rect: Rect{Min: Point{X: 10, Y: 20}, Max: Point{X: 30, Y: 20}}, want: true},
		{name: "unordered X", rect: Rect{Min: Point{X: 30, Y: 20}, Max: Point{X: 10, Y: 50}}, want: false},
		{name: "unordered Y", rect: Rect{Min: Point{X: 10, Y: 50}, Max: Point{X: 30, Y: 20}}, want: false},
		{name: "nan min X", rect: Rect{Min: Point{X: math.NaN(), Y: 20}, Max: Point{X: 30, Y: 50}}, want: false},
		{name: "inf max Y", rect: Rect{Min: Point{X: 10, Y: 20}, Max: Point{X: 30, Y: math.Inf(1)}}, want: false},
		{name: "-inf min Y", rect: Rect{Min: Point{X: 10, Y: math.Inf(-1)}, Max: Point{X: 30, Y: 50}}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			g.Expect(tt.rect.Valid()).To(Equal(tt.want))
		})
	}
}

func TestRectEmpty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		rect Rect
		want bool
	}{
		{name: "zero value", rect: Rect{}, want: true},
		{name: "ordinary rectangle", rect: Rect{Min: Point{X: 10, Y: 20}, Max: Point{X: 30, Y: 50}}, want: false},
		{name: "zero width", rect: Rect{Min: Point{X: 10, Y: 20}, Max: Point{X: 10, Y: 50}}, want: true},
		{name: "zero height", rect: Rect{Min: Point{X: 10, Y: 20}, Max: Point{X: 30, Y: 20}}, want: true},
		{name: "invalid unordered", rect: Rect{Min: Point{X: 30, Y: 20}, Max: Point{X: 10, Y: 50}}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			g.Expect(tt.rect.Empty()).To(Equal(tt.want))
		})
	}
}

func TestRectDimensions(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	rect := Rect{Min: Point{X: 10, Y: 20}, Max: Point{X: 30, Y: 50}}

	g.Expect(rect.Width()).To(Equal(20.0))
	g.Expect(rect.Height()).To(Equal(30.0))
	g.Expect(rect.Size()).To(Equal(Size{Width: 20, Height: 30}))
}

func TestRectCenter(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	rect := Rect{Min: Point{X: 10, Y: 20}, Max: Point{X: 30, Y: 50}}

	g.Expect(rect.Center()).To(Equal(Point{X: 20, Y: 35}))
}

func TestRectContains(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		rect  Rect
		point Point
		want  bool
	}{
		{name: "min corner is inclusive", rect: Rect{Min: Point{X: 10, Y: 20}, Max: Point{X: 30, Y: 50}}, point: Point{X: 10, Y: 20}, want: true},
		{name: "max corner is inclusive", rect: Rect{Min: Point{X: 10, Y: 20}, Max: Point{X: 30, Y: 50}}, point: Point{X: 30, Y: 50}, want: true},
		{name: "interior point", rect: Rect{Min: Point{X: 10, Y: 20}, Max: Point{X: 30, Y: 50}}, point: Point{X: 20, Y: 35}, want: true},
		{name: "outside on X", rect: Rect{Min: Point{X: 10, Y: 20}, Max: Point{X: 30, Y: 50}}, point: Point{X: 31, Y: 35}, want: false},
		{name: "outside on Y", rect: Rect{Min: Point{X: 10, Y: 20}, Max: Point{X: 30, Y: 50}}, point: Point{X: 20, Y: 51}, want: false},
		{name: "invalid rectangle", rect: Rect{Min: Point{X: 30, Y: 20}, Max: Point{X: 10, Y: 50}}, point: Point{X: 20, Y: 35}, want: false},
		{name: "invalid point", rect: Rect{Min: Point{X: 10, Y: 20}, Max: Point{X: 30, Y: 50}}, point: Point{X: math.NaN(), Y: 35}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			g.Expect(tt.rect.Contains(tt.point)).To(Equal(tt.want))
		})
	}
}

func TestRectTranslate(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	rect := Rect{Min: Point{X: 10, Y: 20}, Max: Point{X: 30, Y: 50}}

	got := rect.Translate(Vector{X: 5, Y: -5})

	g.Expect(got).To(Equal(Rect{
		Min: Point{X: 15, Y: 15},
		Max: Point{X: 35, Y: 45},
	}))
}

func TestRectTranslateByZeroVectorIsIdentity(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	rect := Rect{Min: Point{X: 10, Y: 20}, Max: Point{X: 30, Y: 50}}

	g.Expect(rect.Translate(Vector{})).To(Equal(rect))
}

func TestRectInset(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	rect := Rect{Min: Point{X: 10, Y: 20}, Max: Point{X: 30, Y: 50}}

	inset, ok := rect.Inset(5)
	g.Expect(ok).To(BeTrue())
	g.Expect(inset).To(Equal(Rect{
		Min: Point{X: 15, Y: 25},
		Max: Point{X: 25, Y: 45},
	}))
}

func TestRectInsetNegativeAmountExpands(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	rect := Rect{Min: Point{X: 10, Y: 20}, Max: Point{X: 30, Y: 50}}

	expanded, ok := rect.Inset(-5)
	g.Expect(ok).To(BeTrue())
	g.Expect(expanded).To(Equal(Rect{
		Min: Point{X: 5, Y: 15},
		Max: Point{X: 35, Y: 55},
	}))
}

func TestRectInsetOverInsetFails(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	rect := Rect{Min: Point{X: 10, Y: 20}, Max: Point{X: 30, Y: 50}}

	_, ok := rect.Inset(20)
	g.Expect(ok).To(BeFalse())
}

func TestRectInsetInvalidValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		rect   Rect
		amount float64
	}{
		{name: "invalid rectangle", rect: Rect{Min: Point{X: 30, Y: 20}, Max: Point{X: 10, Y: 50}}, amount: 1},
		{name: "nan amount", rect: Rect{Min: Point{X: 10, Y: 20}, Max: Point{X: 30, Y: 50}}, amount: math.NaN()},
		{name: "infinite amount", rect: Rect{Min: Point{X: 10, Y: 20}, Max: Point{X: 30, Y: 50}}, amount: math.Inf(1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			_, ok := tt.rect.Inset(tt.amount)
			g.Expect(ok).To(BeFalse())
		})
	}
}

func TestRectExpandToInclude(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	rect := Rect{Min: Point{X: 10, Y: 20}, Max: Point{X: 30, Y: 50}}

	got, ok := rect.ExpandToInclude(Point{X: 40, Y: 10})
	g.Expect(ok).To(BeTrue())
	g.Expect(got).To(Equal(Rect{
		Min: Point{X: 10, Y: 10},
		Max: Point{X: 40, Y: 50},
	}))
}

func TestRectExpandToIncludeInteriorPointIsIdentity(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	rect := Rect{Min: Point{X: 10, Y: 20}, Max: Point{X: 30, Y: 50}}

	got, ok := rect.ExpandToInclude(Point{X: 20, Y: 35})
	g.Expect(ok).To(BeTrue())
	g.Expect(got).To(Equal(rect))
}

func TestRectExpandToIncludeInvalidOperands(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	valid := Rect{Min: Point{X: 10, Y: 20}, Max: Point{X: 30, Y: 50}}
	invalid := Rect{Min: Point{X: 30, Y: 20}, Max: Point{X: 10, Y: 50}}

	_, ok := invalid.ExpandToInclude(Point{X: 0, Y: 0})
	g.Expect(ok).To(BeFalse())

	_, ok = valid.ExpandToInclude(Point{X: math.NaN(), Y: 0})
	g.Expect(ok).To(BeFalse())
}

func TestRectUnion(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	a := Rect{Min: Point{X: 10, Y: 20}, Max: Point{X: 30, Y: 50}}
	b := Rect{Min: Point{X: 25, Y: 5}, Max: Point{X: 40, Y: 30}}

	got, ok := a.Union(b)
	g.Expect(ok).To(BeTrue())
	g.Expect(got).To(Equal(Rect{
		Min: Point{X: 10, Y: 5},
		Max: Point{X: 40, Y: 50},
	}))
}

func TestRectUnionInvalidOperands(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	valid := Rect{Min: Point{X: 10, Y: 20}, Max: Point{X: 30, Y: 50}}
	invalid := Rect{Min: Point{X: 30, Y: 20}, Max: Point{X: 10, Y: 50}}

	_, ok := valid.Union(invalid)
	g.Expect(ok).To(BeFalse())

	_, ok = invalid.Union(valid)
	g.Expect(ok).To(BeFalse())
}

func TestRectOperations(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	rect := Rect{Min: Point{X: 10, Y: 20}, Max: Point{X: 30, Y: 50}}

	g.Expect(rect.Size()).To(Equal(Size{Width: 20, Height: 30}))
	g.Expect(rect.Center()).To(Equal(Point{X: 20, Y: 35}))
	g.Expect(rect.Contains(rect.Min)).To(BeTrue())
	g.Expect(rect.Contains(rect.Max)).To(BeTrue())

	inset, ok := rect.Inset(5)
	g.Expect(ok).To(BeTrue())
	g.Expect(inset).To(Equal(Rect{
		Min: Point{X: 15, Y: 25},
		Max: Point{X: 25, Y: 45},
	}))
}

func TestRectMethodsDoNotMutateReceiver(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	original := Rect{Min: Point{X: 10, Y: 20}, Max: Point{X: 30, Y: 50}}
	want := original

	_ = original.Valid()
	_ = original.Empty()
	_ = original.Width()
	_ = original.Height()
	_ = original.Size()
	_ = original.Center()
	_ = original.Contains(Point{X: 20, Y: 35})
	_ = original.Translate(Vector{X: 1, Y: 1})
	_, _ = original.Inset(1)
	_, _ = original.ExpandToInclude(Point{X: 100, Y: 100})
	_, _ = original.Union(Rect{Min: Point{X: 0, Y: 0}, Max: Point{X: 5, Y: 5}})

	g.Expect(original).To(Equal(want))
}
