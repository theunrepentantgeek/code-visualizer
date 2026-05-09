package canvas

import (
	"image/color"
	"testing"

	. "github.com/onsi/gomega"
)

func TestTextColourFor_DarkOnLightFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	lightFill := color.RGBA{R: 240, G: 240, B: 240, A: 255}
	textCol := TextColourFor(lightFill)

	g.Expect(textCol.R).To(BeNumerically("<", 100))
	g.Expect(textCol.G).To(BeNumerically("<", 100))
	g.Expect(textCol.B).To(BeNumerically("<", 100))
}

func TestTextColourFor_LightOnDarkFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	darkFill := color.RGBA{R: 20, G: 20, B: 20, A: 255}
	textCol := TextColourFor(darkFill)

	g.Expect(textCol.R).To(BeNumerically(">", 150))
	g.Expect(textCol.G).To(BeNumerically(">", 150))
	g.Expect(textCol.B).To(BeNumerically(">", 150))
}

func TestTextColourFor_MidGrey_ReturnsDark(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	midFill := color.RGBA{R: 200, G: 200, B: 200, A: 255}
	textCol := TextColourFor(midFill)

	g.Expect(textCol).To(Equal(color.RGBA{R: 0, G: 0, B: 0, A: 255}))
}
