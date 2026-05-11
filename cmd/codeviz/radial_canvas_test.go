package main

import (
	"bytes"
	"cmp"
	"encoding/xml"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"slices"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/radialtree"
)

func radialTestFile(name, ext string, size int64) *model.File {
	f := &model.File{Name: name, Extension: ext}
	f.SetQuantity(filesystem.FileSize, size)
	f.SetClassification(filesystem.FileType, ext)

	return f
}

func TestBuildRadialInks_NumericFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			radialTestFile("a.go", "go", 100),
			radialTestFile("b.go", "go", 200),
		},
	}

	inks := buildRadialInks(
		root, filesystem.FileSize, palette.Temperature, "", "",
	)

	g.Expect(inks.fill.Info().Kind).To(Equal(canvas.InkNumeric))
	g.Expect(inks.border.Info().Kind).To(Equal(canvas.InkFixed))
}

func TestBuildRadialInks_CategoricalFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			radialTestFile("a.go", "go", 100),
			radialTestFile("b.rs", "rs", 200),
		},
	}

	inks := buildRadialInks(
		root, filesystem.FileType, palette.Categorization, "", "",
	)

	g.Expect(inks.fill.Info().Kind).To(Equal(canvas.InkCategorical))
}

func TestBuildRadialInks_WithBorder(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			radialTestFile("a.go", "go", 100),
			radialTestFile("b.rs", "rs", 200),
		},
	}

	inks := buildRadialInks(
		root,
		filesystem.FileSize, palette.Temperature,
		filesystem.FileSize, palette.Temperature,
	)

	g.Expect(inks.fill.Info().Kind).To(Equal(canvas.InkNumeric))
	g.Expect(inks.border.Info().Kind).NotTo(Equal(canvas.InkFixed))
}

func radialTestRoot() *model.Directory {
	return &model.Directory{
		Name: "flat",
		Files: []*model.File{
			radialTestFile("small.txt", "txt", 5),
			radialTestFile("medium.go", "go", 100),
			radialTestFile("large.rs", "rs", 1000),
		},
	}
}

func TestRenderRadialToCanvas_PNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := radialTestRoot()
	nodes := radialtree.Layout(root, 800, filesystem.FileSize, radialtree.LabelNone)
	inks := buildRadialInks(root, filesystem.FileSize, palette.Temperature, "", "")
	cv := renderRadialToCanvas(&nodes, root, 800, inks)

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

func TestRenderRadialToCanvas_SVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := radialTestRoot()
	nodes := radialtree.Layout(root, 400, filesystem.FileSize, radialtree.LabelNone)
	inks := buildRadialInks(root, filesystem.FileSize, palette.Temperature, "", "")
	cv := renderRadialToCanvas(&nodes, root, 400, inks)

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

func TestRenderRadialToCanvas_NestedDirs(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "project",
		Files: []*model.File{
			radialTestFile("readme.md", "md", 50),
		},
		Dirs: []*model.Directory{
			{
				Name: "src",
				Files: []*model.File{
					radialTestFile("main.go", "go", 200),
					radialTestFile("util.go", "go", 80),
				},
			},
		},
	}

	nodes := radialtree.Layout(root, 800, filesystem.FileSize, radialtree.LabelAll)
	inks := buildRadialInks(root, filesystem.FileSize, palette.Temperature, "", "")
	cv := renderRadialToCanvas(&nodes, root, 800, inks)

	out := filepath.Join(t.TempDir(), "nested.png")
	err := cv.Render(out)
	g.Expect(err).NotTo(HaveOccurred())

	info, err := os.Stat(out)
	g.Expect(err).NotTo(HaveOccurred())

	if info != nil {
		g.Expect(info.Size()).To(BeNumerically(">", 0))
	}
}

