package treemap_test

import (
	"fmt"
	"image/color"
	"math"
	"slices"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/mock"
	canvasmodel "github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/treemap"
)

// expectedHeaderFills lists the exact rail fill colours, darkest to lightest,
// in the order selected by VisibleDepth % len(expectedHeaderFills). Kept in
// sync with the private treemap.headerFills table so these black-box tests
// can assert on depth-selected colours without importing internal state.
var expectedHeaderFills = [5]color.RGBA{ //nolint:gochecknoglobals // test fixture data
	{R: 0x20, G: 0x26, B: 0x31, A: 0xFF},
	{R: 0x2F, G: 0x3B, B: 0x4D, A: 0xFF},
	{R: 0x3D, G: 0x52, B: 0x68, A: 0xFF},
	{R: 0x51, G: 0x6A, B: 0x7D, A: 0xFF},
	{R: 0x5F, G: 0x78, B: 0x88, A: 0xFF},
}

func TestRenderToCanvas_DrawsTopDirectoryChrome(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	backend := renderDirectoryChrome(t, treemap.DirectoryChrome{
		Orientation: treemap.DirectoryLabelTop,
		Text:        "source",
		Rail:        treemap.RectangleBounds{X: 10, Y: 10, W: 80, H: 20},
		Content:     treemap.RectangleBounds{X: 14, Y: 30, W: 72, H: 56},
	}, 0)

	g.Expect(hasRectangle(
		backend.Calls,
		geometry.Point{X: 10, Y: 10},
		canvas.Size{Width: 80, Height: 20},
	)).To(BeTrue())
	g.Expect(hasText(backend.Calls, mock.Call{
		Pos:      geometry.Point{X: 50, Y: 20},
		Text:     "source",
		FontSize: 12,
		Anchor:   canvas.AnchorMiddle,
		Rotation: 0,
	})).To(BeTrue())
}

func TestRenderToCanvas_DrawsLeftDirectoryChrome(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	backend := renderDirectoryChrome(t, treemap.DirectoryChrome{
		Orientation: treemap.DirectoryLabelLeft,
		Text:        "source",
		Rail:        treemap.RectangleBounds{X: 10, Y: 10, W: 20, H: 80},
		Content:     treemap.RectangleBounds{X: 30, Y: 14, W: 56, H: 72},
	}, 0)

	g.Expect(hasRectangle(
		backend.Calls,
		geometry.Point{X: 10, Y: 10},
		canvas.Size{Width: 20, Height: 80},
	)).To(BeTrue())
	g.Expect(hasText(backend.Calls, mock.Call{
		Pos:      geometry.Point{X: 20, Y: 50},
		Text:     "source",
		FontSize: 12,
		Anchor:   canvas.AnchorMiddle,
		Rotation: -math.Pi / 2,
	})).To(BeTrue())
}

func TestRenderToCanvas_OmitsDirectoryChromeWhenOrientationIsNone(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	backend := renderDirectoryChrome(t, treemap.DirectoryChrome{
		Orientation: treemap.DirectoryLabelNone,
		Content:     treemap.RectangleBounds{X: 14, Y: 14, W: 72, H: 72},
	}, 0)

	g.Expect(callsNamed(backend.Calls, "DrawText")).To(BeEmpty())
	g.Expect(hasRectangle(
		backend.Calls,
		geometry.Point{X: 10, Y: 10},
		canvas.Size{Width: 80, Height: 20},
	)).To(BeFalse())
	g.Expect(hasRectangle(
		backend.Calls,
		geometry.Point{X: 10, Y: 10},
		canvas.Size{Width: 20, Height: 80},
	)).To(BeFalse())
	g.Expect(hasRectangle(
		backend.Calls,
		geometry.Point{X: 10, Y: 10},
		canvas.Size{Width: 80, Height: 80},
	)).To(BeTrue())
}

func TestRenderToCanvas_TopRailUsesDepthPaletteAcrossAllPaletteDepths(t *testing.T) {
	t.Parallel()

	for depth := range expectedHeaderFills {
		t.Run(fmt.Sprintf("depth=%d", depth), func(t *testing.T) {
			t.Parallel()

			g := NewGomegaWithT(t)
			backend := renderDirectoryChrome(t, treemap.DirectoryChrome{
				Orientation: treemap.DirectoryLabelTop,
				Text:        "source",
				Rail:        treemap.RectangleBounds{X: 10, Y: 10, W: 80, H: 20},
				Content:     treemap.RectangleBounds{X: 14, Y: 30, W: 72, H: 56},
			}, depth)

			fill, ok := railFillAt(
				backend.Calls,
				geometry.Point{X: 10, Y: 10},
				canvas.Size{Width: 80, Height: 20},
			)
			g.Expect(ok).To(BeTrue())
			g.Expect(fill).To(Equal(expectedHeaderFills[depth]))
		})
	}
}

