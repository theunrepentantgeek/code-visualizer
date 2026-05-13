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

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
)

func TestRasterBackend_DrawRectangle_ProducesValidPNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(200, 200)
	red := color.RGBA{R: 255, A: 255}
	blk := color.RGBA{A: 255}

	b.DrawRectangle(
		model.Position{X: 10, Y: 10},
		model.Size{Width: 80, Height: 60},
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
		model.Position{X: 100, Y: 100},
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
		model.Position{X: 100, Y: 50},
		"hello", blk, 14.0,
		model.AnchorMiddle, 0,
	)

	out := filepath.Join(t.TempDir(), "text.png")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	info, err := os.Stat(out)
	g.Expect(err).NotTo(HaveOccurred())

	if info != nil {
		g.Expect(info.Size()).To(BeNumerically(">", 0))
	}
}

func TestRasterBackend_DrawLine_ProducesValidPNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(200, 200)
	blk := color.RGBA{A: 255}

	b.DrawLine(
		model.Position{X: 0, Y: 0},
		model.Position{X: 200, Y: 200},
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
		[]model.Position{
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

	if info != nil {
		g.Expect(info.Size()).To(BeNumerically(">", 0))
	}
}

func TestRasterBackend_Finish_UnsupportedFormat(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(100, 100)
	out := filepath.Join(t.TempDir(), "test.bmp")

	err := b.Finish(out)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("unsupported"))
	}
}

func TestRasterBackend_ImplementsBackendInterface(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(100, 100)
	g.Expect(b).NotTo(BeNil())
}

func TestRasterBackend_DrawArcText_ProducesValidPNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(400, 400)
	blk := color.RGBA{A: 255}

	b.DrawArcText(
		model.Position{X: 200, Y: 200},
		100, "hello", blk, 14.0,
	)

	out := filepath.Join(t.TempDir(), "arctext.png")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	img := loadImage(t, out)
	g.Expect(img.Bounds().Dx()).To(Equal(400))
}

func TestRasterBackend_DrawArcText_FontSizeZero_ProducesValidPNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(400, 400)
	blk := color.RGBA{A: 255}

	b.DrawArcText(
		model.Position{X: 200, Y: 200},
		100, "hello", blk, 0,
	)

	out := filepath.Join(t.TempDir(), "arctext-zero.png")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	img := loadImage(t, out)
	g.Expect(img.Bounds().Dx()).To(Equal(400))
}

func TestRasterBackend_DrawText_FontSizeZero_ProducesValidPNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(200, 100)
	blk := color.RGBA{A: 255}

	b.DrawText(
		model.Position{X: 100, Y: 50},
		"hello", blk, 0,
		model.AnchorMiddle, 0,
	)

	out := filepath.Join(t.TempDir(), "text-zero.png")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	info, err := os.Stat(out)
	g.Expect(err).NotTo(HaveOccurred())

	if info != nil {
		g.Expect(info.Size()).To(BeNumerically(">", 0))
	}
}

func TestRasterBackend_DrawText_RespectsCustomFontSize(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Render the same text at two different font sizes and verify both produce
	// valid output with different pixel content (larger font = more ink).
	small := New(200, 100)
	blk := color.RGBA{A: 255}

	small.DrawText(
		model.Position{X: 100, Y: 50},
		"Hello", blk, 10.0,
		model.AnchorMiddle, 0,
	)

	outSmall := filepath.Join(t.TempDir(), "text-small.png")
	err := small.Finish(outSmall)
	g.Expect(err).NotTo(HaveOccurred())

	large := New(200, 100)

	large.DrawText(
		model.Position{X: 100, Y: 50},
		"Hello", blk, 28.0,
		model.AnchorMiddle, 0,
	)

	outLarge := filepath.Join(t.TempDir(), "text-large.png")
	err = large.Finish(outLarge)
	g.Expect(err).NotTo(HaveOccurred())

	infoSmall, err := os.Stat(outSmall)
	g.Expect(err).NotTo(HaveOccurred())

	infoLarge, err := os.Stat(outLarge)
	g.Expect(err).NotTo(HaveOccurred())

	// Larger text produces a different (usually larger) PNG due to more ink.
	g.Expect(infoSmall).NotTo(BeNil())
	g.Expect(infoLarge).NotTo(BeNil())

	if infoSmall != nil && infoLarge != nil {
		g.Expect(infoSmall.Size()).NotTo(Equal(infoLarge.Size()))
	}
}

func TestRasterBackend_DrawDisc_SemiTransparentOverWhite_ProducesCorrectBlend(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Reproduce issue #228/#231: semi-transparent colours must be treated as
	// non-premultiplied when passed to gg, otherwise the gg raster painter
	// receives invalid premultiplied values and compositing produces wrong results.
	//
	// This test draws a white background and then a semi-transparent blue disc
	// centred at (50,50). The pixel at the centre should be a light blue tint,
	// NOT white (which would indicate the transparency was silently discarded)
	// and NOT black (which would indicate incorrect premultiplied-alpha math).
	b := New(100, 100)

	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	// Semi-transparent blue, stored as non-premultiplied in color.RGBA.
	semiBlue := color.RGBA{R: 0, G: 0, B: 255, A: 64}

	b.DrawRectangle(
		model.Position{X: 0, Y: 0},
		model.Size{Width: 100, Height: 100},
		white, white, 0,
	)

	b.DrawDisc(
		model.Position{X: 50, Y: 50},
		40, semiBlue, semiBlue, 0,
	)

	out := filepath.Join(t.TempDir(), "semi-transparent.png")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	img := loadImage(t, out)

	// Read the pixel at the centre of the disc.
	r, gg2, bv, _ := img.At(50, 50).RGBA()
	rB := uint8(r >> 8)
	gB := uint8(gg2 >> 8)
	bB := uint8(bv >> 8)

	// The pixel should have a blue tint (B > R, B > G), not pure white or black.
	g.Expect(bB).To(BeNumerically(">", rB), "blue channel should dominate over red")
	g.Expect(bB).To(BeNumerically(">", gB), "blue channel should dominate over green")
	// The background is white, so R and G should still be quite high
	// (a 25%-opacity blue disc over white gives R=G≈191, B=255).
	g.Expect(rB).To(BeNumerically(">", 150), "red should be high (near-white background)")
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
