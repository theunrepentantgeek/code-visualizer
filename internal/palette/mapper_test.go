package palette

import (
	"image/color"
	"testing"

	. "github.com/onsi/gomega"
)

func TestMapNumeric_MinToFirstColour(t *testing.T) {
	g := NewGomegaWithT(t)

	p := GetPalette(Neutral)
	// bucketIdx=0 out of 3 buckets
	col := MapNumericToColour(0, 3, p)
	g.Expect(col).To(Equal(p.Colours[0]))
}

func TestMapNumeric_MaxToLastColour(t *testing.T) {
	g := NewGomegaWithT(t)

	p := GetPalette(Neutral)
	// bucketIdx=2 out of 3 buckets → should map to last palette colour
	col := MapNumericToColour(2, 3, p)
	lastIdx := len(p.Colours) - 1
	g.Expect(col).To(Equal(p.Colours[lastIdx]))
}

func TestMapNumeric_MedianToMiddleColour(t *testing.T) {
	g := NewGomegaWithT(t)

	p := GetPalette(Neutral)
	// bucketIdx=1 out of 3 buckets → should be in the middle
	col := MapNumericToColour(1, 3, p)
	g.Expect(col.A).To(Equal(uint8(255)))
	// For neutral palette, middle bucket should be mid-grey
	g.Expect(col.R).To(BeNumerically(">", 50))
	g.Expect(col.R).To(BeNumerically("<", 200))
}

func TestMapCategorical_DistinctValues(t *testing.T) {
	g := NewGomegaWithT(t)

	p := GetPalette(Neutral)
	values := []string{"go", "rs", "py"}

	mapper := NewCategoricalMapper(values, p)
	colours := map[color.RGBA]bool{}
	for _, v := range values {
		col := mapper.Map(v)
		colours[col] = true
		g.Expect(col.A).To(Equal(uint8(255)))
	}
	g.Expect(colours).To(HaveLen(3))
}

func TestMapCategorical_WrapAround(t *testing.T) {
	g := NewGomegaWithT(t)

	p := GetPalette(Neutral) // 9 steps
	values := make([]string, 15)
	for i := range values {
		values[i] = string(rune('a' + i))
	}

	mapper := NewCategoricalMapper(values, p)
	for _, v := range values {
		col := mapper.Map(v)
		g.Expect(col.A).To(Equal(uint8(255)))
	}
}
