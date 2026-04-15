package radialtree

import (
	"math"
	"sort"
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

func TestLayoutRootIsAtCentre(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name:  "root",
		Files: []*model.File{makeFile("only.go", 100)},
	}

	node := Layout(root, 800, filesystem.FileSize, LabelAll)
	g.Expect(node.X).To(BeNumerically("==", 0))
	g.Expect(node.Y).To(BeNumerically("==", 0))
}

func TestLayoutChildrenInRing(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			makeFile("a.go", 100),
			makeFile("b.go", 100),
			makeFile("c.go", 100),
		},
	}

	node := Layout(root, 800, filesystem.FileSize, LabelAll)
	g.Expect(node.Children).To(HaveLen(3))

	var radii []float64
	for _, child := range node.Children {
		r := math.Sqrt(child.X*child.X + child.Y*child.Y)
		g.Expect(r).To(BeNumerically(">", 0))
		radii = append(radii, r)
	}

	// All three children should be at approximately the same radius.
	g.Expect(radii[0]).To(BeNumerically("~", radii[1], radii[0]*0.01))
	g.Expect(radii[0]).To(BeNumerically("~", radii[2], radii[0]*0.01))
}

func TestLayoutSingleFile(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name:  "root",
		Files: []*model.File{makeFile("only.go", 100)},
	}

	node := Layout(root, 800, filesystem.FileSize, LabelAll)
	g.Expect(node.Children).To(HaveLen(1))
	g.Expect(node.Children[0].DiscRadius).To(BeNumerically(">", 0))
}

func TestLayoutAnglesFullCircle(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			makeFile("a.go", 100),
			makeFile("b.go", 100),
			makeFile("c.go", 100),
			makeFile("d.go", 100),
		},
	}

	node := Layout(root, 800, filesystem.FileSize, LabelAll)
	g.Expect(node.Children).To(HaveLen(4))

	angles := make([]float64, 4)
	for i, child := range node.Children {
		angles[i] = child.Angle
	}
	sort.Float64s(angles)

	// 4 equal-weight files should be spaced ~π/2 apart
	expectedGap := 2 * math.Pi / 4
	for i := 0; i < 3; i++ {
		gap := angles[i+1] - angles[i]
		g.Expect(gap).To(BeNumerically("~", expectedGap, expectedGap*0.05),
			"angles[%d] to angles[%d] gap should be ~%.3f", i, i+1, expectedGap)
	}
}

func TestLayoutNestedDepth(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	inner := &model.Directory{
		Name:  "sub",
		Files: []*model.File{makeFile("inner.go", 200)},
	}
	root := &model.Directory{
		Name: "root",
		Dirs: []*model.Directory{inner},
	}

	node := Layout(root, 800, filesystem.FileSize, LabelAll)

	// Root is at centre (radius 0).
	g.Expect(node.X).To(BeNumerically("==", 0))
	g.Expect(node.Y).To(BeNumerically("==", 0))

	g.Expect(node.Children).To(HaveLen(1))
	subNode := node.Children[0]
	subRadius := math.Sqrt(subNode.X*subNode.X + subNode.Y*subNode.Y)
	g.Expect(subRadius).To(BeNumerically(">", 0))

	g.Expect(subNode.Children).To(HaveLen(1))
	fileNode := subNode.Children[0]
	fileRadius := math.Sqrt(fileNode.X*fileNode.X + fileNode.Y*fileNode.Y)
	g.Expect(fileRadius).To(BeNumerically(">", subRadius))
}

func TestLayoutDiscSizeScalesWithMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			makeFile("small.go", 100),
			makeFile("large.go", 1000),
		},
	}

	node := Layout(root, 800, filesystem.FileSize, LabelAll)
	g.Expect(node.Children).To(HaveLen(2))

	var smallDisc, largeDisc float64

	for _, child := range node.Children {
		switch child.Label {
		case "small.go":
			smallDisc = child.DiscRadius
		case "large.go":
			largeDisc = child.DiscRadius
		}
	}

	g.Expect(largeDisc).To(BeNumerically(">", smallDisc))
}

func TestLayoutLabelAll(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name:  "root",
		Files: []*model.File{makeFile("only.go", 100)},
	}

	node := Layout(root, 800, filesystem.FileSize, LabelAll)
	g.Expect(node.ShowLabel).To(BeTrue())
	g.Expect(node.Children).To(HaveLen(1))
	g.Expect(node.Children[0].ShowLabel).To(BeTrue())
}

