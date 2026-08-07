package spiral_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/spiral"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func TestResolveMetrics_SizeOnly(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sizeStr := "file-size"
	common := &stages.CommonState{}
	viz := &spiral.State{}
	cfg := &config.Spiral{Size: &sizeStr}

	g.Expect(spiral.ResolveMetrics(common, viz, cfg)).To(Succeed())
	g.Expect(viz.Size).To(Equal(metric.Name("file-size")))
	// Spiral does not fall back FillMetric to Size; without an explicit Fill
	// the spiral renders without a fill metric.
	g.Expect(viz.FillMetric).To(Equal(metric.Name("")))
	g.Expect(common.Requested.BaseMetrics).To(ConsistOf(metric.Name("file-size")))
}

func TestResolveMetrics_NilSizeExcludesSizeFromRequested(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// When Size is nil the spiral defaults to commit-count.
	// Only fill and border contribute to Requested.
	common := &stages.CommonState{}
	viz := &spiral.State{}
	cfg := &config.Spiral{
		Fill: &config.MetricSpec{Metric: "file-type"},
	}

	g.Expect(spiral.ResolveMetrics(common, viz, cfg)).To(Succeed())
	g.Expect(viz.Size).To(Equal(metric.Name("")))
	g.Expect(common.Requested.BaseMetrics).To(ConsistOf(metric.Name("file-type")))
}

func TestResolveMetrics_FillMetricSetWhenFillConfigured(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sizeStr := "file-size"
	common := &stages.CommonState{}
	viz := &spiral.State{}
	cfg := &config.Spiral{
		Size: &sizeStr,
		Fill: &config.MetricSpec{Metric: "file-type"},
	}

	g.Expect(spiral.ResolveMetrics(common, viz, cfg)).To(Succeed())
	g.Expect(viz.FillMetric).To(Equal(metric.Name("file-type")))
}

func TestResolveMetrics_FillOverridesSizeAsFillMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sizeStr := "file-size"
	common := &stages.CommonState{}
	viz := &spiral.State{}
	cfg := &config.Spiral{
		Size: &sizeStr,
		Fill: &config.MetricSpec{Metric: "file-type"},
	}

	g.Expect(spiral.ResolveMetrics(common, viz, cfg)).To(Succeed())
	g.Expect(viz.FillMetric).To(Equal(metric.Name("file-type")))
	g.Expect(common.Requested.BaseMetrics).To(ContainElements(metric.Name("file-size"), metric.Name("file-type")))
}

func TestResolveMetrics_DefaultsResolutionToDaily(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sizeStr := "file-size"
	common := &stages.CommonState{}
	viz := &spiral.State{}
	cfg := &config.Spiral{Size: &sizeStr}

	g.Expect(spiral.ResolveMetrics(common, viz, cfg)).To(Succeed())
	g.Expect(viz.Resolution).To(Equal(spiral.Daily))
}

func TestResolveMetrics_HourlyResolution(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sizeStr := "file-size"
	res := "hourly"
	common := &stages.CommonState{}
	viz := &spiral.State{}
	cfg := &config.Spiral{Size: &sizeStr, Resolution: &res}

	g.Expect(spiral.ResolveMetrics(common, viz, cfg)).To(Succeed())
	g.Expect(viz.Resolution).To(Equal(spiral.Hourly))
}

func TestResolveMetrics_SurfaceDefaultsToFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	enabled := true
	common := &stages.CommonState{}
	viz := &spiral.State{}
	cfg := &config.Spiral{
		Fill:    &config.MetricSpec{Metric: "file-lines", Palette: palette.Foliage},
		Surface: &enabled,
	}

	g.Expect(spiral.ResolveMetrics(common, viz, cfg)).To(Succeed())
	g.Expect(viz.SurfaceEnabled).To(BeTrue())
	g.Expect(viz.SurfaceMetric).To(Equal(metric.Name("file-lines")))
	g.Expect(viz.SurfacePalette).To(Equal(palette.Foliage))
	g.Expect(common.Requested.BaseMetrics).To(ConsistOf(metric.Name("file-lines")))
}

