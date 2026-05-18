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
	s := &bubbletree.State{
		Config: &config.Bubbletree{Size: &sizeStr},
	}

	g.Expect(bubbletree.ResolveMetrics(s)).To(Succeed())
	g.Expect(s.Size).To(Equal(metric.Name("file-size")))
	g.Expect(s.FillMetric).To(Equal(metric.Name("file-size")))
	g.Expect(s.Common().Requested).To(ConsistOf(metric.Name("file-size")))
}

func TestResolveMetrics_FillOverridesSizeAsFillMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sizeStr := "file-size"
	s := &bubbletree.State{
		Config: &config.Bubbletree{
			Size: &sizeStr,
			Fill: &config.MetricSpec{Metric: "file-type"},
		},
	}

	g.Expect(bubbletree.ResolveMetrics(s)).To(Succeed())
	g.Expect(s.FillMetric).To(Equal(metric.Name("file-type")))
	g.Expect(s.Common().Requested).To(ContainElements(metric.Name("file-size"), metric.Name("file-type")))
}

func TestResolveMetrics_DefaultsLabelsToFoldersOnly(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sizeStr := "file-size"
	s := &bubbletree.State{
		Config: &config.Bubbletree{Size: &sizeStr},
	}

	g.Expect(bubbletree.ResolveMetrics(s)).To(Succeed())
	g.Expect(s.Labels).To(Equal(bubbletree.LabelFoldersOnly))
}

func TestResolveMetrics_LabelsCanBeOverridden(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sizeStr := "file-size"
	labelsStr := "all"
	s := &bubbletree.State{
		Config: &config.Bubbletree{Size: &sizeStr, Labels: &labelsStr},
	}

	g.Expect(bubbletree.ResolveMetrics(s)).To(Succeed())
	g.Expect(s.Labels).To(Equal(bubbletree.LabelAll))
}

func TestState_CommonReturnsEmbeddedPointer(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	s := &bubbletree.State{}
	c := s.Common()
	c.Width = 42
	g.Expect(s.CommonState.Width).To(Equal(42))
}

func TestState_IncludeBinary(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	on := &bubbletree.State{IncludeBinaryFiles: true}
	off := &bubbletree.State{IncludeBinaryFiles: false}

	g.Expect(on.IncludeBinary()).To(BeTrue())
	g.Expect(off.IncludeBinary()).To(BeFalse())

	var _ stages.BinaryFilterToggler = on
}
