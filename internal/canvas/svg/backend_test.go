package svg

import (
	"image/color"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
)

func TestSVGBackend_DrawRectangle_ProducesValidSVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(200, 200)
	red := color.RGBA{R: 255, A: 255}
	blk := color.RGBA{A: 255}

	b.DrawRectangle(
		model.Position{X: 10, Y: 10},
		model.Size{Width: 80, Height: 60},
		model.SolidFill{Color: red}, model.SolidFill{Color: blk}, 2.0,
	)

	out := filepath.Join(t.TempDir(), "rect.svg")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	content := readFile(t, out)
	g.Expect(content).To(ContainSubstring("<svg"))
	g.Expect(content).To(ContainSubstring("<rect"))
	g.Expect(content).To(ContainSubstring("</svg>"))
}

func TestSVGBackend_DrawDisc_ProducesValidSVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(200, 200)
	blue := color.RGBA{B: 255, A: 255}
	blk := color.RGBA{A: 255}

	b.DrawDisc(
		model.Position{X: 100, Y: 100},
		50, model.SolidFill{Color: blue}, model.SolidFill{Color: blk}, 1.0,
	)

	out := filepath.Join(t.TempDir(), "disc.svg")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	content := readFile(t, out)
	g.Expect(content).To(ContainSubstring("<circle"))
}

func TestSVGBackend_DrawText_ProducesValidSVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(200, 100)
	blk := color.RGBA{A: 255}

	b.DrawText(
		model.Position{X: 100, Y: 50},
		"hello", blk, 14.0,
		model.AnchorMiddle, 0,
	)

	out := filepath.Join(t.TempDir(), "text.svg")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	content := readFile(t, out)
	g.Expect(content).To(ContainSubstring("<text"))
	g.Expect(content).To(ContainSubstring("hello"))
}

func TestSVGBackend_DrawLine_ProducesValidSVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(200, 200)
	blk := color.RGBA{A: 255}

	b.DrawLine(
		model.Position{X: 0, Y: 0},
		model.Position{X: 200, Y: 200},
		blk, 2.0,
	)

	out := filepath.Join(t.TempDir(), "line.svg")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	content := readFile(t, out)
	g.Expect(content).To(ContainSubstring("<line"))
}

func TestSVGBackend_DrawPath_ProducesValidSVG(t *testing.T) {
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

	out := filepath.Join(t.TempDir(), "path.svg")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	content := readFile(t, out)
	g.Expect(content).To(ContainSubstring("<path"))
}

func TestSVGBackend_DrawArcText_ProducesValidSVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(400, 400)
	blk := color.RGBA{A: 255}

	b.DrawArcText(
		model.Position{X: 200, Y: 200},
		100, "hello", blk, 14.0,
	)

	out := filepath.Join(t.TempDir(), "arctext.svg")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	content := readFile(t, out)
	g.Expect(content).To(ContainSubstring("<textPath"))
}

func TestSVGBackend_DrawText_FontSizeZero_UsesDefault(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(200, 100)
	blk := color.RGBA{A: 255}

	b.DrawText(
		model.Position{X: 100, Y: 50},
		"hello", blk, 0,
		model.AnchorMiddle, 0,
	)

	out := filepath.Join(t.TempDir(), "text-zero.svg")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	content := readFile(t, out)
	g.Expect(content).To(ContainSubstring("<text"))
	g.Expect(content).NotTo(ContainSubstring(`font-size="0`))
	g.Expect(content).To(ContainSubstring(`font-size="12.0"`))
}

func TestSVGBackend_DrawText_FontSizeNegative_UsesDefault(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(200, 100)
	blk := color.RGBA{A: 255}

	b.DrawText(
		model.Position{X: 100, Y: 50},
		"hello", blk, -5.0,
		model.AnchorMiddle, 0,
	)

	out := filepath.Join(t.TempDir(), "text-neg.svg")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	content := readFile(t, out)
	g.Expect(content).NotTo(ContainSubstring(`font-size="0`))
	g.Expect(content).NotTo(ContainSubstring(`font-size="-`))
	g.Expect(content).To(ContainSubstring(`font-size="12.0"`))
}

func TestSVGBackend_DrawArcText_FontSizeZero_UsesDefault(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(400, 400)
	blk := color.RGBA{A: 255}

	b.DrawArcText(
		model.Position{X: 200, Y: 200},
		100, "hello", blk, 0,
	)

	out := filepath.Join(t.TempDir(), "arctext-zero.svg")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	content := readFile(t, out)
	g.Expect(content).To(ContainSubstring("<textPath"))
	g.Expect(content).NotTo(ContainSubstring(`font-size="0`))
	g.Expect(content).To(ContainSubstring(`font-size="12.0"`))
}

func TestSVGBackend_DrawArcText_PathGoesOverTop(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(400, 400)
	blk := color.RGBA{A: 255}

	// Circle centred at (200, 200) with radius 100.
	// The arc path should start at the left side (100, 200) and end at the
	// right side (300, 200) so that the 50% midpoint is at the top (200, 100),
	// placing the text label at the top of the circle.
	b.DrawArcText(
		model.Position{X: 200, Y: 200},
		100, "hello", blk, 14.0,
	)

	out := filepath.Join(t.TempDir(), "arctext-top.svg")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	content := readFile(t, out)
	// Arc starts at left: M <center.X - arcR>, <center.Y> — arcR = 100-14 = 86
	g.Expect(content).To(ContainSubstring("M114.00,200.00"))
	// Arc ends at right: ... <center.X + arcR>, <center.Y>
	g.Expect(content).To(ContainSubstring("286.00,200.00"))
	// sweep-flag=1 (clockwise, through the top), large-arc-flag=0 (half-circle)
	g.Expect(content).To(ContainSubstring("0 0,1"))
}

func TestSVGBackend_DrawArcText_CentersGlyphsOnPath(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(400, 400)
	blk := color.RGBA{A: 255}

	b.DrawArcText(
		model.Position{X: 200, Y: 200},
		100, "hello", blk, 14.0,
	)

	out := filepath.Join(t.TempDir(), "arctext-centered.svg")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	content := readFile(t, out)
	g.Expect(content).To(ContainSubstring(`dominant-baseline="middle"`))
}

func TestSVGBackend_ImplementsBackendInterface(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(100, 100)
	g.Expect(b).NotTo(BeNil())
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return string(data)
}