func TestResolveMetrics_SurfaceMetricOverridesFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	common := &stages.CommonState{}
	viz := &spiral.State{}
	cfg := &config.Spiral{
		Fill: &config.MetricSpec{Metric: "file-lines"},
		SurfaceMetric: &config.MetricSpec{
			Metric:  "file-size",
			Palette: palette.Temperature,
		},
	}

	g.Expect(spiral.ResolveMetrics(common, viz, cfg)).To(Succeed())
	g.Expect(viz.SurfaceEnabled).To(BeTrue())
	g.Expect(viz.SurfaceMetric).To(Equal(metric.Name("file-size")))
	g.Expect(viz.SurfacePalette).To(Equal(palette.Temperature))
	g.Expect(common.Requested.BaseMetrics).To(ConsistOf(
		metric.Name("file-lines"), metric.Name("file-size"),
	))
}

func TestResolveMetrics_SurfaceDisabledHasNoMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	common := &stages.CommonState{}
	viz := &spiral.State{}

	g.Expect(spiral.ResolveMetrics(common, viz, &config.Spiral{})).To(Succeed())
	g.Expect(viz.SurfaceEnabled).To(BeFalse())
	g.Expect(viz.SurfaceMetric).To(BeEmpty())
}

func TestAggregateBucketMetricsStage_UsesRequestedDescriptorForExpressionFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	const expressionMetric = metric.Name("expression.metric.count")

	first := &model.File{Name: "a.go"}
	first.SetQuantity(expressionMetric, 2)

	second := &model.File{Name: "b.go"}
	second.SetQuantity(expressionMetric, 1)

	common := &stages.CommonState{
		Requested: stages.RequestedMetrics{
			Expressions: []provider.ResolvedMetric{{
				ResultName: expressionMetric,
				ResultKind: metric.Quantity,
			}},
		},
	}
	viz := &spiral.State{
		FillMetric: expressionMetric,
		Buckets: []spiral.TimeBucket{{
			Files: []*model.File{first, second},
		}},
	}

	g.Expect(spiral.AggregateBucketMetricsStage(common, viz)).To(Succeed())
	g.Expect(viz.Buckets[0].FillValue).To(Equal(3.0))
}

func TestBuildInksStage_SurfaceUsesFillInkForSameMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	common := &stages.CommonState{}
	viz := &spiral.State{
		Buckets:        []spiral.TimeBucket{{FillValue: 4, SurfaceValue: 4}},
		FillMetric:     "file-lines",
		FillPalette:    palette.Foliage,
		SurfaceEnabled: true,
		SurfaceMetric:  "file-lines",
		SurfacePalette: palette.Foliage,
	}

	g.Expect(spiral.BuildInksStage(common, viz)).To(Succeed())
	g.Expect(viz.SurfaceInk).To(BeIdenticalTo(viz.Inks.Fill))
}

func TestBuildInksStage_BuildsIndependentSurfaceInkForSameMetricWithDifferentPalette(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	common := &stages.CommonState{
		Requested: stages.RequestedMetrics{
			BaseMetrics: []metric.Name{"file-lines"},
		},
	}
	viz := &spiral.State{
		Buckets: []spiral.TimeBucket{
			{FillValue: 1, SurfaceValue: 1},
			{FillValue: 10, SurfaceValue: 10},
		},
		FillMetric:     "file-lines",
		FillPalette:    palette.Foliage,
		SurfaceEnabled: true,
		SurfaceMetric:  "file-lines",
		SurfacePalette: palette.Temperature,
	}

	g.Expect(spiral.BuildInksStage(common, viz)).To(Succeed())
	g.Expect(viz.SurfaceInk).NotTo(BeIdenticalTo(viz.Inks.Fill))
	g.Expect(viz.SurfaceInk.Dip(inks.MeasureValue(10))).NotTo(Equal(viz.Inks.Fill.Dip(inks.MeasureValue(10))))
}

