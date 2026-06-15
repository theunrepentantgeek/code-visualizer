package inks_test

import (
	"image/color"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
)

func TestMain(m *testing.M) {
	filesystem.Register()
	m.Run()
}

func fallbackColour() color.RGBA {
	return color.RGBA{R: 0xCC, G: 0xCC, B: 0xCC, A: 0xFF}
}

func makeFile(name, ext string, size int64) *model.File {
	f := &model.File{Path: "/p/" + name, Name: name, Extension: ext}
	f.SetQuantity(filesystem.FileSize, size)
	f.SetClassification(filesystem.FileType, ext)

	return f
}

func descriptorFor(t *testing.T, name metric.Name) provider.BaseMetricDescriptor {
	t.Helper()

	d, ok := provider.GetBase(name)
	if !ok {
		t.Fatalf("descriptor not found for %q", name)
	}

	return d
}

func TestCollectDistinctTypes_ReturnsSortedTypes(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Path: "/p", Name: "p",
		Files: []*model.File{
			makeFile("z.go", "go", 1),
			makeFile("a.md", "md", 1),
			makeFile("m.txt", "txt", 1),
		},
	}

	g.Expect(inks.CollectDistinctTypes(root, filesystem.FileType)).
		To(Equal([]string{"go", "md", "txt"}))
}

func TestCollectNumericValues_WalksAllFiles(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Path: "/p",
		Files: []*model.File{
			makeFile("a.go", "go", 10),
			makeFile("b.go", "go", 20),
		},
		Dirs: []*model.Directory{
			{Path: "/p/sub", Files: []*model.File{makeFile("c.go", "go", 30)}},
		},
	}

	g.Expect(inks.CollectNumericValues(root, filesystem.FileSize)).
		To(ConsistOf(10.0, 20.0, 30.0))
}

func TestBuildMetricInk_NumericKind(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Path: "/p",
		Files: []*model.File{
			makeFile("a.go", "go", 100),
			makeFile("b.go", "go", 200),
		},
	}

	ink := inks.BuildMetricInk(root, descriptorFor(t, filesystem.FileSize), palette.Temperature, fallbackColour())

	g.Expect(ink.Info().Kind).To(Equal(canvas.InkNumeric))
}

func TestBuildMetricInk_CategoricalKind(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Path: "/p",
		Files: []*model.File{
			makeFile("a.go", "go", 100),
			makeFile("b.rs", "rs", 200),
		},
	}

	ink := inks.BuildMetricInk(root, descriptorFor(t, filesystem.FileType), palette.Categorization, fallbackColour())

	g.Expect(ink.Info().Kind).To(Equal(canvas.InkCategorical))
}

func TestBuildMetricInk_UnknownMetricFallsBackToFixed(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{Path: "/p"}

	ink := inks.BuildMetricInk(root, provider.BaseMetricDescriptor{}, palette.Temperature, fallbackColour())

	g.Expect(ink.Info().Kind).To(Equal(canvas.InkFixed))
}

func TestBuildMetricInk_EmptyNumericFallsBackToFixed(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Numeric descriptor exists, but the tree has no files → no values.
	root := &model.Directory{Path: "/p"}

	ink := inks.BuildMetricInk(root, descriptorFor(t, filesystem.FileSize), palette.Temperature, fallbackColour())

	g.Expect(ink.Info().Kind).To(Equal(canvas.InkFixed))
}

func TestMetricValueForFile_NumericInk(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	file := makeFile("a.go", "go", 42)
	root := &model.Directory{Path: "/p", Files: []*model.File{file}}
	ink := inks.BuildMetricInk(root, descriptorFor(t, filesystem.FileSize), palette.Temperature, fallbackColour())

	mv := inks.MetricValueForFile(file, ink)

	g.Expect(mv.Kind).To(Equal(metric.Quantity))
	g.Expect(mv.Quantity).To(Equal(42))
}

func TestMetricValueForFile_CategoricalInk(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	file := makeFile("a.go", "go", 1)
	root := &model.Directory{Path: "/p", Files: []*model.File{file}}
	ink := inks.BuildMetricInk(root, descriptorFor(t, filesystem.FileType), palette.Categorization, fallbackColour())

	mv := inks.MetricValueForFile(file, ink)

	g.Expect(mv.Kind).To(Equal(metric.Classification))
	g.Expect(mv.Category).To(Equal("go"))
}

func TestMetricValueForFile_NilFile(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(inks.MetricValueForFile(nil, canvas.FixedInk(fallbackColour()))).
		To(Equal(canvas.MetricValue{}))
}

func TestMetricValueForFile_FixedInk(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	file := makeFile("a.go", "go", 1)

	g.Expect(inks.MetricValueForFile(file, canvas.FixedInk(fallbackColour()))).
		To(Equal(canvas.MetricValue{}))
}
