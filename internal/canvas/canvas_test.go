package canvas

import (
	"image/color"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

type fillAwareInk struct {
	fill     model.Fill
	gotValue inks.MetricValue
	gotFocus model.Point
}

func (*fillAwareInk) Dip(inks.MetricValue) color.RGBA {
	return color.RGBA{R: 255, A: 255}
}

func (ink *fillAwareInk) Fill(value inks.MetricValue, focus model.Point) model.Fill {
	ink.gotValue = value
	ink.gotFocus = focus

	return ink.fill
}

func (*fillAwareInk) Info() inks.Info {
	return inks.Info{Kind: inks.KindFixed}
}

func (*fillAwareInk) Boundaries() []float64          { return nil }
func (*fillAwareInk) Palette() palette.ColourPalette { return palette.ColourPalette{} }
func (*fillAwareInk) Categories() []string           { return nil }

func TestCanvas_AddRectangle_DispatchesToBackend(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(800, 600)
	focus := model.Point{X: 0.2, Y: 0.8}
	fillValue := inks.MeasureValue(0.75)
	gradient := model.RadialGradientFill{
		Center: color.RGBA{R: 255, A: 255},
		Edge:   color.RGBA{B: 255, A: 255},
		Focus:  focus,
	}
	fillInk := &fillAwareInk{fill: gradient}
	spec := &RectangleSpec{
		ShapeStyle: ShapeStyle{
			Fill:        fillInk,
			Border:      inks.FixedInk(black),
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
		Border: inks.MeasureValue(1.0),
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
			Fill:        inks.FixedInk(blue),
			Border:      inks.FixedInk(black),
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
		Ink:      inks.FixedInk(black),
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
		Stroke:      inks.FixedInk(black),
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
		Stroke:      inks.FixedInk(black),
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
		Ink:      inks.FixedInk(color.RGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xFF}),
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
			Fill:   inks.FixedInk(white),
			Border: inks.FixedInk(white),
		},
	}

	fgSpec := &RectangleSpec{
		ShapeStyle: ShapeStyle{
			Fill:   inks.FixedInk(black),
			Border: inks.FixedInk(black),
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
			Fill:   inks.FixedInk(red),
			Border: inks.FixedInk(red),
		},
	}

	spec2 := &RectangleSpec{
		ShapeStyle: ShapeStyle{
			Fill:   inks.FixedInk(green),
			Border: inks.FixedInk(green),
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
	ink := inks.NumericInk("test-metric", []float64{10, 50, 90}, pal)

	spec := &RectangleSpec{
		ShapeStyle: ShapeStyle{
			Fill:   ink,
			Border: inks.FixedInk(black),
		},
	}

	c.AddRectangle(LayerContent, Rectangle{
		Spec: spec,
		W:    100,
		H:    100,
		Fill: inks.MeasureValue(10),
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
			Fill:   inks.FixedInk(white),
			Border: inks.FixedInk(black),
		},
	}

	lineSpec := &LineSpec{
		Stroke:      inks.FixedInk(black),
		StrokeWidth: 1.0,
	}

	textSpec := &TextSpec{
		Ink:      inks.FixedInk(black),
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
			Fill:   inks.FixedInk(white),
			Border: inks.FixedInk(black),
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
			Fill:   inks.FixedInk(white),
			Border: inks.FixedInk(black),
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
			Fill:   inks.FixedInk(white),
			Border: inks.FixedInk(black),
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
			Fill:   inks.FixedInk(white),
			Border: inks.FixedInk(white),
		},
	}

	c.AddRectangle(LayerBackground, Rectangle{
		Spec: bgSpec,
		W:    800, H: 600,
	})

	lineSpec := &LineSpec{
		Stroke:      inks.FixedInk(color.RGBA{R: 200, G: 200, B: 200, A: 255}),
		StrokeWidth: 1.0,
	}

	c.AddLine(LayerStructure, Line{
		Spec: lineSpec,
		X1:   0, Y1: 300, X2: 800, Y2: 300,
	})

	pal := palette.GetPalette(palette.Temperature)
	fillInk := inks.NumericInk("test-metric", []float64{10, 20, 30, 40, 50}, pal)

	rectSpec := &RectangleSpec{
		ShapeStyle: ShapeStyle{
			Fill:        fillInk,
			Border:      inks.FixedInk(black),
			BorderWidth: 1.0,
		},
	}

	c.AddRectangle(LayerContent, Rectangle{
		Spec: rectSpec,
		X:    50, Y: 50, W: 200, H: 150,
		Fill: inks.MeasureValue(10),
	})

	c.AddRectangle(LayerContent, Rectangle{
		Spec: rectSpec,
		X:    300, Y: 50, W: 200, H: 150,
		Fill: inks.MeasureValue(50),
	})

	discSpec := &DiscSpec{
		ShapeStyle: ShapeStyle{
			Fill:        inks.FixedInk(color.RGBA{R: 100, G: 200, B: 100, A: 255}),
			Border:      inks.FixedInk(black),
			BorderWidth: 1.0,
		},
	}

	c.AddDisc(LayerContent, Disc{
		Spec: discSpec,
		X:    650, Y: 125, Radius: 60,
	})

	textSpec := &TextSpec{
		Ink:      inks.FixedInk(black),
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
		Stroke:      inks.FixedInk(color.RGBA{R: 255, G: 100, B: 100, A: 255}),
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
	fillInk := inks.NumericInk("file-size", []float64{10, 50, 100}, pal)

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

	// Should contain text calls for the label and metric on separate lines.
	hasLabel := false
	hasMetric := false

	for _, call := range mb.calls {
		if call.method == "DrawText" && call.text == "Fill" {
			hasLabel = true
		}

		if call.method == "DrawText" && call.text == "file-size" {
			hasMetric = true
		}
	}

	g.Expect(hasLabel).To(BeTrue(), "expected label text 'Fill'")
	g.Expect(hasMetric).To(BeTrue(), "expected metric text 'file-size'")
}

func TestCanvas_SetLegend_WithLabelSample_RendersSampleBeforeEntries(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(800, 600)
	pal := palette.GetPalette(palette.Temperature)
	fillInk := inks.NumericInk("file-size", []float64{10, 50, 100}, pal)

	c.SetLegend(LegendConfig{
		Position:    model.LegendPositionBottomRight,
		Orientation: model.LegendOrientationVertical,
		LabelSample: []string{"file-name", "file-size", "file-type"},
		Entries: []LegendEntry{
			{Role: LegendRoleFill, MetricName: "file-size", Ink: fillInk},
		},
	})

	mb := newMockBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())

	var sampleY float64

	var sampleFound bool

	var titleY float64

	var titleFound bool

	for _, call := range mb.calls {
		if call.method == "DrawText" && call.text == "file-name" {
			sampleY = call.pos.Y
			sampleFound = true
		}

		if call.method == "DrawText" && call.text == "Fill" {
			titleY = call.pos.Y
			titleFound = true
		}
	}

	g.Expect(sampleFound).To(BeTrue())
	g.Expect(titleFound).To(BeTrue())
	g.Expect(sampleY).To(BeNumerically("<", titleY))
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
			Fill:   inks.FixedInk(white),
			Border: inks.FixedInk(white),
		},
	}

	c.AddRectangle(LayerBackground, Rectangle{
		Spec: bgSpec,
		W:    800, H: 600,
	})

	discSpec := &DiscSpec{
		ShapeStyle: ShapeStyle{
			Fill:        inks.FixedInk(color.RGBA{R: 100, B: 200, A: 255}),
			Border:      inks.FixedInk(black),
			BorderWidth: 2.0,
		},
	}

	c.AddDisc(LayerContent, Disc{
		Spec: discSpec,
		X:    400, Y: 300, Radius: 100,
	})

	textSpec := &TextSpec{
		Ink:      inks.FixedInk(black),
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

func TestCanvas_SetFooter_RendersTextAtBottom(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(800, 600)
	c.SetFooter("Generated by codeviz at 2026-06-01")

	mb := newMockBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(mb.calls).To(HaveLen(1))
	g.Expect(mb.calls[0].method).To(Equal("DrawText"))
	g.Expect(mb.calls[0].text).To(Equal("Generated by codeviz at 2026-06-01"))
	g.Expect(mb.calls[0].pos.X).To(Equal(400.0)) // width/2
	g.Expect(mb.calls[0].anchor).To(Equal(AnchorMiddle))
}

func TestCanvas_SetFooter_EmptyString_NoFooterRendered(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(800, 600)
	c.SetFooter("some text")
	c.SetFooter("") // clear it

	mb := newMockBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.calls).To(BeEmpty())
}