func TestRenderToCanvas_LeftRailWrapsPaletteAtPaletteLength(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	backend := renderDirectoryChrome(t, treemap.DirectoryChrome{
		Orientation: treemap.DirectoryLabelLeft,
		Text:        "source",
		Rail:        treemap.RectangleBounds{X: 10, Y: 10, W: 20, H: 80},
		Content:     treemap.RectangleBounds{X: 30, Y: 14, W: 56, H: 72},
	}, len(expectedHeaderFills))

	fill, ok := railFillAt(
		backend.Calls,
		geometry.Point{X: 10, Y: 10},
		canvas.Size{Width: 20, Height: 80},
	)
	g.Expect(ok).To(BeTrue())
	g.Expect(fill).To(Equal(expectedHeaderFills[0]))
}

// TestRenderToCanvas_TopRailAtNegativeDepthUsesDarkestFillWithoutPanicking is
// a regression test: dirRailSpecForDepth must be total over negative
// VisibleDepth values (Go's % operator can return negative results for a
// negative dividend, which previously panicked on out-of-range index).
// VisibleDepth -1 identifies the synthetic root directory, but a rail-bearing
// directory can also carry a negative VisibleDepth if layout logic changes;
// either way, rendering must clamp to the darkest fill rather than panic. A
// regression here fails this test with a panic, since nothing recovers it.
func TestRenderToCanvas_TopRailAtNegativeDepthUsesDarkestFillWithoutPanicking(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	backend := renderDirectoryChrome(t, treemap.DirectoryChrome{
		Orientation: treemap.DirectoryLabelTop,
		Text:        "source",
		Rail:        treemap.RectangleBounds{X: 10, Y: 10, W: 80, H: 20},
		Content:     treemap.RectangleBounds{X: 14, Y: 30, W: 72, H: 56},
	}, -1)

	fill, ok := railFillAt(
		backend.Calls,
		geometry.Point{X: 10, Y: 10},
		canvas.Size{Width: 80, Height: 20},
	)
	g.Expect(ok).To(BeTrue())
	g.Expect(fill).To(Equal(expectedHeaderFills[0]))
}

func TestRenderToCanvas_SameDepthSiblingRailsShareFill(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	backend := renderSiblingDirectoryChrome(t)

	first, ok := railFillAt(backend.Calls, geometry.Point{X: 10, Y: 10}, canvas.Size{Width: 35, Height: 20})
	g.Expect(ok).To(BeTrue())
	second, ok := railFillAt(backend.Calls, geometry.Point{X: 55, Y: 10}, canvas.Size{Width: 35, Height: 20})
	g.Expect(ok).To(BeTrue())

	g.Expect(first).To(Equal(expectedHeaderFills[0]))
	g.Expect(second).To(Equal(expectedHeaderFills[0]))
}

// renderDirectoryChrome renders a root directory with a single "source" child
// directory using the given chrome and visibleDepth. The root itself always
// carries VisibleDepth -1 (border-only chrome), matching layoutRoot.
func renderDirectoryChrome(t *testing.T, chrome treemap.DirectoryChrome, visibleDepth int) *mock.Backend {
	t.Helper()

	root := &model.Directory{
		Name: "",
		Dirs: []*model.Directory{
			{
				Name:  "source",
				Files: []*model.File{makeTestFile("main.go", "go", 100)},
			},
		},
	}
	rects := treemap.TreemapRectangle{
		X:            0,
		Y:            0,
		W:            100,
		H:            100,
		VisibleDepth: -1,
		IsDirectory:  true,
		Chrome: treemap.DirectoryChrome{
			Orientation: treemap.DirectoryLabelNone,
			Content:     treemap.RectangleBounds{X: 4, Y: 4, W: 92, H: 92},
		},
		Children: []treemap.TreemapRectangle{
			{
				X:            10,
				Y:            10,
				W:            80,
				H:            80,
				VisibleDepth: visibleDepth,
				Label:        "source",
				IsDirectory:  true,
				Chrome:       chrome,
				Children: []treemap.TreemapRectangle{
					{
						X: 14,
						Y: 14,
						W: 72,
						H: 72,
					},
				},
			},
		},
	}

	return renderRectsToBackend(t, rects, root)
}

