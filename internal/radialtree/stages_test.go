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
	s := &radialtree.State{
		Config: &config.Radial{DiscSize: &discSizeStr},
	}

	g.Expect(radialtree.ResolveMetrics(s)).To(Succeed())
	g.Expect(s.DiscSize).To(Equal(metric.Name("file-size")))
	// Without an explicit Fill, fill metric defaults to disc size.
	g.Expect(s.FillMetric).To(Equal(metric.Name("file-size")))
	g.Expect(s.Common().Requested).To(ConsistOf(metric.Name("file-size")))
}

func TestResolveRadialMetrics_FillOverridesDiscSizeAsFillMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	discSizeStr := "file-size"
	s := &radialtree.State{
		Config: &config.Radial{
			DiscSize: &discSizeStr,
			Fill:     &config.MetricSpec{Metric: "file-type"},
		},
	}

	g.Expect(radialtree.ResolveMetrics(s)).To(Succeed())
	g.Expect(s.FillMetric).To(Equal(metric.Name("file-type")))
	g.Expect(s.Common().Requested).To(ContainElements(metric.Name("file-size"), metric.Name("file-type")))
}

func TestResolveRadialMetrics_LabelsDefaultToAll(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	discSizeStr := "file-size"
	s := &radialtree.State{
		Config: &config.Radial{DiscSize: &discSizeStr},
	}

	g.Expect(radialtree.ResolveMetrics(s)).To(Succeed())
	g.Expect(s.Labels).To(Equal(radialtree.LabelAll))
}

func TestResolveRadialMetrics_LabelsNoneExplicit(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	discSizeStr := "file-size"
	labelStr := string(radialtree.LabelNone)
	s := &radialtree.State{
		Config: &config.Radial{
			DiscSize: &discSizeStr,
			Labels:   &labelStr,
		},
	}

	g.Expect(radialtree.ResolveMetrics(s)).To(Succeed())
	g.Expect(s.Labels).To(Equal(radialtree.LabelNone))
}

func TestRadialState_CommonReturnsEmbeddedPointer(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	s := &radialtree.State{}
	c := s.Common()
	c.Width = 800
	g.Expect(s.CommonState.Width).To(Equal(800))
}

func TestRadialState_IncludeBinary(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	on := &radialtree.State{IncludeBinaryFiles: true}
	off := &radialtree.State{IncludeBinaryFiles: false}

	g.Expect(on.IncludeBinary()).To(BeTrue())
	g.Expect(off.IncludeBinary()).To(BeFalse())

	var _ stages.BinaryFilterToggler = on
}
