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

func TestComputeAggregations_DeclarationLevel_EmptyDirNoError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{Name: "root"}

	resolved := provider.ResolvedMetric{
		Expression: metric.MetricExpression{Base: "cyclomatic-complexity", Aggregation: metric.AggMean},
		Descriptor: provider.BaseMetricDescriptor{
			Name:  "cyclomatic-complexity",
			Kind:  metric.Quantity,
			Level: metric.LevelDeclaration,
		},
		SourceLevel:      metric.LevelDeclaration,
		TargetLevel:      metric.LevelDirectory,
		ResultKind:       metric.Measure,
		ResultName:       "cyclomatic-complexity.mean",
		NeedsAggregation: true,
	}

	err := stages.ComputeAggregations(root, []provider.ResolvedMetric{resolved})
	g.Expect(err).NotTo(HaveOccurred())

	// No declarations → no metric set
	_, ok := root.Measure("cyclomatic-complexity.mean")
	g.Expect(ok).To(BeFalse())
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

// ---------------------------------------------------------------------------
// Declaration-level aggregation tests
// ---------------------------------------------------------------------------

func TestComputeAggregations_DeclarationLevel_CountPublicMethods(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	decl1 := &model.Declaration{Name: "Foo", Kind: "method", Visibility: "public"}
	decl2 := &model.Declaration{Name: "bar", Kind: "method", Visibility: "private"}
	decl3 := &model.Declaration{Name: "Baz", Kind: "method", Visibility: "public"}

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			{Name: "a.go", Declarations: []*model.Declaration{decl1, decl2}},
			{Name: "b.go", Declarations: []*model.Declaration{decl3}},
		},
	}

	filterFunc := func(filter metric.FilterName, node any) bool {
		d, ok := node.(*model.Declaration)
		if !ok {
			return false
		}

		return d.MatchesFilter(filter)
	}

	resolved := provider.ResolvedMetric{
		Expression: metric.MetricExpression{
			Filter:      "public",
			Base:        "methods",
			Aggregation: metric.AggCount,
		},
		Descriptor: provider.BaseMetricDescriptor{
			Name:       "methods",
			Kind:       metric.Quantity,
			Level:      metric.LevelDeclaration,
			Filters:    []metric.FilterName{"public", "private"},
			FilterFunc: filterFunc,
		},
		SourceLevel:      metric.LevelDeclaration,
		TargetLevel:      metric.LevelDirectory,
		ResultKind:       metric.Quantity,
		ResultName:       "public.methods.count",
		NeedsAggregation: true,
	}

	err := stages.ComputeAggregations(root, []provider.ResolvedMetric{resolved})
	g.Expect(err).NotTo(HaveOccurred())

	v, ok := root.Quantity("public.methods.count")
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(Equal(int64(2)))
}

func TestComputeAggregations_DeclarationLevel_MeanCyclomaticComplexity(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	d1 := &model.Declaration{Name: "A", Kind: "function", Visibility: "public"}
	d1.SetQuantity("cyclomatic-complexity", 2)

	d2 := &model.Declaration{Name: "B", Kind: "function", Visibility: "public"}
	d2.SetQuantity("cyclomatic-complexity", 4)

	d3 := &model.Declaration{Name: "C", Kind: "method", Visibility: "private"}
	d3.SetQuantity("cyclomatic-complexity", 6)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			{Name: "a.go", Declarations: []*model.Declaration{d1, d2}},
			{Name: "b.go", Declarations: []*model.Declaration{d3}},
		},
	}

	resolved := provider.ResolvedMetric{
		Expression: metric.MetricExpression{
			Base:        "cyclomatic-complexity",
			Aggregation: metric.AggMean,
		},
		Descriptor: provider.BaseMetricDescriptor{
			Name:  "cyclomatic-complexity",
			Kind:  metric.Quantity,
			Level: metric.LevelDeclaration,
		},
		SourceLevel:      metric.LevelDeclaration,
		TargetLevel:      metric.LevelDirectory,
		ResultKind:       metric.Measure,
		ResultName:       "cyclomatic-complexity.mean",
		NeedsAggregation: true,
	}

	err := stages.ComputeAggregations(root, []provider.ResolvedMetric{resolved})
	g.Expect(err).NotTo(HaveOccurred())

	v, ok := root.Measure("cyclomatic-complexity.mean")
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(BeNumerically("~", 4.0, 0.01))
}

