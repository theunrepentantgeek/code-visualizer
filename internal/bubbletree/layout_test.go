package bubbletree

import (
	"math"
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

// allChildren collects every descendant BubbleNode via depth-first walk.
func allChildren(node *BubbleNode) []BubbleNode {
	var result []BubbleNode

	for i := range node.Children {
		result = append(result, node.Children[i])
		result = append(result, allChildren(&node.Children[i])...)
	}

	return result
}

// assertContainment verifies that every child circle is geometrically inside its
// parent circle (distance + childRadius <= parentRadius + tolerance).
func assertContainment(g Gomega, parent BubbleNode) {
	for _, child := range parent.Children {
		dist := math.Sqrt((child.X-parent.X)*(child.X-parent.X) + (child.Y-parent.Y)*(child.Y-parent.Y))
		g.Expect(dist + child.Radius).To(
			BeNumerically("<=", parent.Radius+1.0),
			"child %q must be contained in parent %q", child.Label, parent.Label,
		)
		// Recurse into nested directories.
		assertContainment(g, child)
	}
}

// assertNoOverlap verifies that no two sibling circles overlap
// (distance between centres >= sum of radii - tolerance).
func assertNoOverlap(g Gomega, parent BubbleNode) {
	for i := 0; i < len(parent.Children); i++ {
		for j := i + 1; j < len(parent.Children); j++ {
			a := parent.Children[i]
			b := parent.Children[j]
			dist := math.Sqrt((a.X-b.X)*(a.X-b.X) + (a.Y-b.Y)*(a.Y-b.Y))
			g.Expect(dist).To(
				BeNumerically(">=", a.Radius+b.Radius-1.0),
				"siblings %q and %q must not overlap", a.Label, b.Label,
			)
		}
	}

	// Recurse into child directories.
	for _, child := range parent.Children {
		if child.IsDirectory {
			assertNoOverlap(g, child)
		}
	}
}

func TestLayoutRootEnclosure(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			makeFile("a.go", 200),
			makeFile("b.go", 500),
			makeFile("c.go", 300),
		},
	}

	node := Layout(root, 1920, 1080, filesystem.FileSize, LabelAll)
	g.Expect(node.Radius).To(BeNumerically(">", 0))
	g.Expect(node.IsDirectory).To(BeTrue())
	assertContainment(g, node)
}

func TestLayoutNoOverlap(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			makeFile("a.go", 100),
			makeFile("b.go", 200),
			makeFile("c.go", 300),
			makeFile("d.go", 400),
			makeFile("e.go", 500),
		},
	}

	node := Layout(root, 1920, 1080, filesystem.FileSize, LabelAll)
	assertNoOverlap(g, node)
}

func TestLayoutRadiusScaling(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			makeFile("small.go", 100),
			makeFile("large.go", 1000),
		},
	}

	node := Layout(root, 1920, 1080, filesystem.FileSize, LabelAll)
	g.Expect(node.Children).To(HaveLen(2))

	var smallRadius, largeRadius float64

	for _, child := range node.Children {
		switch child.Label {
		case "small.go":
			smallRadius = child.Radius
		case "large.go":
			largeRadius = child.Radius
		default:
		}
	}

	g.Expect(largeRadius).To(BeNumerically(">", smallRadius))
}

func TestLayoutNestingDepth(t *testing.T) {
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

	node := Layout(root, 1920, 1080, filesystem.FileSize, LabelAll)
	g.Expect(node.IsDirectory).To(BeTrue())
	g.Expect(node.Radius).To(BeNumerically(">", 0))
	g.Expect(node.Children).To(HaveLen(1))

	subNode := node.Children[0]
	g.Expect(subNode.IsDirectory).To(BeTrue())
	g.Expect(subNode.Radius).To(BeNumerically(">", 0))
	g.Expect(subNode.Children).To(HaveLen(1))

	fileNode := subNode.Children[0]
	g.Expect(fileNode.IsDirectory).To(BeFalse())
	g.Expect(fileNode.Radius).To(BeNumerically(">", 0))

	// Nested containment: sub inside root, file inside sub.
	assertContainment(g, node)
}

