package canvas

import (
	"image/color"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

func baseInkForTest(t *testing.T, ink Ink) *baseInk {
	t.Helper()

	base, ok := ink.(*baseInk)
	if !ok {
		t.Fatalf("expected *baseInk, got %T", ink)
	}

	return base
}

func TestNumericInk_Boundaries_ReturnsBucketValues(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	values := []float64{10, 20, 30, 40, 50, 60, 70, 80, 90}
	pal := palette.GetPalette(palette.Neutral)
	ink := NumericInk("test-metric", values, pal)

	boundaries := baseInkForTest(t, ink).Boundaries()
	g.Expect(boundaries).NotTo(BeEmpty())
}

func TestFixedInk_Boundaries_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ink := FixedInk(white)
	g.Expect(baseInkForTest(t, ink).Boundaries()).To(BeNil())
}

func TestCategoricalInk_Boundaries_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ink := CategoricalInk("test-metric", []string{"go"}, palette.GetPalette(palette.Categorization))
	g.Expect(baseInkForTest(t, ink).Boundaries()).To(BeNil())
}

func TestNumericInk_Palette_ReturnsPalette(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pal := palette.GetPalette(palette.Temperature)
	ink := NumericInk("test-metric", []float64{1, 2}, pal)

	g.Expect(baseInkForTest(t, ink).Palette().Name).To(Equal(palette.Temperature))
}

func TestFixedInk_Palette_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ink := FixedInk(white)
	g.Expect(baseInkForTest(t, ink).Palette().Colours).To(BeEmpty())
}

func TestCategoricalInk_Categories_ReturnsList(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cats := []string{"go", "rs", "py"}
	ink := CategoricalInk("test-metric", cats, palette.GetPalette(palette.Categorization))

	g.Expect(baseInkForTest(t, ink).Categories()).To(Equal(cats))
}

func TestNumericInk_Categories_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ink := NumericInk("test-metric", []float64{1, 2}, palette.GetPalette(palette.Neutral))
	g.Expect(baseInkForTest(t, ink).Categories()).To(BeNil())
}

func TestFixedInk_Categories_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ink := FixedInk(white)
	g.Expect(baseInkForTest(t, ink).Categories()).To(BeNil())
}

func TestInkInfo_Fixed(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ink := FixedInk(color.RGBA{R: 255, A: 255})
	info := ink.Info()
	g.Expect(info.Kind).To(Equal(InkFixed))
	g.Expect(info.MetricName).To(Equal(metric.Name("")))
}

func TestInkInfo_Numeric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ink := NumericInk("file-size", []float64{1, 2, 3}, palette.GetPalette(palette.Neutral))
	info := ink.Info()
	g.Expect(info.Kind).To(Equal(InkNumeric))
	g.Expect(info.MetricName).To(Equal(metric.Name("file-size")))
}

func TestInkInfo_Categorical(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ink := CategoricalInk("file-type", []string{"go", "rs"}, palette.GetPalette(palette.Categorization))
	info := ink.Info()
	g.Expect(info.Kind).To(Equal(InkCategorical))
	g.Expect(info.MetricName).To(Equal(metric.Name("file-type")))
}
