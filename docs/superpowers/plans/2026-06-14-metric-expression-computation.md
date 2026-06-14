# Metric Expression Computation — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make metric expressions (`[filter.]base-metric[.aggregation]`) actually compute aggregated values at directory level, so expressions like `file-bytes.sum` produce correct results in visualizations.

**Architecture:** A new `stages.ComputeAggregations` stage runs after `stages.RunProviders`. It walks the directory tree, collects source-level values from all descendant files, applies the aggregation function, and stores the result on each directory's MetricContainer. The `CommonState.Requested` field changes from `[]metric.Name` to a `RequestedMetrics` struct that separates base provider names from aggregation expressions.

**Tech Stack:** Go 1.26.1, Gomega (assertions), eris (error wrapping), existing metric/provider/model packages.

---

## File Structure

| File | Responsibility |
|------|---------------|
| `internal/stages/requested.go` (create) | `RequestedMetrics` struct + `ClassifyRequestedMetrics` function |
| `internal/stages/requested_test.go` (create) | Tests for classification logic |
| `internal/stages/aggregation.go` (create) | `ComputeAggregations` stage implementation |
| `internal/stages/aggregation_test.go` (create) | Tests for aggregation computation |
| `internal/stages/common.go` (modify) | Change `Requested []metric.Name` → `Requested RequestedMetrics` |
| `internal/stages/providers.go` (modify) | Use `Requested.LegacyNames()` for provider.Run |
| `internal/stages/metrics.go` (modify) | `CollectRequestedMetrics` returns `RequestedMetrics` |
| `internal/stages/metrics_test.go` (modify) | Update test assertions for new return type |
| `internal/treemap/stages.go` (modify) | Adapt to new Requested type |
| `internal/radialtree/stages.go` (modify) | Adapt to new Requested type |
| `internal/bubbletree/stages.go` (modify) | Adapt to new Requested type |
| `internal/scatter/stages.go` (modify) | Adapt to new Requested type |
| `internal/scatter/stages_test.go` (modify) | Update test assertions |
| `internal/spiral/stages.go` (modify) | Adapt to new Requested type |
| `cmd/codeviz/treemap_cmd.go` (modify) | Add `stages.ComputeAggregations` after `RunProviders` |
| `cmd/codeviz/radialtree_cmd.go` (modify) | Add `stages.ComputeAggregations` after `RunProviders` |
| `cmd/codeviz/bubbletree_cmd.go` (modify) | Add `stages.ComputeAggregations` after `RunProviders` |
| `cmd/codeviz/scatter_cmd.go` (modify) | Add `stages.ComputeAggregations` after `RunProviders` |
| `cmd/codeviz/spiral_cmd.go` (modify) | Add `stages.ComputeAggregations` after `RunProviders` |

---

### Task 1: RequestedMetrics Type

**Files:**
- Create: `internal/stages/requested.go`
- Create: `internal/stages/requested_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/stages/requested_test.go
package stages_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func TestClassifyRequestedMetrics_BareMetricGoesToLegacy(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	names := []metric.Name{"file-bytes"}
	result := stages.ClassifyRequestedMetrics(names, metric.LevelDirectory)

	g.Expect(result.LegacyNames()).To(ContainElement(metric.Name("file-bytes")))
	g.Expect(result.Expressions).To(BeEmpty())
}

func TestClassifyRequestedMetrics_ExpressionWithAggregation(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	names := []metric.Name{"file-bytes.sum"}
	result := stages.ClassifyRequestedMetrics(names, metric.LevelDirectory)

	g.Expect(result.Expressions).To(HaveLen(1))
	g.Expect(result.Expressions[0].Expression.Base).To(Equal(metric.Name("file-bytes")))
	g.Expect(result.Expressions[0].Expression.Aggregation).To(Equal(metric.AggSum))
	g.Expect(result.BaseMetrics).To(ContainElement(metric.Name("file-bytes")))
}

func TestClassifyRequestedMetrics_UnresolvableExpressionGoesToLegacy(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// "file-bytes" without aggregation at directory level requires aggregation,
	// so it won't resolve. It should fall through to legacy.
	names := []metric.Name{"not-a-real-metric"}
	result := stages.ClassifyRequestedMetrics(names, metric.LevelDirectory)

	g.Expect(result.LegacyNames()).To(ContainElement(metric.Name("not-a-real-metric")))
	g.Expect(result.Expressions).To(BeEmpty())
}

func TestClassifyRequestedMetrics_MixedSet(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	names := []metric.Name{"file-bytes", "file-bytes.sum", "file-lines.max"}
	result := stages.ClassifyRequestedMetrics(names, metric.LevelDirectory)

	g.Expect(result.LegacyNames()).To(ContainElement(metric.Name("file-bytes")))
	g.Expect(result.Expressions).To(HaveLen(2))
	g.Expect(result.BaseMetrics).To(ContainElement(metric.Name("file-bytes")))
	g.Expect(result.BaseMetrics).To(ContainElement(metric.Name("file-lines")))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/stages/ -run TestClassifyRequested -v`
