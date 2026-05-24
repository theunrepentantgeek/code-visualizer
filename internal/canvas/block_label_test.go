package canvas

import (
	"image/color"
	"testing"

	. "github.com/onsi/gomega"
)

func TestCanvas_AddBlockLabel_CentersMultilineText(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(200, 120)
	c.AddBlockLabel(LayerOverlay, BlockLabel{
		X:     20,
		Y:     30,
		W:     160,
		H:     60,
		Lines: []string{"alpha.go", "128"},
		Ink:   color.RGBA{A: 255},
	}, FormatSVG)

	mb := newMockBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.calls).To(HaveLen(2))
	g.Expect(mb.calls[0].method).To(Equal("DrawText"))
	g.Expect(mb.calls[1].method).To(Equal("DrawText"))
	g.Expect(mb.calls[0].text).To(Equal("alpha.go"))
	g.Expect(mb.calls[1].text).To(Equal("128"))
	g.Expect(mb.calls[0].pos.X).To(BeNumerically("~", 100.0, 0.01))
	g.Expect(mb.calls[1].pos.X).To(BeNumerically("~", 100.0, 0.01))
	g.Expect((mb.calls[0].pos.Y + mb.calls[1].pos.Y) / 2).To(BeNumerically("~", 60.0, 0.01))
	g.Expect(mb.calls[0].fontSize).To(BeNumerically(">", 0.0))
	g.Expect(mb.calls[0].fontSize).To(Equal(mb.calls[1].fontSize))
	g.Expect(mb.calls[0].anchor).To(Equal(AnchorMiddle))
	g.Expect(mb.calls[1].anchor).To(Equal(AnchorMiddle))
}

func TestCanvas_AddBlockLabel_GreeksTinyRasterLabels(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(40, 20)
	c.AddBlockLabel(LayerOverlay, BlockLabel{
		X:     5,
		Y:     5,
		W:     30,
		H:     8,
		Lines: []string{"a.go", "42"},
		Ink:   color.RGBA{A: 255},
	}, FormatPNG)

	mb := newMockBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.calls).To(HaveLen(2))
	g.Expect(mb.calls[0].method).To(Equal("DrawLine"))
	g.Expect(mb.calls[1].method).To(Equal("DrawLine"))
}

func TestCanvas_AddBlockLabel_OmitsUnreadableRasterLabels(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(20, 10)
	c.AddBlockLabel(LayerOverlay, BlockLabel{
		X:     2,
		Y:     2,
		W:     16,
		H:     1.5,
		Lines: []string{"a.go"},
		Ink:   color.RGBA{A: 255},
	}, FormatPNG)

	mb := newMockBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.calls).To(BeEmpty())
}

func TestCanvas_AddBlockLabel_KeepsTinySVGLabelsVisible(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(40, 20)
	c.AddBlockLabel(LayerOverlay, BlockLabel{
		X:     5,
		Y:     5,
		W:     30,
		H:     8,
		Lines: []string{"a.go", "42"},
		Ink:   color.RGBA{A: 255},
	}, FormatSVG)

	mb := newMockBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.calls).To(HaveLen(2))
	g.Expect(mb.calls[0].method).To(Equal("DrawText"))
	g.Expect(mb.calls[1].method).To(Equal("DrawText"))
}
