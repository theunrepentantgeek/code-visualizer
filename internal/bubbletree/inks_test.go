package bubbletree_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/bubbletree"
	pkginks "github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func TestMain(m *testing.M) {
	filesystem.Register()
	m.Run()
}

func makeFile(name, ext string, size int64) *model.File {
	f := &model.File{Name: name, Extension: ext}
	f.SetQuantity(filesystem.FileSize, size)
	f.SetClassification(filesystem.FileType, ext)

	return f
}

func TestBuildInks_DefaultColours(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name:  "root",
		Files: []*model.File{makeFile("a.go", "go", 100)},
	}

	inks := bubbletree.BuildInks(root, stages.RequestedMetrics{}, "", "", "", "")

	g.Expect(inks.Fill.Info().Kind).To(Equal(pkginks.KindFixed))
	g.Expect(inks.Border.Info().Kind).To(Equal(pkginks.KindFixed))
}

func TestBuildInks_NumericFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			makeFile("a.go", "go", 100),
			makeFile("b.go", "go", 200),
		},
	}

	inks := bubbletree.BuildInks(root, stages.RequestedMetrics{}, filesystem.FileSize, palette.Temperature, "", "")

	g.Expect(inks.Fill.Info().Kind).To(Equal(pkginks.KindNumeric))
	g.Expect(inks.Border.Info().Kind).To(Equal(pkginks.KindFixed))
}

func TestBuildInks_BorderMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			makeFile("a.go", "go", 100),
			makeFile("b.rs", "rs", 200),
		},
	}

	inks := bubbletree.BuildInks(
		root,
		stages.RequestedMetrics{},
		filesystem.FileSize, palette.Temperature,
		filesystem.FileType, palette.Categorization,
	)

	g.Expect(inks.Fill.Info().Kind).To(Equal(pkginks.KindNumeric))
	g.Expect(inks.Border.Info().Kind).To(Equal(pkginks.KindCategorical))
}