Expected: compilation failure (type not defined)

- [ ] **Step 3: Write the implementation**

```go
// internal/stages/requested.go
package stages

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// RequestedMetrics separates user-requested metric names into expressions
// that need aggregation and legacy names that go directly to provider.Run.
type RequestedMetrics struct {
	// BaseMetrics are the base metric names extracted from resolved expressions.
	// These must be run by provider.Run to populate source-level data.
	BaseMetrics []metric.Name
	// Expressions are fully resolved metrics that need aggregation computation.
	Expressions []provider.ResolvedMetric
	// Legacy are metric names that couldn't be parsed/resolved as expressions
	// and should be passed to provider.Run as-is (backward compatibility).
	Legacy []metric.Name
}

// LegacyNames returns all metric names that should be passed to provider.Run:
// both the base metrics needed by expressions AND the legacy unresolved names.
func (r RequestedMetrics) LegacyNames() []metric.Name {
	seen := make(map[metric.Name]bool, len(r.BaseMetrics)+len(r.Legacy))

	var result []metric.Name

	for _, n := range r.BaseMetrics {
		if !seen[n] {
			seen[n] = true
			result = append(result, n)
		}
	}

	for _, n := range r.Legacy {
		if !seen[n] {
			seen[n] = true
			result = append(result, n)
		}
	}

	return result
}

// ClassifyRequestedMetrics takes a flat list of metric name strings and
// classifies each as either a resolvable expression or a legacy metric name.
func ClassifyRequestedMetrics(names []metric.Name, targetLevel metric.MetricLevel) RequestedMetrics {
	var result RequestedMetrics

	baseSeen := make(map[metric.Name]bool)

	for _, name := range names {
		expr, parseErr := metric.ParseExpression(string(name))
		if parseErr != nil {
			result.Legacy = append(result.Legacy, name)
			continue
		}

		resolved, resolveErr := provider.ResolveExpression(expr, targetLevel)
		if resolveErr != nil {
			result.Legacy = append(result.Legacy, name)
			continue
		}

		if !resolved.NeedsAggregation {
			// Bare metric at native level — treat as legacy (provider handles it directly).
			result.Legacy = append(result.Legacy, name)
			continue
		}

		result.Expressions = append(result.Expressions, resolved)

		if !baseSeen[expr.Base] {
			baseSeen[expr.Base] = true
			result.BaseMetrics = append(result.BaseMetrics, expr.Base)
		}
	}

	return result
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/stages/ -run TestClassifyRequested -v`
Expected: all 4 tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/stages/requested.go internal/stages/requested_test.go
git commit -m "feat: add RequestedMetrics type and ClassifyRequestedMetrics"
```

---

### Task 2: ComputeAggregations Stage

**Files:**
- Create: `internal/stages/aggregation.go`
- Create: `internal/stages/aggregation_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/stages/aggregation_test.go
package stages_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func TestComputeAggregations_SumFileBytes(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			fileWithQuantity("a.go", "file-bytes", 100),
			fileWithQuantity("b.go", "file-bytes", 200),
		},
	}

	expr, _ := metric.ParseExpression("file-bytes.sum")
	resolved, _ := provider.ResolveExpression(expr, metric.LevelDirectory)

	err := stages.ComputeAggregations(root, []provider.ResolvedMetric{resolved})
	g.Expect(err).To(Succeed())

	val, ok := root.Measure(metric.Name("file-bytes.sum"))
	g.Expect(ok).To(BeTrue())
	g.Expect(val).To(BeNumerically("~", 300.0, 0.001))
}

