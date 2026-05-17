package radialtree_test

import (
	"bytes"
	"encoding/xml"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/radialtree"
)

func radialTestRoot() *model.Directory {
	return &model.Directory{
		Name: "flat",
		Files: []*model.File{
			makeFile("small.txt", "txt", 5),
			makeFile("medium.go", "go", 100),
			makeFile("large.rs", "rs", 1000),
		},
	}
}

func TestRenderToCanvas_PNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := radialTestRoot()
	nodes := radialtree.Layout(root, 800, filesystem.FileSize, radialtree.LabelNone)
	inks := radialtree.BuildInks(root, filesystem.FileSize, palette.Temperature, "", "")
	cv := radialtree.RenderToCanvas(&nodes, root, 800, inks)

	out := filepath.Join(t.TempDir(), "radial.png")
	err := cv.Render(out)
	g.Expect(err).NotTo(HaveOccurred())

	f, err := os.Open(out)
	g.Expect(err).NotTo(HaveOccurred())

	defer f.Close()

	_, format, err := image.DecodeConfig(f)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(format).To(Equal("png"))
}

func TestRenderToCanvas_SVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := radialTestRoot()
	nodes := radialtree.Layout(root, 400, filesystem.FileSize, radialtree.LabelNone)
	inks := radialtree.BuildInks(root, filesystem.FileSize, palette.Temperature, "", "")
	cv := radialtree.RenderToCanvas(&nodes, root, 400, inks)

	out := filepath.Join(t.TempDir(), "radial.svg")
	err := cv.Render(out)
	g.Expect(err).NotTo(HaveOccurred())

	data, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())

	decoder := xml.NewDecoder(bytes.NewReader(data))

	var rootElement string

	for {
		tok, xmlErr := decoder.Token()
		if xmlErr != nil {
			break
		}

		if se, ok := tok.(xml.StartElement); ok {
			rootElement = se.Name.Local

			break
		}
	}

	g.Expect(rootElement).To(Equal("svg"))
}

func TestRenderToCanvas_NestedDirs(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "project",
		Files: []*model.File{
			makeFile("readme.md", "md", 50),
		},
		Dirs: []*model.Directory{
			{
				Name: "src",
				Files: []*model.File{
					makeFile("main.go", "go", 200),
					makeFile("util.go", "go", 80),
				},
			},
		},
	}

	nodes := radialtree.Layout(root, 800, filesystem.FileSize, radialtree.LabelAll)
	inks := radialtree.BuildInks(root, filesystem.FileSize, palette.Temperature, "", "")
	cv := radialtree.RenderToCanvas(&nodes, root, 800, inks)

	out := filepath.Join(t.TempDir(), "nested.png")
	err := cv.Render(out)
	g.Expect(err).NotTo(HaveOccurred())

	info, err := os.Stat(out)
	g.Expect(err).NotTo(HaveOccurred())

	if info != nil {
		g.Expect(info.Size()).To(BeNumerically(">", 0))
	}
}

func TestRenderToCanvas_EmptyDir(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{Name: "empty"}

	nodes := radialtree.Layout(root, 400, filesystem.FileSize, radialtree.LabelNone)
	inks := radialtree.BuildInks(root, filesystem.FileSize, palette.Temperature, "", "")
	cv := radialtree.RenderToCanvas(&nodes, root, 400, inks)

	out := filepath.Join(t.TempDir(), "empty.png")
	err := cv.Render(out)
	g.Expect(err).NotTo(HaveOccurred())

	info, err := os.Stat(out)
	g.Expect(err).NotTo(HaveOccurred())

	if info != nil {
		g.Expect(info.Size()).To(BeNumerically(">", 0))
	}
}
