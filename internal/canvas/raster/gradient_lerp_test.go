package raster

import (
	"image/color"
	"testing"

	. "github.com/onsi/gomega"
)

func TestGradientLerp_AtZero_ReturnsCenter(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	lerp := newGradientLerp(
		color.RGBA{R: 255, G: 200, B: 100, A: 255},
		color.RGBA{R: 0, G: 0, B: 0, A: 255},
	)

	result := lerp.at(0)

	g.Expect(result).To(Equal(color.RGBA{R: 255, G: 200, B: 100, A: 255}))
}

func TestGradientLerp_AtOne_ReturnsEdge(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	lerp := newGradientLerp(
		color.RGBA{R: 0, G: 0, B: 0, A: 255},
		color.RGBA{R: 100, G: 150, B: 200, A: 128},
	)

	result := lerp.at(1)

	g.Expect(result).To(Equal(color.RGBA{R: 100, G: 150, B: 200, A: 128}))
}

func TestGradientLerp_AtHalf_InterpolatesMidpoint(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	lerp := newGradientLerp(
		color.RGBA{R: 0, G: 0, B: 0, A: 0},
		color.RGBA{R: 200, G: 100, B: 50, A: 255},
	)

	result := lerp.at(0.5)

	g.Expect(result.R).To(BeNumerically("~", 100, 1))
	g.Expect(result.G).To(BeNumerically("~", 50, 1))
	g.Expect(result.B).To(BeNumerically("~", 25, 1))
	g.Expect(result.A).To(BeNumerically("~", 127, 1))
}

func TestGradientLerp_SameColors_AlwaysReturnsSameColor(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := color.RGBA{R: 128, G: 64, B: 32, A: 255}
	lerp := newGradientLerp(c, c)

	g.Expect(lerp.at(0)).To(Equal(c))
	g.Expect(lerp.at(0.5)).To(Equal(c))
	g.Expect(lerp.at(1)).To(Equal(c))
}