func TestComputeAggregations_MeanFileBytes(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			fileWithQuantity("a.go", "file-bytes", 100),
			fileWithQuantity("b.go", "file-bytes", 300),
		},
	}

	expr, _ := metric.ParseExpression("file-bytes.mean")
	resolved, _ := provider.ResolveExpression(expr, metric.LevelDirectory)

	err := stages.ComputeAggregations(root, []provider.ResolvedMetric{resolved})
	g.Expect(err).To(Succeed())

	val, ok := root.Measure(metric.Name("file-bytes.mean"))
	g.Expect(ok).To(BeTrue())
	g.Expect(val).To(BeNumerically("~", 200.0, 0.001))
}

func TestComputeAggregations_RecursiveCollection(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	child := &model.Directory{
		Name: "child",
		Files: []*model.File{
			fileWithQuantity("c.go", "file-bytes", 50),
		},
	}
	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			fileWithQuantity("a.go", "file-bytes", 100),
		},
		Dirs: []*model.Directory{child},
	}

	expr, _ := metric.ParseExpression("file-bytes.sum")
	resolved, _ := provider.ResolveExpression(expr, metric.LevelDirectory)

	err := stages.ComputeAggregations(root, []provider.ResolvedMetric{resolved})
	g.Expect(err).To(Succeed())

	// Root should include file in child directory.
	rootVal, ok := root.Measure(metric.Name("file-bytes.sum"))
	g.Expect(ok).To(BeTrue())
	g.Expect(rootVal).To(BeNumerically("~", 150.0, 0.001))

	// Child should have its own aggregate.
	childVal, ok := child.Measure(metric.Name("file-bytes.sum"))
	g.Expect(ok).To(BeTrue())
	g.Expect(childVal).To(BeNumerically("~", 50.0, 0.001))
}

func TestComputeAggregations_EmptyDirectoryNoMetricSet(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{Name: "empty"}

	expr, _ := metric.ParseExpression("file-bytes.sum")
	resolved, _ := provider.ResolveExpression(expr, metric.LevelDirectory)

	err := stages.ComputeAggregations(root, []provider.ResolvedMetric{resolved})
	g.Expect(err).To(Succeed())

	_, ok := root.Measure(metric.Name("file-bytes.sum"))
	g.Expect(ok).To(BeFalse())
}

func TestComputeAggregations_MaxFileBytes(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			fileWithQuantity("a.go", "file-bytes", 100),
			fileWithQuantity("b.go", "file-bytes", 500),
			fileWithQuantity("c.go", "file-bytes", 200),
		},
	}

	expr, _ := metric.ParseExpression("file-bytes.max")
	resolved, _ := provider.ResolveExpression(expr, metric.LevelDirectory)

	err := stages.ComputeAggregations(root, []provider.ResolvedMetric{resolved})
	g.Expect(err).To(Succeed())

	val, ok := root.Measure(metric.Name("file-bytes.max"))
	g.Expect(ok).To(BeTrue())
	g.Expect(val).To(BeNumerically("~", 500.0, 0.001))
}

func TestComputeAggregations_ModeClassification(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			fileWithClassification("a.go", "file-type", "go"),
			fileWithClassification("b.go", "file-type", "go"),
			fileWithClassification("c.py", "file-type", "python"),
		},
	}

	expr, _ := metric.ParseExpression("file-type.mode")
	resolved, _ := provider.ResolveExpression(expr, metric.LevelDirectory)

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
			fileWithClassification("a.go", "file-type", "go"),
			fileWithClassification("b.go", "file-type", "go"),
			fileWithClassification("c.py", "file-type", "python"),
		},
	}

	expr, _ := metric.ParseExpression("file-type.distinct")
	resolved, _ := provider.ResolveExpression(expr, metric.LevelDirectory)

	err := stages.ComputeAggregations(root, []provider.ResolvedMetric{resolved})
	g.Expect(err).To(Succeed())

	val, ok := root.Quantity(metric.Name("file-type.distinct"))
	g.Expect(ok).To(BeTrue())
	g.Expect(val).To(Equal(int64(2)))
}