func TestLayoutLabelAll(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name:  "root",
		Files: []*model.File{makeFile("only.go", 100)},
	}

	node := Layout(root, 1920, 1080, filesystem.FileSize, LabelAll)
	g.Expect(node.ShowLabel).To(BeTrue())
	g.Expect(node.Children).To(HaveLen(1))
	g.Expect(node.Children[0].ShowLabel).To(BeTrue())
}

func TestLayoutLabelFoldersOnly(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	inner := &model.Directory{
		Name:  "sub",
		Files: []*model.File{makeFile("inner.go", 100)},
	}
	root := &model.Directory{
		Name: "root",
		Dirs: []*model.Directory{inner},
	}

	node := Layout(root, 1920, 1080, filesystem.FileSize, LabelFoldersOnly)
	g.Expect(node.IsDirectory).To(BeTrue())
	g.Expect(node.ShowLabel).To(BeTrue())

	g.Expect(node.Children).To(HaveLen(1))
	subNode := node.Children[0]
	g.Expect(subNode.IsDirectory).To(BeTrue())
	g.Expect(subNode.ShowLabel).To(BeTrue())

	g.Expect(subNode.Children).To(HaveLen(1))
	fileNode := subNode.Children[0]
	g.Expect(fileNode.IsDirectory).To(BeFalse())
	g.Expect(fileNode.ShowLabel).To(BeFalse())
}

func TestLayoutLabelNone(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name:  "root",
		Files: []*model.File{makeFile("only.go", 100)},
	}

	node := Layout(root, 1920, 1080, filesystem.FileSize, LabelNone)
	g.Expect(node.ShowLabel).To(BeFalse())
	g.Expect(node.Children).To(HaveLen(1))
	g.Expect(node.Children[0].ShowLabel).To(BeFalse())
}

func TestLayoutEmptyDirectory(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{Name: "empty"}

	// Should not panic.
	node := Layout(root, 1920, 1080, filesystem.FileSize, LabelAll)
	g.Expect(node.Radius).To(BeNumerically(">", 0))
	g.Expect(node.IsDirectory).To(BeTrue())
	g.Expect(node.Children).To(BeEmpty())
}

func TestLayoutSingleFile(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name:  "root",
		Files: []*model.File{makeFile("only.go", 100)},
	}

	node := Layout(root, 1920, 1080, filesystem.FileSize, LabelAll)
	g.Expect(node.Children).To(HaveLen(1))

	child := node.Children[0]
	g.Expect(child.Radius).To(BeNumerically(">", 0))

	// Single child should be roughly centred in the parent.
	dist := math.Sqrt((child.X-node.X)*(child.X-node.X) + (child.Y-node.Y)*(child.Y-node.Y))
	g.Expect(dist).To(BeNumerically("<", node.Radius))

	// Must be contained.
	g.Expect(dist + child.Radius).To(BeNumerically("<=", node.Radius+1.0))
}

func TestLayoutLargeFlatDirectory(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	files := make([]*model.File, 20)
	for i := range files {
		files[i] = makeFile("file"+string(rune('a'+i))+".go", int64(100+i*50))
	}

	root := &model.Directory{
		Name:  "root",
		Files: files,
	}

	node := Layout(root, 1920, 1080, filesystem.FileSize, LabelAll)
	g.Expect(node.Children).To(HaveLen(20))

	// All children must be contained.
	assertContainment(g, node)
	// No siblings should overlap.
	assertNoOverlap(g, node)
}

func TestLayoutZeroMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// File with no metric value set (zero/missing).
	emptyFile := &model.File{Name: "empty.go"}

	root := &model.Directory{
		Name:  "root",
		Files: []*model.File{emptyFile},
	}

	node := Layout(root, 1920, 1080, filesystem.FileSize, LabelAll)
	g.Expect(node.Children).To(HaveLen(1))
	// Should have a positive radius (minimum floor), not zero.
	g.Expect(node.Children[0].Radius).To(BeNumerically(">", 0))
}

