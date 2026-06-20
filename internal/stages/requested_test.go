package stages_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func TestClassifyRequestedMetrics_BareMetricGoesToBaseMetrics(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	names := []metric.Name{"file-size"}
	result := stages.ClassifyRequestedMetrics(names, metric.LevelDirectory)

	g.Expect(result.BaseMetrics).To(ContainElement(metric.Name("file-size")))
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

func TestClassifyRequestedMetrics_UnresolvableExpressionGoesToBaseMetrics(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	names := []metric.Name{"not-a-real-metric"}
	result := stages.ClassifyRequestedMetrics(names, metric.LevelDirectory)

	g.Expect(result.BaseMetrics).To(ContainElement(metric.Name("not-a-real-metric")))
	g.Expect(result.Expressions).To(BeEmpty())
}

func TestClassifyRequestedMetrics_MixedSet(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	names := []metric.Name{"file-size", "file-size.sum", "file-lines.max"}
	result := stages.ClassifyRequestedMetrics(names, metric.LevelDirectory)

	g.Expect(result.Expressions).To(HaveLen(2))
	g.Expect(result.BaseMetrics).To(ContainElements(metric.Name("file-size"), metric.Name("file-lines")))
}

// ---------------------------------------------------------------------------
// HasDeclarationExpressions
// ---------------------------------------------------------------------------

func TestRequestedMetrics_HasDeclarationExpressions_FalseWhenEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	r := stages.RequestedMetrics{}
	g.Expect(r.HasDeclarationExpressions()).To(BeFalse())
}

func TestRequestedMetrics_HasDeclarationExpressions_FalseForFileLevelOnly(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	r := stages.RequestedMetrics{
		Expressions: []provider.ResolvedMetric{
			{SourceLevel: metric.LevelFile},
		},
	}
	g.Expect(r.HasDeclarationExpressions()).To(BeFalse())
}

func TestRequestedMetrics_HasDeclarationExpressions_TrueWhenPresent(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	r := stages.RequestedMetrics{
		Expressions: []provider.ResolvedMetric{
			{SourceLevel: metric.LevelFile},
			{SourceLevel: metric.LevelDeclaration},
		},
	}
	g.Expect(r.HasDeclarationExpressions()).To(BeTrue())
}

// ---------------------------------------------------------------------------
// HasCommitExpressions
// ---------------------------------------------------------------------------

func TestRequestedMetrics_HasCommitExpressions_FalseWhenEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	r := stages.RequestedMetrics{}
	g.Expect(r.HasCommitExpressions()).To(BeFalse())
}

func TestRequestedMetrics_HasCommitExpressions_FalseForFileLevelOnly(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	r := stages.RequestedMetrics{
		Expressions: []provider.ResolvedMetric{
			{SourceLevel: metric.LevelFile},
		},
	}
	g.Expect(r.HasCommitExpressions()).To(BeFalse())
}

func TestRequestedMetrics_HasCommitExpressions_TrueWhenPresent(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	r := stages.RequestedMetrics{
		Expressions: []provider.ResolvedMetric{
			{SourceLevel: metric.LevelFile},
			{SourceLevel: metric.LevelCommit},
		},
	}
	g.Expect(r.HasCommitExpressions()).To(BeTrue())
}

// ---------------------------------------------------------------------------
// DescriptorFor
// ---------------------------------------------------------------------------

func TestRequestedMetrics_DescriptorFor_ReturnsExpressionDescriptor(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	r := stages.RequestedMetrics{
		Expressions: []provider.ResolvedMetric{
			{
				ResultName: "file-size.sum",
				ResultKind: metric.Measure,
			},
		},
	}
	d, ok := r.DescriptorFor("file-size.sum")

	g.Expect(ok).To(BeTrue())
	g.Expect(d.Name).To(Equal(metric.Name("file-size.sum")))
	g.Expect(d.Kind).To(Equal(metric.Measure))
}

func TestRequestedMetrics_DescriptorFor_ExpressionTakesPriorityOverRegistry(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// "file-size" exists in the registry with Kind Quantity; we shadow it with
	// a Measure expression to verify the expression slot wins.
	r := stages.RequestedMetrics{
		Expressions: []provider.ResolvedMetric{
			{
				ResultName: "file-size",
				ResultKind: metric.Measure,
			},
		},
	}
	d, ok := r.DescriptorFor("file-size")

	g.Expect(ok).To(BeTrue())
	g.Expect(d.Kind).To(Equal(metric.Measure))
}

func TestRequestedMetrics_DescriptorFor_FallsBackToRegistry(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// No expressions — must find "file-size" via the provider registry.
	r := stages.RequestedMetrics{}
	d, ok := r.DescriptorFor("file-size")

	g.Expect(ok).To(BeTrue())
	g.Expect(d.Name).To(Equal(metric.Name("file-size")))
}

func TestRequestedMetrics_DescriptorFor_ReturnsFalseForUnknown(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	r := stages.RequestedMetrics{}
	_, ok := r.DescriptorFor("not-a-real-metric")

	g.Expect(ok).To(BeFalse())
}
