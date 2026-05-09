package svg

import (
	"image/color"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/bevan/code-visualizer/internal/canvas"
)

func TestSVGBackend_DrawRectangle_ProducesValidSVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(200, 200)
	red := color.RGBA{R: 255, A: 255}
	blk := color.RGBA{A: 255}

	b.DrawRectangle(
		canvas.Position{X: 10, Y: 10},
		canvas.Size{Width: 80, Height: 60},
		red, blk, 2.0,
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
		canvas.Position{X: 100, Y: 100},
		50, blue, blk, 1.0,
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
		canvas.Position{X: 100, Y: 50},
		"hello", blk, 14.0,
		canvas.AnchorMiddle, 0,
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
		canvas.Position{X: 0, Y: 0},
		canvas.Position{X: 200, Y: 200},
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
		[]canvas.Position{
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
		canvas.Position{X: 200, Y: 200},
		100, "hello", blk, 14.0,
	)

	out := filepath.Join(t.TempDir(), "arctext.svg")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	content := readFile(t, out)
	g.Expect(content).To(ContainSubstring("<textPath"))
}

func TestSVGBackend_ImplementsBackendInterface(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var b canvas.Backend = New(100, 100)
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
