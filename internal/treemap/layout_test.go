package treemap

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/bevan/code-visualizer/internal/scan"
)

func TestLayoutSingleFile(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := scan.DirectoryNode{
		Name: "root",
		Files: []scan.FileNode{
			{Name: "only.go", Size: 100},
		},
	}

	rects := Layout(root, 1920, 1080)
	g.Expect(rects.Children).To(HaveLen(1))
	// Single file should occupy most of the available area
	child := rects.Children[0]
	g.Expect(child.W).To(BeNumerically(">", 0))
	g.Expect(child.H).To(BeNumerically(">", 0))
}

func TestLayoutProportionalAreas(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := scan.DirectoryNode{
		Name: "root",
		Files: []scan.FileNode{
			{Name: "big.go", Size: 900},
			{Name: "small.go", Size: 100},
		},
	}

	rects := Layout(root, 1000, 1000)
	// Find the two file rectangles
	var bigRect, smallRect TreemapRectangle

	for _, c := range rects.Children {
		if !c.IsDirectory {
			switch c.Label {
			case "big.go":
				bigRect = c
			case "small.go":
				smallRect = c
			default:
				// other files
			}
		}
	}

	bigArea := bigRect.W * bigRect.H
	smallArea := smallRect.W * smallRect.H
	// big should be roughly 9x the small (within tolerance for padding)
	ratio := bigArea / smallArea
	g.Expect(ratio).To(BeNumerically("~", 9.0, 2.0))
}

func TestLayoutNestedDirs(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := scan.DirectoryNode{
		Name: "root",
		Files: []scan.FileNode{
			{Name: "top.go", Size: 100},
		},
		Dirs: []scan.DirectoryNode{
			{
				Name: "sub",
				Files: []scan.FileNode{
					{Name: "inner.go", Size: 200},
				},
			},
		},
	}

	rects := Layout(root, 1920, 1080)
	// Should have children (file + directory group)
	g.Expect(len(rects.Children)).To(BeNumerically(">=", 2))

	// Find the directory child
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

	root := scan.DirectoryNode{
		Name: "root",
		Files: []scan.FileNode{
			{Name: "normal.go", Size: 1000},
			{Name: "empty.go", Size: 0},
		},
	}

	rects := Layout(root, 1920, 1080)
	// Zero-size file should get a minimum rectangle (FR-013)
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
