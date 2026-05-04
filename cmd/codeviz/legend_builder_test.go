package main

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/provider/filesystem"
	"github.com/bevan/code-visualizer/internal/render"
)

func TestResolveLegendOptions_EmptyDefaults(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pos, orient := resolveLegendOptions("", "")
	g.Expect(pos).To(Equal(render.LegendPositionBottomRight))
	g.Expect(orient).To(Equal(render.LegendOrientationVertical))
}

func TestResolveLegendOptions_ExplicitValues(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pos, orient := resolveLegendOptions("top-left", "horizontal")
	g.Expect(pos).To(Equal(render.LegendPositionTopLeft))
	g.Expect(orient).To(Equal(render.LegendOrientationHorizontal))
}

func TestResolveLegendOptions_PositionOnly_DerivesOrientation(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pos, orient := resolveLegendOptions("top-center", "")
	g.Expect(pos).To(Equal(render.LegendPositionTopCenter))
	g.Expect(orient).To(Equal(render.LegendOrientationHorizontal))
}

func TestResolveLegendOptions_None_DisablesLegend(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pos, _ := resolveLegendOptions("none", "")
	g.Expect(pos).To(Equal(render.LegendPositionNone))
}

func TestBuildLegendInfo_NonePosition_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := makeLegendTestRoot()
	info := buildLegendInfo(
		render.LegendPositionNone, render.LegendOrientationVertical,
		"file-size", "temperature",
		"", "",
		"file-lines", root,
	)

	g.Expect(info).To(BeNil())
}

func TestBuildLegendInfo_FillOnly_SingleEntry(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := makeLegendTestRoot()
	info := buildLegendInfo(
		render.LegendPositionBottomRight, render.LegendOrientationVertical,
		"file-size", "temperature",
		"", "",
		"file-size", root,
	)

	if info == nil {
		t.Fatal("expected non-nil LegendInfo")
	} else {
		g.Expect(info.Entries).To(HaveLen(1))
		g.Expect(info.Entries[0].Role()).To(Equal("Fill"))
		g.Expect(info.Entries[0].MetricName()).To(Equal("file-size"))
	}
}

func TestBuildLegendInfo_FillAndBorder_TwoEntries(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := makeLegendTestRoot()
	info := buildLegendInfo(
		render.LegendPositionBottomRight, render.LegendOrientationVertical,
		"file-size", "temperature",
		"file-type", "categorization",
		"file-size", root,
	)

	if info == nil {
		t.Fatal("expected non-nil LegendInfo")
	} else {
		g.Expect(info.Entries).To(HaveLen(2))
		g.Expect(info.Entries[0].Role()).To(Equal("Fill"))
		g.Expect(info.Entries[1].Role()).To(Equal("Border"))
		g.Expect(info.Entries[1].MetricName()).To(Equal("file-type"))
	}
}

func TestBuildLegendInfo_DifferentSizeMetric_AddsEntry(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := makeLegendTestRoot()
	info := buildLegendInfo(
		render.LegendPositionBottomRight, render.LegendOrientationVertical,
		"file-size", "temperature",
		"", "",
		"file-lines", root,
	)

	if info == nil {
		t.Fatal("expected non-nil LegendInfo")
	} else {
		g.Expect(info.Entries).To(HaveLen(2))
		g.Expect(info.Entries[1].Role()).To(Equal("Size"))
		g.Expect(info.Entries[1].MetricName()).To(Equal("file-lines"))
	}
}

func TestBuildLegendInfo_SameSizeAsFill_NoSizeEntry(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := makeLegendTestRoot()
	info := buildLegendInfo(
		render.LegendPositionBottomRight, render.LegendOrientationVertical,
		"file-size", "temperature",
		"", "",
		"file-size", root,
	)

	if info == nil {
		t.Fatal("expected non-nil LegendInfo")
	} else {
		g.Expect(info.Entries).To(HaveLen(1))
		g.Expect(info.Entries[0].Role()).To(Equal("Fill"))
		g.Expect(info.Entries[0].MetricName()).To(Equal("file-size"))
	}
}

func TestBuildLegendInfo_Classification_HasCategories(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := makeLegendTestRoot()
	info := buildLegendInfo(
		render.LegendPositionBottomRight, render.LegendOrientationVertical,
		"file-type", "categorization",
		"", "",
		"file-size", root,
	)

	if info == nil {
		t.Fatal("expected non-nil LegendInfo")
	} else {
		g.Expect(info.Entries).To(HaveLen(2))
		g.Expect(info.Entries[0].Kind()).To(Equal(metric.Classification))
		g.Expect(info.Entries[0].Categories()).NotTo(BeEmpty())
	}
}

func TestBuildLegendInfo_NoMetrics_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := makeLegendTestRoot()
	info := buildLegendInfo(
		render.LegendPositionBottomRight, render.LegendOrientationVertical,
		"", "",
		"", "",
		"", root,
	)

	g.Expect(info).To(BeNil())
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
