package legend_test

import (
	"image/color"
	"slices"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	model0 "github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/walk"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
)

func TestResolveLegendOptions_EmptyDefaults(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pos, orient := legend.ResolveOptions("", "")
	g.Expect(pos).To(Equal(model0.LegendPositionBottomRight))
	g.Expect(orient).To(Equal(model0.LegendOrientationVertical))
}

func TestResolveLegendOptions_ExplicitValues(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pos, orient := legend.ResolveOptions("top-left", "horizontal")
	g.Expect(pos).To(Equal(model0.LegendPositionTopLeft))
	g.Expect(orient).To(Equal(model0.LegendOrientationHorizontal))
}

func TestResolveLegendOptions_PositionOnly_DerivesOrientation(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pos, orient := legend.ResolveOptions("top-center", "")
	g.Expect(pos).To(Equal(model0.LegendPositionTopCenter))
	g.Expect(orient).To(Equal(model0.LegendOrientationHorizontal))
}

func TestResolveLegendOptions_None_DisablesLegend(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pos, _ := legend.ResolveOptions("none", "")
	g.Expect(pos).To(Equal(model0.LegendPositionNone))
}

func TestBuildLegendConfig_NonePosition_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := legend.Build(
		model0.LegendPositionNone, model0.LegendOrientationVertical,
		canvas.FixedInk(color.RGBA{A: 255}), "file-size",
		canvas.FixedInk(color.RGBA{A: 255}), "",
		"file-lines",
	)

	g.Expect(cfg).To(BeNil())
}

func TestBuildLegendConfig_FillOnly_SingleEntry(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := makeLegendTestRoot()
	pal := palette.GetPalette(palette.Temperature)
	values := collectNumericValues(root, "file-size")
	fillInk := canvas.NumericInk("file-size", values, pal)

	cfg := legend.Build(
		model0.LegendPositionBottomRight, model0.LegendOrientationVertical,
		fillInk, "file-size",
		canvas.FixedInk(color.RGBA{A: 255}), "",
		"file-size",
	)

	if cfg == nil {
		t.Fatal("expected non-nil LegendConfig")
	} else {
		g.Expect(cfg.Entries).To(HaveLen(1))
		g.Expect(cfg.Entries[0].Role).To(Equal(canvas.LegendRoleFill))
		g.Expect(cfg.Entries[0].MetricName).To(Equal("file-size"))
	}
}

func TestBuildLegendConfig_FillAndBorder_TwoEntries(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := makeLegendTestRoot()
	pal := palette.GetPalette(palette.Temperature)
	values := collectNumericValues(root, "file-size")
	fillInk := canvas.NumericInk("file-size", values, pal)

	types := collectDistinctTypes(root, "file-type")
	catPal := palette.GetPalette(palette.Categorization)
	borderInk := canvas.CategoricalInk("file-type", types, catPal)

	cfg := legend.Build(
		model0.LegendPositionBottomRight, model0.LegendOrientationVertical,
		fillInk, "file-size",
		borderInk, "file-type",
		"file-size",
	)

	if cfg == nil {
		t.Fatal("expected non-nil LegendConfig")
	} else {
		g.Expect(cfg.Entries).To(HaveLen(2))
		g.Expect(cfg.Entries[0].Role).To(Equal(canvas.LegendRoleFill))
		g.Expect(cfg.Entries[1].Role).To(Equal(canvas.LegendRoleBorder))
	}
}

func TestBuildLegendConfig_DifferentSizeMetric_AddsEntry(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := makeLegendTestRoot()
	pal := palette.GetPalette(palette.Temperature)
	values := collectNumericValues(root, "file-size")
	fillInk := canvas.NumericInk("file-size", values, pal)

	cfg := legend.Build(
		model0.LegendPositionBottomRight, model0.LegendOrientationVertical,
		fillInk, "file-size",
		canvas.FixedInk(color.RGBA{A: 255}), "",
		"file-lines",
	)

	if cfg == nil {
		t.Fatal("expected non-nil LegendConfig")
	} else {
		g.Expect(cfg.Entries).To(HaveLen(2))
		g.Expect(cfg.Entries[1].Role).To(Equal(canvas.LegendRoleSize))
		g.Expect(cfg.Entries[1].MetricName).To(Equal("file-lines"))
	}
}

func TestBuildLegendConfig_SameSizeAsFill_NoSizeEntry(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := makeLegendTestRoot()
	pal := palette.GetPalette(palette.Temperature)
	values := collectNumericValues(root, "file-size")
	fillInk := canvas.NumericInk("file-size", values, pal)

	cfg := legend.Build(
		model0.LegendPositionBottomRight, model0.LegendOrientationVertical,
		fillInk, "file-size",
		canvas.FixedInk(color.RGBA{A: 255}), "",
		"file-size",
	)

	if cfg == nil {
		t.Fatal("expected non-nil LegendConfig")
	} else {
		g.Expect(cfg.Entries).To(HaveLen(1))
	}
}

func TestBuildLegendConfig_NoMetrics_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := legend.Build(
		model0.LegendPositionBottomRight, model0.LegendOrientationVertical,
		canvas.FixedInk(color.RGBA{A: 255}), "",
		canvas.FixedInk(color.RGBA{A: 255}), "",
		"",
	)

	g.Expect(cfg).To(BeNil())
}

func makeLegendTestRoot() *model.Directory {
	f1 := &model.File{Name: "main.go", Extension: "go"}
	f1.SetQuantity(filesystem.FileSize, 500)
	f1.SetQuantity(filesystem.FileLines, 50)
	f1.SetClassification(filesystem.FileType, "go")

	f2 := &model.File{Name: "lib.rs", Extension: "rs"}
	f2.SetQuantity(filesystem.FileSize, 1000)
	f2.SetQuantity(filesystem.FileLines, 100)
	f2.SetClassification(filesystem.FileType, "rs")

	f3 := &model.File{Name: "app.py", Extension: "py"}
	f3.SetQuantity(filesystem.FileSize, 200)
	f3.SetQuantity(filesystem.FileLines, 20)
	f3.SetClassification(filesystem.FileType, "py")

	return &model.Directory{
		Name:  "root",
		Files: []*model.File{f1, f2, f3},
	}
}

func extractNumeric(f *model.File, m metric.Name) float64 {
	if v, ok := f.Measure(m); ok {
		return v
	}

	return 0
}

//nolint:unparam // m kept for symmetry with collectDistinctTypes
func collectNumericValues(root *model.Directory, m metric.Name) []float64 {
	var values []float64

	walk.Files(root, func(f *model.File) {
		values = append(values, extractNumeric(f, m))
	})

	return values
}

func collectDistinctTypes(root *model.Directory, m metric.Name) []string {
	seen := map[string]bool{}

	walk.Files(root, func(f *model.File) {
		if v, ok := f.Classification(m); ok {
			seen[v] = true
		}
	})

	types := make([]string, 0, len(seen))
	for t := range seen {
		types = append(types, t)
	}

	slices.Sort(types)

	return types
}