func TestRenderRadialToCanvas_EmptyDir(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{Name: "empty"}

	nodes := radialtree.Layout(root, 400, filesystem.FileSize, radialtree.LabelNone)
	inks := buildRadialInks(root, filesystem.FileSize, palette.Temperature, "", "")
	cv := renderRadialToCanvas(&nodes, root, 400, inks)

	out := filepath.Join(t.TempDir(), "empty.png")
	err := cv.Render(out)
	g.Expect(err).NotTo(HaveOccurred())

	info, err := os.Stat(out)
	g.Expect(err).NotTo(HaveOccurred())

	if info != nil {
		g.Expect(info.Size()).To(BeNumerically(">", 0))
	}
}

func TestCollectRadialDiscs_SortOrder(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			radialTestFile("tiny.go", "go", 10),
			radialTestFile("huge.go", "go", 10000),
			radialTestFile("mid.go", "go", 500),
		},
	}

	nodes := radialtree.Layout(root, 800, filesystem.FileSize, radialtree.LabelNone)

	cx := float64(800) / 2.0
	cy := float64(800) / 2.0
	entries := collectRadialDiscs(&nodes, root, cx, cy)

	g.Expect(len(entries)).To(BeNumerically(">=", 2))

	// Sort largest-first, mirroring addRadialDiscs production code
	slices.SortFunc(entries, func(a, b radialDiscEntry) int {
		return cmp.Compare(b.node.DiscRadius, a.node.DiscRadius)
	})

	// Verify entries are sorted largest-first by disc radius
	for i := range len(entries) - 1 {
		g.Expect(entries[i].node.DiscRadius).To(
			BeNumerically(">=", entries[i+1].node.DiscRadius),
			"entries should be sorted largest disc first",
		)
	}

	// Verify at least two distinct radii exist, proving the metric drives sizing
	radii := make(map[float64]struct{}, len(entries))
	for _, e := range entries {
		radii[e.node.DiscRadius] = struct{}{}
	}

	g.Expect(len(radii)).To(BeNumerically(">=", 2),
		"expected at least 2 distinct disc radii to confirm metric drives sizing",
	)
}

func TestRenderRadialToCanvas_DirBorderUsesFixedInk(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "project",
		Files: []*model.File{
			radialTestFile("main.go", "go", 200),
		},
		Dirs: []*model.Directory{
			{
				Name: "src",
				Files: []*model.File{
					radialTestFile("lib.go", "go", 300),
				},
			},
		},
	}

	// Build inks with a border metric configured.
	inks := buildRadialInks(
		root,
		filesystem.FileSize, palette.Temperature,
		filesystem.FileSize, palette.Temperature,
	)

	// Precondition: the border ink must be metric-driven, not fixed.
	g.Expect(inks.border.Info().Kind).NotTo(Equal(canvas.InkFixed),
		"precondition: border ink should be metric-driven when a border metric is configured")

	nodes := radialtree.Layout(root, 800, filesystem.FileSize, radialtree.LabelAll)

	cx := float64(800) / 2.0
	cy := cx
	entries := collectRadialDiscs(&nodes, root, cx, cy)

	// Find a directory entry and a file entry.
	var dirEntry *radialDiscEntry

	var fileEntry *radialDiscEntry

	for i := range entries {
		if entries[i].isDir && dirEntry == nil {
			dirEntry = &entries[i]
		}

		if !entries[i].isDir && entries[i].file != nil && fileEntry == nil {
			fileEntry = &entries[i]
		}
	}

	g.Expect(dirEntry).NotTo(BeNil(), "should have at least one directory disc")
	g.Expect(fileEntry).NotTo(BeNil(), "should have at least one file disc")

	// Directory border must resolve to radialDefaultBorder (fixed ink),
	// not the metric ink's lowest bucket.
	dirBorderInk := canvas.FixedInk(radialDefaultBorder)
	g.Expect(dirBorderInk.Dip(canvas.MetricValue{})).To(Equal(radialDefaultBorder),
		"directory disc border should resolve to radialDefaultBorder")

	// File border should follow the metric ink.
	if fileEntry != nil && fileEntry.file != nil {
		fileMV := metricValueForFile(fileEntry.file, inks.border)
		fileBorderColour := inks.border.Dip(fileMV)
		g.Expect(fileBorderColour).NotTo(Equal(radialDefaultBorder),
			"file disc border should follow the metric ink, not the fixed default")
	}
}
