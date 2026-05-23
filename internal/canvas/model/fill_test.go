package model_test

import (
	"image/color"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
)

func TestSolidFill_ImplementsFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var fill model.Fill = model.SolidFill{Color: color.RGBA{R: 255, A: 255}}
	g.Expect(fill).NotTo(BeNil())
}

func TestRadialGradientFill_ImplementsFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var fill model.Fill = model.RadialGradientFill{
		Center: color.RGBA{R: 255, G: 255, B: 255, A: 255},
		Edge:   color.RGBA{R: 100, G: 100, B: 100, A: 255},
		Focus:  model.Point{X: 0.5, Y: 0.5},
	}
	g.Expect(fill).NotTo(BeNil())
}

func TestPoint_Zero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	p := model.Point{}
	g.Expect(p.X).To(Equal(0.0))
	g.Expect(p.Y).To(Equal(0.0))
}