func TestComputeAggregations_DeclarationLevel_FilterExcludesNonMatching(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	d1 := &model.Declaration{Name: "Pub", Kind: "function", Visibility: "public"}
	d1.SetQuantity("cyclomatic-complexity", 10)

	d2 := &model.Declaration{Name: "priv", Kind: "function", Visibility: "private"}
	d2.SetQuantity("cyclomatic-complexity", 2)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			{Name: "a.go", Declarations: []*model.Declaration{d1, d2}},
		},
	}

	filterFunc := func(filter metric.FilterName, node any) bool {
		d, ok := node.(*model.Declaration)
		if !ok {
			return false
		}

		return d.MatchesFilter(filter)
	}

	resolved := provider.ResolvedMetric{
		Expression: metric.MetricExpression{
			Filter:      "public",
			Base:        "cyclomatic-complexity",
			Aggregation: metric.AggMax,
		},
		Descriptor: provider.BaseMetricDescriptor{
			Name:       "cyclomatic-complexity",
			Kind:       metric.Quantity,
			Level:      metric.LevelDeclaration,
			Filters:    []metric.FilterName{"public", "private"},
			FilterFunc: filterFunc,
		},
		SourceLevel:      metric.LevelDeclaration,
		TargetLevel:      metric.LevelDirectory,
		ResultKind:       metric.Quantity,
		ResultName:       "public.cyclomatic-complexity.max",
		NeedsAggregation: true,
	}

	err := stages.ComputeAggregations(root, []provider.ResolvedMetric{resolved})
	g.Expect(err).NotTo(HaveOccurred())

	v, ok := root.Quantity("public.cyclomatic-complexity.max")
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(Equal(int64(10)))
}

// ---------------------------------------------------------------------------
// Commit-level aggregation tests
// ---------------------------------------------------------------------------

func TestComputeAggregations_CommitLevel_MaxLinesChanged(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c1 := &model.Commit{Hash: "aaa"}
	c1.SetQuantity("lines-changed", 10)

	c2 := &model.Commit{Hash: "bbb"}
	c2.SetQuantity("lines-changed", 50)

	c3 := &model.Commit{Hash: "ccc"}
	c3.SetQuantity("lines-changed", 25)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			{Name: "a.go", Commits: []*model.Commit{c1, c2}},
			{Name: "b.go", Commits: []*model.Commit{c3}},
		},
	}

	resolved := provider.ResolvedMetric{
		Expression: metric.MetricExpression{
			Base:        "lines-changed",
			Aggregation: metric.AggMax,
		},
		Descriptor: provider.BaseMetricDescriptor{
			Name:  "lines-changed",
			Kind:  metric.Quantity,
			Level: metric.LevelCommit,
		},
		SourceLevel:      metric.LevelCommit,
		TargetLevel:      metric.LevelDirectory,
		ResultKind:       metric.Quantity,
		ResultName:       "lines-changed.max",
		NeedsAggregation: true,
	}

	err := stages.ComputeAggregations(root, []provider.ResolvedMetric{resolved})
	g.Expect(err).NotTo(HaveOccurred())

	v, ok := root.Quantity("lines-changed.max")
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(Equal(int64(50)))
}

func TestComputeAggregations_CommitLevel_MeanLinesAdded(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c1 := &model.Commit{Hash: "a"}
	c1.SetQuantity("lines-added", 10)

	c2 := &model.Commit{Hash: "b"}
	c2.SetQuantity("lines-added", 20)

	c3 := &model.Commit{Hash: "c"}
	c3.SetQuantity("lines-added", 30)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			{Name: "x.go", Commits: []*model.Commit{c1, c2, c3}},
		},
	}

	resolved := provider.ResolvedMetric{
		Expression: metric.MetricExpression{
			Base:        "lines-added",
			Aggregation: metric.AggMean,
		},
		Descriptor: provider.BaseMetricDescriptor{
			Name:  "lines-added",
			Kind:  metric.Quantity,
			Level: metric.LevelCommit,
		},
		SourceLevel:      metric.LevelCommit,
		TargetLevel:      metric.LevelDirectory,
		ResultKind:       metric.Measure,
		ResultName:       "lines-added.mean",
		NeedsAggregation: true,
	}

	err := stages.ComputeAggregations(root, []provider.ResolvedMetric{resolved})
	g.Expect(err).NotTo(HaveOccurred())

	v, ok := root.Measure("lines-added.mean")
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(BeNumerically("~", 20.0, 0.01))
}
