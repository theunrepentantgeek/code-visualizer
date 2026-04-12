package treemap

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/provider/filesystem"
)

func makeFile(name string, size int) *model.File {
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
	g.Expect(emptyRect.W).To(BeNumerically(">", 0))
	g.Expect(emptyRect.H).To(BeNumerically(">", 0))
}
