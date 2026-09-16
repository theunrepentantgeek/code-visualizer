package stages_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/git"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
	"github.com/theunrepentantgeek/code-visualizer/internal/viz"
)

func TestMain(m *testing.M) {
	filesystem.Register()
	git.Register()
	m.Run()
}

// ---------------------------------------------------------------------------
// CollectRequestedMetrics
// ---------------------------------------------------------------------------

func TestCollectRequestedMetrics_SizeOnly(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	got := stages.CollectRequestedMetrics("file-size", nil, nil)

	g.Expect(got.BaseMetrics).To(ConsistOf(metric.Name("file-size")))
}

func TestCollectRequestedMetrics_SizeAndFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	fill := &config.MetricSpec{Metric: "file-lines"}
	got := stages.CollectRequestedMetrics("file-size", fill, nil)

	g.Expect(got.BaseMetrics).To(ContainElements(metric.Name("file-size"), metric.Name("file-lines")))
}

func TestCollectRequestedMetrics_SizeAndBorder(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	border := &config.MetricSpec{Metric: "file-type"}
	got := stages.CollectRequestedMetrics("file-size", nil, border)

	g.Expect(got.BaseMetrics).To(ContainElements(metric.Name("file-size"), metric.Name("file-type")))
}

func TestCollectRequestedMetrics_DeduplicatesFillEqualsSize(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	fill := &config.MetricSpec{Metric: "file-size"}
	got := stages.CollectRequestedMetrics("file-size", fill, nil)

	g.Expect(got.BaseMetrics).To(HaveLen(1))
	g.Expect(got.BaseMetrics).To(ConsistOf(metric.Name("file-size")))
}

func TestCollectRequestedMetrics_AllThreeDistinct(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	fill := &config.MetricSpec{Metric: "file-lines"}
	border := &config.MetricSpec{Metric: "file-type"}
	got := stages.CollectRequestedMetrics("file-size", fill, border)

	g.Expect(got.BaseMetrics).To(ContainElements(
		metric.Name("file-size"),
		metric.Name("file-lines"),
		metric.Name("file-type"),
	))
}

func TestResolveColourEncoding(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	g.Expect(stages.ResolveColourEncoding(nil, "")).To(Equal(viz.ColourEncoding{}))
	g.Expect(stages.ResolveColourEncoding(nil, "file-type")).To(Equal(viz.ColourEncoding{
		Metric:  metric.Name("file-type"),
		Palette: palette.Categorization,
	}))
	g.Expect(stages.ResolveColourEncoding(
		&config.MetricSpec{Metric: "file-size", Palette: "terrain"},
		"file-type",
	)).To(Equal(viz.ColourEncoding{
		Metric:  metric.Name("file-size"),
		Palette: palette.PaletteName("terrain"),
	}))
}

// ---------------------------------------------------------------------------
// ResolveFillPalette
// ---------------------------------------------------------------------------

func TestResolveFillPalette_ExplicitPaletteUsed(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	fill := &config.MetricSpec{Metric: "file-size", Palette: "terrain"}
	got := stages.ResolveFillPalette(fill, "file-size")

	g.Expect(got).To(Equal(palette.PaletteName("terrain")))
}

func TestResolveFillPalette_NilSpecFallsBackToProviderDefault(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// file-type is registered with palette.Categorization as default.
	got := stages.ResolveFillPalette(nil, "file-type")

	g.Expect(got).NotTo(BeEmpty())
	g.Expect(got).NotTo(Equal(palette.Neutral))
}

func TestResolveFillPalette_UnknownMetricReturnsNeutral(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	got := stages.ResolveFillPalette(nil, "not-a-real-metric")

	g.Expect(got).To(Equal(palette.Neutral))
}

// ---------------------------------------------------------------------------
// Expression metric palette inheritance
// ---------------------------------------------------------------------------

func TestResolveFillPalette_ExpressionMetricInheritsBasePalette(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// commit-count is a git metric with DefaultPalette: palette.Temperature.
	// commit-count.sum is an aggregation expression; it should inherit Temperature.
	got := stages.ResolveFillPalette(nil, "commit-count.sum")

	g.Expect(got).NotTo(Equal(palette.Neutral))
	g.Expect(got).To(Equal(palette.Temperature))
}
