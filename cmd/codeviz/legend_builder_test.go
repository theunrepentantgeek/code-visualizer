package main

import (
	"image/color"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/palette"
	"github.com/bevan/code-visualizer/internal/provider/filesystem"
	"github.com/bevan/code-visualizer/internal/render"
)

func TestBuildNumericLegendRow_Quantity(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := metric.ComputeBuckets([]float64{10, 50, 100, 200, 500}, 3)
	numBuckets := len(buckets.Boundaries) + 1
	pal := palette.GetPalette(palette.Temperature)

	row := render.BuildNumericLegendRow("file-size", metric.Quantity, buckets, numBuckets, pal)

	g.Expect(row.MetricName).To(Equal("file-size"))
	g.Expect(row.Kind).To(Equal(metric.Quantity))
	g.Expect(row.Colours).To(HaveLen(numBuckets))
	g.Expect(row.Breakpoints).To(Equal(buckets.Boundaries))

	// All colours must be fully opaque.
	for _, c := range row.Colours {
		g.Expect(c.A).To(Equal(uint8(0xFF)))
	}
}

func TestBuildNumericLegendRow_Measure(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := metric.ComputeBuckets([]float64{0.1, 0.3, 0.5, 0.7, 0.9}, 5)
	numBuckets := len(buckets.Boundaries) + 1
	pal := palette.GetPalette(palette.GoodBad)

	row := render.BuildNumericLegendRow("freshness", metric.Measure, buckets, numBuckets, pal)

	g.Expect(row.MetricName).To(Equal("freshness"))
	g.Expect(row.Kind).To(Equal(metric.Measure))
	g.Expect(row.Colours).To(HaveLen(numBuckets))
	g.Expect(row.Breakpoints).To(Equal(buckets.Boundaries))
	// Categories must be empty for numeric metrics.
	g.Expect(row.Categories).To(BeEmpty())
}

func TestBuildNumericLegendRow_SingleBucket(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// All same value produces zero boundaries.
	buckets := metric.ComputeBuckets([]float64{42, 42, 42}, 3)
	numBuckets := len(buckets.Boundaries) + 1
	pal := palette.GetPalette(palette.Neutral)

	row := render.BuildNumericLegendRow("uniform", metric.Quantity, buckets, numBuckets, pal)

	g.Expect(row.Colours).To(HaveLen(numBuckets))
	g.Expect(row.Breakpoints).To(BeEmpty())
}

func TestBuildNumericLegendRow_DifferentPalettes(t *testing.T) {
	t.Parallel()

	buckets := metric.ComputeBuckets([]float64{1, 2, 3, 4, 5}, 3)
	numBuckets := len(buckets.Boundaries) + 1

	for _, name := range []palette.PaletteName{
		palette.Temperature,
		palette.Neutral,
		palette.GoodBad,
		palette.Foliage,
	} {
		t.Run(string(name), func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			pal := palette.GetPalette(name)
			row := render.BuildNumericLegendRow("m", metric.Quantity, buckets, numBuckets, pal)

			g.Expect(row.Colours).To(HaveLen(numBuckets))

			// Adjacent bucket colours should differ for ordered palettes.
			if len(row.Colours) >= 2 {
				first := row.Colours[0]
				last := row.Colours[len(row.Colours)-1]
				g.Expect(first).NotTo(Equal(last),
					"first and last bucket colours should differ for palette %s", name)
			}
		})
	}
}

func TestBuildCategoricalLegendRow(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cats := []string{".go", ".rs", ".py", ".js"}
	pal := palette.GetPalette(palette.Categorization)

	row := render.BuildCategoricalLegendRow("file-type", cats, pal)

	g.Expect(row.MetricName).To(Equal("file-type"))
	g.Expect(row.Kind).To(Equal(metric.Classification))
	g.Expect(row.Colours).To(HaveLen(4))
	g.Expect(row.Categories).To(Equal(cats))
	g.Expect(row.Breakpoints).To(BeEmpty())
}

func TestBuildCategoricalLegendRow_SingleCategory(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cats := []string{"go"}
	pal := palette.GetPalette(palette.Categorization)

	row := render.BuildCategoricalLegendRow("single", cats, pal)

	g.Expect(row.Colours).To(HaveLen(1))
	g.Expect(row.Categories).To(Equal([]string{"go"}))
}

func TestBuildCategoricalLegendRow_ManyCategories(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cats := make([]string, 20)
	for i := range cats {
		cats[i] = "type-" + string(rune('a'+i%26))
	}

	pal := palette.GetPalette(palette.Categorization)

	row := render.BuildCategoricalLegendRow("many-types", cats, pal)

	g.Expect(row.Colours).To(HaveLen(20))
	g.Expect(row.Categories).To(HaveLen(20))
}

func TestBuildCategoricalLegendRow_ColoursMatchMapper(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cats := []string{"a", "b", "c"}
	pal := palette.GetPalette(palette.Categorization)

	row := render.BuildCategoricalLegendRow("mapped", cats, pal)

	// The colours in the row should match what CategoricalMapper produces.
	mapper := palette.NewCategoricalMapper(cats, pal)
	for i, cat := range cats {
		g.Expect(row.Colours[i]).To(Equal(mapper.Map(cat)))
	}
}

