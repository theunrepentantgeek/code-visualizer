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
