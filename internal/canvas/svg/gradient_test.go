package svg_test

import (
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	svgbackend "github.com/theunrepentantgeek/code-visualizer/internal/canvas/svg"
)

func TestSVGBackend_DrawRectangle_WithRadialGradientFill_EmitsGradient(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	backend := svgbackend.New(200, 200)

	fill := model.RadialGradientFill{
		Center: color.RGBA{R: 255, G: 255, B: 255, A: 255},
		Edge:   color.RGBA{R: 100, G: 100, B: 100, A: 255},
		Focus:  model.Point{X: 0.35, Y: 0.35},
	}
	border := model.SolidFill{Color: color.RGBA{A: 255}}

	backend.DrawRectangle(
		model.Position{X: 10, Y: 10},
		model.Size{Width: 180, Height: 180},
		fill, border, 1.0,
	)

	tmp := filepath.Join(t.TempDir(), "gradient.svg")
	err := backend.Finish(tmp)
	g.Expect(err).NotTo(HaveOccurred())

	data, err := os.ReadFile(tmp)
	g.Expect(err).NotTo(HaveOccurred())

	svg := string(data)
	g.Expect(svg).To(ContainSubstring("<radialGradient"))
	g.Expect(svg).To(ContainSubstring(`fx="35.0%"`))
	g.Expect(svg).To(ContainSubstring(`fy="35.0%"`))
	g.Expect(svg).To(ContainSubstring(`fill="url(#`))
	g.Expect(strings.Count(svg, "<stop")).To(BeNumerically(">=", 2))
}

func TestSVGBackend_DrawDisc_WithRadialGradientFill_EmitsGradient(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	backend := svgbackend.New(200, 200)
	backend.DrawDisc(
		model.Position{X: 100, Y: 100},
		80,
		model.RadialGradientFill{
			Center: color.RGBA{R: 255, G: 255, B: 255, A: 255},
			Edge:   color.RGBA{R: 100, G: 100, B: 100, A: 255},
			Focus:  model.Point{X: 0.5, Y: 0.5},
		},
		model.SolidFill{Color: color.RGBA{A: 255}}, 0,
	)

	tmp := filepath.Join(t.TempDir(), "disc-gradient.svg")
	err := backend.Finish(tmp)
	g.Expect(err).NotTo(HaveOccurred())

	data, err := os.ReadFile(tmp)
	g.Expect(err).NotTo(HaveOccurred())

	svg := string(data)
	g.Expect(svg).To(ContainSubstring("<radialGradient"))
	g.Expect(svg).To(ContainSubstring(`fill="url(#`))
	g.Expect(svg).To(ContainSubstring("<circle"))
	g.Expect(strings.Count(svg, "<stop")).To(BeNumerically(">=", 2))
}

func TestSVGBackend_DeduplicatesIdenticalRadialGradients(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	backend := svgbackend.New(600, 200)

	sharedFill := model.RadialGradientFill{
		Center: color.RGBA{R: 255, G: 255, B: 255, A: 255},
		Edge:   color.RGBA{R: 100, G: 100, B: 100, A: 255},
		Focus:  model.Point{X: 0.5, Y: 0.5},
	}
	uniqueFill := model.RadialGradientFill{
		Center: color.RGBA{R: 200, G: 100, B: 50, A: 255},
		Edge:   color.RGBA{R: 50, G: 25, B: 10, A: 255},
		Focus:  model.Point{X: 0.3, Y: 0.3},
	}
	border := model.SolidFill{Color: color.RGBA{A: 255}}

	// Draw 5 rectangles with sharedFill and 1 with uniqueFill.
	for i := range 5 {
		backend.DrawRectangle(
			model.Position{X: float64(i * 100), Y: 0},
			model.Size{Width: 90, Height: 200},
			sharedFill, border, 1.0,
		)
	}

	backend.DrawRectangle(
		model.Position{X: 500, Y: 0},
		model.Size{Width: 90, Height: 200},
		uniqueFill, border, 1.0,
	)

	tmp := filepath.Join(t.TempDir(), "dedup.svg")
	err := backend.Finish(tmp)
	g.Expect(err).NotTo(HaveOccurred())

	data, err := os.ReadFile(tmp)
	g.Expect(err).NotTo(HaveOccurred())

	svg := string(data)

	// 5 identical + 1 unique → exactly 2 gradient definitions, not 6.
	g.Expect(strings.Count(svg, "<radialGradient")).To(Equal(2))
}

func TestSVGBackend_DeduplicatesGradientAcrossRectAndDisc(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	backend := svgbackend.New(300, 200)

	fill := model.RadialGradientFill{
		Center: color.RGBA{R: 255, G: 200, B: 100, A: 255},
		Edge:   color.RGBA{R: 100, G: 80, B: 40, A: 255},
		Focus:  model.Point{X: 0.4, Y: 0.4},
	}
	border := model.SolidFill{Color: color.RGBA{A: 255}}

	backend.DrawRectangle(
		model.Position{X: 0, Y: 0},
		model.Size{Width: 100, Height: 200},
		fill, border, 1.0,
	)
	backend.DrawDisc(
		model.Position{X: 200, Y: 100},
		80,
		fill, border, 0,
	)

	tmp := filepath.Join(t.TempDir(), "rect-disc-dedup.svg")
	err := backend.Finish(tmp)
	g.Expect(err).NotTo(HaveOccurred())

	data, err := os.ReadFile(tmp)
	g.Expect(err).NotTo(HaveOccurred())

	svg := string(data)

	// Same gradient used for both rect and disc → only one definition.
	g.Expect(strings.Count(svg, "<radialGradient")).To(Equal(1))
}
