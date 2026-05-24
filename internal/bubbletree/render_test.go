package bubbletree_test

import (
	"bytes"
	"encoding/xml"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/bubbletree"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	canvasmodel "github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
)

type capturedDisc struct {
	radius float64
}

type capturedArcText struct {
	text   string
	radius float64
}

type captureBackend struct {
	discs    []capturedDisc
	arcTexts []capturedArcText
}

func (*captureBackend) DrawRectangle(canvas.Position, canvas.Size, canvasmodel.Fill, canvasmodel.Fill, float64) {
}

func (c *captureBackend) DrawDisc(
	_ canvas.Position,
	radius float64,
	_ canvasmodel.Fill,
	_ canvasmodel.Fill,
	_ float64,
) {
	c.discs = append(c.discs, capturedDisc{radius: radius})
}

func (*captureBackend) DrawLine(canvas.Position, canvas.Position, color.RGBA, float64) {}

func (*captureBackend) DrawPath([]canvas.Position, color.RGBA, float64) {}

func (*captureBackend) DrawText(canvas.Position, string, color.RGBA, float64, canvas.TextAnchor, float64) {
}

func (c *captureBackend) DrawArcText(
	_ canvas.Position,
	radius float64,
	text string,
	_ color.RGBA,
	_ float64,
) {
	c.arcTexts = append(c.arcTexts, capturedArcText{text: text, radius: radius})
}

func (*captureBackend) Finish(string) error { return nil }

func testRoot() *model.Directory {
	mainFile := makeFile("main.go", "go", 100)
	mainFile.Path = "root/main.go"

	styleFile := makeFile("style.css", "css", 50)
	styleFile.Path = "root/style.css"

	libFile := makeFile("lib.go", "go", 200)
	libFile.Path = "root/pkg/lib.go"

	return &model.Directory{
		Name:  "root",
		Path:  "root",
		Files: []*model.File{mainFile, styleFile},
		Dirs: []*model.Directory{
			{
				Name:  "pkg",
				Path:  "root/pkg",
				Files: []*model.File{libFile},
			},
		},
	}
}

func decodeImage(t *testing.T, path string) image.Image {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open image: %v", err)
	}

	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		t.Fatalf("decode image: %v", err)
	}

	return img
}

func hasNonWhitePixelInRect(img image.Image, minX, minY, maxX, maxY int) bool {
	bounds := img.Bounds()
	minX = max(minX, bounds.Min.X)
	minY = max(minY, bounds.Min.Y)
	maxX = min(maxX, bounds.Max.X)
	maxY = min(maxY, bounds.Max.Y)

	for y := minY; y < maxY; y++ {
		for x := minX; x < maxX; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			if a != 0xffff || r != 0xffff || g != 0xffff || b != 0xffff {
				return true
			}
		}
	}

	return false
}

func mustParseFloat(t *testing.T, value string) float64 {
	t.Helper()

	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		t.Fatalf("parse float %q: %v", value, err)
	}

	return f
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

func TestRenderBubbleToCanvas_DirectoryLabelsUseReservedBandOutsideBubble(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := testRoot()
	nodes := bubbletree.Layout(root, 1000, 800, filesystem.FileSize, bubbletree.LabelFoldersOnly)
	inks := bubbletree.BuildInks(root, filesystem.FileSize, palette.Temperature, "", "")
	cv := bubbletree.RenderToCanvas(&nodes, root, 1000, 800, inks)

	backend := &captureBackend{}
	g.Expect(cv.RenderTo(backend)).To(Succeed())

	g.Expect(nodes.Children).To(HaveLen(3))

	var pkgNode *bubbletree.BubbleNode

	for i := range nodes.Children {
		if nodes.Children[i].IsDirectory {
			pkgNode = &nodes.Children[i]

			break
		}
	}

	g.Expect(pkgNode).NotTo(BeNil())

	if pkgNode == nil {
		return
	}

	maxDiscRadius := 0.0
	for _, disc := range backend.discs {
		if disc.radius > maxDiscRadius {
			maxDiscRadius = disc.radius
		}
	}

	g.Expect(maxDiscRadius).To(BeNumerically("~", pkgNode.Radius-bubbletree.LabelReservation, 0.001))

	var pkgLabelRadius float64

	for _, arcText := range backend.arcTexts {
		if arcText.text == "pkg" {
			pkgLabelRadius = arcText.radius

			break
		}
	}

	g.Expect(pkgLabelRadius).To(BeNumerically(">", maxDiscRadius),
		"directory label should sit above the bubble edge, not on top of it",
	)
	g.Expect(pkgLabelRadius).To(BeNumerically(">", pkgNode.Radius),
		"rendered arc should use the reserved label band outside the bubble",
	)
}

