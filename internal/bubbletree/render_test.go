package bubbletree_test

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/bubbletree"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	canvasmodel "github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
	"github.com/theunrepentantgeek/code-visualizer/internal/viz"
)

type capturedDisc struct {
	geometry    geometry.Circle
	borderWidth float64
	radius      float64
}

type capturedArcText struct {
	text   string
	radius float64
	font   float64
}

type captureBackend struct {
	discs    []capturedDisc
	arcTexts []capturedArcText
}

func (*captureBackend) DrawRectangle(geometry.Rect, canvasmodel.Fill, canvasmodel.Fill, float64) {}

func (c *captureBackend) DrawDisc(
	circle geometry.Circle,
	_ canvasmodel.Fill,
	_ canvasmodel.Fill,
	borderWidth float64,
) {
	c.discs = append(c.discs, capturedDisc{geometry: circle, borderWidth: borderWidth, radius: circle.Radius})
}

func (*captureBackend) DrawPolygon([]geometry.Point, canvasmodel.Fill, canvasmodel.Fill, float64) {}

func (*captureBackend) DrawFilledPath([][]geometry.Point, color.RGBA) {}

func (*captureBackend) DrawLine(geometry.Point, geometry.Point, color.RGBA, float64) {}

func (*captureBackend) DrawPath([]geometry.Point, color.RGBA, float64) {}

func (*captureBackend) DrawText(geometry.Point, string, color.RGBA, float64, canvas.TextAnchor, float64) {
}

func (c *captureBackend) DrawArcText(
	_ geometry.Point,
	radius float64,
	text string,
	_ color.RGBA,
	fontSize float64,
) {
	c.arcTexts = append(c.arcTexts, capturedArcText{text: text, radius: radius, font: fontSize})
}

func (*captureBackend) DrawTooltip(geometry.Rect, string) {}

func (*captureBackend) DrawPolygonTooltip([]geometry.Point, geometry.Point, string) {}

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

