package main

import (
	"image/color"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/bevan/code-visualizer/internal/canvas"
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/palette"
	"github.com/bevan/code-visualizer/internal/provider/filesystem"
)

func TestResolveLegendOptions_EmptyDefaults(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pos, orient := resolveLegendOptions("", "")
	g.Expect(pos).To(Equal(canvas.LegendPositionBottomRight))
	g.Expect(orient).To(Equal(canvas.LegendOrientationVertical))
}

func TestResolveLegendOptions_ExplicitValues(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pos, orient := resolveLegendOptions("top-left", "horizontal")
	g.Expect(pos).To(Equal(canvas.LegendPositionTopLeft))
	g.Expect(orient).To(Equal(canvas.LegendOrientationHorizontal))
}

func TestResolveLegendOptions_PositionOnly_DerivesOrientation(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pos, orient := resolveLegendOptions("top-center", "")
	g.Expect(pos).To(Equal(canvas.LegendPositionTopCenter))
	g.Expect(orient).To(Equal(canvas.LegendOrientationHorizontal))
}

func TestResolveLegendOptions_None_DisablesLegend(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pos, _ := resolveLegendOptions("none", "")
	g.Expect(pos).To(Equal(canvas.LegendPositionNone))
}

func TestBuildLegendConfig_NonePosition_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := buildLegendConfig(
		canvas.LegendPositionNone, canvas.LegendOrientationVertical,
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

	cfg := buildLegendConfig(
		canvas.LegendPositionBottomRight, canvas.LegendOrientationVertical,
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

	cfg := buildLegendConfig(
		canvas.LegendPositionBottomRight, canvas.LegendOrientationVertical,
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

	cfg := buildLegendConfig(
		canvas.LegendPositionBottomRight, canvas.LegendOrientationVertical,
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

	cfg := buildLegendConfig(
		canvas.LegendPositionBottomRight, canvas.LegendOrientationVertical,
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

	cfg := buildLegendConfig(
		canvas.LegendPositionBottomRight, canvas.LegendOrientationVertical,
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