func TestBuildInksStage_BuildsNumericSurfaceInk(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	common := &stages.CommonState{}
	viz := &spiral.State{
		Buckets:        []spiral.TimeBucket{{SurfaceValue: 4}},
		SurfaceEnabled: true,
		SurfaceMetric:  "file-lines",
		SurfacePalette: palette.Foliage,
	}

	g.Expect(spiral.BuildInksStage(common, viz)).To(Succeed())
	g.Expect(viz.SurfaceInk.Info().Kind).To(Equal(inks.KindNumeric))
}

func TestBuildInksStage_UsesRequestedDescriptorForExpressionFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	const expressionMetric = metric.Name("expression.metric.mode")

	common := &stages.CommonState{
		Requested: stages.RequestedMetrics{
			Expressions: []provider.ResolvedMetric{{
				ResultName: expressionMetric,
				ResultKind: metric.Classification,
			}},
		},
	}
	viz := &spiral.State{
		Buckets: []spiral.TimeBucket{
			{FillLabel: "go"},
			{FillLabel: "py"},
		},
		FillMetric:  expressionMetric,
		FillPalette: palette.Categorization,
	}

	g.Expect(spiral.BuildInksStage(common, viz)).To(Succeed())
	g.Expect(viz.Inks.Fill.Info().Kind).To(Equal(inks.KindCategorical))
}

func TestBuildLegendStage_AddsDistinctSurfaceMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	common := &stages.CommonState{RootConfig: config.New()}
	viz := &spiral.State{
		FillMetric:    "file-lines",
		SurfaceMetric: "file-size",
		Inks:          spiral.Inks{Fill: inks.FixedInk(palette.White)},
		SurfaceInk:    inks.FixedInk(palette.White),
	}

	g.Expect(spiral.BuildLegendStage(common, viz)).To(Succeed())
	g.Expect(viz.LegendConfig.Entries).To(HaveLen(3))
	g.Expect(viz.LegendConfig.Entries[2].Role).To(Equal(legend.RoleSurface))
	g.Expect(viz.LegendConfig.Entries[2].MetricName).To(Equal("file-size"))
}

func TestBuildLegendStage_SkipsSurfaceMatchingFillMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	common := &stages.CommonState{RootConfig: config.New()}
	viz := &spiral.State{
		FillMetric:     "file-lines",
		FillPalette:    palette.Foliage,
		SurfaceMetric:  "file-lines",
		SurfacePalette: palette.Foliage,
		Inks:           spiral.Inks{Fill: inks.FixedInk(palette.White)},
		SurfaceInk:     inks.FixedInk(palette.White),
	}

	g.Expect(spiral.BuildLegendStage(common, viz)).To(Succeed())
	g.Expect(viz.LegendConfig.Entries).To(HaveLen(2))
	g.Expect(viz.LegendConfig.Entries[0].Role).To(Equal(legend.RoleFill))
}

func TestBuildLegendStage_AddsSurfaceForSameMetricWithDifferentPalette(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	common := &stages.CommonState{RootConfig: config.New()}
	viz := &spiral.State{
		FillMetric:     "file-lines",
		FillPalette:    palette.Foliage,
		SurfaceMetric:  "file-lines",
		SurfacePalette: palette.Temperature,
		Inks:           spiral.Inks{Fill: inks.FixedInk(palette.White)},
		SurfaceInk:     inks.FixedInk(palette.White),
	}

	g.Expect(spiral.BuildLegendStage(common, viz)).To(Succeed())
	g.Expect(viz.LegendConfig.Entries).To(HaveLen(3))
	g.Expect(viz.LegendConfig.Entries[2].Role).To(Equal(legend.RoleSurface))
	g.Expect(viz.LegendConfig.Entries[2].MetricName).To(Equal("file-lines"))
}
