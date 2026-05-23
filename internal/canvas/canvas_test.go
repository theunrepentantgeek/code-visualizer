package canvas

import (
	"image/color"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

type fillAwareInk struct {
	fill     model.Fill
	gotValue MetricValue
	gotFocus model.Point
}

func (*fillAwareInk) Dip(MetricValue) color.RGBA {
	return color.RGBA{R: 255, A: 255}
}

func (ink *fillAwareInk) Fill(value MetricValue, focus model.Point) model.Fill {
	ink.gotValue = value
	ink.gotFocus = focus

	return ink.fill
}

func (*fillAwareInk) Info() InkInfo {
	return InkInfo{Kind: InkFixed}
}

func (*fillAwareInk) legendEntryKind() model.LegendEntryKind {
	return model.LegendEntryNumeric
}

func (*fillAwareInk) legendSwatches() []model.LegendSwatch {
	return nil
}

func TestCanvas_AddRectangle_DispatchesToBackend(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(800, 600)
	focus := model.Point{X: 0.2, Y: 0.8}
	fillValue := MeasureValue(0.75)
	gradient := model.RadialGradientFill{
		Center: color.RGBA{R: 255, A: 255},
		Edge:   color.RGBA{B: 255, A: 255},
		Focus:  focus,
	}
	fillInk := &fillAwareInk{fill: gradient}
	spec := &RectangleSpec{
		ShapeStyle: ShapeStyle{
			Fill:        fillInk,
			Border:      FixedInk(black),
			BorderWidth: 2.0,
		},
	}

	c.AddRectangle(LayerContent, Rectangle{
		Spec:   spec,
		X:      10,
		Y:      20,
		W:      100,
		H:      50,
		Fill:   fillValue,
		Focus:  focus,
		Border: MeasureValue(1.0),
	})

	mb := newMockBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.calls).To(HaveLen(1))
	g.Expect(mb.calls[0].method).To(Equal("DrawRectangle"))
	g.Expect(mb.calls[0].rawFill).To(Equal(model.Fill(gradient)))
	g.Expect(fillInk.gotValue).To(Equal(fillValue))
	g.Expect(fillInk.gotFocus).To(Equal(focus))
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

func TestAddArcText_DispatchesToBackend(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(400, 400)
	spec := &ArcTextSpec{
		Ink:      FixedInk(color.RGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xFF}),
		FontSize: 14,
	}

	c.AddArcText(LayerOverlay, ArcText{
		Spec:   spec,
		X:      200,
		Y:      200,
		Radius: 100,
		Text:   "hello",
	})

	mock := newMockBackend()
	err := c.RenderTo(mock)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mock.calls).To(HaveLen(1))
	g.Expect(mock.calls[0].method).To(Equal("DrawArcText"))
	g.Expect(mock.calls[0].text).To(Equal("hello"))
	g.Expect(mock.calls[0].pos).To(Equal(Position{X: 200, Y: 200}))
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
	ink := NumericInk("test-metric", []float64{10, 50, 90}, pal)

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

func TestCanvas_Render_PNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(200, 200)
	spec := &RectangleSpec{
		ShapeStyle: ShapeStyle{
			Fill:   FixedInk(white),
			Border: FixedInk(black),
		},
	}

	c.AddRectangle(LayerBackground, Rectangle{
		Spec: spec,
		W:    200,
		H:    200,
	})

	out := filepath.Join(t.TempDir(), "output.png")
	err := c.Render(out)
	g.Expect(err).NotTo(HaveOccurred())

	info, statErr := os.Stat(out)
	g.Expect(statErr).NotTo(HaveOccurred())

	if info != nil {
		g.Expect(info.Size()).To(BeNumerically(">", 0))
	}
}

func TestCanvas_Render_SVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(200, 200)
	spec := &RectangleSpec{
		ShapeStyle: ShapeStyle{
			Fill:   FixedInk(white),
			Border: FixedInk(black),
		},
	}

	c.AddRectangle(LayerBackground, Rectangle{
		Spec: spec,
		W:    200,
		H:    200,
	})

	out := filepath.Join(t.TempDir(), "output.svg")
	err := c.Render(out)
	g.Expect(err).NotTo(HaveOccurred())

	data, readErr := os.ReadFile(out)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(data)).To(ContainSubstring("<svg"))
}

func TestCanvas_Render_JPG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(200, 200)
	spec := &RectangleSpec{
		ShapeStyle: ShapeStyle{
			Fill:   FixedInk(white),
			Border: FixedInk(black),
		},
	}

	c.AddRectangle(LayerBackground, Rectangle{
		Spec: spec,
		W:    200,
		H:    200,
	})

	out := filepath.Join(t.TempDir(), "output.jpg")
	err := c.Render(out)
	g.Expect(err).NotTo(HaveOccurred())

	info, statErr := os.Stat(out)
	g.Expect(statErr).NotTo(HaveOccurred())

	if info != nil {
		g.Expect(info.Size()).To(BeNumerically(">", 0))
	}
}

func TestCanvas_Render_UnsupportedFormat(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(100, 100)
	err := c.Render("output.bmp")
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("unsupported"))
	}
}

