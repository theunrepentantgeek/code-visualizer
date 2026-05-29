package radialtree_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/radialtree"
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

	inks := radialtree.BuildInks(root, "", "", "", "")

	g.Expect(inks.Fill.Info().Kind).To(Equal(canvas.InkFixed))
	g.Expect(inks.Border.Info().Kind).To(Equal(canvas.InkFixed))
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

	inks := radialtree.BuildInks(root, filesystem.FileSize, palette.Temperature, "", "")

	g.Expect(inks.Fill.Info().Kind).To(Equal(canvas.InkNumeric))
	g.Expect(inks.Border.Info().Kind).To(Equal(canvas.InkFixed))
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

	inks := radialtree.BuildInks(root, filesystem.FileType, palette.Categorization, "", "")

	g.Expect(inks.Fill.Info().Kind).To(Equal(canvas.InkCategorical))
	g.Expect(inks.Border.Info().Kind).To(Equal(canvas.InkFixed))
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

	inks := radialtree.BuildInks(
		root,
		filesystem.FileSize, palette.Temperature,
		filesystem.FileType, palette.Categorization,
	)

	g.Expect(inks.Fill.Info().Kind).To(Equal(canvas.InkNumeric))
	g.Expect(inks.Border.Info().Kind).To(Equal(canvas.InkCategorical))
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

	inks := radialtree.BuildInks(
		root,
		filesystem.FileSize, palette.Temperature,
		filesystem.FileSize, palette.Temperature,
	)

	g.Expect(inks.Fill.Info().Kind).To(Equal(canvas.InkNumeric))
	g.Expect(inks.Border.Info().Kind).To(Equal(canvas.InkNumeric))
}
