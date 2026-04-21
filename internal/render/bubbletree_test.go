package render

import (
	"bytes"
	"encoding/xml"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/sebdah/goldie/v2"

	"github.com/bevan/code-visualizer/internal/bubbletree"
)

// sampleBubbleTree returns a small deterministic BubbleNode tree for render smoke tests.
// Positions are absolute pixel coordinates; we build directly rather than going through Layout.
func sampleBubbleTree() bubbletree.BubbleNode {
	return bubbletree.BubbleNode{
		X: 0, Y: 0, Radius: 200,
		Label: "root", ShowLabel: true, IsDirectory: true,
		FillColour: color.RGBA{R: 200, G: 200, B: 255, A: 40},
		Children: []bubbletree.BubbleNode{
			{
				X: -60, Y: 0, Radius: 80,
				Label: "src", ShowLabel: true, IsDirectory: true,
				FillColour: color.RGBA{R: 180, G: 220, B: 255, A: 40},
				Children: []bubbletree.BubbleNode{
					{
						X: -80, Y: -20, Radius: 20,
						Label: "main.go", ShowLabel: false, IsDirectory: false,
						FillColour: color.RGBA{R: 100, G: 200, B: 100, A: 255},
					},
					{
						X: -40, Y: 20, Radius: 15,
						Label: "utils.go", ShowLabel: false, IsDirectory: false,
						FillColour: color.RGBA{R: 200, G: 100, B: 100, A: 255},
					},
				},
			},
			{
				X: 60, Y: 0, Radius: 30,
				Label: "README.md", ShowLabel: false, IsDirectory: false,
				FillColour: color.RGBA{R: 150, G: 150, B: 200, A: 255},
			},
		},
	}
}

func TestRenderBubble_PNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := sampleBubbleTree()
	out := filepath.Join(t.TempDir(), "bubble.png")

	err := RenderBubble(&root, 800, 600, out, nil)
	g.Expect(err).NotTo(HaveOccurred())

	f, err := os.Open(out)
	g.Expect(err).NotTo(HaveOccurred())

	defer f.Close()

	_, format, err := image.DecodeConfig(f)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(format).To(Equal("png"))
}

func TestRenderBubble_JPG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := sampleBubbleTree()
	out := filepath.Join(t.TempDir(), "bubble.jpg")

	err := RenderBubble(&root, 800, 600, out, nil)
	g.Expect(err).NotTo(HaveOccurred())

	f, err := os.Open(out)
	g.Expect(err).NotTo(HaveOccurred())

	defer f.Close()

	_, format, err := image.DecodeConfig(f)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(format).To(Equal("jpeg"))
}

func TestRenderBubble_SVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := sampleBubbleTree()
	out := filepath.Join(t.TempDir(), "bubble.svg")

	err := RenderBubble(&root, 800, 600, out, nil)
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

func TestRenderBubble_GoldenFile(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := sampleBubbleTree()
	out := filepath.Join(t.TempDir(), "bubble-golden.png")

	err := RenderBubble(&root, 800, 600, out, nil)
	g.Expect(err).NotTo(HaveOccurred())

	actual, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())

	gld := goldie.New(t, goldie.WithFixtureDir("testdata"), goldie.WithNameSuffix(".png"))
	gld.Assert(t, "bubble-tree", actual)
}
