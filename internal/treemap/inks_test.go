package treemap_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
	"github.com/theunrepentantgeek/code-visualizer/internal/treemap"
	"github.com/theunrepentantgeek/code-visualizer/internal/viz"
)

func colourEncoding(name metric.Name, pal palette.PaletteName) viz.ColourEncoding {
	return viz.ColourEncoding{Metric: name, Palette: pal}
}

func TestBuildTreemapInks_DefaultColours(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name:  "root",
		Files: []*model.File{makeTestFile("a.go", "go", 100)},
	}

	is := treemap.BuildInks(root, stages.RequestedMetrics{}, viz.NoColourEncoding, viz.NoColourEncoding)

	g.Expect(is.Fill.Info().Kind).To(Equal(inks.KindFixed))
	g.Expect(is.Border.Info().Kind).To(Equal(inks.KindFixed))
}

func TestBuildTreemapInks_NumericFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			makeTestFile("a.go", "go", 100),
			makeTestFile("b.go", "go", 200),
		},
	}

	is := treemap.BuildInks(
		root,
		stages.RequestedMetrics{},
		colourEncoding(filesystem.FileSize, palette.Temperature),
		viz.NoColourEncoding,
	)

	g.Expect(is.Fill.Info().Kind).To(Equal(inks.KindNumeric))
	g.Expect(is.Border.Info().Kind).To(Equal(inks.KindFixed))
}

func TestBuildTreemapInks_CategoricalFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			makeTestFile("a.go", "go", 100),
			makeTestFile("b.rs", "rs", 200),
		},
	}

	is := treemap.BuildInks(
		root,
		stages.RequestedMetrics{},
		colourEncoding(filesystem.FileType, palette.Categorization),
		viz.NoColourEncoding,
	)

	g.Expect(is.Fill.Info().Kind).To(Equal(inks.KindCategorical))
	g.Expect(is.Border.Info().Kind).To(Equal(inks.KindFixed))
}

func TestBuildTreemapInks_BorderMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			makeTestFile("a.go", "go", 100),
			makeTestFile("b.rs", "rs", 200),
		},
	}

	is := treemap.BuildInks(
		root, stages.RequestedMetrics{},
		colourEncoding(filesystem.FileSize, palette.Temperature),
		colourEncoding(filesystem.FileType, palette.Categorization),
	)

	g.Expect(is.Fill.Info().Kind).To(Equal(inks.KindNumeric))
	g.Expect(is.Border.Info().Kind).To(Equal(inks.KindCategorical))
}

func TestBuildTreemapInks_NumericBorder(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			makeTestFile("small.go", "go", 50),
			makeTestFile("large.go", "go", 5000),
		},
	}

	is := treemap.BuildInks(
		root, stages.RequestedMetrics{},
		colourEncoding(filesystem.FileSize, palette.Temperature),
		colourEncoding(filesystem.FileSize, palette.Temperature),
	)

	g.Expect(is.Fill.Info().Kind).To(Equal(inks.KindNumeric))
	g.Expect(is.Border.Info().Kind).To(Equal(inks.KindNumeric))
}
