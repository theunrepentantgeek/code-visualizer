package geometry

import (
	"math"
	"testing"

	. "github.com/onsi/gomega"
)

func TestSizeValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		size Size
		want bool
	}{
		{name: "zero value", size: Size{}, want: true},
		{name: "finite positive dimensions", size: Size{Width: 16, Height: 9}, want: true},
		{name: "one zero dimension", size: Size{Width: 16}, want: true},
		{name: "negative width", size: Size{Width: -1, Height: 9}, want: false},
		{name: "negative height", size: Size{Width: 16, Height: -1}, want: false},
		{name: "nan width", size: Size{Width: math.NaN(), Height: 9}, want: false},
		{name: "nan height", size: Size{Width: 16, Height: math.NaN()}, want: false},
		{name: "positive infinity width", size: Size{Width: math.Inf(1), Height: 9}, want: false},
		{name: "negative infinity height", size: Size{Width: 16, Height: math.Inf(-1)}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			g.Expect(tt.size.Valid()).To(Equal(tt.want))
		})
	}
}

func TestSizeEmpty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		size Size
		want bool
	}{
		{name: "zero value", size: Size{}, want: true},
		{name: "positive dimensions", size: Size{Width: 16, Height: 9}, want: false},
		{name: "zero height", size: Size{Width: 16}, want: true},
		{name: "zero width", size: Size{Height: 9}, want: true},
		{name: "negative dimension", size: Size{Width: -1, Height: 9}, want: false},
		{name: "nan dimension", size: Size{Width: math.NaN(), Height: 9}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			g.Expect(tt.size.Empty()).To(Equal(tt.want))
		})
	}
}

func TestSizeArea(t *testing.T) {
	t.Parallel()

	assertFloatClose(t, (Size{Width: 16, Height: 9}).Area(), 144)
}

func TestSizeScale(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	got := (Size{Width: 16, Height: 9}).Scale(0.5)

	g.Expect(got).To(Equal(Size{Width: 8, Height: 4.5}))
}

func TestSizeAspectRatio(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ratio, ok := (Size{Width: 16, Height: 9}).AspectRatio()
	g.Expect(ok).To(BeTrue())
	g.Expect(ratio).To(Equal(16.0 / 9.0))

	_, ok = (Size{Width: 16}).AspectRatio()
	g.Expect(ok).To(BeFalse())
	_, ok = (Size{Width: -1, Height: 9}).AspectRatio()
	g.Expect(ok).To(BeFalse())
}

func TestSizeAspectRatioInvalidValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		size Size
	}{
		{name: "zero value", size: Size{}},
		{name: "nan height", size: Size{Width: 16, Height: math.NaN()}},
		{name: "positive infinity width", size: Size{Width: math.Inf(1), Height: 9}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			_, ok := tt.size.AspectRatio()
			g.Expect(ok).To(BeFalse())
		})
	}
}

func TestSizeMethodsDoNotMutateReceiver(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	original := Size{Width: 16, Height: 9}
	want := original

	_ = original.Valid()
	_ = original.Empty()
	_ = original.Area()
	_ = original.Scale(2)
	_, _ = original.AspectRatio()

	g.Expect(original).To(Equal(want))
}
