package canvas_test

import (
	"image/color"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/mock"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

var black = color.RGBA{A: 255}

type fillAwareInk struct {
	fillValue model.Fill
	gotValue  inks.MetricValue
	gotFocus  model.Point
}

func (*fillAwareInk) Dip(inks.MetricValue) color.RGBA {
	return color.RGBA{R: 255, A: 255}
}

func (ink *fillAwareInk) Fill(value inks.MetricValue, focus model.Point) model.Fill {
	ink.gotValue = value
	ink.gotFocus = focus

	return ink.fillValue
}

func (*fillAwareInk) Info() inks.Info {
	return inks.Info{Kind: inks.KindFixed}
}

func (*fillAwareInk) LegendData() (model.LegendEntryKind, []model.LegendSwatch) {
	return model.LegendEntryNumeric, nil
}

func TestCanvas_AddRectangle_DispatchesToBackend(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := canvas.NewCanvas(800, 600)
	focus := model.Point{X: 0.2, Y: 0.8}
	fillValue := inks.MeasureValue(0.75)
	gradient := model.RadialGradientFill{
		Center: color.RGBA{R: 255, A: 255},
		Edge:   color.RGBA{B: 255, A: 255},
		Focus:  focus,
	}
	fillInk := &fillAwareInk{fillValue: gradient}
	spec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        fillInk,
			Border:      inks.FixedInk(black),
			BorderWidth: 2.0,
		},
	}

	c.AddRectangle(canvas.LayerContent, canvas.Rectangle{
		Spec:   spec,
		X:      10,
		Y:      20,
		W:      100,
		H:      50,
		Fill:   fillValue,
		Focus:  focus,
		Border: inks.MeasureValue(1.0),
	})

	mb := mock.NewBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.Calls).To(HaveLen(1))
	g.Expect(mb.Calls[0].Method).To(Equal("DrawRectangle"))
	g.Expect(mb.Calls[0].RawFill).To(Equal(model.Fill(gradient)))
	g.Expect(fillInk.gotValue).To(Equal(fillValue))
	g.Expect(fillInk.gotFocus).To(Equal(focus))
}

func TestCanvas_AddDisc_DispatchesToBackend(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := canvas.NewCanvas(800, 600)
	blue := color.RGBA{B: 255, A: 255}
	spec := &canvas.DiscSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        inks.FixedInk(blue),
			Border:      inks.FixedInk(black),
			BorderWidth: 1.0,
		},
	}

	c.AddDisc(canvas.LayerContent, canvas.Disc{
		Spec:   spec,
		X:      400,
		Y:      300,
		Radius: 50,
	})

	mb := mock.NewBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.Calls).To(HaveLen(1))
	g.Expect(mb.Calls[0].Method).To(Equal("DrawDisc"))
	g.Expect(mb.Calls[0].Fill).To(Equal(blue))
}

func TestCanvas_AddText_DispatchesToBackend(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := canvas.NewCanvas(800, 600)
	spec := &canvas.TextSpec{
		Ink:      inks.FixedInk(black),
		FontSize: 14,
		Anchor:   canvas.AnchorMiddle,
	}

	c.AddText(canvas.LayerOverlay, canvas.Text{
		Spec:    spec,
		X:       100,
		Y:       200,
		Content: "hello",
	})

	mb := mock.NewBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.Calls).To(HaveLen(1))
	g.Expect(mb.Calls[0].Method).To(Equal("DrawText"))
	g.Expect(mb.Calls[0].Text).To(Equal("hello"))
}

func TestCanvas_AddLine_DispatchesToBackend(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := canvas.NewCanvas(800, 600)
	spec := &canvas.LineSpec{
		Stroke:      inks.FixedInk(black),
		StrokeWidth: 1.0,
	}

	c.AddLine(canvas.LayerStructure, canvas.Line{
		Spec: spec,
		X1:   0, Y1: 0, X2: 100, Y2: 100,
	})

	mb := mock.NewBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.Calls).To(HaveLen(1))
	g.Expect(mb.Calls[0].Method).To(Equal("DrawLine"))
}

