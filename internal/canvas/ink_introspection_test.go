package canvas

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/bevan/code-visualizer/internal/palette"
)

func TestNumericInk_Boundaries_ReturnsBucketValues(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	values := []float64{10, 20, 30, 40, 50, 60, 70, 80, 90}
	pal := palette.GetPalette(palette.Neutral)
	ink := NumericInk(values, pal)

	boundaries := ink.Boundaries()
	g.Expect(boundaries).NotTo(BeEmpty())
}

func TestFixedInk_Boundaries_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ink := FixedInk(white)
	g.Expect(ink.Boundaries()).To(BeNil())
}

func TestCategoricalInk_Boundaries_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ink := CategoricalInk([]string{"go"}, palette.GetPalette(palette.Categorization))
	g.Expect(ink.Boundaries()).To(BeNil())
}

func TestNumericInk_Palette_ReturnsPalette(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pal := palette.GetPalette(palette.Temperature)
	ink := NumericInk([]float64{1, 2}, pal)

	g.Expect(ink.Palette().Name).To(Equal(palette.Temperature))
}

func TestFixedInk_Palette_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ink := FixedInk(white)
	g.Expect(ink.Palette().Colours).To(BeEmpty())
}

func TestCategoricalInk_Categories_ReturnsList(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cats := []string{"go", "rs", "py"}
	ink := CategoricalInk(cats, palette.GetPalette(palette.Categorization))

	g.Expect(ink.Categories()).To(Equal(cats))
}

func TestNumericInk_Categories_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ink := NumericInk([]float64{1, 2}, palette.GetPalette(palette.Neutral))
	g.Expect(ink.Categories()).To(BeNil())
}

func TestFixedInk_Categories_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ink := FixedInk(white)
	g.Expect(ink.Categories()).To(BeNil())
}
