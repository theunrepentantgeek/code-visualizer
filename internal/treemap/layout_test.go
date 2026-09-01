package treemap

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
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
	g.Expect(rects.Children[0].Bounds.Width()).To(BeNumerically(">", 0))
	g.Expect(rects.Children[0].Bounds.Height()).To(BeNumerically(">", 0))
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
	g.Expect(rects.Chrome.Content).To(Equal(geometry.Rect{Min: geometry.Point{X: 4, Y: 4}, Max: geometry.Point{X: 196, Y: 96}}))
	g.Expect(rects.Children).To(HaveLen(1))
	g.Expect(rects.Children[0].Bounds.Min.Y).To(BeNumerically(">=", 4.0))
	g.Expect(rects.Children[0].Bounds.Min.Y).To(BeNumerically("<", directoryRailThickness))
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

	ratio := (bigRect.Bounds.Width() * bigRect.Bounds.Height()) / (smallRect.Bounds.Width() * smallRect.Bounds.Height())
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

func TestLayoutAssignsVisibleDepthToDirectoriesAndFiles(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			makeFile("root.go", 100),
		},
		Dirs: []*model.Directory{
			{
				Name: "src",
				Dirs: []*model.Directory{
					{
						Name:  "internal",
						Files: []*model.File{makeFile("internal.go", 50)},
					},
				},
			},
			{
				Name:  "cmd",
				Files: []*model.File{makeFile("main.go", 75)},
			},
		},
	}

	rects := Layout(root, 800, 600, filesystem.FileSize)
	g.Expect(rects.VisibleDepth).To(Equal(-1))

	src := findDirRect(rects, "src")
	cmd := findDirRect(rects, "cmd")

	g.Expect(src).NotTo(BeNil())
	g.Expect(cmd).NotTo(BeNil())

	if src == nil || cmd == nil {
		return
	}

	internal := findDirRect(*src, "internal")
	g.Expect(internal).NotTo(BeNil())

	if internal == nil {
		return
	}

	g.Expect(src.VisibleDepth).To(Equal(0))
	g.Expect(cmd.VisibleDepth).To(Equal(0))
	g.Expect(internal.VisibleDepth).To(Equal(1))

	rootFile := func() *TreemapRectangle {
		for i := range rects.Children {
			if !rects.Children[i].IsDirectory && rects.Children[i].Label == "root.go" {
				return &rects.Children[i]
			}
		}

		return nil
	}()

	g.Expect(rootFile).NotTo(BeNil())

	if rootFile == nil {
		return
	}

	g.Expect(rootFile.VisibleDepth).To(Equal(0))
}

//nolint:dupl // similar structure, different axis config and assertions
func TestLayoutNestedDirectoryChrome_WideUsesTopChrome(t *testing.T) {
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
	g.Expect(dirRect.Children[0].Bounds.Min.Y).To(BeNumerically(">=", dirRect.Bounds.Min.Y+directoryRailThickness))
}

//nolint:dupl // similar structure, different axis config and assertions
func TestLayoutNestedDirectoryChrome_TallUsesLeftChrome(t *testing.T) {
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
	g.Expect(dirRect.Children[0].Bounds.Min.X).To(BeNumerically(">=", dirRect.Bounds.Min.X+directoryRailThickness))
}

func TestLayoutNestedDirectoryChrome_OmittedRailReclaimsSpace(t *testing.T) {
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
	g.Expect(dirRect.Children[0].Bounds.Min.Y).To(BeNumerically("<", dirRect.Bounds.Min.Y+directoryRailThickness))
}

func TestLayoutNestedDirectoryChrome_ChildrenInsideContent(t *testing.T) {
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
		g.Expect(child.Bounds.Min.X).To(BeNumerically(">=", content.Min.X))
		g.Expect(child.Bounds.Min.Y).To(BeNumerically(">=", content.Min.Y))
		g.Expect(child.Bounds.Max.X).To(BeNumerically("<=", content.Max.X))
		g.Expect(child.Bounds.Max.Y).To(BeNumerically("<=", content.Max.Y))
	}
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

	g.Expect(emptyRect.Bounds.Width()).To(BeNumerically(">", 0))
	g.Expect(emptyRect.Bounds.Height()).To(BeNumerically(">", 0))
}