func TestLayoutUniformMetric(t *testing.T) {
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

	node := Layout(root, 1920, 1080, filesystem.FileSize, LabelAll)
	g.Expect(node.Children).To(HaveLen(3))

	// All file circles should have the same radius.
	radius0 := node.Children[0].Radius
	g.Expect(radius0).To(BeNumerically(">", 0))

	for _, child := range node.Children[1:] {
		g.Expect(child.Radius).To(BeNumerically("~", radius0, 0.001))
	}
}

func TestLayoutFitsWithinCanvas(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			makeFile("a.go", 200),
			makeFile("b.go", 300),
			makeFile("c.go", 100),
		},
	}

	width, height := 1920, 1080

	node := Layout(root, width, height, filesystem.FileSize, LabelAll)

	// The root circle's bounding box must fit within the canvas.
	g.Expect(node.X - node.Radius).To(BeNumerically(">=", -1.0))
	g.Expect(node.Y - node.Radius).To(BeNumerically(">=", -1.0))
	g.Expect(node.X + node.Radius).To(BeNumerically("<=", float64(width)+1.0))
	g.Expect(node.Y + node.Radius).To(BeNumerically("<=", float64(height)+1.0))
}

func TestLayoutRootLabel(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name:  "myproject",
		Files: []*model.File{makeFile("only.go", 100)},
	}

	node := Layout(root, 1920, 1080, filesystem.FileSize, LabelAll)
	g.Expect(node.Label).To(Equal("myproject"))
}

func TestLayoutRootIsDirectory(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name:  "root",
		Files: []*model.File{makeFile("a.go", 100)},
	}

	node := Layout(root, 1920, 1080, filesystem.FileSize, LabelAll)
	g.Expect(node.IsDirectory).To(BeTrue())

	g.Expect(node.Children).To(HaveLen(1))
	g.Expect(node.Children[0].IsDirectory).To(BeFalse())
}

func TestLayoutDeepNesting(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Build a 3-level deep tree: root → mid → leaf (with file)
	leaf := &model.Directory{
		Name:  "leaf",
		Files: []*model.File{makeFile("deep.go", 100)},
	}
	mid := &model.Directory{
		Name: "mid",
		Dirs: []*model.Directory{leaf},
	}
	root := &model.Directory{
		Name: "root",
		Dirs: []*model.Directory{mid},
	}

	node := Layout(root, 1920, 1080, filesystem.FileSize, LabelAll)

	// Walk three levels down.
	g.Expect(node.Children).To(HaveLen(1))
	midNode := node.Children[0]
	g.Expect(midNode.IsDirectory).To(BeTrue())
	g.Expect(midNode.Children).To(HaveLen(1))

	leafNode := midNode.Children[0]
	g.Expect(leafNode.IsDirectory).To(BeTrue())
	g.Expect(leafNode.Children).To(HaveLen(1))
	g.Expect(leafNode.Children[0].IsDirectory).To(BeFalse())

	// Full containment must hold at every level.
	assertContainment(g, node)
}

func TestLayoutMixedFilesAndDirs(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sub := &model.Directory{
		Name:  "pkg",
		Files: []*model.File{makeFile("lib.go", 300)},
	}
	root := &model.Directory{
		Name:  "root",
		Files: []*model.File{makeFile("main.go", 200)},
		Dirs:  []*model.Directory{sub},
	}

	node := Layout(root, 1920, 1080, filesystem.FileSize, LabelAll)
	g.Expect(node.Children).To(HaveLen(2))

	var dirChild *BubbleNode

	for i, child := range node.Children {
		if child.IsDirectory {
			dirChild = &node.Children[i]

			break
		}
	}

	g.Expect(dirChild).NotTo(BeNil())

	if dirChild == nil {
		return
	}

	g.Expect(dirChild.Label).To(Equal("pkg"))
	g.Expect(dirChild.Children).To(HaveLen(1))

	// No overlap between the file and the directory bubble.
	assertNoOverlap(g, node)
	assertContainment(g, node)
}
