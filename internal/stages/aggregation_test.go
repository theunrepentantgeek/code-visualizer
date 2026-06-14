package stages_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func TestComputeAggregations_SumFileSize(t *testing.T) {
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

	err := stages.ComputeAggregations(root, []provider.ResolvedMetric{resolved})
	g.Expect(err).To(Succeed())

	val, ok := root.Quantity(metric.Name("file-size.sum"))
	g.Expect(ok).To(BeTrue())
	g.Expect(val).To(Equal(int64(300)))
}

func TestComputeAggregations_MeanFileSize(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			fileWithQuantity("a.go", 100),
			fileWithQuantity("b.go", 300),
		},
	}

	resolved := resolveMetricForTest(t, "file-size.mean")

	err := stages.ComputeAggregations(root, []provider.ResolvedMetric{resolved})
	g.Expect(err).To(Succeed())

	val, ok := root.Measure(metric.Name("file-size.mean"))
	g.Expect(ok).To(BeTrue())
	g.Expect(val).To(BeNumerically("~", 200.0, 0.001))
}

func TestComputeAggregations_RecursiveCollection(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	child := &model.Directory{
		Name: "child",
		Files: []*model.File{
			fileWithQuantity("c.go", 50),
		},
	}
	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			fileWithQuantity("a.go", 100),
		},
		Dirs: []*model.Directory{child},
	}

	resolved := resolveMetricForTest(t, "file-size.sum")

	err := stages.ComputeAggregations(root, []provider.ResolvedMetric{resolved})
	g.Expect(err).To(Succeed())

	rootVal, ok := root.Quantity(metric.Name("file-size.sum"))
	g.Expect(ok).To(BeTrue())
	g.Expect(rootVal).To(Equal(int64(150)))

	childVal, ok := child.Quantity(metric.Name("file-size.sum"))
	g.Expect(ok).To(BeTrue())
	g.Expect(childVal).To(Equal(int64(50)))
}

func TestComputeAggregations_EmptyDirectoryNoMetricSet(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{Name: "empty"}

	resolved := resolveMetricForTest(t, "file-size.sum")

	err := stages.ComputeAggregations(root, []provider.ResolvedMetric{resolved})
	g.Expect(err).To(Succeed())

	_, ok := root.Quantity(metric.Name("file-size.sum"))
	g.Expect(ok).To(BeFalse())
}

func TestComputeAggregations_MaxFileSize(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			fileWithQuantity("a.go", 100),
			fileWithQuantity("b.go", 500),
			fileWithQuantity("c.go", 200),
		},
	}

	resolved := resolveMetricForTest(t, "file-size.max")

	err := stages.ComputeAggregations(root, []provider.ResolvedMetric{resolved})
	g.Expect(err).To(Succeed())

	val, ok := root.Quantity(metric.Name("file-size.max"))
	g.Expect(ok).To(BeTrue())
	g.Expect(val).To(Equal(int64(500)))
}

func TestComputeAggregations_ModeClassification(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			fileWithClassification("a.go", "go"),
			fileWithClassification("b.go", "go"),
			fileWithClassification("c.py", "python"),
		},
	}

	resolved := resolveMetricForTest(t, "file-type.mode")

	err := stages.ComputeAggregations(root, []provider.ResolvedMetric{resolved})
	g.Expect(err).To(Succeed())

	val, ok := root.Classification(metric.Name("file-type.mode"))
	g.Expect(ok).To(BeTrue())
	g.Expect(val).To(Equal("go"))
}

func TestComputeAggregations_DistinctClassification(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			fileWithClassification("a.go", "go"),
			fileWithClassification("b.go", "go"),
			fileWithClassification("c.py", "python"),
		},
	}

	resolved := resolveMetricForTest(t, "file-type.distinct")

	err := stages.ComputeAggregations(root, []provider.ResolvedMetric{resolved})
	g.Expect(err).To(Succeed())

	val, ok := root.Quantity(metric.Name("file-type.distinct"))
	g.Expect(ok).To(BeTrue())
	g.Expect(val).To(Equal(int64(2)))
}

func TestComputeAggregations_UnsupportedLevelReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{Name: "root"}

	resolved := provider.ResolvedMetric{
		Expression:       metric.MetricExpression{Base: "cyclomatic-complexity", Aggregation: metric.AggMean},
		SourceLevel:      metric.LevelDeclaration,
		TargetLevel:      metric.LevelDirectory,
		ResultKind:       metric.Measure,
		ResultName:       "cyclomatic-complexity.mean",
		NeedsAggregation: true,
	}

	err := stages.ComputeAggregations(root, []provider.ResolvedMetric{resolved})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("declaration-level")))
}

func TestComputeAggregations_UnsupportedClassificationAggregationReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			fileWithClassification("a.go", "go"),
		},
	}

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

	err := stages.ComputeAggregations(root, []provider.ResolvedMetric{resolved})
	g.Expect(err).To(MatchError(ContainSubstring(`classification aggregation "sum"`)))
}

func TestComputeAggregations_NoExpressionsReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{Name: "root"}

	err := stages.ComputeAggregations(root, nil)
	g.Expect(err).To(Succeed())
}

func resolveMetricForTest(t *testing.T, raw string) provider.ResolvedMetric {
	t.Helper()

	expr, err := metric.ParseExpression(raw)
	if err != nil {
		t.Fatalf("ParseExpression(%q): %v", raw, err)
	}

	resolved, err := provider.ResolveExpression(expr, metric.LevelDirectory)
	if err != nil {
		t.Fatalf("ResolveExpression(%q): %v", raw, err)
	}

	return resolved
}

func fileWithQuantity(name string, value int64) *model.File {
	f := &model.File{Name: name}
	f.SetQuantity("file-size", value)

	return f
}

func fileWithClassification(name string, value string) *model.File {
	f := &model.File{Name: name}
	f.SetClassification("file-type", value)

	return f
}
