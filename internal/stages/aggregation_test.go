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

func TestComputeAggregations_FilteredFileLevelMetricUsesFilteredKey(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Files store filtered values under the composite key "stdlib.imports"
	f1 := &model.File{Name: "a.go"}
	f1.SetQuantity("stdlib.imports", 3)

	f2 := &model.File{Name: "b.go"}
	f2.SetQuantity("stdlib.imports", 5)

	root := &model.Directory{
		Name:  "root",
		Files: []*model.File{f1, f2},
	}

	resolved := provider.ResolvedMetric{
		Expression: metric.MetricExpression{
			Filter:      "stdlib",
			Base:        "imports",
			Aggregation: metric.AggSum,
		},
		Descriptor: provider.BaseMetricDescriptor{
			Name:  "imports",
			Kind:  metric.Quantity,
			Level: metric.LevelFile,
		},
		SourceLevel:      metric.LevelFile,
		TargetLevel:      metric.LevelDirectory,
		ResultKind:       metric.Quantity,
		ResultName:       "stdlib.imports.sum",
		NeedsAggregation: true,
	}

	err := stages.ComputeAggregations(root, []provider.ResolvedMetric{resolved})
	g.Expect(err).NotTo(HaveOccurred())

	val, ok := root.Quantity("stdlib.imports.sum")
	g.Expect(ok).To(BeTrue())
	g.Expect(val).To(Equal(int64(8)))
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

	// Directory should have the flat count across all files.
	v, ok := root.Quantity("public.methods.count")
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(Equal(int64(2)))

	// Per-file values should also be set.
	fv, fok := root.Files[0].Quantity("public.methods.count")
	g.Expect(fok).To(BeTrue())
	g.Expect(fv).To(Equal(int64(1))) // a.go has 1 public method (Foo)

	fv2, fok2 := root.Files[1].Quantity("public.methods.count")
	g.Expect(fok2).To(BeTrue())
	g.Expect(fv2).To(Equal(int64(1))) // b.go has 1 public method (Baz)
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

// ---------------------------------------------------------------------------
// Declaration-level classification aggregation tests
// ---------------------------------------------------------------------------

func TestComputeAggregations_DeclarationLevel_Classification_ModeAggregation(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	d1 := &model.Declaration{Name: "A", Kind: "function"}
	d1.SetClassification("visibility", "public")

	d2 := &model.Declaration{Name: "B", Kind: "function"}
	d2.SetClassification("visibility", "public")

	d3 := &model.Declaration{Name: "C", Kind: "function"}
	d3.SetClassification("visibility", "private")

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			{Name: "a.go", Declarations: []*model.Declaration{d1, d2, d3}},
		},
	}

	resolved := provider.ResolvedMetric{
		Expression: metric.MetricExpression{
			Base:        "visibility",
			Aggregation: metric.AggMode,
		},
		Descriptor: provider.BaseMetricDescriptor{
			Name:  "visibility",
			Kind:  metric.Classification,
			Level: metric.LevelDeclaration,
		},
		SourceLevel:      metric.LevelDeclaration,
		TargetLevel:      metric.LevelDirectory,
		ResultKind:       metric.Classification,
		ResultName:       "visibility.mode",
		NeedsAggregation: true,
	}

	err := stages.ComputeAggregations(root, []provider.ResolvedMetric{resolved})
	g.Expect(err).To(Succeed())

	// File should have the mode classification from its declarations.
	fv, fok := root.Files[0].Classification("visibility.mode")
	g.Expect(fok).To(BeTrue())
	g.Expect(fv).To(Equal("public"))

	// Directory should aggregate all declarations flat.
	dv, dok := root.Classification("visibility.mode")
	g.Expect(dok).To(BeTrue())
	g.Expect(dv).To(Equal("public"))
}

