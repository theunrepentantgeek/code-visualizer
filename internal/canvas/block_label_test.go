package canvas_test

import (
	"image/color"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/mock"
	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
)

func TestCanvas_AddBlockLabel_CentersMultilineText(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := canvas.NewCanvas(200, 120)
	c.AddBlockLabel(canvas.LayerOverlay, canvas.BlockLabel{
		Bounds: geometry.Rect{Min: geometry.NewPoint(20, 30), Max: geometry.NewPoint(180, 90)},
		Lines:  []string{"alpha.go", "128"},
		Ink:    color.RGBA{A: 255},
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

func TestCanvas_AddBlockLabel_TinyRasterLabelStrategy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		format canvas.ImageFormat
		method string
	}{
		{name: "PNG greeks tiny raster labels", format: canvas.FormatPNG, method: "DrawLine"},
		{name: "SVG keeps tiny labels visible", format: canvas.FormatSVG, method: "DrawText"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			c := canvas.NewCanvas(40, 20)
			c.AddBlockLabel(canvas.LayerOverlay, canvas.BlockLabel{
				Bounds: geometry.Rect{Min: geometry.NewPoint(5, 5), Max: geometry.NewPoint(35, 13)},
				Lines:  []string{"a.go", "42"},
				Ink:    color.RGBA{A: 255},
			}, tt.format)

			mb := mock.NewBackend()
			err := c.RenderTo(mb)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(mb.Calls).To(HaveLen(2))
			g.Expect(mb.Calls[0].Method).To(Equal(tt.method))
			g.Expect(mb.Calls[1].Method).To(Equal(tt.method))
		})
	}
}

func TestCanvas_AddBlockLabel_PreservesTinyRasterTextWhenRequested(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := canvas.NewCanvas(40, 20)
	c.AddBlockLabel(canvas.LayerOverlay, canvas.BlockLabel{
		Bounds:       geometry.Rect{Min: geometry.NewPoint(5, 5), Max: geometry.NewPoint(35, 13)},
		Lines:        []string{"a.go", "0"},
		Ink:          color.RGBA{A: 255},
		PreserveText: true,
	}, canvas.FormatPNG)

	mb := mock.NewBackend()
	g.Expect(c.RenderTo(mb)).To(Succeed())
	g.Expect(mb.Calls).To(HaveLen(2))

	for _, call := range mb.Calls {
		g.Expect(call.Method).To(Equal("DrawText"))
		g.Expect(call.Pos.X).To(BeNumerically(">=", 5.0))
		g.Expect(call.Pos.X).To(BeNumerically("<=", 35.0))
		g.Expect(call.Pos.Y).To(BeNumerically(">=", 5.0))
		g.Expect(call.Pos.Y).To(BeNumerically("<=", 13.0))
	}

	g.Expect(mb.Calls[1].Text).To(Equal("0"))
}

func TestCanvas_AddBlockLabel_UsesExplicitLayoutSize(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := canvas.NewCanvas(40, 20)
	c.AddBlockLabel(canvas.LayerOverlay, canvas.BlockLabel{
		Bounds:       geometry.Rect{Min: geometry.NewPoint(5, 5), Max: geometry.NewPoint(35, 15)},
		LayoutSize:   geometry.Size{Width: 20, Height: 10},
		Lines:        []string{"a.go"},
		Ink:          color.RGBA{A: 255},
		PreserveText: true,
	}, canvas.FormatPNG)

	mb := mock.NewBackend()
	g.Expect(c.RenderTo(mb)).To(Succeed())
	g.Expect(mb.Calls).To(HaveLen(1))
	g.Expect(mb.Calls[0].Method).To(Equal("DrawText"))
	g.Expect(mb.Calls[0].Pos.X).To(Equal(15.0))
	g.Expect(mb.Calls[0].Pos.Y).To(Equal(10.0))
}

func TestCanvas_AddBlockLabel_OmitsUnreadableRasterLabels(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := canvas.NewCanvas(20, 10)
	c.AddBlockLabel(canvas.LayerOverlay, canvas.BlockLabel{
		Bounds: geometry.Rect{Min: geometry.NewPoint(2, 2), Max: geometry.NewPoint(18, 3.5)},
		Lines:  []string{"a.go"},
		Ink:    color.RGBA{A: 255},
	}, canvas.FormatPNG)

	mb := mock.NewBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.Calls).To(BeEmpty())
}
