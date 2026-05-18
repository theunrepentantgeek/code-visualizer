package treemap_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/treemap"
)

func TestBuildTreemapInks_DefaultColours(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name:  "root",
		Files: []*model.File{makeTestFile("a.go", "go", 100)},
	}

	inks := treemap.BuildInks(root, "", "", "", "")

	g.Expect(inks.Fill.Info().Kind).To(Equal(canvas.InkFixed))
	g.Expect(inks.Border.Info().Kind).To(Equal(canvas.InkFixed))
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

	inks := treemap.BuildInks(root, filesystem.FileSize, palette.Temperature, "", "")

	g.Expect(inks.Fill.Info().Kind).To(Equal(canvas.InkNumeric))
	g.Expect(inks.Border.Info().Kind).To(Equal(canvas.InkFixed))
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

	inks := treemap.BuildInks(root, filesystem.FileType, palette.Categorization, "", "")

	g.Expect(inks.Fill.Info().Kind).To(Equal(canvas.InkCategorical))
	g.Expect(inks.Border.Info().Kind).To(Equal(canvas.InkFixed))
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

	inks := treemap.BuildInks(
		root,
		filesystem.FileSize, palette.Temperature,
		filesystem.FileType, palette.Categorization,
	)

	g.Expect(inks.Fill.Info().Kind).To(Equal(canvas.InkNumeric))
	g.Expect(inks.Border.Info().Kind).To(Equal(canvas.InkCategorical))
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

	inks := treemap.BuildInks(
		root,
		filesystem.FileSize, palette.Temperature,
		filesystem.FileSize, palette.Temperature,
	)

	g.Expect(inks.Fill.Info().Kind).To(Equal(canvas.InkNumeric))
	g.Expect(inks.Border.Info().Kind).To(Equal(canvas.InkNumeric))
}