func TestOffsetRects_ShiftsCoordinates(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	rect := TreemapRectangle{
		Bounds:      geometry.Rect{Min: geometry.Point{X: 10, Y: 20}, Max: geometry.Point{X: 110, Y: 70}},
		IsDirectory: true,
		Chrome: DirectoryChrome{
			Orientation: DirectoryLabelTop,
			Rail:        geometry.Rect{Min: geometry.Point{X: 10, Y: 20}, Max: geometry.Point{X: 110, Y: 40}},
			Content:     geometry.Rect{Min: geometry.Point{X: 14, Y: 40}, Max: geometry.Point{X: 106, Y: 66}},
		},
	}
	OffsetRects(&rect, geometry.Vector{X: 30, Y: 40})
	g.Expect(rect.Bounds.Min.X).To(Equal(40.0))
	g.Expect(rect.Bounds.Min.Y).To(Equal(60.0))
	g.Expect(rect.Bounds.Width()).To(Equal(100.0))
	g.Expect(rect.Bounds.Height()).To(Equal(50.0))
	g.Expect(rect.VisibleDepth).To(Equal(0))
	g.Expect(rect.Chrome.Rail).To(Equal(geometry.Rect{Min: geometry.Point{X: 40, Y: 60}, Max: geometry.Point{X: 140, Y: 80}}))
	g.Expect(rect.Chrome.Content).To(Equal(geometry.Rect{Min: geometry.Point{X: 44, Y: 80}, Max: geometry.Point{X: 136, Y: 106}}))
}

func TestOffsetRects_ShiftsChildrenRecursively(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	rect := TreemapRectangle{
		Bounds: geometry.Rect{Min: geometry.Point{X: 0, Y: 0}, Max: geometry.Point{X: 200, Y: 100}},
		Children: []TreemapRectangle{
			{Bounds: geometry.Rect{Min: geometry.Point{X: 5, Y: 5}, Max: geometry.Point{X: 95, Y: 95}}},
			{
				Bounds: geometry.Rect{Min: geometry.Point{X: 100, Y: 5}, Max: geometry.Point{X: 190, Y: 95}},
				Children: []TreemapRectangle{
					{Bounds: geometry.Rect{Min: geometry.Point{X: 105, Y: 10}, Max: geometry.Point{X: 145, Y: 50}}},
				},
			},
		},
	}

	OffsetRects(&rect, geometry.Vector{X: 50, Y: 100})
	g.Expect(rect.Bounds.Min.X).To(Equal(50.0))
	g.Expect(rect.Bounds.Min.Y).To(Equal(100.0))
	g.Expect(rect.Children[0].Bounds.Min.X).To(Equal(55.0))
	g.Expect(rect.Children[0].Bounds.Min.Y).To(Equal(105.0))
	g.Expect(rect.Children[1].Bounds.Min.X).To(Equal(150.0))
	g.Expect(rect.Children[1].Bounds.Min.Y).To(Equal(105.0))
	g.Expect(rect.Children[1].Children[0].Bounds.Min.X).To(Equal(155.0))
	g.Expect(rect.Children[1].Children[0].Bounds.Min.Y).To(Equal(110.0))
}

func TestOffsetRects_ZeroOffset_NoChange(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	rect := TreemapRectangle{Bounds: geometry.Rect{Min: geometry.Point{X: 10, Y: 20}, Max: geometry.Point{X: 110, Y: 70}}}
	OffsetRects(&rect, geometry.Vector{X: 0, Y: 0})
	g.Expect(rect.Bounds.Min.X).To(Equal(10.0))
	g.Expect(rect.Bounds.Min.Y).To(Equal(20.0))
}

func TestOffsetRects_PreservesVisibleDepth(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	rect := TreemapRectangle{
		Bounds:       geometry.Rect{Min: geometry.Point{X: 10, Y: 20}, Max: geometry.Point{X: 110, Y: 70}},
		IsDirectory:  true,
		VisibleDepth: 3,
		Chrome: DirectoryChrome{
			Orientation: DirectoryLabelTop,
			Rail:        geometry.Rect{Min: geometry.Point{X: 10, Y: 20}, Max: geometry.Point{X: 110, Y: 40}},
			Content:     geometry.Rect{Min: geometry.Point{X: 14, Y: 40}, Max: geometry.Point{X: 106, Y: 66}},
		},
		Children: []TreemapRectangle{
			{VisibleDepth: 0},
		},
	}

	OffsetRects(&rect, geometry.Vector{X: 30, Y: 40})

	g.Expect(rect.VisibleDepth).To(Equal(3))
	g.Expect(rect.Children[0].VisibleDepth).To(Equal(0))
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

	ratio := (highRect.Bounds.Width() * highRect.Bounds.Height()) / (lowRect.Bounds.Width() * lowRect.Bounds.Height())
	g.Expect(ratio).To(BeNumerically("~", 9.0, 2.0))
}