func TestComputeAggregations_DeclarationLevel_Classification_DistinctAggregation(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	d1 := &model.Declaration{Name: "A", Kind: "function"}
	d1.SetClassification("visibility", "public")

	d2 := &model.Declaration{Name: "B", Kind: "function"}
	d2.SetClassification("visibility", "public")

	d3 := &model.Declaration{Name: "C", Kind: "method"}
	d3.SetClassification("visibility", "private")

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			{Name: "a.go", Declarations: []*model.Declaration{d1, d2}},
			{Name: "b.go", Declarations: []*model.Declaration{d3}},
		},
	}

	resolved := provider.ResolvedMetric{
		Expression: metric.MetricExpression{
			Base:        "visibility",
			Aggregation: metric.AggDistinct,
		},
		Descriptor: provider.BaseMetricDescriptor{
			Name:  "visibility",
			Kind:  metric.Classification,
			Level: metric.LevelDeclaration,
		},
		SourceLevel:      metric.LevelDeclaration,
		TargetLevel:      metric.LevelDirectory,
		ResultKind:       metric.Quantity,
		ResultName:       "visibility.distinct",
		NeedsAggregation: true,
	}

	err := stages.ComputeAggregations(root, []provider.ResolvedMetric{resolved})
	g.Expect(err).To(Succeed())

	// a.go has 2 declarations but only 1 distinct value ("public").
	fv, fok := root.Files[0].Quantity("visibility.distinct")
	g.Expect(fok).To(BeTrue())
	g.Expect(fv).To(Equal(int64(1)))

	// b.go has 1 declaration with 1 distinct value ("private").
	fv2, fok2 := root.Files[1].Quantity("visibility.distinct")
	g.Expect(fok2).To(BeTrue())
	g.Expect(fv2).To(Equal(int64(1)))

	// Directory sees "public" and "private" → 2 distinct values.
	dv, dok := root.Quantity("visibility.distinct")
	g.Expect(dok).To(BeTrue())
	g.Expect(dv).To(Equal(int64(2)))
}

func TestComputeAggregations_DeclarationLevel_Classification_EmptyDeclarationsNoMetricSet(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{Name: "root"}

	resolved := provider.ResolvedMetric{
		Expression: metric.MetricExpression{
			Base:        "visibility",
			Aggregation: metric.AggMode,
		},
		Descriptor: provider.BaseMetricDescriptor{
			Name:  "visibility",
			Kind:  metric.Classification,
			Level: metric.LevelDeclaration,
		},
		SourceLevel:      metric.LevelDeclaration,
		TargetLevel:      metric.LevelDirectory,
		ResultKind:       metric.Classification,
		ResultName:       "visibility.mode",
		NeedsAggregation: true,
	}

	err := stages.ComputeAggregations(root, []provider.ResolvedMetric{resolved})
	g.Expect(err).To(Succeed())

	_, ok := root.Classification("visibility.mode")
	g.Expect(ok).To(BeFalse())
}

func TestComputeAggregations_DeclarationLevel_Classification_UnsupportedAggReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	d1 := &model.Declaration{Name: "A", Kind: "function"}
	d1.SetClassification("visibility", "public")

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			{Name: "a.go", Declarations: []*model.Declaration{d1}},
		},
	}

	resolved := provider.ResolvedMetric{
		Expression: metric.MetricExpression{
			Base:        "visibility",
			Aggregation: metric.AggSum,
		},
		Descriptor: provider.BaseMetricDescriptor{
			Name:  "visibility",
			Kind:  metric.Classification,
			Level: metric.LevelDeclaration,
		},
		SourceLevel:      metric.LevelDeclaration,
		TargetLevel:      metric.LevelDirectory,
		ResultKind:       metric.Classification,
		ResultName:       "visibility.sum",
		NeedsAggregation: true,
	}

	err := stages.ComputeAggregations(root, []provider.ResolvedMetric{resolved})
	g.Expect(err).To(MatchError(ContainSubstring(`classification aggregation "sum"`)))
}

