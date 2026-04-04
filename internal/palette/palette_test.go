package palette

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestNeutralPalette_StepCount(t *testing.T) {
	g := NewGomegaWithT(t)

	p := GetPalette(Neutral)
	g.Expect(p.Colours).To(HaveLen(9))
	g.Expect(p.Ordered).To(BeTrue())
	g.Expect(p.Name).To(Equal(Neutral))
}

func TestNeutralPalette_BlackToWhite(t *testing.T) {
	g := NewGomegaWithT(t)

	p := GetPalette(Neutral)
	// First step should be near black
	g.Expect(p.Colours[0].R).To(BeNumerically("<=", 30))
	g.Expect(p.Colours[0].G).To(BeNumerically("<=", 30))
	g.Expect(p.Colours[0].B).To(BeNumerically("<=", 30))
	// Last step should be near white
	g.Expect(p.Colours[8].R).To(BeNumerically(">=", 225))
	g.Expect(p.Colours[8].G).To(BeNumerically(">=", 225))
	g.Expect(p.Colours[8].B).To(BeNumerically(">=", 225))
}

func TestNeutralPalette_MonotonicallyIncreasing(t *testing.T) {
	g := NewGomegaWithT(t)

	p := GetPalette(Neutral)
	for i := 1; i < len(p.Colours); i++ {
		g.Expect(p.Colours[i].R).To(BeNumerically(">=", p.Colours[i-1].R),
			"step %d should be >= step %d", i, i-1)
	}
}

func TestPaletteName_IsValid(t *testing.T) {
	g := NewGomegaWithT(t)

	g.Expect(Neutral.IsValid()).To(BeTrue())
	g.Expect(Categorization.IsValid()).To(BeTrue())
	g.Expect(Temperature.IsValid()).To(BeTrue())
	g.Expect(GoodBad.IsValid()).To(BeTrue())
	g.Expect(PaletteName("invalid").IsValid()).To(BeFalse())
}
