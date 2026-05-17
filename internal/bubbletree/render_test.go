package bubbletree_test

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

	"github.com/theunrepentantgeek/code-visualizer/internal/bubbletree"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
)

func testRoot() *model.Directory {
	return &model.Directory{
		Path: "root",
		Files: []*model.File{
			makeFile("main.go", "go", 100),
			makeFile("style.css", "css", 50),
		},
		Dirs: []*model.Directory{
			{
				Path:  "root/pkg",
				Files: []*model.File{makeFile("lib.go", "go", 200)},
			},
		},
	}
}

func TestRenderBubbleToCanvas_PNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := testRoot()
	nodes := bubbletree.Layout(root, 1000, 800, filesystem.FileSize, bubbletree.LabelFoldersOnly)
	inks := bubbletree.BuildInks(root, filesystem.FileSize, palette.Temperature, "", "")
	cv := bubbletree.RenderToCanvas(&nodes, root, 1000, 800, inks)

	out := filepath.Join(t.TempDir(), "bubble.png")
	g.Expect(cv.Render(out)).To(Succeed())

	f, err := os.Open(out)
	g.Expect(err).NotTo(HaveOccurred())

	defer f.Close()

	_, format, err := image.DecodeConfig(f)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(format).To(Equal("png"))
}

func TestRenderBubbleToCanvas_SVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := testRoot()
	nodes := bubbletree.Layout(root, 800, 600, filesystem.FileSize, bubbletree.LabelFoldersOnly)
	inks := bubbletree.BuildInks(root, filesystem.FileSize, palette.Temperature, "", "")
	cv := bubbletree.RenderToCanvas(&nodes, root, 800, 600, inks)

	out := filepath.Join(t.TempDir(), "bubble.svg")
	g.Expect(cv.Render(out)).To(Succeed())

	data, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())

	dec := xml.NewDecoder(bytes.NewReader(data))

	var rootElement string

	for {
		tok, xmlErr := dec.Token()
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

func TestRenderBubbleToCanvas_JPG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := testRoot()
	nodes := bubbletree.Layout(root, 400, 300, filesystem.FileSize, bubbletree.LabelFoldersOnly)
	inks := bubbletree.BuildInks(root, filesystem.FileSize, palette.Temperature, "", "")
	cv := bubbletree.RenderToCanvas(&nodes, root, 400, 300, inks)

	out := filepath.Join(t.TempDir(), "bubble.jpg")
	g.Expect(cv.Render(out)).To(Succeed())

	f, err := os.Open(out)
	g.Expect(err).NotTo(HaveOccurred())

	defer f.Close()

	_, format, err := image.DecodeConfig(f)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(format).To(Equal("jpeg"))
}

func TestRenderBubbleToCanvas_EmptyDirectory(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{Path: "empty"}
	nodes := bubbletree.Layout(root, 800, 800, filesystem.FileSize, bubbletree.LabelFoldersOnly)
	inks := bubbletree.BuildInks(root, "", "", "", "")

	cv := bubbletree.RenderToCanvas(&nodes, root, 800, 800, inks)

	out := filepath.Join(t.TempDir(), "empty.png")
	g.Expect(cv.Render(out)).To(Succeed())

	info, err := os.Stat(out)
	g.Expect(err).NotTo(HaveOccurred())

	if info != nil {
		g.Expect(info.Size()).To(BeNumerically(">", 0))
	}
}

func TestRenderBubbleToCanvas_LabelsAll(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := testRoot()
	nodes := bubbletree.Layout(root, 1000, 800, filesystem.FileSize, bubbletree.LabelAll)
	inks := bubbletree.BuildInks(root, filesystem.FileSize, palette.Temperature, "", "")
	cv := bubbletree.RenderToCanvas(&nodes, root, 1000, 800, inks)

	out := filepath.Join(t.TempDir(), "labels-all.svg")
	g.Expect(cv.Render(out)).To(Succeed())

	data, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())

	// LabelAll should emit <text> elements for file labels in the SVG.
	g.Expect(bytes.Contains(data, []byte("<text"))).To(BeTrue(),
		"expected SVG to contain at least one <text> element with LabelAll")
}

func TestRenderBubbleToCanvas_LabelsNone(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := testRoot()
	nodes := bubbletree.Layout(root, 1000, 800, filesystem.FileSize, bubbletree.LabelNone)
	inks := bubbletree.BuildInks(root, filesystem.FileSize, palette.Temperature, "", "")
	cv := bubbletree.RenderToCanvas(&nodes, root, 1000, 800, inks)

	out := filepath.Join(t.TempDir(), "labels-none.svg")
	g.Expect(cv.Render(out)).To(Succeed())

	data, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())

	// LabelNone should emit no <text> or <textPath> elements.
	g.Expect(bytes.Contains(data, []byte("<text"))).To(BeFalse(),
		"expected SVG to contain no <text> elements with LabelNone")
}

func TestRenderBubbleToCanvas_CategoricalFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := testRoot()
	nodes := bubbletree.Layout(root, 1000, 800, filesystem.FileSize, bubbletree.LabelFoldersOnly)
	inks := bubbletree.BuildInks(
		root,
		filesystem.FileType, palette.Categorization,
		filesystem.FileSize, palette.Temperature,
	)
	cv := bubbletree.RenderToCanvas(&nodes, root, 1000, 800, inks)

	out := filepath.Join(t.TempDir(), "cat.png")
	g.Expect(cv.Render(out)).To(Succeed())

	f, err := os.Open(out)
	g.Expect(err).NotTo(HaveOccurred())

	defer f.Close()

	_, format, err := image.DecodeConfig(f)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(format).To(Equal("png"))
}
