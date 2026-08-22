package treemap_test

import (
	"image/color"
	"math"
	"slices"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/treemap"
)

func TestRenderToCanvas_DrawsTopDirectoryChrome(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	backend := renderDirectoryChrome(t, treemap.DirectoryChrome{
		Orientation: treemap.DirectoryLabelTop,
		Text:        "source",
		Rail:        treemap.RectangleBounds{X: 10, Y: 10, W: 80, H: 20},
		Content:     treemap.RectangleBounds{X: 14, Y: 30, W: 72, H: 56},
	})

	g.Expect(hasRectangle(backend.rectangles, canvas.Position{X: 10, Y: 10}, canvas.Size{Width: 80, Height: 20})).To(BeTrue())
	g.Expect(hasText(backend.texts, textCall{
		pos:      canvas.Position{X: 14, Y: 20},
		text:     "source",
		fontSize: 12,
		anchor:   canvas.AnchorStart,
		rotation: 0,
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
	})

	g.Expect(hasRectangle(backend.rectangles, canvas.Position{X: 10, Y: 10}, canvas.Size{Width: 20, Height: 80})).To(BeTrue())
	g.Expect(hasText(backend.texts, textCall{
		pos:      canvas.Position{X: 20, Y: 86},
		text:     "source",
		fontSize: 12,
		anchor:   canvas.AnchorStart,
		rotation: -math.Pi / 2,
	})).To(BeTrue())
}

func TestRenderToCanvas_OmitsDirectoryChromeWhenOrientationIsNone(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	backend := renderDirectoryChrome(t, treemap.DirectoryChrome{
		Orientation: treemap.DirectoryLabelNone,
		Content:     treemap.RectangleBounds{X: 14, Y: 14, W: 72, H: 72},
	})

	g.Expect(backend.texts).To(BeEmpty())
	g.Expect(hasRectangle(backend.rectangles, canvas.Position{X: 10, Y: 10}, canvas.Size{Width: 80, Height: 20})).To(BeFalse())
	g.Expect(hasRectangle(backend.rectangles, canvas.Position{X: 10, Y: 10}, canvas.Size{Width: 20, Height: 80})).To(BeFalse())
	g.Expect(hasRectangle(backend.rectangles, canvas.Position{X: 10, Y: 10}, canvas.Size{Width: 80, Height: 80})).To(BeTrue())
}

func renderDirectoryChrome(t *testing.T, chrome treemap.DirectoryChrome) *captureBackend {
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
		X:           0,
		Y:           0,
		W:           100,
		H:           100,
		IsDirectory: true,
		Chrome: treemap.DirectoryChrome{
			Orientation: treemap.DirectoryLabelNone,
			Content:     treemap.RectangleBounds{X: 4, Y: 4, W: 92, H: 92},
		},
		Children: []treemap.TreemapRectangle{
			{
				X:           10,
				Y:           10,
				W:           80,
				H:           80,
				Label:       "source",
				IsDirectory: true,
				Chrome:      chrome,
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
	is := treemap.Inks{
		Fill:   inks.FixedInk(color.RGBA{R: 0x88, A: 0xFF}),
		Border: inks.FixedInk(color.RGBA{A: 0xFF}),
	}
	cv := treemap.RenderToCanvas(rects, root, 100, 100, is, filesystem.FileSize)
	backend := &captureBackend{}

	NewGomegaWithT(t).Expect(cv.RenderTo(backend)).To(Succeed())

	return backend
}

func hasRectangle(rectangles []rectangleCall, pos canvas.Position, size canvas.Size) bool {
	for _, rectangle := range rectangles {
		if rectangle.pos == pos && rectangle.size == size {
			return true
		}
	}

	return false
}

func hasText(texts []textCall, want textCall) bool {
	return slices.Contains(texts, want)
}
