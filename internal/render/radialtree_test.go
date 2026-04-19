package render

import (
	"bytes"
	"encoding/xml"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/provider/filesystem"
	"github.com/bevan/code-visualizer/internal/radialtree"
)

func TestRenderRadial_FlatDir(t *testing.T) {
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
	err := RenderRadial(&node, 800, out)
	g.Expect(err).NotTo(HaveOccurred())

	info, err := os.Stat(out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(info).NotTo(BeNil())

	if info == nil {
		return
	}

	g.Expect(info.Size()).To(BeNumerically(">", 0))
}

func TestRenderRadial_NestedDir(t *testing.T) {
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
	err := RenderRadial(&node, 800, out)
	g.Expect(err).NotTo(HaveOccurred())

	info, err := os.Stat(out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(info).NotTo(BeNil())

	if info == nil {
		return
	}

	g.Expect(info.Size()).To(BeNumerically(">", 0))
}

func TestRenderRadial_LabelModes(t *testing.T) {
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
			err := RenderRadial(&node, 400, out)
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

func TestRenderRadial_EmptyDir(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{Name: "empty"}
	node := radialtree.Layout(root, 400, filesystem.FileSize, radialtree.LabelAll)
	out := filepath.Join(t.TempDir(), "empty-radial.png")
	err := RenderRadial(&node, 400, out)
	g.Expect(err).NotTo(HaveOccurred())

	info, err := os.Stat(out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(info).NotTo(BeNil())

	if info == nil {
		return
	}

	g.Expect(info.Size()).To(BeNumerically(">", 0))
}

func TestRenderRadial_JPG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name:  "flat",
		Files: []*model.File{makeFile("a.go", "go", 100), makeFile("b.go", "go", 200)},
	}

	node := radialtree.Layout(root, 400, filesystem.FileSize, radialtree.LabelAll)
	out := filepath.Join(t.TempDir(), "radial.jpg")

	err := RenderRadial(&node, 400, out)
	g.Expect(err).NotTo(HaveOccurred())

	f, err := os.Open(out)
	g.Expect(err).NotTo(HaveOccurred())

	defer f.Close()

	_, format, err := image.DecodeConfig(f)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(format).To(Equal("jpeg"))
}

func TestRenderRadial_SVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name:  "flat",
		Files: []*model.File{makeFile("a.go", "go", 100), makeFile("b.go", "go", 200)},
	}

	node := radialtree.Layout(root, 400, filesystem.FileSize, radialtree.LabelAll)
	out := filepath.Join(t.TempDir(), "radial.svg")

	err := RenderRadial(&node, 400, out)
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

	g.Expect(rootElement).To(Equal("svg"), "SVG output should have an <svg> root element")
}

func TestRenderRadial_PNG_DecodesAsPNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name:  "flat",
		Files: []*model.File{makeFile("a.go", "go", 100)},
	}

	node := radialtree.Layout(root, 400, filesystem.FileSize, radialtree.LabelAll)
	out := filepath.Join(t.TempDir(), "radial.png")

	err := RenderRadial(&node, 400, out)
	g.Expect(err).NotTo(HaveOccurred())

	f, err := os.Open(out)
	g.Expect(err).NotTo(HaveOccurred())

	defer f.Close()

	_, format, err := image.DecodeConfig(f)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(format).To(Equal("png"))
}
