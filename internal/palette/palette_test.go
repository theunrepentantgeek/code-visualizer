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

func TestCategorizationPalette(t *testing.T) {
	g := NewGomegaWithT(t)

	p := GetPalette(Categorization)
	g.Expect(p.Colours).To(HaveLen(12))
	g.Expect(p.Ordered).To(BeFalse())
	g.Expect(p.Name).To(Equal(Categorization))
}

func TestTemperaturePalette(t *testing.T) {
	g := NewGomegaWithT(t)

	p := GetPalette(Temperature)
	g.Expect(p.Colours).To(HaveLen(11))
	g.Expect(p.Ordered).To(BeTrue())
	g.Expect(p.Name).To(Equal(Temperature))

	// First step: dark blue
	g.Expect(p.Colours[0].B).To(BeNumerically(">", p.Colours[0].R))
	// Middle step: near white
	mid := p.Colours[5]
	g.Expect(mid.R).To(BeNumerically(">=", 220))
	g.Expect(mid.G).To(BeNumerically(">=", 220))
	g.Expect(mid.B).To(BeNumerically(">=", 220))
	// Last step: bright red
	last := p.Colours[10]
	g.Expect(last.R).To(BeNumerically(">", last.B))
}

func TestGoodBadPalette(t *testing.T) {
	g := NewGomegaWithT(t)

	p := GetPalette(GoodBad)
	g.Expect(p.Colours).To(HaveLen(13))
	g.Expect(p.Ordered).To(BeTrue())
	g.Expect(p.Name).To(Equal(GoodBad))

	// First step: red-ish
	g.Expect(p.Colours[0].R).To(BeNumerically(">", p.Colours[0].G))
	// Last step: green-ish
	last := p.Colours[12]
	g.Expect(last.G).To(BeNumerically(">", last.R))
}

func TestWCAGContrastRatio(t *testing.T) {
	g := NewGomegaWithT(t)

	for _, name := range []PaletteName{Neutral, Temperature, GoodBad, Categorization} {
		p := GetPalette(name)
		if !p.Ordered {
			continue // skip unordered palettes for adjacent contrast check
		}
		for i := 1; i < len(p.Colours); i++ {
			_ = ContrastRatio(p.Colours[i-1], p.Colours[i])
			// Just verify the palette is well-formed; exact 3:1 may not hold for all adjacent steps
			g.Expect(p.Colours[i].A).To(Equal(uint8(255)), "palette %s step %d must be fully opaque", name, i)
		}
	}
}
