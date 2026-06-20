package treemap

import (
	"image/color"
	"testing"

	. "github.com/onsi/gomega"

	pkginks "github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
)

func TestBuildBlockLabels_IncludesOnlyConfiguredMetricLines(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	file := &model.File{Name: "alpha.go", Extension: "go"}
	file.SetQuantity(filesystem.FileSize, 128)
	file.SetClassification(filesystem.FileType, "go")
	file.SetQuantity(filesystem.FileLines, 12)

	root := &model.Directory{Name: "root", Files: []*model.File{file}}
	rects := TreemapRectangle{
		Label:       "root",
		IsDirectory: true,
		Children: []TreemapRectangle{{
			X:     10,
			Y:     20,
			W:     120,
			H:     60,
			Label: "alpha.go",
		}},
	}

	labels := buildBlockLabels(rects, root, pkginks.FixedInk(color.RGBA{R: 255, G: 255, B: 255, A: 255}), LabelMetrics{
		Size:   filesystem.FileSize,
		Fill:   filesystem.FileType,
		Border: filesystem.FileLines,
	})
	g.Expect(labels).To(HaveLen(1))

	if len(labels) == 0 {
		return
	}

	g.Expect(labels[0].Lines).To(Equal([]string{"alpha.go", "128", "go", "12"}))
}

func TestBuildBlockLabels_OmitsUnconfiguredMetrics(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	file := &model.File{Name: "beta.go", Extension: "go"}
	file.SetQuantity(filesystem.FileSize, 64)
	file.SetClassification(filesystem.FileType, "go")

	root := &model.Directory{Name: "root", Files: []*model.File{file}}
	rects := TreemapRectangle{
		Label:       "root",
		IsDirectory: true,
		Children: []TreemapRectangle{{
			X:     0,
			Y:     0,
			W:     100,
			H:     40,
			Label: "beta.go",
		}},
	}

	labels := buildBlockLabels(
		rects,
		root,
		pkginks.FixedInk(color.RGBA{A: 255}),
		LabelMetrics{Size: filesystem.FileSize},
	)
	g.Expect(labels).To(HaveLen(1))

	if len(labels) == 0 {
		return
	}

	g.Expect(labels[0].Lines).To(Equal([]string{"beta.go", "64"}))
}
