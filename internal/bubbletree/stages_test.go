package bubbletree_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/bubbletree"
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func TestResolveMetrics_SizeOnly(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sizeStr := "file-size"
	common := &stages.CommonState{}
	viz := &bubbletree.State{}
	cfg := &config.Bubbletree{Size: &sizeStr}

	g.Expect(bubbletree.ResolveMetrics(common, viz, cfg)).To(Succeed())
	g.Expect(viz.Size).To(Equal(metric.Name("file-size")))
	g.Expect(viz.FillMetric).To(Equal(metric.Name("file-size")))
	g.Expect(common.Requested.LegacyNames()).To(ConsistOf(metric.Name("file-size")))
}

func TestResolveMetrics_FillOverridesSizeAsFillMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sizeStr := "file-size"
	common := &stages.CommonState{}
	viz := &bubbletree.State{}
	cfg := &config.Bubbletree{
		Size: &sizeStr,
		Fill: &config.MetricSpec{Metric: "file-type"},
	}

	g.Expect(bubbletree.ResolveMetrics(common, viz, cfg)).To(Succeed())
	g.Expect(viz.FillMetric).To(Equal(metric.Name("file-type")))
	g.Expect(common.Requested.LegacyNames()).To(ContainElements(metric.Name("file-size"), metric.Name("file-type")))
}

func TestResolveMetrics_DefaultsLabelsToFoldersOnly(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sizeStr := "file-size"
	common := &stages.CommonState{}
	viz := &bubbletree.State{}
	cfg := &config.Bubbletree{Size: &sizeStr}

	g.Expect(bubbletree.ResolveMetrics(common, viz, cfg)).To(Succeed())
	g.Expect(viz.Labels).To(Equal(bubbletree.LabelFoldersOnly))
}

func TestResolveMetrics_LabelsCanBeOverridden(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sizeStr := "file-size"
	labelsStr := "all"
	common := &stages.CommonState{}
	viz := &bubbletree.State{}
	cfg := &config.Bubbletree{Size: &sizeStr, Labels: &labelsStr}

	g.Expect(bubbletree.ResolveMetrics(common, viz, cfg)).To(Succeed())
	g.Expect(viz.Labels).To(Equal(bubbletree.LabelAll))
}
