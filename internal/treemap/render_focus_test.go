package treemap_test

import (
	"image/color"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	canvasmodel "github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/treemap"
)

type rectangleCall struct {
	pos  geometry.Point
	size geometry.Size
	fill canvasmodel.Fill
}

type textCall struct {
	pos      geometry.Point
	text     string
	fontSize float64
	anchor   canvas.TextAnchor
	rotation float64
}

type captureBackend struct {
	rectangles []rectangleCall
	texts      []textCall
}

func (b *captureBackend) DrawRectangle(
	bounds geometry.Rect, fill, _ canvasmodel.Fill, _ float64,
) {
	b.rectangles = append(b.rectangles, rectangleCall{pos: bounds.Min, size: bounds.Size(), fill: fill})
}

func (*captureBackend) DrawDisc(geometry.Point, float64, canvasmodel.Fill, canvasmodel.Fill, float64) {
}

func (*captureBackend) DrawPolygon([]geometry.Point, canvasmodel.Fill, canvasmodel.Fill, float64) {}

func (*captureBackend) DrawFilledPath([][]geometry.Point, color.RGBA) {}

func (*captureBackend) DrawLine(geometry.Point, geometry.Point, color.RGBA, float64) {}

func (*captureBackend) DrawPath([]geometry.Point, color.RGBA, float64) {}

func (b *captureBackend) DrawText(
	pos geometry.Point, text string, _ color.RGBA, fontSize float64, anchor canvas.TextAnchor, rotation float64,
) {
	b.texts = append(b.texts, textCall{
		pos:      pos,
		text:     text,
		fontSize: fontSize,
		anchor:   anchor,
		rotation: rotation,
	})
}

func (*captureBackend) DrawArcText(geometry.Point, float64, string, color.RGBA, float64) {}

func (*captureBackend) Finish(string) error { return nil }

func TestRenderToCanvas_ComputesWeightedFocusForGradientFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			makeTestFile("large.go", "go", 75),
			makeTestFile("small.go", "go", 25),
		},
	}
	rects := treemap.TreemapRectangle{
		Bounds: geometry.Rect{Min: geometry.Point{X: 0, Y: 0}, Max: geometry.Point{X: 100, Y: 100}},
		Label:  "root", IsDirectory: true,
		Children: []treemap.TreemapRectangle{
			{Bounds: geometry.Rect{Min: geometry.Point{X: 0, Y: 20}, Max: geometry.Point{X: 50, Y: 100}}},
			{Bounds: geometry.Rect{Min: geometry.Point{X: 50, Y: 20}, Max: geometry.Point{X: 100, Y: 100}}},
		},
	}
	is := treemap.Inks{
		Fill:   inks.NewRadialGradientInk(inks.FixedInk(color.RGBA{R: 200, A: 255})),
		Border: inks.FixedInk(color.RGBA{A: 255}),
	}

	cv := treemap.RenderToCanvas(rects, root, 100, 100, is, filesystem.FileSize)
	backend := &captureBackend{}

	g.Expect(cv.RenderTo(backend)).To(Succeed())

	var gradientCalls []rectangleCall

	for _, call := range backend.rectangles {
		if _, ok := call.fill.(canvasmodel.RadialGradientFill); ok {
			gradientCalls = append(gradientCalls, call)
		}
	}

	g.Expect(gradientCalls).To(HaveLen(2))

	if len(gradientCalls) < 2 {
		return // unreachable; satisfies nilaway
	}

	first, ok := gradientCalls[0].fill.(canvasmodel.RadialGradientFill)
	g.Expect(ok).To(BeTrue())

	second, ok := gradientCalls[1].fill.(canvasmodel.RadialGradientFill)
	g.Expect(ok).To(BeTrue())

	g.Expect(first.Focus.X).To(BeNumerically("~", 0.875, 1e-9))
	g.Expect(first.Focus.Y).To(BeNumerically("~", 0.40625, 1e-9))
	g.Expect(second.Focus.X).To(BeNumerically("~", 0.375, 1e-9))
	g.Expect(second.Focus.Y).To(BeNumerically("~", 0.46875, 1e-9))
}
