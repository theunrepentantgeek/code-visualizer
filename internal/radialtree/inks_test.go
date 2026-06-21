package radialtree_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/radialtree"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func makeRadialFile(name, ext string, size int64) *model.File {
	f := &model.File{Name: name, Extension: ext}
	f.SetQuantity(filesystem.FileSize, size)
	f.SetClassification(filesystem.FileType, ext)

	return f
}

func TestBuildRadialInks_DefaultColours(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name:  "root",
		Files: []*model.File{makeRadialFile("a.go", "go", 100)},
	}

	is := radialtree.BuildInks(root, stages.RequestedMetrics{}, "", "", "", "")

	g.Expect(is.Fill.Info().Kind).To(Equal(inks.KindFixed))
	g.Expect(is.Border.Info().Kind).To(Equal(inks.KindFixed))
}

func TestBuildRadialInks_NumericFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			makeRadialFile("a.go", "go", 100),
			makeRadialFile("b.go", "go", 200),
		},
	}

	is := radialtree.BuildInks(root, stages.RequestedMetrics{}, filesystem.FileSize, palette.Temperature, "", "")

	g.Expect(is.Fill.Info().Kind).To(Equal(inks.KindNumeric))
	g.Expect(is.Border.Info().Kind).To(Equal(inks.KindFixed))
}

func TestBuildRadialInks_CategoricalFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			makeRadialFile("a.go", "go", 100),
			makeRadialFile("b.rs", "rs", 200),
		},
	}

	is := radialtree.BuildInks(root, stages.RequestedMetrics{}, filesystem.FileType, palette.Categorization, "", "")

	g.Expect(is.Fill.Info().Kind).To(Equal(inks.KindCategorical))
	g.Expect(is.Border.Info().Kind).To(Equal(inks.KindFixed))
}

func TestBuildRadialInks_BorderMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			makeRadialFile("a.go", "go", 100),
			makeRadialFile("b.rs", "rs", 200),
		},
	}

	is := radialtree.BuildInks(
		root, stages.RequestedMetrics{},
		filesystem.FileSize, palette.Temperature,
		filesystem.FileType, palette.Categorization,
	)

	g.Expect(is.Fill.Info().Kind).To(Equal(inks.KindNumeric))
	g.Expect(is.Border.Info().Kind).To(Equal(inks.KindCategorical))
}

func TestBuildRadialInks_NumericBorder(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			makeRadialFile("small.go", "go", 50),
			makeRadialFile("large.go", "go", 5000),
		},
	}

	is := radialtree.BuildInks(
		root, stages.RequestedMetrics{},
		filesystem.FileSize, palette.Temperature,
		filesystem.FileSize, palette.Temperature,
	)

	g.Expect(is.Fill.Info().Kind).To(Equal(inks.KindNumeric))
	g.Expect(is.Border.Info().Kind).To(Equal(inks.KindNumeric))
}
