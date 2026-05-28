package raster_test

import (
	"image"
	"image/color"
	_ "image/png"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/raster"
)

func TestRasterBackend_DrawRectangle_WithRadialGradientFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	backend := raster.New(200, 200)

	fill := model.RadialGradientFill{
		Center: color.RGBA{R: 255, G: 255, B: 255, A: 255},
		Edge:   color.RGBA{R: 100, G: 100, B: 100, A: 255},
		Focus:  model.Point{X: 0.5, Y: 0.5},
	}
	border := model.SolidFill{Color: color.RGBA{A: 255}}

	backend.DrawRectangle(
		model.Position{X: 10, Y: 10},
		model.Size{Width: 180, Height: 180},
		fill, border, 1.0,
	)

	tmp := filepath.Join(t.TempDir(), "gradient.png")
	err := backend.Finish(tmp)
	g.Expect(err).NotTo(HaveOccurred())

	info, err := os.Stat(tmp)
	g.Expect(err).NotTo(HaveOccurred())

	if info != nil {
		g.Expect(info.Size()).To(BeNumerically(">", 0))
	}
}

func TestRasterBackend_DrawRectangle_RadialGradientVariesFromEdgeToCenter(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	backend := raster.New(200, 200)
	backend.DrawRectangle(
		model.Position{X: 10, Y: 10},
		model.Size{Width: 180, Height: 180},
		model.RadialGradientFill{
			Center: color.RGBA{R: 255, G: 255, B: 255, A: 255},
			Edge:   color.RGBA{R: 100, G: 100, B: 100, A: 255},
			Focus:  model.Point{X: 0.5, Y: 0.5},
		},
		model.SolidFill{Color: color.RGBA{A: 255}}, 0,
	)

	out := filepath.Join(t.TempDir(), "gradient-variation.png")
	err := backend.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	img := loadImage(t, out)
	center := colorAt(img, 100, 100)
	edge := colorAt(img, 20, 100)

	g.Expect(center.R).To(BeNumerically(">", edge.R))
	g.Expect(center.G).To(BeNumerically(">", edge.G))
	g.Expect(center.B).To(BeNumerically(">", edge.B))
}

func TestRasterBackend_DrawRectangle_RadialGradientWithOffCenterFocus_CoversFarCorner(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	backend := raster.New(200, 200)
	backend.DrawRectangle(
		model.Position{X: 10, Y: 10},
		model.Size{Width: 180, Height: 180},
		model.RadialGradientFill{
			Center: color.RGBA{R: 255, G: 255, B: 255, A: 255},
			Edge:   color.RGBA{B: 255, A: 255},
			Focus:  model.Point{X: 0.1, Y: 0.1},
		},
		model.SolidFill{Color: color.RGBA{A: 255}}, 0,
	)

	out := filepath.Join(t.TempDir(), "gradient-off-center.png")
	err := backend.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	img := loadImage(t, out)
	farCorner := colorAt(img, 189, 189)

	g.Expect(farCorner.A).To(BeNumerically(">", 0))
	g.Expect(farCorner.B).To(BeNumerically(">", farCorner.R))
	g.Expect(farCorner.B).To(BeNumerically(">", farCorner.G))
}

func TestRasterBackend_DrawDisc_WithRadialGradientFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	backend := raster.New(200, 200)
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

	out := filepath.Join(t.TempDir(), "disc-gradient.png")
	err := backend.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	img := loadImage(t, out)
	center := colorAt(img, 100, 100)
	edge := colorAt(img, 25, 100)

	// Center should be brighter than the edge.
	g.Expect(center.R).To(BeNumerically(">", edge.R))
	g.Expect(center.G).To(BeNumerically(">", edge.G))
	g.Expect(center.B).To(BeNumerically(">", edge.B))
}


func loadImage(t *testing.T, path string) image.Image {
	t.Helper()

	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()

	img, _, err := image.Decode(file)
	if err != nil {
		t.Fatal(err)
	}

	return img
}

func colorAt(img image.Image, x, y int) color.RGBA {
	r, g, b, a := img.At(x, y).RGBA()

	return color.RGBA{
		R: uint8(r >> 8), //nolint:gosec // RGBA returns 16-bit channel values; truncating to 8-bit is intentional.
		G: uint8(g >> 8), //nolint:gosec // RGBA returns 16-bit channel values; truncating to 8-bit is intentional.
		B: uint8(b >> 8), //nolint:gosec // RGBA returns 16-bit channel values; truncating to 8-bit is intentional.
		A: uint8(a >> 8), //nolint:gosec // RGBA returns 16-bit channel values; truncating to 8-bit is intentional.
	}
}
