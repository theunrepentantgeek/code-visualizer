package treemap_test

import (
	"fmt"
	"image/color"
	"math"
	"slices"
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
		Rail:        geometry.Rect{Min: geometry.Point{X: 10, Y: 10}, Max: geometry.Point{X: 90, Y: 30}},
		Content:     geometry.Rect{Min: geometry.Point{X: 14, Y: 30}, Max: geometry.Point{X: 86, Y: 86}},
	}, 0)

	g.Expect(hasRectangle(
		backend.rectangles,
		geometry.Point{X: 10, Y: 10},
		canvas.Size{Width: 80, Height: 20},
	)).To(BeTrue())
	g.Expect(hasText(backend.texts, textCall{
		pos:      geometry.Point{X: 50, Y: 20},
		text:     "source",
		fontSize: 12,
		anchor:   canvas.AnchorMiddle,
		rotation: 0,
	})).To(BeTrue())
}

func TestRenderToCanvas_DrawsLeftDirectoryChrome(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	backend := renderDirectoryChrome(t, treemap.DirectoryChrome{
		Orientation: treemap.DirectoryLabelLeft,
		Text:        "source",
		Rail:        geometry.Rect{Min: geometry.Point{X: 10, Y: 10}, Max: geometry.Point{X: 30, Y: 90}},
		Content:     geometry.Rect{Min: geometry.Point{X: 30, Y: 14}, Max: geometry.Point{X: 86, Y: 86}},
	}, 0)

	g.Expect(hasRectangle(
		backend.rectangles,
		geometry.Point{X: 10, Y: 10},
		canvas.Size{Width: 20, Height: 80},
	)).To(BeTrue())
	g.Expect(hasText(backend.texts, textCall{
		pos:      geometry.Point{X: 20, Y: 50},
		text:     "source",
		fontSize: 12,
		anchor:   canvas.AnchorMiddle,
		rotation: -math.Pi / 2,
	})).To(BeTrue())
}

func TestRenderToCanvas_OmitsDirectoryChromeWhenOrientationIsNone(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	backend := renderDirectoryChrome(t, treemap.DirectoryChrome{
		Orientation: treemap.DirectoryLabelNone,
		Content:     geometry.Rect{Min: geometry.Point{X: 14, Y: 14}, Max: geometry.Point{X: 86, Y: 86}},
	}, 0)

	g.Expect(backend.texts).To(BeEmpty())
	g.Expect(hasRectangle(
		backend.rectangles,
		geometry.Point{X: 10, Y: 10},
		canvas.Size{Width: 80, Height: 20},
	)).To(BeFalse())
	g.Expect(hasRectangle(
		backend.rectangles,
		geometry.Point{X: 10, Y: 10},
		canvas.Size{Width: 20, Height: 80},
	)).To(BeFalse())
	g.Expect(hasRectangle(
		backend.rectangles,
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
				Rail:        geometry.Rect{Min: geometry.Point{X: 10, Y: 10}, Max: geometry.Point{X: 90, Y: 30}},
				Content:     geometry.Rect{Min: geometry.Point{X: 14, Y: 30}, Max: geometry.Point{X: 86, Y: 86}},
			}, depth)

			fill, ok := railFillAt(
				backend.rectangles,
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
		Rail:        geometry.Rect{Min: geometry.Point{X: 10, Y: 10}, Max: geometry.Point{X: 30, Y: 90}},
		Content:     geometry.Rect{Min: geometry.Point{X: 30, Y: 14}, Max: geometry.Point{X: 86, Y: 86}},
	}, len(expectedHeaderFills))

	fill, ok := railFillAt(
		backend.rectangles,
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
		Rail:        geometry.Rect{Min: geometry.Point{X: 10, Y: 10}, Max: geometry.Point{X: 90, Y: 30}},
		Content:     geometry.Rect{Min: geometry.Point{X: 14, Y: 30}, Max: geometry.Point{X: 86, Y: 86}},
	}, -1)

	fill, ok := railFillAt(
		backend.rectangles,
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

	first, ok := railFillAt(backend.rectangles, geometry.Point{X: 10, Y: 10}, canvas.Size{Width: 35, Height: 20})
	g.Expect(ok).To(BeTrue())
	second, ok := railFillAt(backend.rectangles, geometry.Point{X: 55, Y: 10}, canvas.Size{Width: 35, Height: 20})
	g.Expect(ok).To(BeTrue())

	g.Expect(first).To(Equal(expectedHeaderFills[0]))
	g.Expect(second).To(Equal(expectedHeaderFills[0]))
}