func TestRenderBubbleToCanvas_EmptyLabelledDirectoryKeepsVisibleBubble(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Path: "root",
		Dirs: []*model.Directory{{
			Name: "a",
			Path: "root/a",
		}},
	}
	inks := bubbletree.BuildInks(root, filesystem.FileSize, palette.Temperature, "", "")
	nodes := bubbletree.Layout(root, 800, 600, filesystem.FileSize, bubbletree.LabelFoldersOnly)
	cv := bubbletree.RenderToCanvas(&nodes, root, 800, 600, inks)

	backend := &captureBackend{}
	g.Expect(cv.RenderTo(backend)).To(Succeed())

	g.Expect(backend.discs).NotTo(BeEmpty())
	g.Expect(backend.discs[0].radius).To(BeNumerically(">", 0),
		"empty labelled directory should still render as a visible bubble",
	)

	var childLabelRadius float64

	for _, arcText := range backend.arcTexts {
		if arcText.text == "a" {
			childLabelRadius = arcText.radius

			break
		}
	}

	g.Expect(childLabelRadius).To(BeNumerically(">", 0),
		"empty labelled directory should still render its label",
	)
	g.Expect(childLabelRadius).To(BeNumerically(">", backend.discs[0].radius),
		"empty labelled directory should reserve label space above its bubble",
	)
}

func TestRenderBubbleToCanvas_RasterPlacesDirectoryLabelInReservedBand(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Path: "root",
		Dirs: []*model.Directory{{
			Name: "pkg",
			Path: "root/pkg",
		}},
	}
	inks := bubbletree.BuildInks(root, filesystem.FileSize, palette.Temperature, "", "")
	nodes := bubbletree.Layout(root, 800, 600, filesystem.FileSize, bubbletree.LabelFoldersOnly)
	cv := bubbletree.RenderToCanvas(&nodes, root, 800, 600, inks)

	var dirNode *bubbletree.BubbleNode

	for i := range nodes.Children {
		if nodes.Children[i].IsDirectory {
			dirNode = &nodes.Children[i]

			break
		}
	}

	g.Expect(dirNode).NotTo(BeNil())

	if dirNode == nil {
		return
	}

	out := filepath.Join(t.TempDir(), "label-band.png")
	g.Expect(cv.Render(out)).To(Succeed())

	img := decodeImage(t, out)
	minX := int(math.Floor(dirNode.X - dirNode.Radius/2))
	maxX := int(math.Ceil(dirNode.X + dirNode.Radius/2))
	minY := int(math.Floor(dirNode.Y - dirNode.Radius))
	maxY := int(math.Ceil(dirNode.Y - dirNode.Radius + bubbletree.LabelReservation))

	g.Expect(hasNonWhitePixelInRect(img, minX, minY, maxX, maxY)).To(BeTrue(),
		"expected raster output to place the directory label in the reserved band above the bubble",
	)
}

func TestRenderBubbleToCanvas_SVGKeepsRootLabelPathWithinCanvas(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Path: "root",
		Files: []*model.File{
			makeFile("main.go", "go", 100),
			makeFile("style.css", "css", 50),
		},
	}
	inks := bubbletree.BuildInks(root, filesystem.FileSize, palette.Temperature, "", "")
	nodes := bubbletree.Layout(root, 800, 600, filesystem.FileSize, bubbletree.LabelFoldersOnly)
	cv := bubbletree.RenderToCanvas(&nodes, root, 800, 600, inks)

	out := filepath.Join(t.TempDir(), "root-label.svg")
	g.Expect(cv.Render(out)).To(Succeed())

	data, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(bytes.Contains(data, []byte(">root</textPath>"))).To(BeTrue())

	pathPattern := regexp.MustCompile(
		`<path id="[^"]+" d="M([0-9.\-]+),([0-9.\-]+) A[0-9.\-]+,[0-9.\-]+ 0 0,1 ([0-9.\-]+),([0-9.\-]+)"`,
	)
	match := pathPattern.FindStringSubmatch(string(data))
	g.Expect(match).To(HaveLen(5))

	startX := mustParseFloat(t, match[1])
	startY := mustParseFloat(t, match[2])
	endX := mustParseFloat(t, match[3])
	endY := mustParseFloat(t, match[4])

	g.Expect(startX).To(BeNumerically(">=", 0.0))
	g.Expect(startY).To(BeNumerically(">=", 0.0))
	g.Expect(endX).To(BeNumerically("<=", 800.0))
	g.Expect(endY).To(BeNumerically("<=", 600.0))
}
