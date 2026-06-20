package spiral_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	pkginks "github.com/theunrepentantgeek/code-visualizer/internal/inks"
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

func TestResolveMetrics_DefaultsLabelsToLaps(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sizeStr := "file-size"
	common := &stages.CommonState{}
	viz := &spiral.State{}
	cfg := &config.Spiral{Size: &sizeStr}

	g.Expect(spiral.ResolveMetrics(common, viz, cfg)).To(Succeed())
	g.Expect(viz.Labels).To(Equal(spiral.LabelLaps))
}

func TestResolveMetrics_LabelsAllCanBeSet(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sizeStr := "file-size"
	lblStr := "all"
	common := &stages.CommonState{}
	viz := &spiral.State{}
	cfg := &config.Spiral{Size: &sizeStr, Labels: &lblStr}

	g.Expect(spiral.ResolveMetrics(common, viz, cfg)).To(Succeed())
	g.Expect(viz.Labels).To(Equal(spiral.LabelAll))
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
	g.Expect(viz.Inks.Fill.Info().Kind).To(Equal(pkginks.KindCategorical))
}
