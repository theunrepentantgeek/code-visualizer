package treemap_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
	"github.com/theunrepentantgeek/code-visualizer/internal/treemap"
)

func TestResolveMetrics_SizeOnly(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sizeStr := "file-size"
	s := &treemap.State{
		Config: &config.Treemap{Size: &sizeStr},
	}

	g.Expect(treemap.ResolveMetrics(s)).To(Succeed())
	g.Expect(s.Size).To(Equal(metric.Name("file-size")))
	g.Expect(s.FillMetric).To(Equal(metric.Name("file-size")))
	g.Expect(s.Common().Requested).To(ConsistOf(metric.Name("file-size")))
}

func TestResolveMetrics_FillOverridesSizeAsFillMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sizeStr := "file-size"
	s := &treemap.State{
		Config: &config.Treemap{
			Size: &sizeStr,
			Fill: &config.MetricSpec{Metric: "file-type"},
		},
	}

	g.Expect(treemap.ResolveMetrics(s)).To(Succeed())
	g.Expect(s.FillMetric).To(Equal(metric.Name("file-type")))
	g.Expect(s.Common().Requested).To(ContainElements(metric.Name("file-size"), metric.Name("file-type")))
}

func TestState_CommonReturnsEmbeddedPointer(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	s := &treemap.State{}
	c := s.Common()
	c.Width = 42
	g.Expect(s.CommonState.Width).To(Equal(42))
}

func TestState_IncludeBinary(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	on := &treemap.State{IncludeBinaryFiles: true}
	off := &treemap.State{IncludeBinaryFiles: false}

	g.Expect(on.IncludeBinary()).To(BeTrue())
	g.Expect(off.IncludeBinary()).To(BeFalse())

	var _ stages.BinaryFilterToggler = on
}