// --- helpers ---

func fileWithQuantity(name string, metricName metric.Name, value int64) *model.File {
	f := &model.File{Name: name}
	f.SetQuantity(metricName, value)

	return f
}

func fileWithClassification(name string, metricName metric.Name, value string) *model.File {
	f := &model.File{Name: name}
	f.SetClassification(metricName, value)

	return f
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/stages/ -run TestComputeAggregations -v`
Expected: compilation failure (ComputeAggregations not defined)

- [ ] **Step 3: Write the implementation**

```go
// internal/stages/aggregation.go
package stages

import (
	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// ComputeAggregations walks the directory tree and computes aggregated metric
// values for each resolved expression. Each directory gets its own aggregate
// computed from all its descendant source-level nodes.
func ComputeAggregations(root *model.Directory, expressions []provider.ResolvedMetric) error {
	if len(expressions) == 0 {
		return nil
	}

	for _, resolved := range expressions {
		if resolved.SourceLevel != metric.LevelFile {
			return eris.Errorf(
				"aggregation of %s-level metric %q is not yet supported (requires model changes)",
				resolved.SourceLevel, resolved.Expression.Base,
			)
		}

		aggregateDirectory(root, resolved)
	}

	return nil
}

// aggregateDirectory recursively computes the aggregate for one directory.
func aggregateDirectory(dir *model.Directory, resolved provider.ResolvedMetric) {
	// Recurse into children first (post-order not required, but keeps it tidy).
	for _, child := range dir.Dirs {
		aggregateDirectory(child, resolved)
	}

	switch resolved.ResultKind {
	case metric.Quantity, metric.Measure:
		aggregateNumeric(dir, resolved)
	case metric.Classification:
		aggregateClassification(dir, resolved)
	}
}

// aggregateNumeric collects numeric values from all descendant files and applies
// the aggregation function.
func aggregateNumeric(dir *model.Directory, resolved provider.ResolvedMetric) {
	values := collectNumericValues(dir, resolved.Expression.Base, resolved.Descriptor.Kind)
	if len(values) == 0 {
		return
	}

	result := applyNumericAggregation(resolved.Expression.Aggregation, values)

	switch resolved.ResultKind {
	case metric.Quantity:
		dir.SetQuantity(resolved.ResultName, int64(result))
	case metric.Measure:
		dir.SetMeasure(resolved.ResultName, result)
	}
}

// aggregateClassification collects string values from all descendant files and
// applies mode or distinct.
func aggregateClassification(dir *model.Directory, resolved provider.ResolvedMetric) {
	values := collectClassificationValues(dir, resolved.Expression.Base)
	if len(values) == 0 {
		return
	}

	switch resolved.Expression.Aggregation {
	case metric.AggMode:
		dir.SetClassification(resolved.ResultName, metric.AggregateMode(values))
	case metric.AggDistinct:
		dir.SetQuantity(resolved.ResultName, int64(metric.AggregateDistinct(values)))
	}
}

// collectNumericValues walks all descendant files and collects the named metric
// as float64 values.
func collectNumericValues(dir *model.Directory, name metric.Name, kind metric.Kind) []float64 {
	var values []float64

	model.WalkFiles(dir, func(f *model.File) {
		switch kind {
		case metric.Quantity:
			if v, ok := f.Quantity(name); ok {
				values = append(values, float64(v))
			}
		case metric.Measure:
			if v, ok := f.Measure(name); ok {
				values = append(values, v)
			}
		}
	})

	return values
}

// collectClassificationValues walks all descendant files and collects the named
// classification metric values.
func collectClassificationValues(dir *model.Directory, name metric.Name) []string {
	var values []string

	model.WalkFiles(dir, func(f *model.File) {
		if v, ok := f.Classification(name); ok {
			values = append(values, v)
		}
	})

	return values
}

// applyNumericAggregation dispatches to the appropriate aggregation function.
func applyNumericAggregation(agg metric.AggregationName, values []float64) float64 {
	switch agg {
	case metric.AggSum:
		return metric.AggregateSum(values)
	case metric.AggMin:
		return metric.AggregateMin(values)
	case metric.AggMax:
		return metric.AggregateMax(values)
	case metric.AggMean:
		return metric.AggregateMean(values)
	case metric.AggCount:
		return metric.AggregateCount(values)
	case metric.AggRange:
		return metric.AggregateRange(values)
	default:
		return 0
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/stages/ -run TestComputeAggregations -v`
Expected: all 7 tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/stages/aggregation.go internal/stages/aggregation_test.go
git commit -m "feat: add ComputeAggregations stage"
```

---

### Task 3: Change CommonState.Requested to RequestedMetrics

**Files:**
- Modify: `internal/stages/common.go`
- Modify: `internal/stages/providers.go`

- [ ] **Step 1: Update CommonState**

In `internal/stages/common.go`, change line 56:

```go
// Before:
Requested     []metric.Name    // viz-specific ResolveMetrics

// After:
Requested     RequestedMetrics // viz-specific ResolveMetrics
```

- [ ] **Step 2: Update RunProviders to use LegacyNames()**

In `internal/stages/providers.go`, change the `provider.Run` call:

```go
// Before:
return eris.Wrap(provider.Run(c.Root, c.Requested, metric.File, metricProg), "failed to load metrics")

// After:
return eris.Wrap(provider.Run(c.Root, c.Requested.LegacyNames(), metric.File, metricProg), "failed to load metrics")
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/stages/`
Expected: compilation failure — callers that assign `[]metric.Name` to `c.Requested` must be updated (Tasks 4+5 fix this).

- [ ] **Step 4: Commit (will not compile yet — that's expected)**

```bash
git add internal/stages/common.go internal/stages/providers.go
git commit -m "refactor: change CommonState.Requested to RequestedMetrics

Callers updated in subsequent commits."
```

---

### Task 4: Update CollectRequestedMetrics

**Files:**
- Modify: `internal/stages/metrics.go`
- Modify: `internal/stages/metrics_test.go`

- [ ] **Step 1: Update CollectRequestedMetrics to return RequestedMetrics**

Replace the function in `internal/stages/metrics.go`:

```go
// CollectRequestedMetrics returns the classified set of metrics
// implied by size + optional fill + optional border specs.
func CollectRequestedMetrics(size metric.Name, fill *config.MetricSpec, border *config.MetricSpec) RequestedMetrics {
	seen := map[metric.Name]bool{size: true}
	names := []metric.Name{size}

	for _, spec := range []*config.MetricSpec{fill, border} {
		if name := spec.MetricName(); name != "" && !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}

	return ClassifyRequestedMetrics(names, metric.LevelDirectory)
}
```

- [ ] **Step 2: Update tests in metrics_test.go**

The existing tests check for `[]metric.Name` return. Update them to use the new `RequestedMetrics` type. Replace the CollectRequestedMetrics test section:

```go
// ---------------------------------------------------------------------------
// CollectRequestedMetrics
// ---------------------------------------------------------------------------

func TestCollectRequestedMetrics_SizeOnly(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	got := stages.CollectRequestedMetrics("file-bytes", nil, nil)

	// file-bytes at directory level requires aggregation, but it's a bare name
	// without explicit aggregation, so it goes to legacy (resolution fails with
	// "requires aggregation" error because no aggregation was specified).
	g.Expect(got.LegacyNames()).To(ContainElement(metric.Name("file-bytes")))
}

func TestCollectRequestedMetrics_SizeAndFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	fill := &config.MetricSpec{Metric: "file-lines"}
	got := stages.CollectRequestedMetrics("file-bytes", fill, nil)

	g.Expect(got.LegacyNames()).To(ContainElements(metric.Name("file-bytes"), metric.Name("file-lines")))
}

func TestCollectRequestedMetrics_SizeAndBorder(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	border := &config.MetricSpec{Metric: "file-type"}
	got := stages.CollectRequestedMetrics("file-bytes", nil, border)

	g.Expect(got.LegacyNames()).To(ContainElements(metric.Name("file-bytes"), metric.Name("file-type")))
}

func TestCollectRequestedMetrics_DeduplicatesFillEqualsSize(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	fill := &config.MetricSpec{Metric: "file-bytes"}
	got := stages.CollectRequestedMetrics("file-bytes", fill, nil)

	g.Expect(got.LegacyNames()).To(HaveLen(1))
	g.Expect(got.LegacyNames()).To(ContainElement(metric.Name("file-bytes")))
}

func TestCollectRequestedMetrics_AllThreeDistinct(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	fill := &config.MetricSpec{Metric: "file-lines"}
	border := &config.MetricSpec{Metric: "file-type"}
	got := stages.CollectRequestedMetrics("file-bytes", fill, border)

	g.Expect(got.LegacyNames()).To(ContainElements(
		metric.Name("file-bytes"),
		metric.Name("file-lines"),
		metric.Name("file-type"),
	))
}

func TestCollectRequestedMetrics_ExpressionClassified(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	fill := &config.MetricSpec{Metric: "file-bytes.sum"}
	got := stages.CollectRequestedMetrics("file-bytes", nil, fill)

	g.Expect(got.Expressions).To(HaveLen(1))
	g.Expect(got.Expressions[0].Expression.Aggregation).To(Equal(metric.AggSum))
	g.Expect(got.LegacyNames()).To(ContainElement(metric.Name("file-bytes")))
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/stages/ -run TestCollectRequestedMetrics -v`
Expected: all tests PASS

- [ ] **Step 4: Commit**

```bash
git add internal/stages/metrics.go internal/stages/metrics_test.go
git commit -m "refactor: CollectRequestedMetrics returns RequestedMetrics"
```

---

### Task 5: Update Viz-Specific ResolveMetrics Functions

**Files:**
- Modify: `internal/treemap/stages.go`
- Modify: `internal/radialtree/stages.go`
- Modify: `internal/bubbletree/stages.go`
- Modify: `internal/scatter/stages.go`
- Modify: `internal/scatter/stages_test.go`
- Modify: `internal/spiral/stages.go`

- [ ] **Step 1: Update treemap/stages.go**

In `internal/treemap/stages.go`, the assignment on line 21 already calls `stages.CollectRequestedMetrics` which now returns `RequestedMetrics`. Since `CommonState.Requested` is now `RequestedMetrics`, this should compile without change. Verify:

```go
// This line should still work as-is:
c.Requested = stages.CollectRequestedMetrics(t.Size, cfg.Fill, cfg.Border)
```

- [ ] **Step 2: Update radialtree/stages.go**

Check `internal/radialtree/stages.go`. It should use the same pattern:
```go
c.Requested = stages.CollectRequestedMetrics(r.DiscSize, cfg.Fill, cfg.Border)
```
This should compile as-is since both sides are now `RequestedMetrics`.

- [ ] **Step 3: Update bubbletree/stages.go**

Same pattern — verify it compiles.

- [ ] **Step 4: Update scatter/stages.go**

The scatter command has its own `collectRequestedMetrics` that returns `[]metric.Name`. Update it:

```go
// Replace the scatter-specific collectRequestedMetrics function.
// It needs to return stages.RequestedMetrics.
func collectRequestedMetrics(xAxis, yAxis, size metric.Name, fill, border *config.MetricSpec) stages.RequestedMetrics {
	seen := map[metric.Name]bool{}

	var names []metric.Name

	for _, n := range []metric.Name{xAxis, yAxis, size} {
		if n != "" && !seen[n] {
			seen[n] = true
			names = append(names, n)
		}
	}

	for _, spec := range []*config.MetricSpec{fill, border} {
		if spec != nil && spec.Metric != "" && !seen[spec.Metric] {
			seen[spec.Metric] = true
			names = append(names, spec.Metric)
		}
	}

	return stages.ClassifyRequestedMetrics(names, metric.LevelDirectory)
}
```

- [ ] **Step 5: Update scatter/stages_test.go**

Update test assertions to use the new type:
```go
// Change assertions from:
g.Expect(common.Requested).To(Equal([]metric.Name{...}))
// To:
g.Expect(common.Requested.LegacyNames()).To(ContainElements(...))
```

- [ ] **Step 6: Update spiral/stages.go**

The spiral has its own `collectRequestedMetrics` returning `[]metric.Name`. Update similarly:

```go
func collectRequestedMetrics(size metric.Name, fill, border *config.MetricSpec) stages.RequestedMetrics {
	if size != "" {
		return stages.CollectRequestedMetrics(size, fill, border)
	}

	seen := map[metric.Name]bool{}

	var names []metric.Name

	for _, spec := range []*config.MetricSpec{fill, border} {
		if spec != nil && spec.Metric != "" && !seen[spec.Metric] {
			seen[spec.Metric] = true
			names = append(names, spec.Metric)
		}
	}

	return stages.ClassifyRequestedMetrics(names, metric.LevelDirectory)
}
```

- [ ] **Step 7: Verify full build compiles**

Run: `go build ./...`
Expected: SUCCESS

- [ ] **Step 8: Run all tests**

Run: `task test`
Expected: all tests PASS

- [ ] **Step 9: Commit**

```bash
git add internal/treemap/stages.go internal/radialtree/stages.go internal/bubbletree/stages.go internal/scatter/stages.go internal/scatter/stages_test.go internal/spiral/stages.go
git commit -m "refactor: update all viz commands for RequestedMetrics type"
```

---

### Task 6: Wire ComputeAggregations Into Pipelines

**Files:**
- Modify: `cmd/codeviz/treemap_cmd.go`
- Modify: `cmd/codeviz/radialtree_cmd.go`
- Modify: `cmd/codeviz/bubbletree_cmd.go`
- Modify: `cmd/codeviz/scatter_cmd.go`
- Modify: `cmd/codeviz/spiral_cmd.go`
- Create: `internal/stages/compute_aggregations_stage.go`

- [ ] **Step 1: Create the pipeline-compatible wrapper**

The pipeline uses `func(*CommonState) error` signatures. Create a wrapper:

```go
// internal/stages/compute_aggregations_stage.go
package stages

import "github.com/rotisserie/eris"

// RunAggregations is the pipeline stage that computes aggregated metrics
// for all expressions in c.Requested.Expressions.
func RunAggregations(c *CommonState) error {
	if len(c.Requested.Expressions) == 0 {
		return nil
	}

	return eris.Wrap(
		ComputeAggregations(c.Root, c.Requested.Expressions),
		"failed to compute metric aggregations",
	)
}
```

- [ ] **Step 2: Add to treemap pipeline**

In `cmd/codeviz/treemap_cmd.go`, add after the `stages.RunProviders` line (line 126):

```go
pipeline.ApplyFuncX(s, stages.RunProviders)
pipeline.ApplyFuncX(s, stages.RunAggregations) // NEW
pipeline.ApplyFuncX(s, stages.FilterBinaryFiles)
```

- [ ] **Step 3: Add to radialtree pipeline**

In `cmd/codeviz/radialtree_cmd.go`, add `pipeline.ApplyFuncX(s, stages.RunAggregations)` after `stages.RunProviders`.

- [ ] **Step 4: Add to bubbletree pipeline**

In `cmd/codeviz/bubbletree_cmd.go`, add `pipeline.ApplyFuncX(s, stages.RunAggregations)` after `stages.RunProviders`.

- [ ] **Step 5: Add to scatter pipeline**

In `cmd/codeviz/scatter_cmd.go`, add `pipeline.ApplyFuncX(s, stages.RunAggregations)` after `stages.RunProviders`.

- [ ] **Step 6: Add to spiral pipeline**

In `cmd/codeviz/spiral_cmd.go`, add `pipeline.ApplyFuncX(s, stages.RunAggregations)` after `stages.RunProviders`.

- [ ] **Step 7: Verify build**

Run: `go build ./...`
Expected: SUCCESS

- [ ] **Step 8: Run all tests**

Run: `task test`
Expected: all tests PASS

- [ ] **Step 9: Commit**

```bash
git add internal/stages/compute_aggregations_stage.go cmd/codeviz/treemap_cmd.go cmd/codeviz/radialtree_cmd.go cmd/codeviz/bubbletree_cmd.go cmd/codeviz/scatter_cmd.go cmd/codeviz/spiral_cmd.go
git commit -m "feat: wire ComputeAggregations stage into all viz pipelines"
```

---

### Task 7: Handle 'distinct' Result Kind Edge Case

The `ComputeAggregations` implementation in Task 2 handles `distinct` within `aggregateClassification` (stores as Quantity). However, `computeResultKind` in `resolution.go` returns `metric.Quantity` for `AggDistinct`, while the aggregation function dispatches on `resolved.ResultKind`. When `ResultKind == Quantity` but the source is a Classification, we need to ensure the dispatcher calls `aggregateClassification` not `aggregateNumeric`.

**Files:**
- Modify: `internal/stages/aggregation.go`
- Create: `internal/stages/aggregation_dispatch_test.go`

- [ ] **Step 1: Write failing test for count on classification**

```go
// internal/stages/aggregation_dispatch_test.go
package stages_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func TestComputeAggregations_CountOnQuantityMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			fileWithQuantity("a.go", "file-bytes", 100),
			fileWithQuantity("b.go", "file-bytes", 200),
			fileWithQuantity("c.go", "file-bytes", 300),
		},
	}

	expr, _ := metric.ParseExpression("file-bytes.count")
	resolved, _ := provider.ResolveExpression(expr, metric.LevelDirectory)

	err := stages.ComputeAggregations(root, []provider.ResolvedMetric{resolved})
	g.Expect(err).To(Succeed())

	// count returns a Quantity: number of files with the metric set
	val, ok := root.Quantity(metric.Name("file-bytes.count"))
	g.Expect(ok).To(BeTrue())
	g.Expect(val).To(Equal(int64(3)))
}
```

- [ ] **Step 2: Run to check behavior**

Run: `go test ./internal/stages/ -run TestComputeAggregations_CountOnQuantity -v`
Expected: may fail depending on current dispatch logic

- [ ] **Step 3: Fix the dispatch in aggregation.go**

Update `aggregateDirectory` to dispatch based on source kind AND aggregation, not just result kind:

```go
// aggregateDirectory recursively computes the aggregate for one directory.
func aggregateDirectory(dir *model.Directory, resolved provider.ResolvedMetric) {
	for _, child := range dir.Dirs {
		aggregateDirectory(child, resolved)
	}

	// Dispatch based on source kind for collection, but store based on result kind.
	if resolved.Descriptor.Kind == metric.Classification {
		aggregateClassification(dir, resolved)
	} else {
		aggregateNumeric(dir, resolved)
	}
}
```

Also update `aggregateClassification` to handle `AggDistinct` storing as Quantity (already done in Task 2) and `aggregateNumeric` to handle `AggCount` result kind:

```go
func aggregateNumeric(dir *model.Directory, resolved provider.ResolvedMetric) {
	values := collectNumericValues(dir, resolved.Expression.Base, resolved.Descriptor.Kind)
	if len(values) == 0 {
		return
	}

	result := applyNumericAggregation(resolved.Expression.Aggregation, values)

	switch resolved.ResultKind {
	case metric.Quantity:
		dir.SetQuantity(resolved.ResultName, int64(result))
	case metric.Measure:
		dir.SetMeasure(resolved.ResultName, result)
	}
}
```

- [ ] **Step 4: Run all aggregation tests**

Run: `go test ./internal/stages/ -run TestComputeAggregations -v`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add internal/stages/aggregation.go internal/stages/aggregation_dispatch_test.go
git commit -m "fix: correct aggregation dispatch for count/distinct on classification metrics"
```

---

### Task 8: Run Full CI

**Files:** None (verification only)

- [ ] **Step 1: Run CI**

Run: `task ci`
Expected: all checks pass (fmt, mod, build, test, lint)

- [ ] **Step 2: Fix any issues**

If there are lint or test failures, fix them and commit fixes individually.

- [ ] **Step 3: Commit any fixes**

```bash
git add -A
git commit -m "fix: resolve CI issues"
```

---

### Task 9: Push and Update PR

**Files:** None (git operations only)

- [ ] **Step 1: Push**

```bash
git push origin feature/metric-expressions-design
```

- [ ] **Step 2: Verify PR is updated**

The existing PR #413 should show the new commits.

---

## Verification Criteria

After all tasks are complete:

1. `task ci` passes clean
2. `go test ./internal/stages/ -run TestComputeAggregations -v` — all aggregation tests pass
3. A config using `file-bytes.sum` as a fill metric compiles, validates, and produces a computed value on directories
4. Legacy metrics (bare `file-bytes`, `file-type`, etc.) continue to work unchanged
5. Expressions referencing declaration-level metrics produce a clear error message (not a crash)
