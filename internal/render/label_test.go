package render

import (
	"image/color"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/bevan/code-visualizer/internal/treemap"
)

func TestLabelFitting_LargeRect(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	rect := treemap.TreemapRectangle{
		W:     300,
		H:     200,
		Label: "main.go",
	}

	show := ShouldShowLabel(rect)
	g.Expect(show).To(BeTrue())
}

func TestLabelFitting_SmallRect(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	rect := treemap.TreemapRectangle{
		W:     10,
		H:     8,
		Label: "main.go",
	}

	show := ShouldShowLabel(rect)
	g.Expect(show).To(BeFalse())
}

func TestTextColour_DarkOnLightFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	lightFill := color.RGBA{R: 240, G: 240, B: 240, A: 255}
	textCol := TextColourFor(lightFill)
	// Dark text on light background
	g.Expect(textCol.R).To(BeNumerically("<", 100))
	g.Expect(textCol.G).To(BeNumerically("<", 100))
	g.Expect(textCol.B).To(BeNumerically("<", 100))
}

func TestTextColour_LightOnDarkFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	darkFill := color.RGBA{R: 20, G: 20, B: 20, A: 255}
	textCol := TextColourFor(darkFill)
	// Light text on dark background
	g.Expect(textCol.R).To(BeNumerically(">", 150))
	g.Expect(textCol.G).To(BeNumerically(">", 150))
	g.Expect(textCol.B).To(BeNumerically(">", 150))
}
