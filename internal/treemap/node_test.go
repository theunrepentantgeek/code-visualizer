package treemap

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/bevan/code-visualizer/internal/scan"
)

func TestDirectoryHeaderBar(t *testing.T) {
	t.Parallel()
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

	if dirRect == nil {
		return
	}

	g.Expect(dirRect.IsDirectory).To(BeTrue())
	g.Expect(dirRect.Label).To(Equal("mydir"))
}

func TestDirectoryPaddingSeparatesGroups(t *testing.T) {
	t.Parallel()
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

	dirA := findDirRect(rects, "dir-a")
	dirB := findDirRect(rects, "dir-b")

	g.Expect(dirA).NotTo(BeNil())
	g.Expect(dirB).NotTo(BeNil())

	if dirA == nil || dirB == nil {
		return
	}

	// Directories should not overlap — there should be padding between them
	separated := rectsAreSeparated(dirA, dirB)
	g.Expect(separated).To(BeTrue(), "directory groups should not overlap")
}

func findDirRect(rects TreemapRectangle, name string) *TreemapRectangle {
	for i, c := range rects.Children {
		if c.IsDirectory && c.Label == name {
			return &rects.Children[i]
		}
	}

	return nil
}

func rectsAreSeparated(a, b *TreemapRectangle) bool {
	aRight := a.X + a.W
	bRight := b.X + b.W
	aBottom := a.Y + a.H
	bBottom := b.Y + b.H

	return aRight <= b.X || bRight <= a.X || aBottom <= b.Y || bBottom <= a.Y
}
