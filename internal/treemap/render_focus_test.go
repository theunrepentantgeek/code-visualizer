package treemap_test

import (
	"image/color"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/mock"
	canvasmodel "github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/treemap"
)

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
		X: 0, Y: 0, W: 100, H: 100,
		Label: "root", IsDirectory: true,
		Children: []treemap.TreemapRectangle{
			{X: 0, Y: 20, W: 50, H: 80},
			{X: 50, Y: 20, W: 50, H: 80},
		},
	}
	is := treemap.Inks{
		Fill:   inks.NewRadialGradientInk(inks.FixedInk(color.RGBA{R: 200, A: 255})),
		Border: inks.FixedInk(color.RGBA{A: 255}),
	}

	cv := treemap.RenderToCanvas(rects, root, 100, 100, is, filesystem.FileSize)
	backend := mock.NewBackend()

	g.Expect(cv.RenderTo(backend)).To(Succeed())

	var gradientCalls []mock.Call

	for _, call := range backend.Calls {
		if call.Method != "DrawRectangle" {
			continue
		}

		if _, ok := call.RawFill.(canvasmodel.RadialGradientFill); ok {
			gradientCalls = append(gradientCalls, call)
		}
	}

	g.Expect(gradientCalls).To(HaveLen(2))

	if len(gradientCalls) < 2 {
		return // unreachable; satisfies nilaway
	}

	first, ok := gradientCalls[0].RawFill.(canvasmodel.RadialGradientFill)
	g.Expect(ok).To(BeTrue())

	second, ok := gradientCalls[1].RawFill.(canvasmodel.RadialGradientFill)
	g.Expect(ok).To(BeTrue())

	g.Expect(first.Focus.X).To(BeNumerically("~", 0.875, 1e-9))
	g.Expect(first.Focus.Y).To(BeNumerically("~", 0.40625, 1e-9))
	g.Expect(second.Focus.X).To(BeNumerically("~", 0.375, 1e-9))
	g.Expect(second.Focus.Y).To(BeNumerically("~", 0.46875, 1e-9))
}
