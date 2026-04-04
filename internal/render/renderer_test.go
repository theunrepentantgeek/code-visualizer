package render

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/bevan/code-visualizer/internal/scan"
	"github.com/bevan/code-visualizer/internal/treemap"
)

func TestRenderFlatDir(t *testing.T) {
	g := NewGomegaWithT(t)

	root := scan.DirectoryNode{
		Name: "flat",
		Files: []scan.FileNode{
			{Name: "small.txt", Size: 5, Extension: "txt", FileType: "txt"},
			{Name: "medium.go", Size: 100, Extension: "go", FileType: "go"},
			{Name: "large.rs", Size: 1000, Extension: "rs", FileType: "rs"},
		},
	}

	rects := treemap.Layout(root, 800, 600)
	out := filepath.Join(t.TempDir(), "flat.png")
	err := RenderPNG(rects, 800, 600, out)
	g.Expect(err).NotTo(HaveOccurred())

	info, err := os.Stat(out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(info.Size()).To(BeNumerically(">", 0))
}

func TestRenderNestedDir(t *testing.T) {
	g := NewGomegaWithT(t)

	root := scan.DirectoryNode{
		Name: "nested",
		Files: []scan.FileNode{
			{Name: "root.txt", Size: 50, Extension: "txt", FileType: "txt"},
		},
		Dirs: []scan.DirectoryNode{
			{
				Name: "sub",
				Files: []scan.FileNode{
					{Name: "child.go", Size: 200, Extension: "go", FileType: "go"},
				},
			},
		},
	}

	rects := treemap.Layout(root, 800, 600)
	out := filepath.Join(t.TempDir(), "nested.png")
	err := RenderPNG(rects, 800, 600, out)
	g.Expect(err).NotTo(HaveOccurred())

	info, err := os.Stat(out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(info.Size()).To(BeNumerically(">", 0))
}
