package canvas

import (
	"image/color"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/bevan/code-visualizer/internal/palette"
)

func TestCanvas_AddRectangle_DispatchesToBackend(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(800, 600)
	red := color.RGBA{R: 255, A: 255}
	spec := &RectangleSpec{
		ShapeStyle: ShapeStyle{
			Fill:        FixedInk(red),
			Border:      FixedInk(black),
			BorderWidth: 2.0,
		},
	}

	c.AddRectangle(LayerContent, Rectangle{
		Spec: spec,
		X:    10, Y: 20, W: 100, H: 50,
	})

	mb := newMockBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.calls).To(HaveLen(1))
	g.Expect(mb.calls[0].method).To(Equal("DrawRectangle"))
	g.Expect(mb.calls[0].fill).To(Equal(red))
}

func TestCanvas_AddDisc_DispatchesToBackend(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(800, 600)
	blue := color.RGBA{B: 255, A: 255}
	spec := &DiscSpec{
		ShapeStyle: ShapeStyle{
			Fill:        FixedInk(blue),
			Border:      FixedInk(black),
			BorderWidth: 1.0,
		},
	}

	c.AddDisc(LayerContent, Disc{
		Spec:   spec,
		X:      400,
		Y:      300,
		Radius: 50,
	})

	mb := newMockBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.calls).To(HaveLen(1))
	g.Expect(mb.calls[0].method).To(Equal("DrawDisc"))
	g.Expect(mb.calls[0].fill).To(Equal(blue))
}

func TestCanvas_AddText_DispatchesToBackend(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(800, 600)
	spec := &TextSpec{
		Ink:      FixedInk(black),
		FontSize: 14,
		Anchor:   AnchorMiddle,
	}

	c.AddText(LayerOverlay, Text{
		Spec:    spec,
		X:       100,
		Y:       200,
		Content: "hello",
	})

	mb := newMockBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.calls).To(HaveLen(1))
	g.Expect(mb.calls[0].method).To(Equal("DrawText"))
	g.Expect(mb.calls[0].text).To(Equal("hello"))
}

func TestCanvas_AddLine_DispatchesToBackend(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(800, 600)
	spec := &LineSpec{
		Stroke:      FixedInk(black),
		StrokeWidth: 1.0,
	}

	c.AddLine(LayerStructure, Line{
		Spec: spec,
		X1:   0, Y1: 0, X2: 100, Y2: 100,
	})

	mb := newMockBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.calls).To(HaveLen(1))
	g.Expect(mb.calls[0].method).To(Equal("DrawLine"))
}

func TestCanvas_AddPath_DispatchesToBackend(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(800, 600)
	spec := &LineSpec{
		Stroke:      FixedInk(black),
		StrokeWidth: 2.0,
	}

	c.AddPath(LayerStructure, Path{
		Spec: spec,
		Points: []Position{
			{X: 0, Y: 0},
			{X: 50, Y: 50},
			{X: 100, Y: 0},
		},
	})

	mb := newMockBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.calls).To(HaveLen(1))
	g.Expect(mb.calls[0].method).To(Equal("DrawPath"))
}

func TestCanvas_LayerOrdering_BackgroundBeforeContent(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(800, 600)
	bgSpec := &RectangleSpec{
		ShapeStyle: ShapeStyle{
			Fill:   FixedInk(white),
			Border: FixedInk(white),
		},
	}

	fgSpec := &RectangleSpec{
		ShapeStyle: ShapeStyle{
			Fill:   FixedInk(black),
			Border: FixedInk(black),
		},
	}

	// Add content first, then background — layer ordering should override insertion order.
	c.AddRectangle(LayerContent, Rectangle{
		Spec: fgSpec,
		X:    0, Y: 0, W: 100, H: 100,
	})
	c.AddRectangle(LayerBackground, Rectangle{
		Spec: bgSpec,
		X:    0, Y: 0, W: 800, H: 600,
	})

	mb := newMockBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.calls).To(HaveLen(2))
	g.Expect(mb.calls[0].fill).To(Equal(white))
	g.Expect(mb.calls[1].fill).To(Equal(black))
}

func TestCanvas_InsertionOrder_WithinSameLayer(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(800, 600)
	red := color.RGBA{R: 255, A: 255}
	green := color.RGBA{G: 255, A: 255}

	spec1 := &RectangleSpec{
		ShapeStyle: ShapeStyle{
			Fill:   FixedInk(red),
			Border: FixedInk(red),
		},
	}

	spec2 := &RectangleSpec{
		ShapeStyle: ShapeStyle{
			Fill:   FixedInk(green),
			Border: FixedInk(green),
		},
	}

	c.AddRectangle(LayerContent, Rectangle{Spec: spec1, W: 100, H: 100})
	c.AddRectangle(LayerContent, Rectangle{Spec: spec2, W: 50, H: 50})

	mb := newMockBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.calls).To(HaveLen(2))
	g.Expect(mb.calls[0].fill).To(Equal(red))
	g.Expect(mb.calls[1].fill).To(Equal(green))
}

func TestCanvas_InkResolution_NumericInk(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(400, 400)
	pal := palette.GetPalette(palette.Neutral)
	ink := NumericInk([]float64{10, 50, 90}, pal)

	spec := &RectangleSpec{
		ShapeStyle: ShapeStyle{
			Fill:   ink,
			Border: FixedInk(black),
		},
	}

	c.AddRectangle(LayerContent, Rectangle{
		Spec: spec,
		W:    100,
		H:    100,
		Fill: MeasureValue(10),
	})

	mb := newMockBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.calls).To(HaveLen(1))
	g.Expect(mb.calls[0].fill.A).To(Equal(uint8(255)))
}

func TestCanvas_MultipleShapeTypes_MixedLayers(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(800, 600)
	rectSpec := &RectangleSpec{
		ShapeStyle: ShapeStyle{
			Fill:   FixedInk(white),
			Border: FixedInk(black),
		},
	}

	lineSpec := &LineSpec{
		Stroke:      FixedInk(black),
		StrokeWidth: 1.0,
	}

	textSpec := &TextSpec{
		Ink:      FixedInk(black),
		FontSize: 12,
	}

	c.AddText(LayerOverlay, Text{Spec: textSpec, Content: "label"})
	c.AddLine(LayerStructure, Line{Spec: lineSpec, X2: 100, Y2: 100})
	c.AddRectangle(LayerBackground, Rectangle{Spec: rectSpec, W: 800, H: 600})

	mb := newMockBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.calls).To(HaveLen(3))
	g.Expect(mb.calls[0].method).To(Equal("DrawRectangle"))
	g.Expect(mb.calls[1].method).To(Equal("DrawLine"))
	g.Expect(mb.calls[2].method).To(Equal("DrawText"))
}

func TestCanvas_Empty_NoErrors(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(100, 100)
	mb := newMockBackend()

	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.calls).To(BeEmpty())
}
