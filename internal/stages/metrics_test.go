package stages_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func TestMain(m *testing.M) {
	filesystem.Register()
	m.Run()
}

// ---------------------------------------------------------------------------
// CollectRequestedMetrics
// ---------------------------------------------------------------------------

func TestCollectRequestedMetrics_SizeOnly(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	got := stages.CollectRequestedMetrics("file-size", nil, nil)

	g.Expect(got).To(ConsistOf(metric.Name("file-size")))
}

func TestCollectRequestedMetrics_SizeAndFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	fill := &config.MetricSpec{Metric: "file-lines"}
	got := stages.CollectRequestedMetrics("file-size", fill, nil)

	g.Expect(got).To(Equal([]metric.Name{"file-size", "file-lines"}))
}

func TestCollectRequestedMetrics_SizeAndBorder(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	border := &config.MetricSpec{Metric: "file-type"}
	got := stages.CollectRequestedMetrics("file-size", nil, border)

	g.Expect(got).To(Equal([]metric.Name{"file-size", "file-type"}))
}

func TestCollectRequestedMetrics_DeduplicatesFillEqualsSize(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	fill := &config.MetricSpec{Metric: "file-size"}
	got := stages.CollectRequestedMetrics("file-size", fill, nil)

	g.Expect(got).To(ConsistOf(metric.Name("file-size")))
}

func TestCollectRequestedMetrics_AllThreeDistinct(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	fill := &config.MetricSpec{Metric: "file-lines"}
	border := &config.MetricSpec{Metric: "file-type"}
	got := stages.CollectRequestedMetrics("file-size", fill, border)

	g.Expect(got).To(Equal([]metric.Name{"file-size", "file-lines", "file-type"}))
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
// ResolveBorderMetricAndPalette
// ---------------------------------------------------------------------------

func TestResolveBorderMetricAndPalette_NilSpecReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	m, p := stages.ResolveBorderMetricAndPalette(nil)

	g.Expect(m).To(BeEmpty())
	g.Expect(p).To(BeEmpty())
}

func TestResolveBorderMetricAndPalette_EmptyMetricReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	border := &config.MetricSpec{}
	m, p := stages.ResolveBorderMetricAndPalette(border)

	g.Expect(m).To(BeEmpty())
	g.Expect(p).To(BeEmpty())
}

func TestResolveBorderMetricAndPalette_KnownMetricUsesProviderDefault(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	border := &config.MetricSpec{Metric: "file-size"}
	m, p := stages.ResolveBorderMetricAndPalette(border)

	g.Expect(m).To(Equal(metric.Name("file-size")))
	g.Expect(p).To(Equal(palette.Neutral))
}

func TestResolveBorderMetricAndPalette_ExplicitPaletteOverridesDefault(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	border := &config.MetricSpec{Metric: "file-size", Palette: "terrain"}
	m, p := stages.ResolveBorderMetricAndPalette(border)

	g.Expect(m).To(Equal(metric.Name("file-size")))
	g.Expect(p).To(Equal(palette.PaletteName("terrain")))
}

func TestResolveBorderMetricAndPalette_UnknownMetricFallsBackToNeutral(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	border := &config.MetricSpec{Metric: "not-registered"}
	m, p := stages.ResolveBorderMetricAndPalette(border)

	g.Expect(m).To(Equal(metric.Name("not-registered")))
	g.Expect(p).To(Equal(palette.Neutral))
}
