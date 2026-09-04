package svg

import (
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
)

func TestSVGBackend_DrawRectangle_ProducesValidSVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(200, 200)
	red := color.RGBA{R: 255, A: 255}
	blk := color.RGBA{A: 255}

	b.DrawRectangle(
		geometry.RectFromPositionSize(geometry.Point{X: 10, Y: 10}, geometry.Size{Width: 80, Height: 60}),
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

func TestSVGBackend_UsesThreeDecimalPrecisionAcrossPrimitives(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(320, 240)
	blk := color.RGBA{A: 255}

	b.DrawRectangle(
		geometry.RectFromPositionSize(
			geometry.Point{X: 10.1234, Y: 20.5678},
			geometry.Size{Width: 30.1234, Height: 40.5678},
		),
		model.RadialGradientFill{
			Center: color.RGBA{R: 10, G: 20, B: 30, A: 255},
			Edge:   color.RGBA{R: 40, G: 50, B: 60, A: 255},
			Focus:  model.GradientPoint{X: 0.333333, Y: 0.666667},
		},
		model.SolidFill{Color: blk},
		0.4567,
	)
	b.DrawDisc(
		geometry.Point{X: 50.1234, Y: 60.5678},
		7.8912,
		model.SolidFill{Color: color.RGBA{R: 200, G: 100, B: 50, A: 255}},
		model.SolidFill{Color: blk},
		0.4567,
	)
	b.DrawPolygon(
		[]geometry.Point{
			{X: 1.2346, Y: 2.3456},
			{X: 3.4567, Y: 4.5678},
			{X: 5.6789, Y: 6.7891},
		},
		model.SolidFill{Color: color.RGBA{R: 1, G: 2, B: 3, A: 255}},
		model.SolidFill{Color: blk},
		0.4567,
	)
	filledPath, ok := b.(interface {
		DrawFilledPath(loops [][]geometry.Point, fill color.RGBA)
	})
	g.Expect(ok).To(BeTrue())
	filledPath.DrawFilledPath([][]geometry.Point{
		{
			{X: 7.1234, Y: 8.2345},
			{X: 9.3456, Y: 10.4567},
			{X: 11.5678, Y: 12.6789},
		},
	}, color.RGBA{R: 9, G: 8, B: 7, A: 255})
	b.DrawLine(
		geometry.Point{X: 12.3456, Y: 13.4567},
		geometry.Point{X: 14.5678, Y: 15.6789},
		color.RGBA{R: 1, G: 2, B: 3, A: 128},
		0.4567,
	)
	b.DrawPath(
		[]geometry.Point{
			{X: 16.1234, Y: 17.2345},
			{X: 18.3456, Y: 19.4567},
			{X: 20.5678, Y: 21.6789},
		},
		blk,
		0.4567,
	)
	b.DrawText(
		geometry.Point{X: 22.1234, Y: 23.2345},
		"rotated", blk, 12.3456,
		model.AnchorMiddle, 0.44879895,
	)
	b.DrawArcText(
		geometry.Point{X: 100.1234, Y: 200.5678},
		40.9876,
		"arc", blk, 9.8765,
	)

	out := filepath.Join(t.TempDir(), "precision.svg")
	g.Expect(b.Finish(out)).To(Succeed())

	content := readFile(t, out)
	for _, want := range []string{
		`fx="33.333%" fy="66.667%"`,
		`x="10.123" y="20.568" width="30.123" height="40.568"`,
		`stroke-width="0.457"`,
		`<circle cx="50.123" cy="60.568" r="7.891"`,
		`points="1.235,2.346 3.457,4.568 5.679,6.789"`,
		`<path d="M 7.123 8.235 L 9.346 10.457 L 11.568 12.679 Z" fill="rgb(9,8,7)"`,
		`rgba(1,2,3,0.502)`,
		`<line x1="12.346" y1="13.457" x2="14.568" y2="15.679"`,
		`<path d="M 16.123 17.235 L 18.346 19.457 L 20.568 21.679" fill="none"`,
		`font-size="12.346"`,
		`rotate(25.714 22.123 23.235)`,
		`M73.136,200.568 A26.988,26.988 0 0,1 127.111,200.568`,
	} {
		g.Expect(content).To(ContainSubstring(want))
	}

	g.Expect(strings.Count(content, "33.333%")).To(Equal(1))
}

func TestSVGBackend_DrawDisc_ProducesValidSVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(200, 200)
	blue := color.RGBA{B: 255, A: 255}
	blk := color.RGBA{A: 255}

	b.DrawDisc(
		geometry.Point{X: 100, Y: 100},
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
		geometry.Point{X: 100, Y: 50},
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
		geometry.Point{X: 0, Y: 0},
		geometry.Point{X: 200, Y: 200},
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
		[]geometry.Point{
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
	g.Expect(content).To(ContainSubstring(`<path d="M 10.000 10.000 L 100.000 50.000 L 190.000 10.000" fill="none"`))
}

func TestSVGBackend_DrawPolygon_ProducesFilledPolygon(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(10, 10)
	red := color.RGBA{R: 255, A: 255}
	black := color.RGBA{A: 255}
	b.DrawPolygon(
		[]geometry.Point{
			{X: 1, Y: 1},
			{X: 9, Y: 1},
			{X: 1, Y: 9},
		},
		model.SolidFill{Color: red}, model.SolidFill{Color: black}, 0.5,
	)

	out := filepath.Join(t.TempDir(), "polygon.svg")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	content := readFile(t, out)
	g.Expect(content).To(ContainSubstring(`<polygon points="1.000,1.000 9.000,1.000 1.000,9.000"`))
	g.Expect(content).To(ContainSubstring(`fill="rgb(255,0,0)"`))
	g.Expect(content).To(ContainSubstring(`stroke="rgb(0,0,0)"`))
	g.Expect(content).To(ContainSubstring(`stroke-width="0.500"`))
}

func TestSVGBackend_DrawPolygon_WithoutBorder_OmitsStrokeAttributes(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(10, 10)
	b.DrawPolygon(
		[]geometry.Point{
			{X: 1, Y: 1},
			{X: 9, Y: 1},
			{X: 1, Y: 9},
		},
		model.SolidFill{Color: color.RGBA{R: 255, A: 255}},
		model.SolidFill{Color: color.RGBA{A: 255}},
		0,
	)

	out := filepath.Join(t.TempDir(), "borderless-polygon.svg")
	g.Expect(b.Finish(out)).To(Succeed())

	content := readFile(t, out)
	g.Expect(content).To(ContainSubstring("<polygon"))
	g.Expect(content).NotTo(ContainSubstring("stroke="))
	g.Expect(content).NotTo(ContainSubstring("stroke-width="))
}

func TestSVGBackend_DrawFilledPath_ProducesBorderlessEvenOddPath(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(10, 10)
	filledPathBackend, ok := b.(interface {
		DrawFilledPath(loops [][]geometry.Point, fill color.RGBA)
	})

	g.Expect(ok).To(BeTrue())

	if !ok {
		return
	}

	filledPathBackend.DrawFilledPath([][]geometry.Point{
		{
			{X: 1, Y: 1},
			{X: 9, Y: 1},
			{X: 9, Y: 9},
			{X: 1, Y: 9},
		},
		{
			{X: 3, Y: 3},
			{X: 7, Y: 3},
			{X: 7, Y: 7},
			{X: 3, Y: 7},
		},
	}, color.RGBA{R: 255, A: 255})

	out := filepath.Join(t.TempDir(), "filled-path.svg")
	g.Expect(b.Finish(out)).To(Succeed())

	content := readFile(t, out)
	expectedPath := `<path d="M 1.000 1.000 L 9.000 1.000 L 9.000 9.000 L 1.000 9.000 Z ` +
		`M 3.000 3.000 L 7.000 3.000 L 7.000 7.000 L 3.000 7.000 Z"`
	g.Expect(content).To(ContainSubstring(expectedPath))
	g.Expect(content).To(ContainSubstring(`fill-rule="evenodd"`))
	g.Expect(content).NotTo(ContainSubstring("stroke="))
}

func TestSVGBackend_DrawArcText_ProducesValidSVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(400, 400)
	blk := color.RGBA{A: 255}

	b.DrawArcText(
		geometry.Point{X: 200, Y: 200},
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
		geometry.Point{X: 100, Y: 50},
		"hello", blk, 0,
		model.AnchorMiddle, 0,
	)

	out := filepath.Join(t.TempDir(), "text-zero.svg")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	content := readFile(t, out)
	g.Expect(content).To(ContainSubstring("<text"))
	g.Expect(content).NotTo(ContainSubstring(`font-size="0`))
	g.Expect(content).To(ContainSubstring(`font-size="12.000"`))
}

func TestSVGBackend_DrawText_FontSizeNegative_UsesDefault(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(200, 100)
	blk := color.RGBA{A: 255}

	b.DrawText(
		geometry.Point{X: 100, Y: 50},
		"hello", blk, -5.0,
		model.AnchorMiddle, 0,
	)

	out := filepath.Join(t.TempDir(), "text-neg.svg")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	content := readFile(t, out)
	g.Expect(content).NotTo(ContainSubstring(`font-size="0`))
	g.Expect(content).NotTo(ContainSubstring(`font-size="-`))
	g.Expect(content).To(ContainSubstring(`font-size="12.000"`))
}

func TestSVGBackend_DrawArcText_FontSizeZero_UsesDefault(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(400, 400)
	blk := color.RGBA{A: 255}

	b.DrawArcText(
		geometry.Point{X: 200, Y: 200},
		100, "hello", blk, 0,
	)

	out := filepath.Join(t.TempDir(), "arctext-zero.svg")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	content := readFile(t, out)
	g.Expect(content).To(ContainSubstring("<textPath"))
	g.Expect(content).NotTo(ContainSubstring(`font-size="0`))
	g.Expect(content).To(ContainSubstring(`font-size="12.000"`))
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
		geometry.Point{X: 200, Y: 200},
		100, "hello", blk, 14.0,
	)

	out := filepath.Join(t.TempDir(), "arctext-top.svg")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	content := readFile(t, out)
	// Arc starts at left: M <center.X - arcR>, <center.Y> — arcR = 100-14 = 86
	g.Expect(content).To(ContainSubstring("M114.000,200.000"))
	// Arc ends at right: ... <center.X + arcR>, <center.Y>
	g.Expect(content).To(ContainSubstring("286.000,200.000"))
	// sweep-flag=1 (clockwise, through the top), large-arc-flag=0 (half-circle)
	g.Expect(content).To(ContainSubstring("0 0,1"))
}

func TestSVGBackend_DrawArcText_CentersGlyphsOnPath(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(400, 400)
	blk := color.RGBA{A: 255}

	b.DrawArcText(
		geometry.Point{X: 200, Y: 200},
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