func TestCanvas_Integration_AllShapeTypes_PNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(800, 600)

	bgSpec := &RectangleSpec{
		ShapeStyle: ShapeStyle{
			Fill:   FixedInk(white),
			Border: FixedInk(white),
		},
	}

	c.AddRectangle(LayerBackground, Rectangle{
		Spec: bgSpec,
		W:    800, H: 600,
	})

	lineSpec := &LineSpec{
		Stroke:      FixedInk(color.RGBA{R: 200, G: 200, B: 200, A: 255}),
		StrokeWidth: 1.0,
	}

	c.AddLine(LayerStructure, Line{
		Spec: lineSpec,
		X1:   0, Y1: 300, X2: 800, Y2: 300,
	})

	pal := palette.GetPalette(palette.Temperature)
	fillInk := NumericInk("test-metric", []float64{10, 20, 30, 40, 50}, pal)

	rectSpec := &RectangleSpec{
		ShapeStyle: ShapeStyle{
			Fill:        fillInk,
			Border:      FixedInk(black),
			BorderWidth: 1.0,
		},
	}

	c.AddRectangle(LayerContent, Rectangle{
		Spec: rectSpec,
		X:    50, Y: 50, W: 200, H: 150,
		Fill: MeasureValue(10),
	})

	c.AddRectangle(LayerContent, Rectangle{
		Spec: rectSpec,
		X:    300, Y: 50, W: 200, H: 150,
		Fill: MeasureValue(50),
	})

	discSpec := &DiscSpec{
		ShapeStyle: ShapeStyle{
			Fill:        FixedInk(color.RGBA{R: 100, G: 200, B: 100, A: 255}),
			Border:      FixedInk(black),
			BorderWidth: 1.0,
		},
	}

	c.AddDisc(LayerContent, Disc{
		Spec: discSpec,
		X:    650, Y: 125, Radius: 60,
	})

	textSpec := &TextSpec{
		Ink:      FixedInk(black),
		FontSize: 14,
		Anchor:   AnchorMiddle,
	}

	c.AddText(LayerOverlay, Text{
		Spec:    textSpec,
		X:       400,
		Y:       500,
		Content: "Canvas Integration Test",
	})

	pathSpec := &LineSpec{
		Stroke:      FixedInk(color.RGBA{R: 255, G: 100, B: 100, A: 255}),
		StrokeWidth: 2.0,
	}

	c.AddPath(LayerStructure, Path{
		Spec: pathSpec,
		Points: []Position{
			{X: 50, Y: 400},
			{X: 200, Y: 350},
			{X: 400, Y: 450},
			{X: 600, Y: 380},
			{X: 750, Y: 420},
		},
	})

	out := filepath.Join(t.TempDir(), "integration.png")
	err := c.Render(out)
	g.Expect(err).NotTo(HaveOccurred())

	info, statErr := os.Stat(out)
	g.Expect(statErr).NotTo(HaveOccurred())

	if info != nil {
		g.Expect(info.Size()).To(BeNumerically(">", 1000))
	}
}

func TestCanvas_SetLegend_DecomposesToPrimitives(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(800, 600)

	pal := palette.GetPalette(palette.Temperature)
	fillInk := NumericInk("file-size", []float64{10, 50, 100}, pal)

	c.SetLegend(LegendConfig{
		Position:    model.LegendPositionBottomRight,
		Orientation: model.LegendOrientationVertical,
		Entries: []LegendEntry{
			{Role: LegendRoleFill, MetricName: "file-size", Ink: fillInk},
		},
	})

	mb := newMockBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())

	// Legend decomposes into: 1 background rect + swatch rects + title text + label texts.
	g.Expect(mb.calls).NotTo(BeEmpty())

	// First call is the background rectangle.
	g.Expect(mb.calls[0].method).To(Equal("DrawRectangle"))

	// Should contain at least one text call for the title.
	hasTitle := false

	for _, call := range mb.calls {
		if call.method == "DrawText" && call.text == "Fill: file-size" {
			hasTitle = true
		}
	}

	g.Expect(hasTitle).To(BeTrue(), "expected title text 'Fill: file-size'")
}

func TestCanvas_NoLegend_NoPrimitives(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(800, 600)
	mb := newMockBackend()

	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.calls).To(BeEmpty())
}

func TestCanvas_Integration_AllShapeTypes_SVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(800, 600)

	bgSpec := &RectangleSpec{
		ShapeStyle: ShapeStyle{
			Fill:   FixedInk(white),
			Border: FixedInk(white),
		},
	}

	c.AddRectangle(LayerBackground, Rectangle{
		Spec: bgSpec,
		W:    800, H: 600,
	})

	discSpec := &DiscSpec{
		ShapeStyle: ShapeStyle{
			Fill:        FixedInk(color.RGBA{R: 100, B: 200, A: 255}),
			Border:      FixedInk(black),
			BorderWidth: 2.0,
		},
	}

	c.AddDisc(LayerContent, Disc{
		Spec: discSpec,
		X:    400, Y: 300, Radius: 100,
	})

	textSpec := &TextSpec{
		Ink:      FixedInk(black),
		FontSize: 16,
		Anchor:   AnchorMiddle,
	}

	c.AddText(LayerOverlay, Text{
		Spec: textSpec,
		X:    400, Y: 300, Content: "SVG Test",
	})

	out := filepath.Join(t.TempDir(), "integration.svg")
	err := c.Render(out)
	g.Expect(err).NotTo(HaveOccurred())

	data, readErr := os.ReadFile(out)
	g.Expect(readErr).NotTo(HaveOccurred())

	content := string(data)
	g.Expect(content).To(ContainSubstring("<svg"))
	g.Expect(content).To(ContainSubstring("<rect"))
	g.Expect(content).To(ContainSubstring("<circle"))
	g.Expect(content).To(ContainSubstring("<text"))
	g.Expect(content).To(ContainSubstring("SVG Test"))
	g.Expect(content).To(ContainSubstring("</svg>"))
}
