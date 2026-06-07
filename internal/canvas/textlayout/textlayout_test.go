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

func TestMeasureStrings_MatchesMeasureStringPerCall(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	lines := []string{"alpha.go", "128", "hello world"}

	const size = 14.0

	batchWidths, batchH := MeasureStrings(lines, size)

	for i, line := range lines {
		w, h := MeasureString(line, size)
		g.Expect(batchWidths[i]).To(BeNumerically("~", w, 0.01), "width mismatch for line %d", i)
		g.Expect(batchH).To(BeNumerically("~", h, 0.01), "height mismatch for line %d", i)
	}
}
