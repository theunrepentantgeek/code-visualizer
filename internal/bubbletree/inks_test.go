package bubbletree_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/bubbletree"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
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

	is := bubbletree.BuildInks(root, stages.RequestedMetrics{}, "", "", "", "")

	g.Expect(is.Fill.Info().Kind).To(Equal(inks.KindFixed))
	g.Expect(is.Border.Info().Kind).To(Equal(inks.KindFixed))
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

	is := bubbletree.BuildInks(root, stages.RequestedMetrics{}, filesystem.FileSize, palette.Temperature, "", "")

	g.Expect(is.Fill.Info().Kind).To(Equal(inks.KindNumeric))
	g.Expect(is.Border.Info().Kind).To(Equal(inks.KindFixed))
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

	is := bubbletree.BuildInks(
		root,
		stages.RequestedMetrics{},
		filesystem.FileSize, palette.Temperature,
		filesystem.FileType, palette.Categorization,
	)

	g.Expect(is.Fill.Info().Kind).To(Equal(inks.KindNumeric))
	g.Expect(is.Border.Info().Kind).To(Equal(inks.KindCategorical))
}