func TestLayoutLabelFoldersOnly(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name:  "root",
		Files: []*model.File{makeFile("only.go", 100)},
	}

	node := Layout(root, 800, filesystem.FileSize, LabelFoldersOnly)
	g.Expect(node.IsDirectory).To(BeTrue())
	g.Expect(node.ShowLabel).To(BeTrue())
	g.Expect(node.Children).To(HaveLen(1))
	g.Expect(node.Children[0].ShowLabel).To(BeFalse())
}

func TestLayoutLabelNone(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name:  "root",
		Files: []*model.File{makeFile("only.go", 100)},
	}

	node := Layout(root, 800, filesystem.FileSize, LabelNone)
	g.Expect(node.ShowLabel).To(BeFalse())
	g.Expect(node.Children).To(HaveLen(1))
	g.Expect(node.Children[0].ShowLabel).To(BeFalse())
}

func TestLayoutEmptyDirectory(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{Name: "empty"}

	// Should not panic.
	node := Layout(root, 800, filesystem.FileSize, LabelAll)
	g.Expect(node.X).To(BeNumerically("==", 0))
	g.Expect(node.Y).To(BeNumerically("==", 0))
}

func TestLayoutRootLabel(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name:  "myroot",
		Files: []*model.File{makeFile("only.go", 100)},
	}

	node := Layout(root, 800, filesystem.FileSize, LabelAll)
	g.Expect(node.Label).To(Equal("myroot"))
}

func TestLayoutCanvasSize(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	makeRoot := func() *model.Directory {
		return &model.Directory{
			Name: "root",
			Files: []*model.File{
				makeFile("a.go", 100),
				makeFile("b.go", 100),
			},
		}
	}

	small := Layout(makeRoot(), 800, filesystem.FileSize, LabelAll)
	large := Layout(makeRoot(), 1600, filesystem.FileSize, LabelAll)

	g.Expect(small.Children).To(HaveLen(2))
	g.Expect(large.Children).To(HaveLen(2))

	smallRadius := math.Sqrt(small.Children[0].X*small.Children[0].X + small.Children[0].Y*small.Children[0].Y)
	largeRadius := math.Sqrt(large.Children[0].X*large.Children[0].X + large.Children[0].Y*large.Children[0].Y)

	g.Expect(largeRadius).To(BeNumerically(">", smallRadius))
}

func TestLayoutZeroMetricUsesMinDisc(t *testing.T) {
t.Parallel()
g := NewGomegaWithT(t)

// File with zero metric value (no SetQuantity called for FileSize)
emptyFile := &model.File{Name: "empty.go"}

root := &model.Directory{
Name:  "root",
Files: []*model.File{emptyFile},
}

node := Layout(root, 800, filesystem.FileSize, LabelAll)
g.Expect(node.Children).To(HaveLen(1))
// Should use the minimum disc size floor, not zero
g.Expect(node.Children[0].DiscRadius).To(BeNumerically("==", minFileDisc))
}

func TestLayoutUniformMetricUsesMidpoint(t *testing.T) {
t.Parallel()
g := NewGomegaWithT(t)

root := &model.Directory{
Name: "root",
Files: []*model.File{
makeFile("a.go", 500),
makeFile("b.go", 500),
makeFile("c.go", 500),
},
}

node := Layout(root, 800, filesystem.FileSize, LabelAll)
g.Expect(node.Children).To(HaveLen(3))

// All discs should be the same size (midpoint between min and max)
radius0 := node.Children[0].DiscRadius
for _, child := range node.Children[1:] {
g.Expect(child.DiscRadius).To(BeNumerically("~", radius0, 0.001))
}
// Midpoint should be > minFileDisc (not minimum)
g.Expect(radius0).To(BeNumerically(">", minFileDisc))
}

func TestComputeLeafCountEmptyDir(t *testing.T) {
t.Parallel()
g := NewGomegaWithT(t)

empty := &model.Directory{Name: "empty"}
g.Expect(computeLeafCount(empty)).To(Equal(0))
}

func TestComputeLeafCountWithFiles(t *testing.T) {
t.Parallel()
g := NewGomegaWithT(t)

dir := &model.Directory{
Name:  "dir",
Files: []*model.File{makeFile("a.go", 100), makeFile("b.go", 200)},
}
g.Expect(computeLeafCount(dir)).To(Equal(2))
}
