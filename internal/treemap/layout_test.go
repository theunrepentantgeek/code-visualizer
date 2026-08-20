package treemap

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
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

func TestLayoutRootUsesBorderOnlyChrome(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name:  "root",
		Files: []*model.File{makeFile("only.go", 100)},
	}

	rects := Layout(root, 200, 100, filesystem.FileSize)

	g.Expect(rects.Chrome.Orientation).To(Equal(DirectoryLabelNone))
	g.Expect(rects.Chrome.Content).To(Equal(RectangleBounds{X: 4, Y: 4, W: 192, H: 92}))
	g.Expect(rects.Children).To(HaveLen(1))
	g.Expect(rects.Children[0].Y).To(BeNumerically(">=", 4.0))
	g.Expect(rects.Children[0].Y).To(BeNumerically("<", directoryRailThickness))
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

func TestLayoutNestedDirectoryChrome(t *testing.T) {
	t.Parallel()

	t.Run("wide nested directory uses top chrome", func(t *testing.T) {
		t.Parallel()
		g := NewGomegaWithT(t)

		root := &model.Directory{
			Name: "root",
			Dirs: []*model.Directory{
				{
					Name:  "source",
					Files: []*model.File{makeFile("main.go", 100)},
				},
			},
		}

		rects := Layout(root, 200, 100, filesystem.FileSize)
		dirRect := findDirRect(rects, "source")

		g.Expect(dirRect).NotTo(BeNil())
		if dirRect == nil {
			return
		}

		g.Expect(dirRect.Chrome.Orientation).To(Equal(DirectoryLabelTop))
		g.Expect(dirRect.Chrome.Text).To(Equal("source"))
		g.Expect(dirRect.Children).To(HaveLen(1))
		g.Expect(dirRect.Children[0].Y).To(BeNumerically(">=", dirRect.Y+directoryRailThickness))
	})

	t.Run("tall nested directory uses left chrome", func(t *testing.T) {
		t.Parallel()
		g := NewGomegaWithT(t)

		root := &model.Directory{
			Name: "root",
			Dirs: []*model.Directory{
				{
					Name:  "source",
					Files: []*model.File{makeFile("main.go", 100)},
				},
			},
		}

		rects := Layout(root, 100, 200, filesystem.FileSize)
		dirRect := findDirRect(rects, "source")

		g.Expect(dirRect).NotTo(BeNil())
		if dirRect == nil {
			return
		}

		g.Expect(dirRect.Chrome.Orientation).To(Equal(DirectoryLabelLeft))
		g.Expect(dirRect.Chrome.Text).To(Equal("source"))
		g.Expect(dirRect.Children).To(HaveLen(1))
		g.Expect(dirRect.Children[0].X).To(BeNumerically(">=", dirRect.X+directoryRailThickness))
	})

	t.Run("small nested directory omits the rail", func(t *testing.T) {
		t.Parallel()
		g := NewGomegaWithT(t)

		root := &model.Directory{
			Name: "root",
			Dirs: []*model.Directory{
				{
					Name:  "source",
					Files: []*model.File{makeFile("main.go", 100)},
				},
			},
		}

		rects := Layout(root, 50, 50, filesystem.FileSize)
		dirRect := findDirRect(rects, "source")

		g.Expect(dirRect).NotTo(BeNil())
		if dirRect == nil {
			return
		}

		g.Expect(dirRect.Chrome.Orientation).To(Equal(DirectoryLabelNone))
		g.Expect(dirRect.Children).To(HaveLen(1))
		g.Expect(dirRect.Children[0].Y).To(BeNumerically("<", dirRect.Y+directoryRailThickness))
	})

	t.Run("children stay inside the exact chrome content bounds", func(t *testing.T) {
		t.Parallel()
		g := NewGomegaWithT(t)

		root := &model.Directory{
			Name: "root",
			Dirs: []*model.Directory{
				{
					Name: "source",
					Files: []*model.File{
						makeFile("main.go", 100),
						makeFile("util.go", 100),
					},
				},
			},
		}

		rects := Layout(root, 200, 100, filesystem.FileSize)
		dirRect := findDirRect(rects, "source")

		g.Expect(dirRect).NotTo(BeNil())
		if dirRect == nil {
			return
		}

		content := dirRect.Chrome.Content
		g.Expect(dirRect.Chrome.Orientation).To(Equal(DirectoryLabelTop))
		g.Expect(dirRect.Children).To(HaveLen(2))

		for _, child := range dirRect.Children {
			g.Expect(child.X).To(BeNumerically(">=", content.X))
			g.Expect(child.Y).To(BeNumerically(">=", content.Y))
			g.Expect(child.X + child.W).To(BeNumerically("<=", content.X+content.W))
			g.Expect(child.Y + child.H).To(BeNumerically("<=", content.Y+content.H))
		}
	})
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

	rect := TreemapRectangle{
		X: 10, Y: 20, W: 100, H: 50,
		IsDirectory: true,
		Chrome: DirectoryChrome{
			Orientation: DirectoryLabelTop,
			Rail:        RectangleBounds{X: 10, Y: 20, W: 100, H: 20},
			Content:     RectangleBounds{X: 14, Y: 40, W: 92, H: 26},
		},
	}
	OffsetRects(&rect, 30, 40)
	g.Expect(rect.X).To(Equal(40.0))
	g.Expect(rect.Y).To(Equal(60.0))
	g.Expect(rect.W).To(Equal(100.0))
	g.Expect(rect.H).To(Equal(50.0))
	g.Expect(rect.Chrome.Rail).To(Equal(RectangleBounds{X: 40, Y: 60, W: 100, H: 20}))
	g.Expect(rect.Chrome.Content).To(Equal(RectangleBounds{X: 44, Y: 80, W: 92, H: 26}))
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

const testMeasureMetric metric.Name = "test-measure"

func makeMeasureFile(name string, value float64) *model.File {
	f := &model.File{Name: name}
	f.SetMeasure(testMeasureMetric, value)

	return f
}

func TestLayoutProportionalAreas_MeasureMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			makeMeasureFile("high.go", 9.0),
			makeMeasureFile("low.go", 1.0),
		},
	}

	rects := Layout(root, 1000, 1000, testMeasureMetric)

	var highRect, lowRect TreemapRectangle

	for _, c := range rects.Children {
		switch c.Label {
		case "high.go":
			highRect = c
		case "low.go":
			lowRect = c
		default:
		}
	}

	ratio := (highRect.W * highRect.H) / (lowRect.W * lowRect.H)
	g.Expect(ratio).To(BeNumerically("~", 9.0, 2.0))
}