func TestComputeAggregations_DeclarationLevel_Classification_RecursiveSubdir(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	d1 := &model.Declaration{Name: "A", Kind: "function"}
	d1.SetClassification("visibility", "public")

	d2 := &model.Declaration{Name: "B", Kind: "function"}
	d2.SetClassification("visibility", "private")

	d3 := &model.Declaration{Name: "C", Kind: "function"}
	d3.SetClassification("visibility", "public")

	sub := &model.Directory{
		Name:  "sub",
		Files: []*model.File{{Name: "b.go", Declarations: []*model.Declaration{d2, d3}}},
	}

	root := &model.Directory{
		Name:  "root",
		Files: []*model.File{{Name: "a.go", Declarations: []*model.Declaration{d1}}},
		Dirs:  []*model.Directory{sub},
	}

	resolved := provider.ResolvedMetric{
		Expression: metric.MetricExpression{
			Base:        "visibility",
			Aggregation: metric.AggDistinct,
		},
		Descriptor: provider.BaseMetricDescriptor{
			Name:  "visibility",
			Kind:  metric.Classification,
			Level: metric.LevelDeclaration,
		},
		SourceLevel:      metric.LevelDeclaration,
		TargetLevel:      metric.LevelDirectory,
		ResultKind:       metric.Quantity,
		ResultName:       "visibility.distinct",
		NeedsAggregation: true,
	}

	err := stages.ComputeAggregations(root, []provider.ResolvedMetric{resolved})
	g.Expect(err).To(Succeed())

	// sub directory sees "private" and "public" → 2.
	sv, sok := sub.Quantity("visibility.distinct")
	g.Expect(sok).To(BeTrue())
	g.Expect(sv).To(Equal(int64(2)))

	// root directory flat-aggregates all 3 declarations (public, private, public) → 2 distinct.
	rv, rok := root.Quantity("visibility.distinct")
	g.Expect(rok).To(BeTrue())
	g.Expect(rv).To(Equal(int64(2)))
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

// Tests for aggregation operations not yet covered: AggMin and AggRange.
// These fill coverage gaps in applyNumericAggregation, ensuring all supported
// aggregation operations are verified. (Related: issue #551 — hardening the
// aggregation framework against aggregate-of-aggregates for derived metrics.)

func TestComputeAggregations_MinFileSize(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			fileWithQuantity("a.go", 300),
			fileWithQuantity("b.go", 100),
			fileWithQuantity("c.go", 200),
		},
	}
	root.AllFileCount = 3

	resolved := fileSizeResolved(metric.AggMin, metric.Quantity, "file-size.min")
	err := stages.ComputeAggregations(root, []provider.ResolvedMetric{resolved})

	g.Expect(err).NotTo(HaveOccurred())

	v, ok := root.Quantity("file-size.min")
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(Equal(int64(100)))
}

func TestComputeAggregations_RangeFileSize(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			fileWithQuantity("a.go", 50),
			fileWithQuantity("b.go", 150),
		},
	}
	root.AllFileCount = 2

	resolved := fileSizeMeasureResolved(metric.AggRange, "file-size.range")
	err := stages.ComputeAggregations(root, []provider.ResolvedMetric{resolved})

	g.Expect(err).NotTo(HaveOccurred())

	v, ok := root.Measure("file-size.range")
	g.Expect(ok).To(BeTrue())
	// Range = max - min = 150 - 50 = 100
	g.Expect(v).To(BeNumerically("~", 100.0, 0.001))
}

// fileSizeResolved returns a ResolvedMetric for file-size with the given aggregation.
func fileSizeResolved(agg metric.AggregationName, resultKind metric.Kind, resultName metric.Name) provider.ResolvedMetric {
	return provider.ResolvedMetric{
		Expression: metric.MetricExpression{
			Base:        "file-size",
			Aggregation: agg,
		},
		Descriptor: provider.BaseMetricDescriptor{
			Name:  "file-size",
			Kind:  metric.Quantity,
			Level: metric.LevelFile,
		},
		SourceLevel:      metric.LevelFile,
		TargetLevel:      metric.LevelDirectory,
		ResultKind:       resultKind,
		ResultName:       resultName,
		NeedsAggregation: true,
	}
}

// fileSizeMeasureResolved returns a ResolvedMetric for file-size yielding a Measure result.
func fileSizeMeasureResolved(agg metric.AggregationName, resultName metric.Name) provider.ResolvedMetric {
	return provider.ResolvedMetric{
		Expression: metric.MetricExpression{
			Base:        "file-size",
			Aggregation: agg,
		},
		Descriptor: provider.BaseMetricDescriptor{
			Name:  "file-size",
			Kind:  metric.Quantity,
			Level: metric.LevelFile,
		},
		SourceLevel:      metric.LevelFile,
		TargetLevel:      metric.LevelDirectory,
		ResultKind:       metric.Measure,
		ResultName:       resultName,
		NeedsAggregation: true,
	}
}