// renderSiblingDirectoryChrome renders a root directory with two sibling
// directories, both at VisibleDepth 0, each with a top rail.
func renderSiblingDirectoryChrome(t *testing.T) *mock.Backend {
	t.Helper()

	root := &model.Directory{
		Name: "",
		Dirs: []*model.Directory{
			{
				Name:  "alpha",
				Files: []*model.File{makeTestFile("a.go", "go", 100)},
			},
			{
				Name:  "beta",
				Files: []*model.File{makeTestFile("b.go", "go", 100)},
			},
		},
	}
	rects := treemap.TreemapRectangle{
		X:            0,
		Y:            0,
		W:            100,
		H:            100,
		VisibleDepth: -1,
		IsDirectory:  true,
		Chrome: treemap.DirectoryChrome{
			Orientation: treemap.DirectoryLabelNone,
			Content:     treemap.RectangleBounds{X: 4, Y: 4, W: 92, H: 92},
		},
		Children: []treemap.TreemapRectangle{
			{
				X:            10,
				Y:            10,
				W:            35,
				H:            80,
				VisibleDepth: 0,
				Label:        "alpha",
				IsDirectory:  true,
				Chrome: treemap.DirectoryChrome{
					Orientation: treemap.DirectoryLabelTop,
					Text:        "alpha",
					Rail:        treemap.RectangleBounds{X: 10, Y: 10, W: 35, H: 20},
					Content:     treemap.RectangleBounds{X: 14, Y: 30, W: 27, H: 56},
				},
				Children: []treemap.TreemapRectangle{
					{X: 14, Y: 30, W: 27, H: 56},
				},
			},
			{
				X:            55,
				Y:            10,
				W:            35,
				H:            80,
				VisibleDepth: 0,
				Label:        "beta",
				IsDirectory:  true,
				Chrome: treemap.DirectoryChrome{
					Orientation: treemap.DirectoryLabelTop,
					Text:        "beta",
					Rail:        treemap.RectangleBounds{X: 55, Y: 10, W: 35, H: 20},
					Content:     treemap.RectangleBounds{X: 59, Y: 30, W: 27, H: 56},
				},
				Children: []treemap.TreemapRectangle{
					{X: 59, Y: 30, W: 27, H: 56},
				},
			},
		},
	}

	return renderRectsToBackend(t, rects, root)
}

func renderRectsToBackend(t *testing.T, rects treemap.TreemapRectangle, root *model.Directory) *mock.Backend {
	t.Helper()

	is := treemap.Inks{
		Fill:   inks.FixedInk(color.RGBA{R: 0x88, A: 0xFF}),
		Border: inks.FixedInk(color.RGBA{A: 0xFF}),
	}
	cv := treemap.RenderToCanvas(rects, root, 100, 100, is, filesystem.FileSize)
	backend := mock.NewBackend()

	NewGomegaWithT(t).Expect(cv.RenderTo(backend)).To(Succeed())

	return backend
}

func hasRectangle(calls []mock.Call, pos geometry.Point, size canvas.Size) bool {
	for _, call := range calls {
		if call.Method == "DrawRectangle" && call.Pos == pos && call.Size == size {
			return true
		}
	}

	return false
}

// railFillAt returns the solid fill colour of the rectangle at pos/size,
// and whether a matching rectangle with a SolidFill was found.
func railFillAt(calls []mock.Call, pos geometry.Point, size canvas.Size) (color.RGBA, bool) {
	for _, call := range calls {
		if call.Method != "DrawRectangle" || call.Pos != pos || call.Size != size {
			continue
		}

		if solid, ok := call.RawFill.(canvasmodel.SolidFill); ok {
			return solid.Color, true
		}
	}

	return color.RGBA{}, false
}

func hasText(calls []mock.Call, want mock.Call) bool {
	return slices.ContainsFunc(calls, func(call mock.Call) bool {
		return call.Method == "DrawText" &&
			call.Pos == want.Pos &&
			call.Text == want.Text &&
			call.FontSize == want.FontSize &&
			call.Anchor == want.Anchor &&
			call.Rotation == want.Rotation
	})
}

func callsNamed(calls []mock.Call, method string) []mock.Call {
	result := make([]mock.Call, 0)

	for _, call := range calls {
		if call.Method == method {
			result = append(result, call)
		}
	}

	return result
}