func TestBuildLegendRow_EmptyMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{Name: "empty"}

	result := buildLegendRow(root, "", palette.Temperature)
	g.Expect(result).To(BeNil())
}

func TestBuildLegendRow_UnknownMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{Name: "root"}

	result := buildLegendRow(root, "nonexistent-metric", palette.Temperature)
	g.Expect(result).To(BeNil())
}

func TestBuildLegendRow_QuantityMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f1 := &model.File{Name: "a.go", Extension: "go"}
	f1.SetQuantity(filesystem.FileSize, 100)

	f2 := &model.File{Name: "b.go", Extension: "go"}
	f2.SetQuantity(filesystem.FileSize, 500)

	root := &model.Directory{
		Name:  "root",
		Files: []*model.File{f1, f2},
	}

	row := buildLegendRow(root, filesystem.FileSize, palette.Temperature)
	g.Expect(row).NotTo(BeNil())

	if row == nil {
		return
	}

	g.Expect(row.MetricName).To(Equal(string(filesystem.FileSize)))
	g.Expect(row.Kind).To(Equal(metric.Quantity))
	g.Expect(row.Colours).NotTo(BeEmpty())
}

func TestBuildLegendRow_ClassificationMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f1 := &model.File{Name: "a.go", Extension: "go"}
	f1.SetClassification(filesystem.FileType, "go")

	f2 := &model.File{Name: "b.rs", Extension: "rs"}
	f2.SetClassification(filesystem.FileType, "rs")

	root := &model.Directory{
		Name:  "root",
		Files: []*model.File{f1, f2},
	}

	row := buildLegendRow(root, filesystem.FileType, palette.Categorization)
	g.Expect(row).NotTo(BeNil())

	if row == nil {
		return
	}

	g.Expect(row.MetricName).To(Equal(string(filesystem.FileType)))
	g.Expect(row.Kind).To(Equal(metric.Classification))
	g.Expect(row.Categories).To(HaveLen(2))
}

func TestBuildLegendRow_NoFilesReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{Name: "empty"}

	row := buildLegendRow(root, filesystem.FileSize, palette.Temperature)
	g.Expect(row).To(BeNil())
}

func TestBuildLegendInfo_NoLegendTrue(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	noLegend := true
	row := &render.LegendRow{
		MetricName: "test",
		Kind:       metric.Quantity,
		Colours:    []color.RGBA{{A: 0xFF}},
	}

	info := buildLegendInfo(&noLegend, row)
	g.Expect(info).To(BeNil())
}

func TestBuildLegendInfo_NoLegendFalse(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	noLegend := false
	row := &render.LegendRow{
		MetricName: "test",
		Kind:       metric.Quantity,
		Colours:    []color.RGBA{{A: 0xFF}},
	}

	info := buildLegendInfo(&noLegend, row)
	g.Expect(info).NotTo(BeNil())

	if info == nil {
		return
	}

	g.Expect(info.Rows).To(HaveLen(1))
}

func TestBuildLegendInfo_NilFlag(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	row := &render.LegendRow{
		MetricName: "test",
		Kind:       metric.Quantity,
		Colours:    []color.RGBA{{A: 0xFF}},
	}

	info := buildLegendInfo(nil, row)
	g.Expect(info).NotTo(BeNil())

	if info == nil {
		return
	}

	g.Expect(info.Rows).To(HaveLen(1))
}

func TestBuildLegendInfo_AllNilRows(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	info := buildLegendInfo(nil, nil, nil, nil)
	g.Expect(info).To(BeNil())
}

func TestBuildLegendInfo_MixedNilAndReal(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	row := &render.LegendRow{
		MetricName: "fill",
		Kind:       metric.Quantity,
		Colours:    []color.RGBA{{A: 0xFF}},
	}

	info := buildLegendInfo(nil, nil, row, nil)
	g.Expect(info).NotTo(BeNil())

	if info == nil {
		return
	}

	g.Expect(info.Rows).To(HaveLen(1))
	g.Expect(info.Rows[0].MetricName).To(Equal("fill"))
}

func TestBuildLegendInfo_TwoRows(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	fillRow := &render.LegendRow{
		MetricName:  "file-size",
		Kind:        metric.Quantity,
		Colours:     []color.RGBA{{A: 0xFF}, {A: 0xFF}},
		Breakpoints: []float64{100},
	}
	borderRow := &render.LegendRow{
		MetricName: "file-type",
		Kind:       metric.Classification,
		Colours:    []color.RGBA{{A: 0xFF}, {A: 0xFF}},
		Categories: []string{"go", "rs"},
	}

	info := buildLegendInfo(nil, fillRow, borderRow)
	g.Expect(info).NotTo(BeNil())

	if info == nil {
		return
	}

	g.Expect(info.Rows).To(HaveLen(2))
	g.Expect(info.Rows[0].MetricName).To(Equal("file-size"))
	g.Expect(info.Rows[1].MetricName).To(Equal("file-type"))
}