func TestCanvas_AddPath_DispatchesToBackend(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := canvas.NewCanvas(800, 600)
	spec := &canvas.LineSpec{
		Stroke:      inks.FixedInk(black),
		StrokeWidth: 2.0,
	}

	c.AddPath(canvas.LayerStructure, canvas.Path{
		Spec: spec,
		Points: []canvas.Position{
			{X: 0, Y: 0},
			{X: 50, Y: 50},
			{X: 100, Y: 0},
		},
	})

	mb := mock.NewBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.Calls).To(HaveLen(1))
	g.Expect(mb.Calls[0].Method).To(Equal("DrawPath"))
}

func TestAddArcText_DispatchesToBackend(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := canvas.NewCanvas(400, 400)
	spec := &canvas.ArcTextSpec{
		Ink:      inks.FixedInk(color.RGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xFF}),
		FontSize: 14,
	}

	c.AddArcText(canvas.LayerOverlay, canvas.ArcText{
		Spec:   spec,
		X:      200,
		Y:      200,
		Radius: 100,
		Text:   "hello",
	})

	mb := mock.NewBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.Calls).To(HaveLen(1))
	g.Expect(mb.Calls[0].Method).To(Equal("DrawArcText"))
	g.Expect(mb.Calls[0].Text).To(Equal("hello"))
	g.Expect(mb.Calls[0].Pos).To(Equal(canvas.Position{X: 200, Y: 200}))
}

func TestCanvas_LayerOrdering_BackgroundBeforeContent(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := canvas.NewCanvas(800, 600)
	bgSpec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:   inks.FixedInk(palette.White),
			Border: inks.FixedInk(palette.White),
		},
	}

	fgSpec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:   inks.FixedInk(black),
			Border: inks.FixedInk(black),
		},
	}

	// Add content first, then background — layer ordering should override insertion order.
	c.AddRectangle(canvas.LayerContent, canvas.Rectangle{
		Spec: fgSpec,
		X:    0, Y: 0, W: 100, H: 100,
	})
	c.AddRectangle(canvas.LayerBackground, canvas.Rectangle{
		Spec: bgSpec,
		X:    0, Y: 0, W: 800, H: 600,
	})

	mb := mock.NewBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.Calls).To(HaveLen(2))
	g.Expect(mb.Calls[0].Fill).To(Equal(palette.White))
	g.Expect(mb.Calls[1].Fill).To(Equal(black))
}

func TestCanvas_InsertionOrder_WithinSameLayer(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := canvas.NewCanvas(800, 600)
	red := color.RGBA{R: 255, A: 255}
	green := color.RGBA{G: 255, A: 255}

	spec1 := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:   inks.FixedInk(red),
			Border: inks.FixedInk(red),
		},
	}

	spec2 := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:   inks.FixedInk(green),
			Border: inks.FixedInk(green),
		},
	}

	c.AddRectangle(canvas.LayerContent, canvas.Rectangle{Spec: spec1, W: 100, H: 100})
	c.AddRectangle(canvas.LayerContent, canvas.Rectangle{Spec: spec2, W: 50, H: 50})

	mb := mock.NewBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.Calls).To(HaveLen(2))
	g.Expect(mb.Calls[0].Fill).To(Equal(red))
	g.Expect(mb.Calls[1].Fill).To(Equal(green))
}

func TestCanvas_InkResolution_NumericInk(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := canvas.NewCanvas(400, 400)
	pal := palette.GetPalette(palette.Neutral)
	ink := inks.NumericInk("test-metric", []float64{10, 50, 90}, pal)

	spec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:   ink,
			Border: inks.FixedInk(black),
		},
	}

	c.AddRectangle(canvas.LayerContent, canvas.Rectangle{
		Spec: spec,
		W:    100,
		H:    100,
		Fill: inks.MeasureValue(10),
	})

	mb := mock.NewBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.Calls).To(HaveLen(1))
	g.Expect(mb.Calls[0].Fill.A).To(Equal(uint8(255)))
}