// renderDirectoryChrome renders a root directory with a single "source" child
// directory using the given chrome and visibleDepth. The root itself always
// carries VisibleDepth -1 (border-only chrome), matching layoutRoot.
func renderDirectoryChrome(t *testing.T, chrome treemap.DirectoryChrome, visibleDepth int) *captureBackend {
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
		Bounds:       geometry.Rect{Min: geometry.Point{X: 0, Y: 0}, Max: geometry.Point{X: 100, Y: 100}},
		VisibleDepth: -1,
		IsDirectory:  true,
		Chrome: treemap.DirectoryChrome{
			Orientation: treemap.DirectoryLabelNone,
			Content:     geometry.Rect{Min: geometry.Point{X: 4, Y: 4}, Max: geometry.Point{X: 96, Y: 96}},
		},
		Children: []treemap.TreemapRectangle{
			{
				Bounds:       geometry.Rect{Min: geometry.Point{X: 10, Y: 10}, Max: geometry.Point{X: 90, Y: 90}},
				VisibleDepth: visibleDepth,
				Label:        "source",
				IsDirectory:  true,
				Chrome:       chrome,
				Children: []treemap.TreemapRectangle{
					{
						Bounds: geometry.Rect{Min: geometry.Point{X: 14, Y: 14}, Max: geometry.Point{X: 86, Y: 86}},
					},
				},
			},
		},
	}

	return renderRectsToBackend(t, rects, root)
}

// renderSiblingDirectoryChrome renders a root directory with two sibling
// directories, both at VisibleDepth 0, each with a top rail.
func renderSiblingDirectoryChrome(t *testing.T) *captureBackend {
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
		Bounds:       geometry.Rect{Min: geometry.Point{X: 0, Y: 0}, Max: geometry.Point{X: 100, Y: 100}},
		VisibleDepth: -1,
		IsDirectory:  true,
		Chrome: treemap.DirectoryChrome{
			Orientation: treemap.DirectoryLabelNone,
			Content:     geometry.Rect{Min: geometry.Point{X: 4, Y: 4}, Max: geometry.Point{X: 96, Y: 96}},
		},
		Children: []treemap.TreemapRectangle{
			{
				Bounds:       geometry.Rect{Min: geometry.Point{X: 10, Y: 10}, Max: geometry.Point{X: 45, Y: 90}},
				VisibleDepth: 0,
				Label:        "alpha",
				IsDirectory:  true,
				Chrome: treemap.DirectoryChrome{
					Orientation: treemap.DirectoryLabelTop,
					Text:        "alpha",
					Rail:        geometry.Rect{Min: geometry.Point{X: 10, Y: 10}, Max: geometry.Point{X: 45, Y: 30}},
					Content:     geometry.Rect{Min: geometry.Point{X: 14, Y: 30}, Max: geometry.Point{X: 41, Y: 86}},
				},
				Children: []treemap.TreemapRectangle{
					{Bounds: geometry.Rect{Min: geometry.Point{X: 14, Y: 30}, Max: geometry.Point{X: 41, Y: 86}}},
				},
			},
			{
				Bounds:       geometry.Rect{Min: geometry.Point{X: 55, Y: 10}, Max: geometry.Point{X: 90, Y: 90}},
				VisibleDepth: 0,
				Label:        "beta",
				IsDirectory:  true,
				Chrome: treemap.DirectoryChrome{
					Orientation: treemap.DirectoryLabelTop,
					Text:        "beta",
					Rail:        geometry.Rect{Min: geometry.Point{X: 55, Y: 10}, Max: geometry.Point{X: 90, Y: 30}},
					Content:     geometry.Rect{Min: geometry.Point{X: 59, Y: 30}, Max: geometry.Point{X: 86, Y: 86}},
				},
				Children: []treemap.TreemapRectangle{
					{Bounds: geometry.Rect{Min: geometry.Point{X: 59, Y: 30}, Max: geometry.Point{X: 86, Y: 86}}},
				},
			},
		},
	}

	return renderRectsToBackend(t, rects, root)
}

func renderRectsToBackend(t *testing.T, rects treemap.TreemapRectangle, root *model.Directory) *captureBackend {
	t.Helper()

	is := treemap.Inks{
		Fill:   inks.FixedInk(color.RGBA{R: 0x88, A: 0xFF}),
		Border: inks.FixedInk(color.RGBA{A: 0xFF}),
	}
	cv := treemap.RenderToCanvas(rects, root, 100, 100, is, filesystem.FileSize)
	backend := &captureBackend{}

	NewGomegaWithT(t).Expect(cv.RenderTo(backend)).To(Succeed())

	return backend
}

func hasRectangle(rectangles []rectangleCall, pos geometry.Point, size canvas.Size) bool {
	for _, rectangle := range rectangles {
		if rectangle.pos == pos && rectangle.size == size {
			return true
		}
	}

	return false
}

// railFillAt returns the solid fill colour of the rectangle at pos/size,
// and whether a matching rectangle with a SolidFill was found.
func railFillAt(rectangles []rectangleCall, pos geometry.Point, size canvas.Size) (color.RGBA, bool) {
	for _, rectangle := range rectangles {
		if rectangle.pos != pos || rectangle.size != size {
			continue
		}

		if solid, ok := rectangle.fill.(canvasmodel.SolidFill); ok {
			return solid.Color, true
		}
	}

	return color.RGBA{}, false
}

func hasText(texts []textCall, want textCall) bool {
	return slices.Contains(texts, want)
}
