package radialtree_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/radialtree"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func TestResolveRadialMetrics_DiscSizeOnly(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	discSizeStr := "file-size"
	common := &stages.CommonState{}
	viz := &radialtree.State{}
	cfg := &config.Radial{DiscSize: &discSizeStr}

	g.Expect(radialtree.ResolveMetrics(common, viz, cfg)).To(Succeed())
	g.Expect(viz.DiscSize).To(Equal(metric.Name("file-size")))
	// Without an explicit Fill, fill metric defaults to disc size.
	g.Expect(viz.FillMetric).To(Equal(metric.Name("file-size")))
	g.Expect(common.Requested).To(ConsistOf(metric.Name("file-size")))
}

func TestResolveRadialMetrics_FillOverridesDiscSizeAsFillMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	discSizeStr := "file-size"
	common := &stages.CommonState{}
	viz := &radialtree.State{}
	cfg := &config.Radial{
		DiscSize: &discSizeStr,
		Fill:     &config.MetricSpec{Metric: "file-type"},
	}

	g.Expect(radialtree.ResolveMetrics(common, viz, cfg)).To(Succeed())
	g.Expect(viz.FillMetric).To(Equal(metric.Name("file-type")))
	g.Expect(common.Requested).To(ContainElements(metric.Name("file-size"), metric.Name("file-type")))
}

func TestResolveRadialMetrics_LabelsDefaultToFolders(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	discSizeStr := "file-size"
	common := &stages.CommonState{}
	viz := &radialtree.State{}
	cfg := &config.Radial{DiscSize: &discSizeStr}

	g.Expect(radialtree.ResolveMetrics(common, viz, cfg)).To(Succeed())
	g.Expect(viz.Labels).To(Equal(radialtree.LabelFoldersOnly))
}

func TestResolveRadialMetrics_LabelsNoneExplicit(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	discSizeStr := "file-size"
	labelStr := string(radialtree.LabelNone)
	common := &stages.CommonState{}
	viz := &radialtree.State{}
	cfg := &config.Radial{
		DiscSize: &discSizeStr,
		Labels:   &labelStr,
	}

	g.Expect(radialtree.ResolveMetrics(common, viz, cfg)).To(Succeed())
	g.Expect(viz.Labels).To(Equal(radialtree.LabelNone))
}