func TestRenderBubbleToCanvas_PNG(t *testing.T) { //nolint:dupl // similar to SVG/JPG variants
	t.Parallel()
	g := NewGomegaWithT(t)

	root := testRoot()
	nodes := bubbletree.Layout(root, 1000, 800, filesystem.FileSize, bubbletree.LabelFoldersOnly)
	inks := bubbletree.BuildInks(
		root, stages.RequestedMetrics{},
		bubbleEncoding(filesystem.FileSize, palette.Temperature),
		viz.NoColourEncoding,
	)
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
	inks := bubbletree.BuildInks(
		root, stages.RequestedMetrics{},
		bubbleEncoding(filesystem.FileSize, palette.Temperature),
		viz.NoColourEncoding,
	)
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

func TestRenderBubbleToCanvas_JPG(t *testing.T) { //nolint:dupl // similar to PNG/SVG variants
	t.Parallel()
	g := NewGomegaWithT(t)

	root := testRoot()
	nodes := bubbletree.Layout(root, 400, 300, filesystem.FileSize, bubbletree.LabelFoldersOnly)
	inks := bubbletree.BuildInks(
		root, stages.RequestedMetrics{},
		bubbleEncoding(filesystem.FileSize, palette.Temperature),
		viz.NoColourEncoding,
	)
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
	inks := bubbletree.BuildInks(root, stages.RequestedMetrics{}, viz.NoColourEncoding, viz.NoColourEncoding)

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
	inks := bubbletree.BuildInks(
		root, stages.RequestedMetrics{},
		bubbleEncoding(filesystem.FileSize, palette.Temperature),
		viz.NoColourEncoding,
	)
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
	inks := bubbletree.BuildInks(
		root, stages.RequestedMetrics{},
		bubbleEncoding(filesystem.FileSize, palette.Temperature),
		viz.NoColourEncoding,
	)
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
		root, stages.RequestedMetrics{},
		bubbleEncoding(filesystem.FileType, palette.Categorization),
		bubbleEncoding(filesystem.FileSize, palette.Temperature),
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
	inks := bubbletree.BuildInks(
		root, stages.RequestedMetrics{},
		bubbleEncoding(filesystem.FileSize, palette.Temperature),
		viz.NoColourEncoding,
	)
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

	g.Expect(maxDiscRadius).To(BeNumerically("~", pkgNode.Geometry.Radius-bubbletree.LabelReservation, 0.001))

	var pkgLabelRadius float64

	for _, arcText := range backend.arcTexts {
		if arcText.text == "pkg" {
			pkgLabelRadius = arcText.radius

			break
		}
	}

	g.Expect(pkgLabelRadius).To(
		BeNumerically(">", maxDiscRadius),
		"directory label should sit above the bubble edge, not on top of it",
	)
	g.Expect(pkgLabelRadius).To(
		BeNumerically(">", pkgNode.Geometry.Radius),
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
	inks := bubbletree.BuildInks(
		root, stages.RequestedMetrics{},
		bubbleEncoding(filesystem.FileSize, palette.Temperature),
		viz.NoColourEncoding,
	)
	nodes := bubbletree.Layout(root, 800, 600, filesystem.FileSize, bubbletree.LabelFoldersOnly)
	cv := bubbletree.RenderToCanvas(&nodes, root, 800, 600, inks)

	backend := &captureBackend{}
	g.Expect(cv.RenderTo(backend)).To(Succeed())

	g.Expect(backend.discs).NotTo(BeEmpty())
	g.Expect(backend.discs[0].radius).To(
		BeNumerically(">", 0),
		"empty labelled directory should still render as a visible bubble",
	)

	var childLabelRadius float64

	for _, arcText := range backend.arcTexts {
		if arcText.text == "a" {
			childLabelRadius = arcText.radius

			break
		}
	}

	g.Expect(childLabelRadius).To(
		BeNumerically(">", 0),
		"empty labelled directory should still render its label",
	)
	g.Expect(childLabelRadius).To(
		BeNumerically(">", backend.discs[0].radius),
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
	inks := bubbletree.BuildInks(
		root, stages.RequestedMetrics{},
		bubbleEncoding(filesystem.FileSize, palette.Temperature),
		viz.NoColourEncoding,
	)
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
	minX := int(math.Floor(dirNode.Geometry.Center.X - dirNode.Geometry.Radius/2))
	maxX := int(math.Ceil(dirNode.Geometry.Center.X + dirNode.Geometry.Radius/2))
	minY := int(math.Floor(dirNode.Geometry.Center.Y - dirNode.Geometry.Radius))
	maxY := int(math.Ceil(dirNode.Geometry.Center.Y - dirNode.Geometry.Radius + bubbletree.LabelReservation))

	g.Expect(hasNonWhitePixelInRect(img, minX, minY, maxX, maxY)).To(
		BeTrue(),
		"expected raster output to place the directory label in the reserved band above the bubble",
	)
}

func TestRenderBubbleToCanvas_DenseFileBordersDoNotOverlap(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{Name: "root"}

	for i := range 40 {
		f := makeFile(fmt.Sprintf("small-%d.go", i), "go", 1)
		f.Path = fmt.Sprintf("root/small-%d.go", i)
		root.Files = append(root.Files, f)
	}

	inks := bubbletree.BuildInks(
		root, stages.RequestedMetrics{},
		bubbleEncoding(filesystem.FileSize, palette.Temperature),
		bubbleEncoding(filesystem.FileSize, palette.Temperature),
	)
	nodes := bubbletree.Layout(root, 30, 20, filesystem.FileSize, bubbletree.LabelNone)
	backend := &captureBackend{}
	g.Expect(bubbletree.RenderToCanvas(&nodes, root, 30, 20, inks).RenderTo(backend)).To(Succeed())
	g.Expect(backend.discs).To(HaveLen(40))

	for i, first := range backend.discs {
		for _, second := range backend.discs[i+1:] {
			distance := first.geometry.Center.DistanceTo(second.geometry.Center)
			outer := first.radius + first.borderWidth/2 + second.radius + second.borderWidth/2
			g.Expect(distance).To(BeNumerically(">=", outer-1e-6),
				"rendered file outlines must not overlap")
		}
	}
}

func TestRenderBubbleToCanvas_ScaledDirectoryLabelFitsReservedBand(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := testRoot()
	inks := bubbletree.BuildInks(root, stages.RequestedMetrics{},
		bubbleEncoding(filesystem.FileSize, palette.Temperature), viz.NoColourEncoding)
	nodes := bubbletree.Layout(root, 80, 60, filesystem.FileSize, bubbletree.LabelFoldersOnly)
	backend := &captureBackend{}
	g.Expect(bubbletree.RenderToCanvas(&nodes, root, 80, 60, inks).RenderTo(backend)).To(Succeed())

	g.Expect(backend.arcTexts).NotTo(BeEmpty())

	dirs, _ := nodes.Index()
	pkg, ok := dirs["root/pkg"]

	if !ok {
		t.Fatal("expected pkg directory in the rendered tree")
	}

	for _, label := range backend.arcTexts {
		if label.text == "pkg" {
			g.Expect(label.radius-canvasmodel.ArcTextInset+label.font/2).To(
				BeNumerically("<=", pkg.Geometry.Radius+1e-6),
				"rendered label must fit its directory's reserved radius",
			)

			return
		}
	}

	t.Fatal("expected a rendered pkg label")
}

func TestRenderBubbleToCanvas_OmitsDirectoryLabelBelowMinimumScaledFontSize(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	file := makeFile("large.go", "go", 10000)
	file.Path = "root/pkg/large.go"
	root := &model.Directory{
		Name: "root",
		Path: "root",
		Dirs: []*model.Directory{{
			Name:  "pkg",
			Path:  "root/pkg",
			Files: []*model.File{file},
		}},
	}
	inks := bubbletree.BuildInks(root, stages.RequestedMetrics{},
		bubbleEncoding(filesystem.FileSize, palette.Temperature), viz.NoColourEncoding)
	nodes := bubbletree.Layout(root, 10, 10, filesystem.FileSize, bubbletree.LabelFoldersOnly)
	backend := &captureBackend{}
	g.Expect(bubbletree.RenderToCanvas(&nodes, root, 10, 10, inks).RenderTo(backend)).To(Succeed())

	g.Expect(nodes.Children).To(HaveLen(1))
	pkg := nodes.Children[0]
	g.Expect(pkg.ShowLabel).To(BeTrue())
	g.Expect(backend.discs).To(HaveLen(2))

	labelBand := pkg.Geometry.Radius - backend.discs[0].radius
	g.Expect(labelBand).To(BeNumerically(">", 0))
	g.Expect(labelBand).To(BeNumerically("<", 7),
		"scaled label band must be smaller than the minimum readable font size")
	g.Expect(backend.arcTexts).To(BeEmpty(),
		"directory labels below the minimum readable font size must be omitted")
}

func denseDirectoryRoot(size int64) *model.Directory {
	root := &model.Directory{Name: "root", Path: "root"}

	for i := range 6 {
		dir := &model.Directory{Name: fmt.Sprintf("group-%d", i), Path: fmt.Sprintf("root/group-%d", i)}
		for j := range 3 {
			name := fmt.Sprintf("pkg-%d-%d", i, j)
			sub := &model.Directory{Name: name, Path: dir.Path + "/" + name}
			file := makeFile("large.go", "go", size)
			file.Path = sub.Path + "/large.go"
			sub.Files = []*model.File{file}
			dir.Dirs = append(dir.Dirs, sub)
		}

		root.Dirs = append(root.Dirs, dir)
	}

	return root
}

func TestRenderBubbleToCanvas_DenseDirectoriesRetainReadableLabels(t *testing.T) {
	t.Parallel()

	for _, size := range []int64{1_000_000, 1_000_000_000_000} {
		t.Run(fmt.Sprintf("file-size-%d", size), func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)
			root := denseDirectoryRoot(size)
			inks := bubbletree.BuildInks(root, stages.RequestedMetrics{},
				bubbleEncoding(filesystem.FileSize, palette.Temperature), viz.NoColourEncoding)
			nodes := bubbletree.Layout(root, 1920, 1080, filesystem.FileSize, bubbletree.LabelFoldersOnly)
			backend := &captureBackend{}
			g.Expect(bubbletree.RenderToCanvas(&nodes, root, 1920, 1080, inks).RenderTo(backend)).To(Succeed())

			dirs, _ := nodes.Index()
			g.Expect(dirs).To(HaveLen(24))
			g.Expect(backend.arcTexts).To(HaveLen(24),
				"every directory in this layout has room for a readable label")

			dirsByLabel := make(map[string]*bubbletree.BubbleNode, len(dirs))
			for _, dir := range dirs {
				dirsByLabel[dir.Label] = dir
			}

			seenLabels := make(map[string]bool, len(dirs))

			for _, label := range backend.arcTexts {
				g.Expect(label.font).To(BeNumerically(">=", 7))

				dir := dirsByLabel[label.text]
				g.Expect(dir).NotTo(BeNil())
				g.Expect(label.radius-canvasmodel.ArcTextInset+label.font/2).To(
					BeNumerically("<=", dir.Geometry.Radius+1e-6),
					"label %q must fit its packed directory radius", label.text,
				)
				seenLabels[label.text] = true
			}

			g.Expect(seenLabels).To(HaveLen(24), "each directory must have its own label")
		})
	}
}

func TestRenderBubbleToCanvas_LargeTreeKeepsLabelsOnLargerDirectories(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	root := &model.Directory{Name: "root", Path: "root"}
	expected := make([]string, 0, 12)

	for i := range 12 {
		name := fmt.Sprintf("area-%02d", i)
		dir := &model.Directory{Name: name, Path: "root/" + name}
		expected = append(expected, name)

		for j := range 100 {
			name := fmt.Sprintf("tiny-%02d-%03d", i, j)
			sub := &model.Directory{Name: name, Path: dir.Path + "/" + name}
			file := makeFile("main.go", "go", 1_000_000)
			file.Path = sub.Path + "/main.go"
			sub.Files = []*model.File{file}
			dir.Dirs = append(dir.Dirs, sub)
		}

		root.Dirs = append(root.Dirs, dir)
	}

	inks := bubbletree.BuildInks(root, stages.RequestedMetrics{},
		bubbleEncoding(filesystem.FileSize, palette.Temperature), viz.NoColourEncoding)
	nodes := bubbletree.Layout(root, 1280, 720, filesystem.FileSize, bubbletree.LabelFoldersOnly)
	backend := &captureBackend{}
	g.Expect(bubbletree.RenderToCanvas(&nodes, root, 1280, 720, inks).RenderTo(backend)).To(Succeed())
	g.Expect(backend.discs).To(HaveLen(2412), "label allocation must retain all directories and files")

	labels := make([]string, 0, len(backend.arcTexts))
	for _, label := range backend.arcTexts {
		g.Expect(label.font).To(BeNumerically(">=", 7))
		labels = append(labels, label.text)
	}

	g.Expect(labels).To(ConsistOf(expected),
		"tiny directories must not prevent larger directories from having readable labels")
}

func deepDirectoryRoot(files []*model.File) *model.Directory {
	root := &model.Directory{Name: "root", Path: "root"}
	dir := root

	for i := range 80 {
		name := fmt.Sprintf("level-%02d", i)
		child := &model.Directory{Name: name, Path: dir.Path + "/" + name}
		dir.Dirs = []*model.Directory{child}
		dir = child
	}

	for _, file := range files {
		file.Path = dir.Path + "/" + file.Name
	}

	dir.Files = files

	return root
}

func TestRenderBubbleToCanvas_DeepTreeRetainsReadableAncestorLabels(t *testing.T) {
	t.Parallel()

	for _, files := range [][]*model.File{nil, {makeFile("main.go", "go", 1_000_000)}} {
		t.Run(fmt.Sprintf("file-count-%d", len(files)), func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)
			root := deepDirectoryRoot(files)
			inks := bubbletree.BuildInks(root, stages.RequestedMetrics{},
				bubbleEncoding(filesystem.FileSize, palette.Temperature), viz.NoColourEncoding)
			nodes := bubbletree.Layout(root, 800, 600, filesystem.FileSize, bubbletree.LabelFoldersOnly)
			backend := &captureBackend{}
			g.Expect(bubbletree.RenderToCanvas(&nodes, root, 800, 600, inks).RenderTo(backend)).To(Succeed())

			labels := make([]string, 0, len(backend.arcTexts))
			for _, label := range backend.arcTexts {
				g.Expect(label.font).To(BeNumerically(">=", 7))
				labels = append(labels, label.text)
			}

			g.Expect(labels).To(ContainElement("level-00"),
				"an unlabelable full hierarchy must still label its largest visible ancestor")
		})
	}
}

func TestRenderBubbleToCanvas_SVGOmitsRootLabel(t *testing.T) {
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
	inks := bubbletree.BuildInks(
		root, stages.RequestedMetrics{},
		bubbleEncoding(filesystem.FileSize, palette.Temperature),
		viz.NoColourEncoding,
	)
	nodes := bubbletree.Layout(root, 800, 600, filesystem.FileSize, bubbletree.LabelFoldersOnly)
	cv := bubbletree.RenderToCanvas(&nodes, root, 800, 600, inks)

	out := filepath.Join(t.TempDir(), "root-label.svg")
	g.Expect(cv.Render(out)).To(Succeed())

	data, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(bytes.Contains(data, []byte(">root</textPath>"))).To(
		BeFalse(),
		"root directory disc is not rendered, so its label should be omitted",
	)
}