func TestCanvas_MultipleShapeTypes_MixedLayers(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := canvas.NewCanvas(800, 600)
	rectSpec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:   inks.FixedInk(palette.White),
			Border: inks.FixedInk(black),
		},
	}

	lineSpec := &canvas.LineSpec{
		Stroke:      inks.FixedInk(black),
		StrokeWidth: 1.0,
	}

	textSpec := &canvas.TextSpec{
		Ink:      inks.FixedInk(black),
		FontSize: 12,
	}

	c.AddText(canvas.LayerOverlay, canvas.Text{Spec: textSpec, Content: "label"})
	c.AddLine(canvas.LayerStructure, canvas.Line{Spec: lineSpec, X2: 100, Y2: 100})
	c.AddRectangle(canvas.LayerBackground, canvas.Rectangle{Spec: rectSpec, W: 800, H: 600})

	mb := mock.NewBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.Calls).To(HaveLen(3))
	g.Expect(mb.Calls[0].Method).To(Equal("DrawRectangle"))
	g.Expect(mb.Calls[1].Method).To(Equal("DrawLine"))
	g.Expect(mb.Calls[2].Method).To(Equal("DrawText"))
}

func TestCanvas_Empty_NoErrors(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := canvas.NewCanvas(100, 100)
	mb := mock.NewBackend()

	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.Calls).To(BeEmpty())
}

func TestCanvas_Render_PNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := canvas.NewCanvas(200, 200)
	spec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:   inks.FixedInk(palette.White),
			Border: inks.FixedInk(black),
		},
	}

	c.AddRectangle(canvas.LayerBackground, canvas.Rectangle{
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

	c := canvas.NewCanvas(200, 200)
	spec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:   inks.FixedInk(palette.White),
			Border: inks.FixedInk(black),
		},
	}

	c.AddRectangle(canvas.LayerBackground, canvas.Rectangle{
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

	c := canvas.NewCanvas(200, 200)
	spec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:   inks.FixedInk(palette.White),
			Border: inks.FixedInk(black),
		},
	}

	c.AddRectangle(canvas.LayerBackground, canvas.Rectangle{
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

	c := canvas.NewCanvas(100, 100)
	err := c.Render("output.bmp")
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("unsupported"))
	}
}

func TestCanvas_Integration_AllShapeTypes_PNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := canvas.NewCanvas(800, 600)

	bgSpec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:   inks.FixedInk(palette.White),
			Border: inks.FixedInk(palette.White),
		},
	}

	c.AddRectangle(canvas.LayerBackground, canvas.Rectangle{
		Spec: bgSpec,
		W:    800, H: 600,
	})

	lineSpec := &canvas.LineSpec{
		Stroke:      inks.FixedInk(color.RGBA{R: 200, G: 200, B: 200, A: 255}),
		StrokeWidth: 1.0,
	}

	c.AddLine(canvas.LayerStructure, canvas.Line{
		Spec: lineSpec,
		X1:   0, Y1: 300, X2: 800, Y2: 300,
	})

	pal := palette.GetPalette(palette.Temperature)
	fillInk := inks.NumericInk("test-metric", []float64{10, 20, 30, 40, 50}, pal)

	rectSpec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        fillInk,
			Border:      inks.FixedInk(black),
			BorderWidth: 1.0,
		},
	}

	c.AddRectangle(canvas.LayerContent, canvas.Rectangle{
		Spec: rectSpec,
		X:    50, Y: 50, W: 200, H: 150,
		Fill: inks.MeasureValue(10),
	})

	c.AddRectangle(canvas.LayerContent, canvas.Rectangle{
		Spec: rectSpec,
		X:    300, Y: 50, W: 200, H: 150,
		Fill: inks.MeasureValue(50),
	})

	discSpec := &canvas.DiscSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        inks.FixedInk(color.RGBA{R: 100, G: 200, B: 100, A: 255}),
			Border:      inks.FixedInk(black),
			BorderWidth: 1.0,
		},
	}

	c.AddDisc(canvas.LayerContent, canvas.Disc{
		Spec: discSpec,
		X:    650, Y: 125, Radius: 60,
	})

	textSpec := &canvas.TextSpec{
		Ink:      inks.FixedInk(black),
		FontSize: 14,
		Anchor:   canvas.AnchorMiddle,
	}

	c.AddText(canvas.LayerOverlay, canvas.Text{
		Spec:    textSpec,
		X:       400,
		Y:       500,
		Content: "Canvas Integration Test",
	})

	pathSpec := &canvas.LineSpec{
		Stroke:      inks.FixedInk(color.RGBA{R: 255, G: 100, B: 100, A: 255}),
		StrokeWidth: 2.0,
	}

	c.AddPath(canvas.LayerStructure, canvas.Path{
		Spec: pathSpec,
		Points: []canvas.Position{
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

func TestCanvas_EmptyCanvas_NoPrimitives(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := canvas.NewCanvas(800, 600)
	mb := mock.NewBackend()

	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.Calls).To(BeEmpty())
}

func TestCanvas_Integration_AllShapeTypes_SVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := canvas.NewCanvas(800, 600)

	bgSpec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:   inks.FixedInk(palette.White),
			Border: inks.FixedInk(palette.White),
		},
	}

	c.AddRectangle(canvas.LayerBackground, canvas.Rectangle{
		Spec: bgSpec,
		W:    800, H: 600,
	})

	discSpec := &canvas.DiscSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        inks.FixedInk(color.RGBA{R: 100, B: 200, A: 255}),
			Border:      inks.FixedInk(black),
			BorderWidth: 2.0,
		},
	}

	c.AddDisc(canvas.LayerContent, canvas.Disc{
		Spec: discSpec,
		X:    400, Y: 300, Radius: 100,
	})

	textSpec := &canvas.TextSpec{
		Ink:      inks.FixedInk(black),
		FontSize: 16,
		Anchor:   canvas.AnchorMiddle,
	}

	c.AddText(canvas.LayerOverlay, canvas.Text{
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

	c := canvas.NewCanvas(800, 600)
	c.SetFooter("Generated by codeviz at 2026-06-01")

	mb := mock.NewBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(mb.Calls).To(HaveLen(1))
	g.Expect(mb.Calls[0].Method).To(Equal("DrawText"))
	g.Expect(mb.Calls[0].Text).To(Equal("Generated by codeviz at 2026-06-01"))
	g.Expect(mb.Calls[0].Pos.X).To(Equal(400.0)) // width/2
	g.Expect(mb.Calls[0].Anchor).To(Equal(canvas.AnchorMiddle))
}

func TestCanvas_SetFooter_EmptyString_NoFooterRendered(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := canvas.NewCanvas(800, 600)
	c.SetFooter("some text")
	c.SetFooter("") // clear it

	mb := mock.NewBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.Calls).To(BeEmpty())
}

