package donuttree_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/donuttree"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func TestResolveMetrics_AggregatesSizeAndDefaultFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	size := "file-lines"
	common := &stages.CommonState{}
	state := &donuttree.State{}
	cfg := &config.DonutTree{Size: &size}

	g.Expect(donuttree.ResolveMetrics(common, state, cfg)).To(Succeed())
	g.Expect(state.SizeMetric).To(Equal(metric.Name("file-lines.sum")))
	g.Expect(state.FillMetric).To(Equal(metric.Name("file-lines.sum")))
	g.Expect(state.BorderMetric).To(BeEmpty())
	g.Expect(common.Requested.Expressions).To(ConsistOf(
		HaveField("ResultName", metric.Name("file-lines.sum")),
	))
}

func TestResolveMetrics_AggregatesExplicitFillAndBorder(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	size := "file-lines"
	common := &stages.CommonState{}
	state := &donuttree.State{}
	cfg := &config.DonutTree{
		Size: &size,
		Fill: &config.MetricSpec{
			Metric:  "file-type",
			Palette: palette.Categorization,
		},
		Border: &config.MetricSpec{
			Metric:  "file-freshness",
			Palette: palette.GoodBad,
		},
	}

	g.Expect(donuttree.ResolveMetrics(common, state, cfg)).To(Succeed())
	g.Expect(state.FillMetric).To(Equal(metric.Name("file-type.mode")))
	g.Expect(state.FillPalette).To(Equal(palette.Categorization))
	g.Expect(state.BorderMetric).To(Equal(metric.Name("file-freshness.sum")))
	g.Expect(state.BorderPalette).To(Equal(palette.GoodBad))
	g.Expect(common.Requested.Expressions).To(ConsistOf(
		HaveField("ResultName", metric.Name("file-lines.sum")),
		HaveField("ResultName", metric.Name("file-type.mode")),
		HaveField("ResultName", metric.Name("file-freshness.sum")),
	))
}

func TestResolveMetrics_PreservesExistingAggregation(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	size := "file-size.sum"
	common := &stages.CommonState{}
	state := &donuttree.State{}
	cfg := &config.DonutTree{Size: &size}

	g.Expect(donuttree.ResolveMetrics(common, state, cfg)).To(Succeed())
	g.Expect(state.SizeMetric).To(Equal(metric.Name("file-size.sum")))
	g.Expect(state.FillMetric).To(Equal(metric.Name("file-size.sum")))
	g.Expect(common.Requested.Expressions).To(ConsistOf(
		HaveField("ResultName", metric.Name("file-size.sum")),
	))
}
