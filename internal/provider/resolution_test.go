package provider

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

func TestResolveExpression_BareMetricAtNativeLevel(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newBaseRegistry()
	reg.register(BaseMetricDescriptor{
		Name:           "file-size",
		Kind:           metric.Quantity,
		Level:          metric.LevelFile,
		Aggregations:   []metric.AggregationName{metric.AggSum, metric.AggMin, metric.AggMax, metric.AggMean},
		DefaultPalette: palette.Neutral,
	})

	expr := metric.MetricExpression{Base: "file-size"}
	resolved, err := resolveExpressionWith(reg, expr, metric.LevelFile)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(resolved.Expression).To(Equal(expr))
	g.Expect(resolved.ResultName).To(Equal(metric.Name("file-size")))
	g.Expect(resolved.NeedsAggregation).To(BeFalse())
}

//nolint:paralleltest // mutates global base registry
func TestResolveExpression_UsesGlobalRegistry(t *testing.T) {
	g := NewGomegaWithT(t)

	ResetBaseRegistryForTesting()
	t.Cleanup(ResetBaseRegistryForTesting)

	RegisterBase(BaseMetricDescriptor{
		Name:         "file-size",
		Kind:         metric.Quantity,
		Level:        metric.LevelFile,
		Aggregations: []metric.AggregationName{metric.AggSum},
	})

	resolved, err := ResolveExpression(metric.MetricExpression{Base: "file-size"}, metric.LevelFile)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(resolved.Descriptor.Name).To(Equal(metric.Name("file-size")))
}

func TestResolveExpression_MetricWithAggregation(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newBaseRegistry()
	reg.register(BaseMetricDescriptor{
		Name:         "file-size",
		Kind:         metric.Quantity,
		Level:        metric.LevelFile,
		Aggregations: []metric.AggregationName{metric.AggSum, metric.AggMin, metric.AggMax, metric.AggMean},
	})

	expr := metric.MetricExpression{Base: "file-size", Aggregation: metric.AggSum}
	resolved, err := resolveExpressionWith(reg, expr, metric.LevelDirectory)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(resolved.NeedsAggregation).To(BeTrue())
	g.Expect(resolved.ResultName).To(Equal(metric.Name("file-size.sum")))
	g.Expect(resolved.ResultKind).To(Equal(metric.Quantity))
}

func TestResolveExpression_MeanChangesKindToMeasure(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newBaseRegistry()
	reg.register(BaseMetricDescriptor{
		Name:         "file-size",
		Kind:         metric.Quantity,
		Level:        metric.LevelFile,
		Aggregations: []metric.AggregationName{metric.AggSum, metric.AggMean},
	})

	expr := metric.MetricExpression{Base: "file-size", Aggregation: metric.AggMean}
	resolved, err := resolveExpressionWith(reg, expr, metric.LevelDirectory)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(resolved.ResultKind).To(Equal(metric.Measure))
}

func TestResolveExpression_SameLevelAggregationRejected(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newBaseRegistry()
	reg.register(BaseMetricDescriptor{
		Name:         "file-size",
		Kind:         metric.Quantity,
		Level:        metric.LevelFile,
		Aggregations: []metric.AggregationName{metric.AggSum, metric.AggMin, metric.AggMax, metric.AggMean},
	})

	// file-size.sum at LevelFile is meaningless — "file-size" is already per-file.
	expr := metric.MetricExpression{Base: "file-size", Aggregation: metric.AggSum}
	_, err := resolveExpressionWith(reg, expr, metric.LevelFile)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("already a per-file metric")))
	g.Expect(err).To(MatchError(ContainSubstring("cross-level")))
}

func TestResolveExpression_InvalidAggregationError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newBaseRegistry()
	reg.register(BaseMetricDescriptor{
		Name:         "file-size",
		Kind:         metric.Quantity,
		Level:        metric.LevelFile,
		Aggregations: []metric.AggregationName{metric.AggSum, metric.AggMin, metric.AggMax, metric.AggMean},
	})

	expr := metric.MetricExpression{Base: "file-size", Aggregation: metric.AggMode}
	_, err := resolveExpressionWith(reg, expr, metric.LevelDirectory)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("mode")))
	g.Expect(err).To(MatchError(ContainSubstring("not a valid aggregation")))
	g.Expect(err).To(MatchError(ContainSubstring("sum, min, max, mean")))
}

func TestResolveExpression_InvalidFilterError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newBaseRegistry()
	reg.register(BaseMetricDescriptor{
		Name:         "types",
		Kind:         metric.Quantity,
		Level:        metric.LevelDeclaration,
		Filters:      []metric.FilterName{"public", "private"},
		Aggregations: []metric.AggregationName{metric.AggCount},
	})

	expr := metric.MetricExpression{Filter: "stdlib", Base: "types", Aggregation: metric.AggCount}
	_, err := resolveExpressionWith(reg, expr, metric.LevelFile)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("stdlib")))
	g.Expect(err).To(MatchError(ContainSubstring("not a valid filter")))
}

func TestResolveExpression_MissingAggregationAtHigherLevel(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newBaseRegistry()
	reg.register(BaseMetricDescriptor{
		Name:         "cyclomatic-complexity",
		Kind:         metric.Quantity,
		Level:        metric.LevelDeclaration,
		Aggregations: []metric.AggregationName{metric.AggSum, metric.AggMax, metric.AggMean},
	})

	expr := metric.MetricExpression{Base: "cyclomatic-complexity"}
	_, err := resolveExpressionWith(reg, expr, metric.LevelFile)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("requires aggregation")))
	g.Expect(err).To(MatchError(ContainSubstring("sum, max, mean")))
}

func TestResolveExpression_UnknownBaseMetricError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newBaseRegistry()

	expr := metric.MetricExpression{Base: "nonexistent"}
	_, err := resolveExpressionWith(reg, expr, metric.LevelFile)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("unknown base metric")))
}

func TestResolveExpression_FilterWithAggregation(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newBaseRegistry()
	reg.register(BaseMetricDescriptor{
		Name:         "types",
		Kind:         metric.Quantity,
		Level:        metric.LevelDeclaration,
		Filters:      []metric.FilterName{"public", "private"},
		Aggregations: []metric.AggregationName{metric.AggCount, metric.AggSum},
	})

	expr := metric.MetricExpression{Filter: "public", Base: "types", Aggregation: metric.AggCount}
	resolved, err := resolveExpressionWith(reg, expr, metric.LevelFile)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(resolved.ResultName).To(Equal(metric.Name("public.types.count")))
	g.Expect(resolved.NeedsAggregation).To(BeTrue())
}

func TestResolveExpression_FilterOnMetricWithNoFilters(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newBaseRegistry()
	reg.register(BaseMetricDescriptor{
		Name:         "file-size",
		Kind:         metric.Quantity,
		Level:        metric.LevelFile,
		Filters:      nil,
		Aggregations: []metric.AggregationName{metric.AggSum},
	})

	expr := metric.MetricExpression{Filter: "stdlib", Base: "file-size", Aggregation: metric.AggSum}
	_, err := resolveExpressionWith(reg, expr, metric.LevelDirectory)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("has no filters")))
}