func TestCanvas_NoFooter_NoExtraDrawCall(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := canvas.NewCanvas(800, 600)

	mb := mock.NewBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.Calls).To(BeEmpty())
}

func TestCanvas_SetTitle_RendersTextAtTop(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := canvas.NewCanvas(800, 600)
	c.SetTitle("My Repository")

	mb := mock.NewBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(mb.Calls).To(HaveLen(1))
	g.Expect(mb.Calls[0].Method).To(Equal("DrawText"))
	g.Expect(mb.Calls[0].Text).To(Equal("My Repository"))
	g.Expect(mb.Calls[0].Pos.X).To(Equal(400.0)) // width/2
	g.Expect(mb.Calls[0].Anchor).To(Equal(canvas.AnchorMiddle))
}

func TestCanvas_SetTitle_EmptyString_NoTitleRendered(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := canvas.NewCanvas(800, 600)
	c.SetTitle("some title")
	c.SetTitle("") // clear it

	mb := mock.NewBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.Calls).To(BeEmpty())
}

func TestCanvas_NoTitle_NoExtraDrawCall(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := canvas.NewCanvas(800, 600)

	mb := mock.NewBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.Calls).To(BeEmpty())
}

func TestCanvas_TitleText_ReturnsSetValue(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := canvas.NewCanvas(800, 600)
	g.Expect(c.TitleText()).To(BeEmpty())

	c.SetTitle("Hello")
	g.Expect(c.TitleText()).To(Equal("Hello"))

	c.SetTitle("")
	g.Expect(c.TitleText()).To(BeEmpty())
}

func TestCanvas_DrawingBounds_Getters_ReturnZerosByDefault(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := canvas.NewCanvas(800, 600)
	g.Expect(c.DrawingMinY()).To(Equal(0))
	g.Expect(c.DrawingMaxY()).To(Equal(600))
}

func TestCanvas_DrawingBounds_Getters_ReturnSetValues(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := canvas.NewCanvas(800, 600)
	c.SetDrawingBounds(40, 560)
	g.Expect(c.DrawingMinY()).To(Equal(40))
	g.Expect(c.DrawingMaxY()).To(Equal(560))
}
