package stages_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func TestClassifyRequestedMetrics_BareMetricGoesToLegacy(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	names := []metric.Name{"file-size"}
	result := stages.ClassifyRequestedMetrics(names, metric.LevelDirectory)

	g.Expect(result.LegacyNames()).To(ContainElement(metric.Name("file-size")))
	g.Expect(result.Expressions).To(BeEmpty())
}

func TestClassifyRequestedMetrics_ExpressionWithAggregation(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	names := []metric.Name{"file-size.sum"}
	result := stages.ClassifyRequestedMetrics(names, metric.LevelDirectory)

	g.Expect(result.Expressions).To(HaveLen(1))
	g.Expect(result.Expressions[0].Expression.Base).To(Equal(metric.Name("file-size")))
	g.Expect(result.Expressions[0].Expression.Aggregation).To(Equal(metric.AggSum))
	g.Expect(result.BaseMetrics).To(ContainElement(metric.Name("file-size")))
}

func TestClassifyRequestedMetrics_UnresolvableExpressionGoesToLegacy(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	names := []metric.Name{"not-a-real-metric"}
	result := stages.ClassifyRequestedMetrics(names, metric.LevelDirectory)

	g.Expect(result.LegacyNames()).To(ContainElement(metric.Name("not-a-real-metric")))
	g.Expect(result.Expressions).To(BeEmpty())
}

func TestClassifyRequestedMetrics_MixedSet(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	names := []metric.Name{"file-size", "file-size.sum", "file-lines.max"}
	result := stages.ClassifyRequestedMetrics(names, metric.LevelDirectory)

	g.Expect(result.LegacyNames()).To(ContainElement(metric.Name("file-size")))
	g.Expect(result.Expressions).To(HaveLen(2))
	g.Expect(result.BaseMetrics).To(ContainElement(metric.Name("file-size")))
	g.Expect(result.BaseMetrics).To(ContainElement(metric.Name("file-lines")))
}
