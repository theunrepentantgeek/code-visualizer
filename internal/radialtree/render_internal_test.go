package radialtree

import (
	"cmp"
	"slices"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	pkginks "github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
)

func makeTestFile(name, ext string, size int64) *model.File {
	f := &model.File{Name: name, Extension: ext}
	f.SetQuantity(filesystem.FileSize, size)
	f.SetClassification(filesystem.FileType, ext)

	return f
}

func TestCollectRadialDiscs_SortOrder(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			makeTestFile("tiny.go", "go", 10),
			makeTestFile("huge.rs", "rs", 10000),
			makeTestFile("mid.txt", "txt", 500),
		},
	}

	nodes := Layout(root, 800, filesystem.FileSize, LabelNone)

	cx := float64(800) / 2.0
	cy := float64(800) / 2.0
	entries := collectRadialDiscs(&nodes, root, cx, cy)

	g.Expect(len(entries)).To(BeNumerically(">=", 2))

	slices.SortFunc(entries, func(a, b radialDiscEntry) int {
		return cmp.Compare(b.node.DiscRadius, a.node.DiscRadius)
	})

	for i := range len(entries) - 1 {
		g.Expect(entries[i].node.DiscRadius).To(
			BeNumerically(">=", entries[i+1].node.DiscRadius),
			"entries should be sorted largest disc first",
		)
	}

	radii := make(map[float64]struct{}, len(entries))
	for _, e := range entries {
		radii[e.node.DiscRadius] = struct{}{}
	}

	g.Expect(len(radii)).To(
		BeNumerically(">=", 2),
		"expected at least 2 distinct disc radii to confirm metric drives sizing",
	)
}

func TestRenderRadialToCanvas_DirBorderUsesFixedInk(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "project",
		Files: []*model.File{
			makeTestFile("main.go", "go", 200),
		},
		Dirs: []*model.Directory{
			{
				Name: "src",
				Files: []*model.File{
					makeTestFile("lib.rs", "rs", 300),
				},
			},
		},
	}

	inks := BuildInks(
		root,
		filesystem.FileSize, palette.Temperature,
		filesystem.FileSize, palette.Temperature,
	)

	g.Expect(inks.Border.Info().Kind).NotTo(Equal(canvas.InkFixed),
		"precondition: border ink should be metric-driven when a border metric is configured")

	nodes := Layout(root, 800, filesystem.FileSize, LabelAll)

	cx := float64(800) / 2.0
	cy := cx
	entries := collectRadialDiscs(&nodes, root, cx, cy)

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

	dirBorderInk := canvas.FixedInk(radialDefaultBorder)
	g.Expect(dirBorderInk.Dip(canvas.MetricValue{})).To(Equal(radialDefaultBorder),
		"directory disc border should resolve to radialDefaultBorder")

	if fileEntry != nil && fileEntry.file != nil {
		fileMV := pkginks.MetricValueForFile(fileEntry.file, inks.Border)
		fileBorderColour := inks.Border.Dip(fileMV)
		g.Expect(fileBorderColour).NotTo(Equal(radialDefaultBorder),
			"file disc border should follow the metric ink, not the fixed default")
	}
}
