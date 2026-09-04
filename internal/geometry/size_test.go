package geometry

import (
	"math"
	"testing"

	. "github.com/onsi/gomega"
)

func TestNewSize_Dimensions_ReturnsSize(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Arrange
	width := 16.0
	height := 9.0

	// Act
	size := NewSize(width, height)

	// Assert
	g.Expect(size).To(Equal(Size{Width: width, Height: height}))
}

func TestSize_VariousDimensions_ReportsValidity(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		size Size
		want bool
	}{
		"zero value":                 {size: Size{}, want: true},
		"finite positive dimensions": {size: Size{Width: 16, Height: 9}, want: true},
		"one zero dimension":         {size: Size{Width: 16}, want: true},
		"negative width":             {size: Size{Width: -1, Height: 9}, want: false},
		"negative height":            {size: Size{Width: 16, Height: -1}, want: false},
		"nan width":                  {size: Size{Width: math.NaN(), Height: 9}, want: false},
		"nan height":                 {size: Size{Width: 16, Height: math.NaN()}, want: false},
		"positive infinity width":    {size: Size{Width: math.Inf(1), Height: 9}, want: false},
		"negative infinity width":    {size: Size{Width: math.Inf(-1), Height: 9}, want: false},
		"positive infinity height":   {size: Size{Width: 16, Height: math.Inf(1)}, want: false},
		"negative infinity height":   {size: Size{Width: 16, Height: math.Inf(-1)}, want: false},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			// Arrange
			size := c.size

			// Act
			valid := size.Valid()

			// Assert
			g.Expect(valid).To(Equal(c.want))
		})
	}
}

func TestSize_VariousDimensions_ReportsEmptiness(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		size Size
		want bool
	}{
		"zero value":          {size: Size{}, want: true},
		"positive dimensions": {size: Size{Width: 16, Height: 9}, want: false},
		"zero height":         {size: Size{Width: 16}, want: true},
		"zero width":          {size: Size{Height: 9}, want: true},
		"negative dimension":  {size: Size{Width: -1, Height: 9}, want: false},
		"nan dimension":       {size: Size{Width: math.NaN(), Height: 9}, want: false},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			// Arrange
			size := c.size

			// Act
			empty := size.Empty()

			// Assert
			g.Expect(empty).To(Equal(c.want))
		})
	}
}

func TestSize_Area_ReturnsProductOfDimensions(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Arrange
	size := Size{Width: 16, Height: 9}

	// Act
	area := size.Area()

	// Assert
	assertFloatClose(g, area, 144)
}

func TestSize_Scale_ReturnsScaledDimensions(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Arrange
	size := Size{Width: 16, Height: 9}

	// Act
	scaled := size.Scale(0.5)

	// Assert
	g.Expect(scaled).To(Equal(Size{Width: 8, Height: 4.5}))
}

func TestSize_AspectRatio_ValidSize_ReturnsRatio(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Arrange
	size := Size{Width: 16, Height: 9}

	// Act
	ratio, ok := size.AspectRatio()

	// Assert
	g.Expect(ok).To(BeTrue())
	g.Expect(ratio).To(Equal(16.0 / 9.0))
}

func TestSize_AspectRatio_ZeroHeight_ReturnsFalse(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Arrange
	size := Size{Width: 16}

	// Act
	_, ok := size.AspectRatio()

	// Assert
	g.Expect(ok).To(BeFalse())
}

func TestSize_AspectRatio_InvalidSize_ReturnsFalse(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		size Size
	}{
		"negative dimension":      {size: Size{Width: -1, Height: 9}},
		"nan height":              {size: Size{Width: 16, Height: math.NaN()}},
		"positive infinity width": {size: Size{Width: math.Inf(1), Height: 9}},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			// Arrange
			size := c.size

			// Act
			_, ok := size.AspectRatio()

			// Assert
			g.Expect(ok).To(BeFalse())
		})
	}
}

func TestSize_ValueMethods_DoNotMutateReceiver(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Arrange
	original := Size{Width: 16, Height: 9}
	want := original

	// Act
	_ = original.Valid()
	_ = original.Empty()
	_ = original.Area()
	_ = original.Scale(2)
	_, _ = original.AspectRatio()

	// Assert
	g.Expect(original).To(Equal(want))
}
