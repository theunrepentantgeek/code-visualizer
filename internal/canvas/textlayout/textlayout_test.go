package textlayout

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestMeasureString_ReturnsPositiveDimensionsForNonEmptyText(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	width, height := MeasureString("hello", 12)
	g.Expect(width).To(BeNumerically(">", 0.0))
	g.Expect(height).To(BeNumerically(">", 0.0))
}

func TestMeasureString_GrowsWithFontSize(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	smallWidth, smallHeight := MeasureString("hello", 10)
	largeWidth, largeHeight := MeasureString("hello", 20)
	g.Expect(largeWidth).To(BeNumerically(">", smallWidth))
	g.Expect(largeHeight).To(BeNumerically(">", smallHeight))
}
