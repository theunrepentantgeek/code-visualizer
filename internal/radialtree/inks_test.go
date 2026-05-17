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

	inks := radialtree.BuildInks(
		root, filesystem.FileSize, palette.Temperature, "", "",
	)

	g.Expect(inks.Fill.Info().Kind).To(Equal(canvas.InkNumeric))
	g.Expect(inks.Border.Info().Kind).To(Equal(canvas.InkFixed))
}

func TestBuildInks_CategoricalFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			makeFile("a.go", "go", 100),
			makeFile("b.rs", "rs", 200),
		},
	}

	inks := radialtree.BuildInks(
		root, filesystem.FileType, palette.Categorization, "", "",
	)

	g.Expect(inks.Fill.Info().Kind).To(Equal(canvas.InkCategorical))
}

func TestBuildInks_WithBorder(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			makeFile("a.go", "go", 100),
			makeFile("b.rs", "rs", 200),
		},
	}

	inks := radialtree.BuildInks(
		root,
		filesystem.FileSize, palette.Temperature,
		filesystem.FileSize, palette.Temperature,
	)

	g.Expect(inks.Fill.Info().Kind).To(Equal(canvas.InkNumeric))
	g.Expect(inks.Border.Info().Kind).NotTo(Equal(canvas.InkFixed))
}
