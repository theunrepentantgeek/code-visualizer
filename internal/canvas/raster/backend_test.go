package raster

import (
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/bevan/code-visualizer/internal/canvas/types"
)

func TestRasterBackend_DrawRectangle_ProducesValidPNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(200, 200)
	red := color.RGBA{R: 255, A: 255}
	blk := color.RGBA{A: 255}

	b.DrawRectangle(
		types.Position{X: 10, Y: 10},
		types.Size{Width: 80, Height: 60},
		red, blk, 2.0,
	)

	out := filepath.Join(t.TempDir(), "rect.png")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	img := loadImage(t, out)
	g.Expect(img.Bounds().Dx()).To(Equal(200))
	g.Expect(img.Bounds().Dy()).To(Equal(200))
}

func TestRasterBackend_DrawDisc_ProducesValidPNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(200, 200)
	blue := color.RGBA{B: 255, A: 255}
	blk := color.RGBA{A: 255}

	b.DrawDisc(
		types.Position{X: 100, Y: 100},
		50, blue, blk, 1.0,
	)

	out := filepath.Join(t.TempDir(), "disc.png")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	img := loadImage(t, out)
	g.Expect(img.Bounds().Dx()).To(Equal(200))
}

func TestRasterBackend_DrawText_ProducesValidPNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(200, 100)
	blk := color.RGBA{A: 255}

	b.DrawText(
		types.Position{X: 100, Y: 50},
		"hello", blk, 14.0,
		types.AnchorMiddle, 0,
	)

	out := filepath.Join(t.TempDir(), "text.png")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	info, err := os.Stat(out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(info.Size()).To(BeNumerically(">", 0))
}

func TestRasterBackend_DrawLine_ProducesValidPNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(200, 200)
	blk := color.RGBA{A: 255}

	b.DrawLine(
		types.Position{X: 0, Y: 0},
		types.Position{X: 200, Y: 200},
		blk, 2.0,
	)

	out := filepath.Join(t.TempDir(), "line.png")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestRasterBackend_DrawPath_ProducesValidPNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(200, 200)
	blk := color.RGBA{A: 255}

	b.DrawPath(
		[]types.Position{
			{X: 10, Y: 10},
			{X: 100, Y: 50},
			{X: 190, Y: 10},
		},
		blk, 1.0,
	)

	out := filepath.Join(t.TempDir(), "path.png")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestRasterBackend_Finish_JPG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(100, 100)
	out := filepath.Join(t.TempDir(), "test.jpg")

	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	info, err := os.Stat(out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(info.Size()).To(BeNumerically(">", 0))
}

func TestRasterBackend_Finish_UnsupportedFormat(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(100, 100)
	out := filepath.Join(t.TempDir(), "test.bmp")

	err := b.Finish(out)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("unsupported"))
}

func TestRasterBackend_ImplementsBackendInterface(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var b types.Backend = New(100, 100)
	g.Expect(b).NotTo(BeNil())
}

func loadImage(t *testing.T, path string) image.Image {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = f.Close() }()

	img, _, err := image.Decode(f)
	if err != nil {
		t.Fatal(err)
	}

	return img
}
