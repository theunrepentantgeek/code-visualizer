package render

import (
	"image/color"
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

func TestRenderWithBorderColour(t *testing.T) {
	g := NewGomegaWithT(t)

	red := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	blue := color.RGBA{R: 0, G: 0, B: 255, A: 255}
	green := color.RGBA{R: 0, G: 255, B: 0, A: 255}

	rects := treemap.TreemapRectangle{
		X: 0, Y: 0, W: 800, H: 600,
		Label: "root", IsDirectory: true,
		Children: []treemap.TreemapRectangle{
			{X: 4, Y: 20, W: 380, H: 576, Label: "a.go", FillColour: red, BorderColour: &blue},
			{X: 388, Y: 20, W: 380, H: 576, Label: "b.go", FillColour: green, BorderColour: &red},
		},
	}

	out := filepath.Join(t.TempDir(), "border.png")
	err := RenderPNG(rects, 800, 600, out)
	g.Expect(err).NotTo(HaveOccurred())

	info, err := os.Stat(out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(info.Size()).To(BeNumerically(">", 0))
}

func TestRenderNoBorderWhenNil(t *testing.T) {
	g := NewGomegaWithT(t)

	rects := treemap.TreemapRectangle{
		X: 0, Y: 0, W: 400, H: 300,
		Label: "root", IsDirectory: true,
		Children: []treemap.TreemapRectangle{
			{X: 4, Y: 20, W: 392, H: 276, Label: "a.go",
				FillColour:   color.RGBA{R: 200, G: 200, B: 200, A: 255},
				BorderColour: nil},
		},
	}

	out := filepath.Join(t.TempDir(), "noborder.png")
	err := RenderPNG(rects, 400, 300, out)
	g.Expect(err).NotTo(HaveOccurred())

	info, err := os.Stat(out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(info.Size()).To(BeNumerically(">", 0))
}
