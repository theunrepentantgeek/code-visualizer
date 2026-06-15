package golang

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
)

func TestIsGoMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(IsGoMetric(Types)).To(BeTrue())
	g.Expect(IsGoMetric(Imports)).To(BeTrue())
	g.Expect(IsGoMetric(CommentRatio)).To(BeTrue())
	g.Expect(IsGoMetric(CyclomaticComplexity)).To(BeTrue())
	g.Expect(IsGoMetric(FunctionLength)).To(BeTrue())
	g.Expect(IsGoMetric(metric.Name("type-count"))).To(BeFalse())
	g.Expect(IsGoMetric(metric.Name("public-method-count"))).To(BeFalse())
	g.Expect(IsGoMetric("file-size")).To(BeFalse())
	g.Expect(IsGoMetric("file-lines")).To(BeFalse())
	g.Expect(IsGoMetric("unknown-metric")).To(BeFalse())
}

func TestAllGoMetricCount(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(allMetrics).To(HaveLen(12))
}
