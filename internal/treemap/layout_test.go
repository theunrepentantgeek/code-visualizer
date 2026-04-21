package treemap

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/provider/filesystem"
)

func makeFile(name string, size int64) *model.File {
	f := &model.File{Name: name}
	f.SetQuantity(filesystem.FileSize, size)

	return f
}

func TestLayoutSingleFile(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name:  "root",
		Files: []*model.File{makeFile("only.go", 100)},
	}

	rects := Layout(root, 1920, 1080, filesystem.FileSize)
	g.Expect(rects.Children).To(HaveLen(1))
	g.Expect(rects.Children[0].W).To(BeNumerically(">", 0))
	g.Expect(rects.Children[0].H).To(BeNumerically(">", 0))
}

func TestLayoutProportionalAreas(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			makeFile("big.go", 900),
			makeFile("small.go", 100),
		},
	}

	rects := Layout(root, 1000, 1000, filesystem.FileSize)

	var bigRect, smallRect TreemapRectangle

	for _, c := range rects.Children {
		switch c.Label {
		case "big.go":
			bigRect = c
		case "small.go":
			smallRect = c
		default:
		}
	}

	ratio := (bigRect.W * bigRect.H) / (smallRect.W * smallRect.H)
	g.Expect(ratio).To(BeNumerically("~", 9.0, 2.0))
}

func TestLayoutNestedDirs(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name:  "root",
		Files: []*model.File{makeFile("top.go", 100)},
		Dirs: []*model.Directory{
			{
				Name:  "sub",
				Files: []*model.File{makeFile("inner.go", 200)},
			},
		},
	}

	rects := Layout(root, 1920, 1080, filesystem.FileSize)
	g.Expect(len(rects.Children)).To(BeNumerically(">=", 2))

	var dirRect *TreemapRectangle

	for i, c := range rects.Children {
		if c.IsDirectory {
			dirRect = &rects.Children[i]

			break
		}
	}

	g.Expect(dirRect).NotTo(BeNil())

	if dirRect == nil {
		return
	}

	g.Expect(dirRect.Label).To(Equal("sub"))
	g.Expect(dirRect.Children).NotTo(BeEmpty())
}

func TestLayoutZeroSizeFile(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			makeFile("normal.go", 1000),
			makeFile("empty.go", 0),
		},
	}

	rects := Layout(root, 1920, 1080, filesystem.FileSize)

	var emptyRect *TreemapRectangle

	for i, c := range rects.Children {
		if c.Label == "empty.go" {
			emptyRect = &rects.Children[i]

			break
		}
	}

	g.Expect(emptyRect).NotTo(BeNil())

	if emptyRect == nil {
		return
	}

	g.Expect(emptyRect.W).To(BeNumerically(">", 0))
	g.Expect(emptyRect.H).To(BeNumerically(">", 0))
}

func TestOffsetRects_ShiftsCoordinates(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	rect := TreemapRectangle{X: 10, Y: 20, W: 100, H: 50}
	OffsetRects(&rect, 30, 40)
	g.Expect(rect.X).To(Equal(40.0))
	g.Expect(rect.Y).To(Equal(60.0))
	g.Expect(rect.W).To(Equal(100.0))
	g.Expect(rect.H).To(Equal(50.0))
}

func TestOffsetRects_ShiftsChildrenRecursively(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	rect := TreemapRectangle{
		X: 0, Y: 0, W: 200, H: 100,
		Children: []TreemapRectangle{
			{X: 5, Y: 5, W: 90, H: 90},
			{
				X: 100, Y: 5, W: 90, H: 90,
				Children: []TreemapRectangle{
					{X: 105, Y: 10, W: 40, H: 40},
				},
			},
		},
	}

	OffsetRects(&rect, 50, 100)
	g.Expect(rect.X).To(Equal(50.0))
	g.Expect(rect.Y).To(Equal(100.0))
	g.Expect(rect.Children[0].X).To(Equal(55.0))
	g.Expect(rect.Children[0].Y).To(Equal(105.0))
	g.Expect(rect.Children[1].X).To(Equal(150.0))
	g.Expect(rect.Children[1].Y).To(Equal(105.0))
	g.Expect(rect.Children[1].Children[0].X).To(Equal(155.0))
	g.Expect(rect.Children[1].Children[0].Y).To(Equal(110.0))
}

func TestOffsetRects_ZeroOffset_NoChange(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	rect := TreemapRectangle{X: 10, Y: 20, W: 100, H: 50}
	OffsetRects(&rect, 0, 0)
	g.Expect(rect.X).To(Equal(10.0))
	g.Expect(rect.Y).To(Equal(20.0))
}
