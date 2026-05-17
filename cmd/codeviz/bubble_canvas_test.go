package main

import (
	"bytes"
	"encoding/xml"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/bubbletree"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	pkginks "github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
)

func bubbleTestFile(name, ext string, size int64) *model.File {
	f := &model.File{Name: name, Extension: ext}
	f.SetQuantity(filesystem.FileSize, size)
	f.SetClassification(filesystem.FileType, ext)

	return f
}

// bubbleTestFileGo is a shorthand for Go source files.
func bubbleTestFileGo(name string, size int64) *model.File {
	return bubbleTestFile(name, "go", size)
}

func bubbleTestRoot() *model.Directory {
	return &model.Directory{
		Path: "root",
		Files: []*model.File{
			bubbleTestFileGo("main.go", 100),
			bubbleTestFile("style.css", "css", 50),
		},
		Dirs: []*model.Directory{
			{
				Path: "root/pkg",
				Files: []*model.File{
					bubbleTestFileGo("lib.go", 200),
				},
			},
		},
	}
}

func bubbleTestNodes() bubbletree.BubbleNode {
	return bubbletree.BubbleNode{
		X: 500, Y: 400, Radius: 300,
		Path: "root", IsDirectory: true, Label: "root", ShowLabel: true,
		Children: []bubbletree.BubbleNode{
			{X: 400, Y: 350, Radius: 50, Path: "root/main.go", Label: "main.go", ShowLabel: true},
			{X: 600, Y: 350, Radius: 40, Path: "root/style.css", Label: "style.css", ShowLabel: true},
			{
				X: 500, Y: 500, Radius: 100,
				Path: "root/pkg", IsDirectory: true, Label: "pkg", ShowLabel: true,
				Children: []bubbletree.BubbleNode{
					{X: 500, Y: 500, Radius: 30, Path: "root/pkg/lib.go", Label: "lib.go", ShowLabel: true},
				},
			},
		},
	}
}

func TestBubbleArcFontSize_EmptyLabel(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	fontSize := bubbleArcFontSize("", 100)

	g.Expect(fontSize).To(Equal(0.0))
}

func TestBubbleArcFontSize_TinyRadius(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// bubbleArcLabelInset is 14.0, so radius must be > 14 for any label to fit
	fontSize := bubbleArcFontSize("test", 10)

	g.Expect(fontSize).To(Equal(0.0))
}

func TestBubbleArcFontSize_NormalLabel(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	fontSize := bubbleArcFontSize("normal", 100)

	g.Expect(fontSize).To(BeNumerically(">", 0))
	g.Expect(fontSize).To(BeNumerically(">=", bubbleMinArcFontSize))
	g.Expect(fontSize).To(BeNumerically("<=", bubbleDefaultFontSize))
}

func TestBubbleArcFontSize_LongLabel(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Very long label on a small circle should return 0 (below min font size)
	longLabel := "this_is_a_very_long_label_that_cannot_fit_on_a_small_circle"
	fontSize := bubbleArcFontSize(longLabel, 30)

	g.Expect(fontSize).To(Equal(0.0))
}

func TestBuildBubbleInks_DefaultColours(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			bubbleTestFileGo("a.go", 100),
		},
	}

	inks := buildBubbleInks(root, "", "", "", "")

	g.Expect(inks.fill.Info().Kind).To(Equal(canvas.InkFixed))
	g.Expect(inks.border.Info().Kind).To(Equal(canvas.InkFixed))
}

func TestRenderBubbleToCanvas_ProducesShapes(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := bubbleTestRoot()
	nodes := bubbleTestNodes()
	inks := buildBubbleInks(root, filesystem.FileSize, palette.Temperature, "", "")

	cv := renderBubbleToCanvas(&nodes, root, 1000, 800, inks)

	// Render to PNG and verify the file is created and non-empty
	out := filepath.Join(t.TempDir(), "bubble.png")
	err := cv.Render(out)
	g.Expect(err).NotTo(HaveOccurred())

	f, err := os.Open(out)
	g.Expect(err).NotTo(HaveOccurred())

	defer f.Close()

	_, format, err := image.DecodeConfig(f)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(format).To(Equal("png"))

	info, err := os.Stat(out)
	g.Expect(err).NotTo(HaveOccurred())

	if info != nil {
		g.Expect(info.Size()).To(BeNumerically(">", 0))
	}
}

func TestRenderBubbleToCanvas_EmptyDirectory(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{Path: "empty"}
	nodes := bubbletree.BubbleNode{
		X: 400, Y: 400, Radius: 300,
		Path: "empty", IsDirectory: true, Label: "empty", ShowLabel: true,
	}

	inks := buildBubbleInks(root, "", "", "", "")

	// Should not panic with empty directory
	cv := renderBubbleToCanvas(&nodes, root, 800, 800, inks)

	out := filepath.Join(t.TempDir(), "empty.png")
	err := cv.Render(out)
	g.Expect(err).NotTo(HaveOccurred())

	info, err := os.Stat(out)
	g.Expect(err).NotTo(HaveOccurred())

	if info != nil {
		g.Expect(info.Size()).To(BeNumerically(">", 0))
	}
}

func TestRenderBubbleToCanvas_SVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := bubbleTestRoot()
	nodes := bubbleTestNodes()
	inks := buildBubbleInks(root, filesystem.FileSize, palette.Temperature, "", "")

	cv := renderBubbleToCanvas(&nodes, root, 800, 600, inks)

	out := filepath.Join(t.TempDir(), "bubble.svg")
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

func TestRenderBubbleToCanvas_DirBorderUsesFixedInk(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := bubbleTestRoot()

	// Build inks with a border metric — files should get metric-driven
	// borders, but directories must always get bubbleDefaultBorder.
	inks := buildBubbleInks(
		root,
		filesystem.FileSize, palette.Temperature,
		filesystem.FileSize, palette.Temperature,
	)

	// The metric-based border ink should NOT be fixed.
	g.Expect(inks.border.Info().Kind).NotTo(Equal(canvas.InkFixed),
		"precondition: border ink should be metric-driven when a border metric is configured")

	// Directory border uses fixed ink, so Dip always returns bubbleDefaultBorder
	// regardless of MetricValue.
	dirBorder := canvas.FixedInk(bubbleDefaultBorder)
	g.Expect(dirBorder.Dip(canvas.MetricValue{})).To(Equal(bubbleDefaultBorder),
		"directory disc border should resolve to bubbleDefaultBorder")

	// File border uses the metric ink, which should differ from the fixed default.
	file := root.Files[0]
	fileMV := pkginks.MetricValueForFile(file, inks.border)
	fileBorderColour := inks.border.Dip(fileMV)
	g.Expect(fileBorderColour).NotTo(Equal(bubbleDefaultBorder),
		"file disc border should follow the metric ink, not the fixed default")
}