func TestCanvas_NoFooter_NoExtraDrawCall(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(800, 600)

	mb := newMockBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.calls).To(BeEmpty())
}

func TestCanvas_SetTitle_RendersTextAtTop(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(800, 600)
	c.SetTitle("My Repository")

	mb := newMockBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(mb.calls).To(HaveLen(1))
	g.Expect(mb.calls[0].method).To(Equal("DrawText"))
	g.Expect(mb.calls[0].text).To(Equal("My Repository"))
	g.Expect(mb.calls[0].pos.X).To(Equal(400.0)) // width/2
	g.Expect(mb.calls[0].anchor).To(Equal(AnchorMiddle))
}

func TestCanvas_SetTitle_EmptyString_NoTitleRendered(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(800, 600)
	c.SetTitle("some title")
	c.SetTitle("") // clear it

	mb := newMockBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.calls).To(BeEmpty())
}

func TestCanvas_NoTitle_NoExtraDrawCall(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(800, 600)

	mb := newMockBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.calls).To(BeEmpty())
}

func TestCanvas_TitleText_ReturnsSetValue(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(800, 600)
	g.Expect(c.TitleText()).To(BeEmpty())

	c.SetTitle("Hello")
	g.Expect(c.TitleText()).To(Equal("Hello"))

	c.SetTitle("")
	g.Expect(c.TitleText()).To(BeEmpty())
}

func TestCanvas_DrawingBounds_Getters_ReturnZerosByDefault(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(800, 600)
	g.Expect(c.DrawingMinY()).To(Equal(0))
	g.Expect(c.DrawingMaxY()).To(Equal(600))
}

func TestCanvas_DrawingBounds_Getters_ReturnSetValues(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(800, 600)
	c.SetDrawingBounds(40, 560)
	g.Expect(c.DrawingMinY()).To(Equal(40))
	g.Expect(c.DrawingMaxY()).To(Equal(560))
}
