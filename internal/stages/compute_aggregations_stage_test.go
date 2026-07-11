package stages_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// ---------------------------------------------------------------------------
// RunAggregations tests
// ---------------------------------------------------------------------------

func TestRunAggregations_NoExpressionsReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := &stages.CommonState{
		Root:      &model.Directory{Name: "root"},
		Requested: stages.RequestedMetrics{},
	}

	g.Expect(stages.RunAggregations(c)).To(Succeed())
}

func TestRunAggregations_WithExpressionsComputesResult(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			fileWithQuantity("a.go", 100),
			fileWithQuantity("b.go", 200),
		},
	}

	resolved := resolveMetricForTest(t, "file-size.sum")

	c := &stages.CommonState{
		Root: root,
		Requested: stages.RequestedMetrics{
			Expressions: []provider.ResolvedMetric{resolved},
		},
	}

	g.Expect(stages.RunAggregations(c)).To(Succeed())

	val, ok := root.Quantity(metric.Name("file-size.sum"))
	g.Expect(ok).To(BeTrue())
	g.Expect(val).To(Equal(int64(300)))
}

func TestRunAggregations_InvalidExpressionReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			fileWithClassification("a.go", "go"),
		},
	}

	// Sum aggregation on a classification metric is unsupported and will error.
	resolved := provider.ResolvedMetric{
		Expression: metric.MetricExpression{
			Base:        "file-type",
			Aggregation: metric.AggSum,
		},
		Descriptor: provider.BaseMetricDescriptor{
			Name:  "file-type",
			Kind:  metric.Classification,
			Level: metric.LevelFile,
		},
		SourceLevel:      metric.LevelFile,
		TargetLevel:      metric.LevelDirectory,
		ResultKind:       metric.Classification,
		ResultName:       "file-type.sum",
		NeedsAggregation: true,
	}

	c := &stages.CommonState{
		Root: root,
		Requested: stages.RequestedMetrics{
			Expressions: []provider.ResolvedMetric{resolved},
		},
	}

	g.Expect(stages.RunAggregations(c)).To(HaveOccurred())
}

// ---------------------------------------------------------------------------
// PopulateDeclarations tests
// ---------------------------------------------------------------------------

func TestPopulateDeclarations_NoDeclarationExpressionsIsNoop(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name:  "root",
		Files: []*model.File{{Name: "a.go"}},
	}

	c := &stages.CommonState{
		Root:      root,
		Requested: stages.RequestedMetrics{},
	}

	// Should succeed without modifying anything.
	g.Expect(stages.PopulateDeclarations(c)).To(Succeed())

	// No declarations should have been populated on the file.
	g.Expect(root.Files[0].Declarations).To(BeEmpty())
}

func TestPopulateDeclarations_WithDeclarationExpressionsDoesNotPanic(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// A file with no content: PopulateDeclarations should not panic even when
	// HasDeclarationExpressions() is true and the files have no source content.
	root := &model.Directory{
		Name:  "root",
		Files: []*model.File{{Name: "empty.go"}},
	}

	declExpr := provider.ResolvedMetric{
		Expression: metric.MetricExpression{
			Base:        "cyclomatic-complexity",
			Aggregation: metric.AggMean,
		},
		Descriptor: provider.BaseMetricDescriptor{
			Name:  "cyclomatic-complexity",
			Kind:  metric.Quantity,
			Level: metric.LevelDeclaration,
		},
		SourceLevel: metric.LevelDeclaration,
		TargetLevel: metric.LevelDirectory,
		ResultKind:  metric.Measure,
		ResultName:  "cyclomatic-complexity.mean",
	}

	c := &stages.CommonState{
		Root: root,
		Requested: stages.RequestedMetrics{
			Expressions: []provider.ResolvedMetric{declExpr},
		},
	}

	g.Expect(stages.PopulateDeclarations(c)).To(Succeed())
}
