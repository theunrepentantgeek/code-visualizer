package canvas_test

import (
	"image/color"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/mock"
)

func TestCanvas_AddBlockLabel_CentersMultilineText(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := canvas.NewCanvas(200, 120)
	c.AddBlockLabel(canvas.LayerOverlay, canvas.BlockLabel{
		X:     20,
		Y:     30,
		W:     160,
		H:     60,
		Lines: []string{"alpha.go", "128"},
		Ink:   color.RGBA{A: 255},
	}, canvas.FormatSVG)

	mb := mock.NewBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.Calls).To(HaveLen(2))
	g.Expect(mb.Calls[0].Method).To(Equal("DrawText"))
	g.Expect(mb.Calls[1].Method).To(Equal("DrawText"))
	g.Expect(mb.Calls[0].Text).To(Equal("alpha.go"))
	g.Expect(mb.Calls[1].Text).To(Equal("128"))
	g.Expect(mb.Calls[0].Pos.X).To(BeNumerically("~", 100.0, 0.01))
	g.Expect(mb.Calls[1].Pos.X).To(BeNumerically("~", 100.0, 0.01))
	g.Expect((mb.Calls[0].Pos.Y + mb.Calls[1].Pos.Y) / 2).To(BeNumerically("~", 60.0, 0.01))
	g.Expect(mb.Calls[0].FontSize).To(BeNumerically(">", 0.0))
	g.Expect(mb.Calls[0].FontSize).To(Equal(mb.Calls[1].FontSize))
	g.Expect(mb.Calls[0].Anchor).To(Equal(canvas.AnchorMiddle))
	g.Expect(mb.Calls[1].Anchor).To(Equal(canvas.AnchorMiddle))
}

func TestCanvas_AddBlockLabel_GreeksTinyRasterLabels(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := canvas.NewCanvas(40, 20)
	c.AddBlockLabel(canvas.LayerOverlay, canvas.BlockLabel{
		X:     5,
		Y:     5,
		W:     30,
		H:     8,
		Lines: []string{"a.go", "42"},
		Ink:   color.RGBA{A: 255},
	}, canvas.FormatPNG)

	mb := mock.NewBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.Calls).To(HaveLen(2))
	g.Expect(mb.Calls[0].Method).To(Equal("DrawLine"))
	g.Expect(mb.Calls[1].Method).To(Equal("DrawLine"))
}

func TestCanvas_AddBlockLabel_OmitsUnreadableRasterLabels(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := canvas.NewCanvas(20, 10)
	c.AddBlockLabel(canvas.LayerOverlay, canvas.BlockLabel{
		X:     2,
		Y:     2,
		W:     16,
		H:     1.5,
		Lines: []string{"a.go"},
		Ink:   color.RGBA{A: 255},
	}, canvas.FormatPNG)

	mb := mock.NewBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.Calls).To(BeEmpty())
}

func TestCanvas_AddBlockLabel_KeepsTinySVGLabelsVisible(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := canvas.NewCanvas(40, 20)
	c.AddBlockLabel(canvas.LayerOverlay, canvas.BlockLabel{
		X:     5,
		Y:     5,
		W:     30,
		H:     8,
		Lines: []string{"a.go", "42"},
		Ink:   color.RGBA{A: 255},
	}, canvas.FormatSVG)

	mb := mock.NewBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.Calls).To(HaveLen(2))
	g.Expect(mb.Calls[0].Method).To(Equal("DrawText"))
	g.Expect(mb.Calls[1].Method).To(Equal("DrawText"))
}
