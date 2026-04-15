package render

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/provider/filesystem"
	"github.com/bevan/code-visualizer/internal/radialtree"
)

func TestRenderRadialPNG_FlatDir(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "flat",
		Files: []*model.File{
			makeFile("small.go", "go", 100),
			makeFile("medium.go", "go", 500),
			makeFile("large.go", "go", 2000),
		},
	}

	node := radialtree.Layout(root, 800, filesystem.FileSize, radialtree.LabelAll)
	out := filepath.Join(t.TempDir(), "flat-radial.png")
	err := RenderRadialPNG(&node, 800, out)
	g.Expect(err).NotTo(HaveOccurred())

	info, err := os.Stat(out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(info).NotTo(BeNil())

	if info == nil {
		return
	}

	g.Expect(info.Size()).To(BeNumerically(">", 0))
}

func TestRenderRadialPNG_NestedDir(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name:  "root",
		Files: []*model.File{makeFile("root.go", "go", 100)},
		Dirs: []*model.Directory{
			{
				Name:  "sub",
				Files: []*model.File{makeFile("child.go", "go", 500)},
			},
		},
	}

	node := radialtree.Layout(root, 800, filesystem.FileSize, radialtree.LabelAll)
	out := filepath.Join(t.TempDir(), "nested-radial.png")
	err := RenderRadialPNG(&node, 800, out)
	g.Expect(err).NotTo(HaveOccurred())

	info, err := os.Stat(out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(info).NotTo(BeNil())

	if info == nil {
		return
	}

	g.Expect(info.Size()).To(BeNumerically(">", 0))
}

func TestRenderRadialPNG_LabelModes(t *testing.T) {
	t.Parallel()

	root := &model.Directory{
		Name:  "root",
		Files: []*model.File{makeFile("a.go", "go", 100), makeFile("b.go", "go", 200)},
	}

	for _, mode := range []radialtree.LabelMode{
		radialtree.LabelAll,
		radialtree.LabelFoldersOnly,
		radialtree.LabelNone,
	} {
		t.Run(string(mode), func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			node := radialtree.Layout(root, 400, filesystem.FileSize, mode)
			out := filepath.Join(t.TempDir(), "labels-"+string(mode)+".png")
			err := RenderRadialPNG(&node, 400, out)
			g.Expect(err).NotTo(HaveOccurred())

			info, err := os.Stat(out)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(info).NotTo(BeNil())

			if info == nil {
				return
			}

			g.Expect(info.Size()).To(BeNumerically(">", 0))
		})
	}
}

func TestRenderRadialPNG_EmptyDir(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{Name: "empty"}
	node := radialtree.Layout(root, 400, filesystem.FileSize, radialtree.LabelAll)
	out := filepath.Join(t.TempDir(), "empty-radial.png")
	err := RenderRadialPNG(&node, 400, out)
	g.Expect(err).NotTo(HaveOccurred())

	info, err := os.Stat(out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(info).NotTo(BeNil())

	if info == nil {
		return
	}

	g.Expect(info.Size()).To(BeNumerically(">", 0))
}
