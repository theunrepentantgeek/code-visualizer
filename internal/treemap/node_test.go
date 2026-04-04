package treemap

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/bevan/code-visualizer/internal/scan"
)

func TestDirectoryHeaderBar(t *testing.T) {
	g := NewGomegaWithT(t)

	root := scan.DirectoryNode{
		Name: "root",
		Dirs: []scan.DirectoryNode{
			{
				Name: "mydir",
				Files: []scan.FileNode{
					{Name: "file.go", Size: 100},
				},
			},
		},
	}

	rects := Layout(root, 1920, 1080)

	// Find the directory rectangle
	var dirRect *TreemapRectangle
	for i, c := range rects.Children {
		if c.IsDirectory && c.Label == "mydir" {
			dirRect = &rects.Children[i]
			break
		}
	}
	g.Expect(dirRect).NotTo(BeNil())
	g.Expect(dirRect.IsDirectory).To(BeTrue())
	g.Expect(dirRect.Label).To(Equal("mydir"))
}

func TestDirectoryPaddingSeparatesGroups(t *testing.T) {
	g := NewGomegaWithT(t)

	root := scan.DirectoryNode{
		Name: "root",
		Dirs: []scan.DirectoryNode{
			{
				Name:  "dir-a",
				Files: []scan.FileNode{{Name: "a.go", Size: 100}},
			},
			{
				Name:  "dir-b",
				Files: []scan.FileNode{{Name: "b.go", Size: 100}},
			},
		},
	}

	rects := Layout(root, 1920, 1080)

	// Find the two directory rectangles
	var dirA, dirB *TreemapRectangle
	for i, c := range rects.Children {
		if c.IsDirectory && c.Label == "dir-a" {
			dirA = &rects.Children[i]
		}
		if c.IsDirectory && c.Label == "dir-b" {
			dirB = &rects.Children[i]
		}
	}
	g.Expect(dirA).NotTo(BeNil())
	g.Expect(dirB).NotTo(BeNil())

	// Directories should not overlap — there should be padding between them
	aRight := dirA.X + dirA.W
	bLeft := dirB.X
	aBottom := dirA.Y + dirA.H
	bTop := dirB.Y
	separated := aRight <= bLeft || bLeft+dirB.W <= dirA.X || aBottom <= bTop || bTop+dirB.H <= dirA.Y
	g.Expect(separated).To(BeTrue(), "directory groups should not overlap")
}
