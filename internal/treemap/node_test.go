package treemap

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
)

func TestSmallDirectoryChromeKeepsInsetContentWithoutRail(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Dirs: []*model.Directory{
			{
				Name:  "my-super-long-directory-name",
				Files: []*model.File{makeFile("file.go", 100)},
			},
		},
	}

	rects := Layout(root, 50, 50, filesystem.FileSize)

	var dirRect *TreemapRectangle

	for i, c := range rects.Children {
		if c.IsDirectory && c.Label == "my-super-long-directory-name" {
			dirRect = &rects.Children[i]

			break
		}
	}

	g.Expect(dirRect).NotTo(BeNil())

	if dirRect == nil {
		return
	}

	g.Expect(dirRect.IsDirectory).To(BeTrue())
	g.Expect(dirRect.Label).To(Equal("my-super-long-directory-name"))
	g.Expect(dirRect.Chrome.Orientation).To(Equal(DirectoryLabelNone))
	g.Expect(dirRect.Chrome.Content.X).To(BeNumerically(">", dirRect.X))
	g.Expect(dirRect.Chrome.Content.Y).To(BeNumerically(">", dirRect.Y))
}

func TestDirectoryPaddingSeparatesGroups(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Dirs: []*model.Directory{
			{
				Name:  "dir-a",
				Files: []*model.File{makeFile("a.go", 100)},
			},
			{
				Name:  "dir-b",
				Files: []*model.File{makeFile("b.go", 100)},
			},
		},
	}

	rects := Layout(root, 1920, 1080, filesystem.FileSize)

	dirA := findDirRect(rects, "dir-a")
	dirB := findDirRect(rects, "dir-b")

	g.Expect(dirA).NotTo(BeNil())
	g.Expect(dirB).NotTo(BeNil())

	if dirA == nil || dirB == nil {
		return
	}

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
