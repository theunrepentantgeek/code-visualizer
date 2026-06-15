# Legacy Provider Removal (Phase 4) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove the legacy `provider.Interface`-based registry and migrate all consumers to the expression/base-metric system.

**Architecture:** Add loader registration to the base registry, rewrite `provider.Run` to use it, migrate all viz packages and CLI commands to resolve metrics through `RequestedMetrics.DescriptorFor()` or the base registry, then delete the legacy infrastructure.

**Tech Stack:** Go 1.26+, Kong (CLI), eris (errors), Gomega (test assertions)

---

### Task 1: Add `declarations` Base Metric

The legacy `declaration-count` has no expression equivalent. Add the missing base metric.

**Files:**
- Modify: `internal/provider/golang/base_metrics.go`
- Modify: `internal/provider/golang/base_metrics.go` (add `Declarations` constant)
- Test: `cmd/codeviz/main_test.go` (existing test infrastructure)

- [ ] **Step 1: Add `Declarations` constant and base metric descriptor**

In `internal/provider/golang/base_metrics.go`, add after line 20 (`FunctionLength`):

```go
Declarations metric.Name = "declarations"
```

And add to `goBaseMetrics` slice (after the `Variables` entry, before `Imports`):

```go
{
    Name:           Declarations,
    Kind:           metric.Quantity,
    Level:          metric.LevelDeclaration,
    Description:    "Count of all declarations (types, functions, methods, constants, variables).",
    Filters:        goVisibilityNames,
    Aggregations:   goDeclCountAggs,
    DefaultPalette: palette.Neutral,
    FilterFunc:     goDeclarationFilter,
},
```

- [ ] **Step 2: Verify expression resolves**

Run: `go test ./internal/provider/... -run TestResolve -count=1 -v 2>&1 | tail -5`
Expected: PASS (no test yet but build succeeds)

Verify manually:
```bash
go test -run "^$" ./cmd/codeviz/ -count=1
```
Expected: builds without error

- [ ] **Step 3: Ensure PopulateDeclarations sets the metric on declarations**

In `internal/provider/golang/declarations.go`, find where each declaration gets metrics set. Add to the loop that processes declarations:

```go
d.SetQuantity(Declarations, 1)
```

This makes every declaration "count as 1" so `declarations.count` aggregates them.

Check existing code to see where `d.SetQuantity(Types, 1)` etc. are set — add `Declarations` in the same place.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/provider/golang/ ./internal/stages/ -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/provider/golang/
git commit -m "feat: add 'declarations' base metric for expression-based total declaration count

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 2: Add `BaseMetricLoader` Type and `RegisterLoader`

**Files:**
- Create: `internal/provider/loader.go`
- Modify: `internal/provider/base_registry.go`
- Create: `internal/provider/loader_test.go`

- [ ] **Step 1: Create `internal/provider/loader.go`**

```go
package provider

import "github.com/theunrepentantgeek/code-visualizer/internal/metric"

// BaseMetricLoader describes a unit of metric loading work.
// A single loader may populate multiple base metrics in one pass.
type BaseMetricLoader struct {
	// Metrics lists the base metric names this loader populates.
	Metrics []metric.Name
	// Dependencies lists base metrics that must be loaded before this loader runs.
	Dependencies []metric.Name
	// Load populates the directory tree with metric values.
	Load LoadFunc
}

// LoadFunc is the function signature for metric loading.
type LoadFunc func(root *model.Directory) error
```

- [ ] **Step 2: Add loader storage to base registry**

In `internal/provider/base_registry.go`, add a `loaders` field to `baseRegistry`:

```go
type baseRegistry struct {
	mu          sync.RWMutex
	descriptors map[metric.Name]BaseMetricDescriptor
	providers   map[metric.Name]ProviderDescriptor
	loaders     []BaseMetricLoader
}
```

Update `newBaseRegistry`:
```go
func newBaseRegistry() *baseRegistry {
	return &baseRegistry{
		descriptors: make(map[metric.Name]BaseMetricDescriptor),
		providers:   make(map[metric.Name]ProviderDescriptor),
		loaders:     nil,
	}
}
```

Add methods:
```go
func (r *baseRegistry) registerLoader(loader BaseMetricLoader) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.loaders = append(r.loaders, loader)
}

func (r *baseRegistry) loadersFor(requested []metric.Name) []BaseMetricLoader {
	r.mu.RLock()
	defer r.mu.RUnlock()

	need := make(map[metric.Name]bool, len(requested))
	for _, n := range requested {
		need[n] = true
	}

	var result []BaseMetricLoader
	for _, l := range r.loaders {
		for _, m := range l.Metrics {
			if need[m] {
				result = append(result, l)
				break
			}
		}
	}

	return result
}
```

Add exported function:
```go
// RegisterLoader adds a metric loader to the global base registry.
func RegisterLoader(loader BaseMetricLoader) {
	globalBaseRegistry.registerLoader(loader)
}

// LoadersFor returns loaders needed to satisfy the requested base metrics.
func LoadersFor(requested []metric.Name) []BaseMetricLoader {
	return globalBaseRegistry.loadersFor(requested)
}
```

- [ ] **Step 3: Write test**

Create `internal/provider/loader_test.go`:

```go
package provider_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

func TestLoadersFor_ReturnsMatchingLoader(t *testing.T) {
	g := NewGomegaWithT(t)
	provider.ResetBaseRegistryForTesting()
	defer provider.ResetBaseRegistryForTesting()

	called := false
	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics: []metric.Name{"test-metric"},
		Load: func(_ *model.Directory) error {
			called = true
			return nil
		},
	})

	loaders := provider.LoadersFor([]metric.Name{"test-metric"})
	g.Expect(loaders).To(HaveLen(1))
	g.Expect(loaders[0].Metrics).To(ContainElement(metric.Name("test-metric")))
}

func TestLoadersFor_SkipsUnrelatedLoader(t *testing.T) {
	g := NewGomegaWithT(t)
	provider.ResetBaseRegistryForTesting()
	defer provider.ResetBaseRegistryForTesting()

	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics: []metric.Name{"other-metric"},
		Load:    func(_ *model.Directory) error { return nil },
	})

	loaders := provider.LoadersFor([]metric.Name{"unrelated"})
	g.Expect(loaders).To(BeEmpty())
}
```

- [ ] **Step 4: Add missing import to loader.go**

Add `"github.com/theunrepentantgeek/code-visualizer/internal/model"` import.

- [ ] **Step 5: Run tests**

Run: `go test ./internal/provider/ -run TestLoaders -count=1 -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/provider/loader.go internal/provider/loader_test.go internal/provider/base_registry.go
git commit -m "feat: add BaseMetricLoader type and RegisterLoader API

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 3: Rewrite `provider.Run` to Use Loaders

**Files:**
- Modify: `internal/provider/run.go`
- Modify: `internal/provider/run_test.go`

- [ ] **Step 1: Add `RunLoaders` function**

Add a new function alongside the existing `Run` (keep `Run` for now as other code still calls it):

```go
// RunLoaders loads the requested base metrics using registered loaders.
// Loaders run in parallel where dependency ordering allows.
func RunLoaders(root *model.Directory, requested []metric.Name, progress MetricProgress) error {
	loaders := LoadersFor(requested)
	if len(loaders) == 0 {
		return nil
	}

	levels, err := topoSortLoaders(loaders)
	if err != nil {
		return err
	}

	for _, level := range levels {
		g := new(errgroup.Group)
		for _, loader := range level {
			g.Go(func() error {
				for _, m := range loader.Metrics {
					if progress != nil {
						progress.OnMetricStarted(m)
					}
				}

				if err := loader.Load(root); err != nil {
					return eris.Wrapf(err, "loader failed for metrics %v", loader.Metrics)
				}

				for _, m := range loader.Metrics {
					if progress != nil {
						progress.OnMetricFinished(m)
					}
				}

				return nil
			})
		}

		if err := g.Wait(); err != nil {
			return err
		}
	}

	return nil
}

func topoSortLoaders(loaders []BaseMetricLoader) ([][]BaseMetricLoader, error) {
	// Build a set of all metrics provided by loaders in this run
	provides := make(map[metric.Name]int) // metric -> loader index
	for i, l := range loaders {
		for _, m := range l.Metrics {
			provides[m] = i
		}
	}

	// Compute in-degree for each loader
	inDegree := make([]int, len(loaders))
	dependents := make(map[int][]int) // loader index -> dependent loader indices
	for i, l := range loaders {
		for _, dep := range l.Dependencies {
			if j, ok := provides[dep]; ok && j != i {
				inDegree[i]++
				dependents[j] = append(dependents[j], i)
			}
		}
	}

	// Kahn's algorithm
	var levels [][]BaseMetricLoader
	processed := 0

	for processed < len(loaders) {
		var level []BaseMetricLoader
		var levelIndices []int

		for i, deg := range inDegree {
			if deg == 0 {
				level = append(level, loaders[i])
				levelIndices = append(levelIndices, i)
			}
		}

		if len(level) == 0 {
			return nil, eris.New("circular dependency detected among metric loaders")
		}

		for _, i := range levelIndices {
			inDegree[i] = -1
			processed++
			for _, dep := range dependents[i] {
				inDegree[dep]--
			}
		}

		levels = append(levels, level)
	}

	return levels, nil
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/provider/ -count=1`
Expected: PASS (existing tests unaffected, new code not yet wired)

- [ ] **Step 3: Commit**

```bash
git add internal/provider/run.go
git commit -m "feat: add RunLoaders for expression-based metric loading

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 4: Register Filesystem Loaders

**Files:**
- Modify: `internal/provider/filesystem/register.go`

- [ ] **Step 1: Add loader registrations**

Replace the legacy `provider.Register` calls with `RegisterLoader`:

```go
package filesystem

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// Register adds all filesystem metric providers and loaders to the global registry.
func Register() {
	// Legacy providers (kept temporarily for backward compat)
	provider.Register(FileSizeProvider{})
	provider.Register(&FileLinesProvider{})
	provider.Register(FileTypeProvider{})

	RegisterBase()

	// New loader registrations
	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics: []metric.Name{FileSize},
		Load:    FileSizeProvider{}.Load,
	})
	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics: []metric.Name{FileLines},
		Load:    (&FileLinesProvider{}).Load,
	})
	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics: []metric.Name{FileType},
		Load:    FileTypeProvider{}.Load,
	})
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/provider/filesystem/ -count=1`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/provider/filesystem/register.go
git commit -m "feat: register filesystem loaders alongside legacy providers

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 5: Register Git Loader (Consolidated)

**Files:**
- Modify: `internal/provider/git/register.go`
- Create: `internal/provider/git/loader.go`

- [ ] **Step 1: Create consolidated git loader**

Create `internal/provider/git/loader.go`:

```go
package git

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

// loadAllFileMetrics runs the git analysis once and populates all 7 file-level
// git metrics in a single pass. This replaces 7 separate legacy providers that
// each independently walked git history.
func loadAllFileMetrics(root *model.Directory) error {
	return walkGitFilesAll(root)
}
```

This function needs to call into the existing git infrastructure. Look at `walkGitFiles` in the git package — it takes a metric name and process function. We need a variant that processes ALL metrics in one walk. Check `internal/provider/git/walk.go` or similar for the implementation pattern.

- [ ] **Step 2: Register the consolidated loader**

In `internal/provider/git/register.go`, add:

```go
func Register() {
	for name := range providerDefs {
		gp := newProvider(name)
		provider.Register(gp)
	}

	RegisterBase()

	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics: []metric.Name{
			FileAge, FileFreshness, AuthorCount, CommitCount,
			TotalLinesAdded, TotalLinesRemoved, CommitDensity,
		},
		Load: loadAllFileMetrics,
	})
}
```

- [ ] **Step 3: Implement `walkGitFilesAll`**

This should mirror the existing `walkGitFiles` but call ALL process functions for each file instead of just one. Look at how `walkGitFiles` works and create a version that invokes every `providerDef.process` on each file.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/provider/git/ -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/provider/git/
git commit -m "feat: register consolidated git loader for all file-level metrics

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 6: Register Go File-Level Loader + Filtered Import Variants

**Files:**
- Create: `internal/provider/golang/file_loader.go`
- Modify: `internal/provider/golang/register.go`

- [ ] **Step 1: Create Go file-level loader**

Create `internal/provider/golang/file_loader.go`:

```go
package golang

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

// loadFileMetrics populates file-level Go metrics (imports, comment-ratio)
// and filtered import variants (stdlib.imports, external.imports, internal.imports).
func loadFileMetrics(root *model.Directory) error {
	walkGoFiles(root, "go-file-metrics", nil, func(name metric.Name, stats *fileStats, f *model.File) {
		// Total imports
		f.SetQuantity(Imports, stats.imports)

		// Filtered import variants (stored under expression result names)
		f.SetQuantity("stdlib.imports", stats.stdlibImports)
		f.SetQuantity("external.imports", stats.externalImports)
		f.SetQuantity("internal.imports", stats.internalImports)

		// Comment ratio
		if stats.commentRatio > 0 {
			f.SetMeasure(CommentRatio, stats.commentRatio)
		}
	})

	return nil
}
```

- [ ] **Step 2: Register the loader**

In `internal/provider/golang/register.go`, add:

```go
func Register() {
	for name := range providerDefs {
		gp := newProvider(name)
		provider.Register(gp)
	}

	RegisterBase()

	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics: []metric.Name{Imports, CommentRatio},
		Load:    loadFileMetrics,
	})
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/provider/golang/ -count=1`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/provider/golang/file_loader.go internal/provider/golang/register.go
git commit -m "feat: register Go file-level loader with filtered import variants

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 7: Wire `stages.RunProviders` to Use `RunLoaders`

**Files:**
- Modify: `internal/stages/providers.go`
- Modify: `internal/stages/requested.go`

- [ ] **Step 1: Update `RunProviders` to call both systems**

During the transition, call both `RunLoaders` (new) and `Run` (legacy):

```go
func RunProviders(c *CommonState) error {
	slog.Info("Calculating metrics")

	metricProg, stopMetricTicker := BuildMetricProgress(c.Flags, model.CountFiles(c.Root))
	defer stopMetricTicker()

	// New loader system for base metrics needed by expressions
	if err := provider.RunLoaders(c.Root, c.Requested.BaseMetrics, metricProg); err != nil {
		return eris.Wrap(err, "failed to load base metrics")
	}

	// Legacy system for backward-compat metrics
	if len(c.Requested.Legacy) > 0 {
		if err := provider.Run(c.Root, c.Requested.Legacy, metric.File, metricProg); err != nil {
			return eris.Wrap(err, "failed to load metrics")
		}
	}

	return nil
}
```

- [ ] **Step 2: Run full test suite**

Run: `go test ./... -count=1 2>&1 | grep -E "^(ok|FAIL)"`
Expected: All PASS

- [ ] **Step 3: Commit**

```bash
git add internal/stages/providers.go
git commit -m "feat: wire RunProviders to call RunLoaders for expression base metrics

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 8: Migrate Scatter to `DescriptorFor`

**Files:**
- Modify: `internal/scatter/inks.go`
- Modify: `internal/scatter/stages.go`
- Modify: `internal/scatter/inks_test.go` (if exists)
- Modify: `cmd/codeviz/scatter_cmd.go`

- [ ] **Step 1: Update `scatter/inks.go`**

Change `BuildInks` to accept `stages.RequestedMetrics`:

```go
func BuildInks(
	dataset Dataset,
	requested stages.RequestedMetrics,
	fillMetric metric.Name,
	fillPaletteName palette.PaletteName,
	borderMetric metric.Name,
	borderPaletteName palette.PaletteName,
) Inks {
```

Change `buildMetricInk` to accept `requested`:

```go
func buildMetricInk(
	files []*model.File,
	requested stages.RequestedMetrics,
	name metric.Name,
	paletteName palette.PaletteName,
	fallback color.RGBA,
) canvas.Ink {
	if name == "" {
		return canvas.FixedInk(fallback)
	}

	descriptor, ok := requested.DescriptorFor(name)
	if !ok {
		return canvas.FixedInk(fallback)
	}
	// ... rest unchanged, use descriptor.Kind
```

Update import from `provider` to `stages`.

- [ ] **Step 2: Update `scatter/stages.go`**

In `resolveAxisSpec`, change `provider.GetDescriptor` to use the base registry:

```go
func resolveAxisSpec(name *string, scale *string) (AxisSpec, error) {
	metricName := metric.Name(stages.PtrString(name))
	base, ok := provider.GetBase(metricName)
	if !ok {
		return AxisSpec{}, eris.Errorf("unknown axis metric %q", metricName)
	}

	spec := AxisSpec{Metric: metricName, Kind: base.Kind}
	// ... rest unchanged
```

In `BuildInksStage` (or wherever `BuildInks` is called), pass `c.Requested`.

- [ ] **Step 3: Update test files**

Add `stages.RequestedMetrics{}` argument to all `BuildInks` calls in test files.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/scatter/ -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/scatter/ cmd/codeviz/scatter_cmd.go
git commit -m "refactor: migrate scatter to DescriptorFor and base registry

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 9: Migrate Spiral to `DescriptorFor`

**Files:**
- Modify: `internal/spiral/inks.go`
- Modify: `internal/spiral/aggregation.go`
- Modify: `internal/spiral/stages.go` (if BuildInks is called there)
- Modify: test files

- [ ] **Step 1: Update `spiral/inks.go`**

Change `buildBucketInk` to accept `stages.RequestedMetrics` and use `requested.DescriptorFor(m)` instead of `provider.GetDescriptor(m, metric.File)`.

Update `BuildInks` signature to pass through `requested`.

- [ ] **Step 2: Update `spiral/aggregation.go`**

Change `aggregateColourMetric` to accept `stages.RequestedMetrics`:

```go
func aggregateColourMetric(
	files []*model.File,
	m metric.Name,
	requested stages.RequestedMetrics,
	numVal *float64,
	catLabel *string,
) {
	if m == "" {
		return
	}

	d, ok := requested.DescriptorFor(m)
	if !ok {
		return
	}
	// ... rest unchanged
```

Update callers (`aggregateBucket`, `AggregateBucketMetrics`) to pass `requested`.

- [ ] **Step 3: Update test files**

Add `stages.RequestedMetrics{}` to test calls.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/spiral/ -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/spiral/
git commit -m "refactor: migrate spiral to DescriptorFor and base registry

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 10: Update CLI Command Validation

**Files:**
- Modify: `cmd/codeviz/bubbletree_cmd.go`
- Modify: `cmd/codeviz/radialtree_cmd.go`
- Modify: `cmd/codeviz/scatter_cmd.go`
- Modify: `cmd/codeviz/spiral_cmd.go`
- Modify: `cmd/codeviz/treemap_cmd.go`

- [ ] **Step 1: Create shared validation helper**

In `cmd/codeviz/treemap_cmd.go` (where `formatMetricNames` lives), replace:

```go
func formatMetricNames() string {
	names := provider.NamesFor(metric.File)
	// ...
}
```

With:

```go
func formatMetricNames() string {
	names := provider.BaseNames()
	strs := make([]string, len(names))
	for i, n := range names {
		strs[i] = string(n)
	}
	return strings.Join(strs, ", ")
}
```

- [ ] **Step 2: Replace `provider.GetDescriptor` in CLI validation**

In each `validateConfig` function, replace:
```go
d, ok := provider.GetDescriptor(metric.Name(size), metric.File)
```
With:
```go
d, ok := provider.GetBase(metric.Name(size))
```

And change the kind check from `d.Kind` (MetricDescriptor) to `d.Kind` (BaseMetricDescriptor) — same field name, works directly.

- [ ] **Step 3: Remove `FindWithHint` usage in `treemap_cmd.go`**

Replace:
```go
p, err := provider.FindWithHint(metric.Name(size), metric.File)
```
With:
```go
d, ok := provider.GetBase(metric.Name(size))
if !ok {
	return eris.Errorf("unknown size metric %q; available metrics: %s", size, formatMetricNames())
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./cmd/codeviz/ -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/codeviz/
git commit -m "refactor: CLI validation uses base registry instead of legacy GetDescriptor

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 11: Update `MetricSpec.Validate` (Remove Legacy Fallback)

**Files:**
- Modify: `internal/config/metric_spec.go`

- [ ] **Step 1: Simplify `validateMetric`**

Replace the existing `validateMetric` with:

```go
func (m *MetricSpec) validateMetric(label string) error {
	name := string(m.Metric)

	// Try expression parse + resolve
	expr, parseErr := metric.ParseExpression(name)
	if parseErr == nil {
		_, resolveErr := provider.ResolveExpression(expr, metric.LevelFile)
		if resolveErr == nil {
			return nil
		}

		return eris.Wrapf(resolveErr, "invalid %s metric", label)
	}

	// If it doesn't parse as an expression, check if it's a known base metric
	if _, ok := provider.GetBase(m.Metric); ok {
		return nil
	}

	// Not found — provide helpful error
	return eris.Errorf(
		"invalid %s metric %q; use expression syntax: [filter.]metric[.aggregation]",
		label, m.Metric,
	)
}
```

- [ ] **Step 2: Remove `provider.Get` and `provider.FindWithHint` imports if unused**

Check if the `provider` import is still needed (it is — for `ResolveExpression` and `GetBase`).

- [ ] **Step 3: Run tests**

Run: `go test ./internal/config/ -count=1`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/config/metric_spec.go
git commit -m "refactor: MetricSpec.Validate uses expression system only, no legacy fallback

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 12: Update `help metrics` Command

**Files:**
- Modify: `cmd/codeviz/help_metrics_cmd.go`

- [ ] **Step 1: Remove legacy metrics section**

Remove `findLegacyMetrics` function and the code that displays it. The `renderHelpMetrics` function should no longer call `provider.AllDescriptors()`.

```go
func renderHelpMetrics() string {
	width := consoleWidth()
	baseSections := buildBaseSections(provider.AllBase())

	content := &strings.Builder{}
	writeWrappedText(content, "Syntax: ", "[filter.]metric[.aggregation]", width)
	writeWrappedText(content, "Examples: ", "file-size.sum, public.types.count, cyclomatic-complexity.max", width)

	for _, section := range providerSectionOrder {
		metrics := baseSections[section.Name]
		if len(metrics) == 0 {
			continue
		}

		content.WriteString("\n")
		writeSectionHeader(content, section.Title)

		providerDescriptor, _ := provider.GetBaseProvider(metrics[0].Name)
		writeProviderFilters(content, providerDescriptor, metrics, width)
		writeBaseMetrics(content, metrics, width)
	}

	return content.String()
}
```

- [ ] **Step 2: Remove unused imports**

Remove `provider.AllDescriptors` import/usage.

- [ ] **Step 3: Run tests**

Run: `go test ./cmd/codeviz/ -run TestHelp -count=1`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add cmd/codeviz/help_metrics_cmd.go
git commit -m "refactor: help metrics shows only base metrics, removes legacy section

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 13: Remove `RequestedMetrics.Legacy` Field

**Files:**
- Modify: `internal/stages/requested.go`

- [ ] **Step 1: Update `ClassifyRequestedMetrics`**

Change the "no aggregation needed" and "parse failed" paths to add to `BaseMetrics` instead of `Legacy`:

```go
func ClassifyRequestedMetrics(names []metric.Name, targetLevel metric.MetricLevel) RequestedMetrics {
	var result RequestedMetrics
	baseSeen := make(map[metric.Name]bool)

	for _, name := range names {
		expr, parseErr := metric.ParseExpression(string(name))
		if parseErr != nil {
			// Treat as bare base metric name
			if !baseSeen[name] {
				baseSeen[name] = true
				result.BaseMetrics = append(result.BaseMetrics, name)
			}
			continue
		}

		resolved, resolveErr := provider.ResolveExpression(expr, targetLevel)
		if resolveErr != nil {
			// Treat as bare base metric name
			if !baseSeen[name] {
				baseSeen[name] = true
				result.BaseMetrics = append(result.BaseMetrics, name)
			}
			continue
		}

		if !resolved.NeedsAggregation {
			// Bare metric or filtered file-level — add base to loading list
			if !baseSeen[expr.Base] {
				baseSeen[expr.Base] = true
				result.BaseMetrics = append(result.BaseMetrics, expr.Base)
			}
			continue
		}

		result.Expressions = append(result.Expressions, resolved)

		if resolved.SourceLevel == metric.LevelFile && !baseSeen[expr.Base] {
			baseSeen[expr.Base] = true
			result.BaseMetrics = append(result.BaseMetrics, expr.Base)
		}
	}

	return result
}
```

- [ ] **Step 2: Remove `Legacy` field and `LegacyNames()` method**

Remove from the struct:
```go
type RequestedMetrics struct {
	BaseMetrics []metric.Name
	Expressions []provider.ResolvedMetric
}
```

Remove `LegacyNames()` method entirely.

- [ ] **Step 3: Update `RunProviders`**

Simplify to only use the new system:
```go
func RunProviders(c *CommonState) error {
	slog.Info("Calculating metrics")
	metricProg, stopMetricTicker := BuildMetricProgress(c.Flags, model.CountFiles(c.Root))
	defer stopMetricTicker()

	return eris.Wrap(
		provider.RunLoaders(c.Root, c.Requested.BaseMetrics, metricProg),
		"failed to load metrics",
	)
}
```

- [ ] **Step 4: Fix any compilation errors from removed `Legacy` field**

Search for `.Legacy` usage and remove/fix.

- [ ] **Step 5: Run tests**

Run: `go test ./... -count=1 2>&1 | grep FAIL`
Expected: No failures

- [ ] **Step 6: Commit**

```bash
git add internal/stages/
git commit -m "refactor: remove RequestedMetrics.Legacy field, all metrics go through base system

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 14: Delete Legacy Registry and Interface

**Files:**
- Delete: `internal/provider/registry.go`
- Delete: `internal/provider/registry_test.go`
- Modify: `internal/provider/provider.go` (remove Interface, MetricDescriptor, Descriptor)
- Modify: `internal/provider/run.go` (remove legacy `Run`, `expandDeps`, `topoSort`, etc.)
- Modify: `internal/provider/run_test.go`
- Modify: Provider `register.go` files (remove `provider.Register` calls)

- [ ] **Step 1: Remove legacy `Register` calls from providers**

In `internal/provider/filesystem/register.go`, remove:
```go
provider.Register(FileSizeProvider{})
provider.Register(&FileLinesProvider{})
provider.Register(FileTypeProvider{})
```

In `internal/provider/git/register.go`, remove:
```go
for name := range providerDefs {
    gp := newProvider(name)
    provider.Register(gp)
}
```

In `internal/provider/golang/register.go`, remove:
```go
for name := range providerDefs {
    gp := newProvider(name)
    provider.Register(gp)
}
```

- [ ] **Step 2: Delete `registry.go` and `registry_test.go`**

```bash
rm internal/provider/registry.go internal/provider/registry_test.go
```

- [ ] **Step 3: Remove `Interface`, `MetricDescriptor`, `Descriptor` from `provider.go`**

Keep only `Loader` interface and `LoadFunc` (if needed). The `MetricDescriptor` type is still used by `DescriptorFor` — check if it can be replaced by `BaseMetricDescriptor` or if we need to keep a slim version.

Actually, `MetricDescriptor` is used by `stages.RequestedMetrics.DescriptorFor()`. Update that to return `BaseMetricDescriptor` or keep `MetricDescriptor` as a slim metadata type.

- [ ] **Step 4: Remove legacy `Run` function from `run.go`**

Keep `RunLoaders`. Remove `Run`, `runWithRegistry`, `expandDeps`, `visitDep`, `topoSort`, `buildDepGraph`, `addEdges`, `computeLevels`, `findReady`, `metricNotFoundError`, `formatNames`, `runProvider`.

- [ ] **Step 5: Fix all compilation errors**

Run `go build ./...` and fix references to removed types/functions. This will likely touch:
- `internal/stages/requested.go` (`DescriptorFor` return type)
- `internal/stages/metrics.go` (uses `provider.GetDescriptor`)
- `cmd/codeviz/` files (if any still reference removed functions)
- Test files

- [ ] **Step 6: Run tests**

Run: `go test ./... -count=1 2>&1 | grep FAIL`
Expected: No failures

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "refactor: delete legacy provider registry, Interface, and Run

Removes 260-line registry.go, provider.Interface type, MetricDescriptor,
and the legacy provider.Run dependency-expansion system. All metric
loading now goes through RegisterLoader/RunLoaders.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 15: Delete Go Provider Legacy Definitions

**Files:**
- Delete: `internal/provider/golang/provider_defs.go`
- Delete: `internal/provider/golang/go_provider.go` (the `goProvider` type)
- Modify: `internal/provider/golang/metrics.go` (keep metric name constants needed by tests)

- [ ] **Step 1: Delete files**

```bash
rm internal/provider/golang/provider_defs.go
rm internal/provider/golang/go_provider.go
```

- [ ] **Step 2: Remove references to `providerDefs`, `goProvider`, `newProvider`**

Check for any imports/usages and remove. The `walkGoFiles` function may still be needed by `file_loader.go` — keep it.

- [ ] **Step 3: Remove unused metric name constants from `metrics.go`**

The legacy constants like `TypeCount`, `PublicTypeCount` etc. are no longer needed externally (they were the legacy provider names). However, keep them if tests reference them. If only the legacy `providerDefs` used them, remove them.

Keep: `Imports`, `CommentRatio` (used by file_loader.go)
Keep: Anything in `base_metrics.go` constants (`Types`, `Methods`, etc.)
Remove: `TypeCount`, `PublicTypeCount`, etc. (the hyphenated legacy names)

- [ ] **Step 4: Fix compilation**

Run: `go build ./...`
Fix any broken references.

- [ ] **Step 5: Run tests**

Run: `go test ./... -count=1 2>&1 | grep FAIL`
Expected: No failures

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "refactor: delete Go provider legacy flat metric definitions (35 metrics)

These are now expressed via the expression system as [filter.]base[.aggregation]
combinations (e.g. public.methods.count, cyclomatic-complexity.max).

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 16: Delete Git Provider Legacy Definitions

**Files:**
- Modify: `internal/provider/git/git_provider.go` (remove `providerDefs`, `gitProvider` type, `newProvider`)

- [ ] **Step 1: Remove legacy `providerDefs` map and `gitProvider` type**

Keep: `walkGitFiles`, `walkGitFilesAll`, repo service code, metric constants.
Remove: `gitProvider` struct, `providerDefs` map, `newProvider`, `providerDef` type.

- [ ] **Step 2: Fix compilation**

Run: `go build ./...`

- [ ] **Step 3: Run tests**

Run: `go test ./internal/provider/git/ -count=1`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/provider/git/
git commit -m "refactor: delete git provider legacy flat metric definitions

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 17: Remove `metric.Target` Type

**Files:**
- Modify: `internal/metric/metric.go`
- Modify: All files referencing `metric.Target`, `metric.File`, `metric.Directory`

- [ ] **Step 1: Identify all usages**

```bash
grep -rn "metric\.Target\|metric\.File\b\|metric\.Directory\b" --include="*.go" | grep -v "_test.go"
```

Most should already be gone after previous tasks. Fix any remaining references.

- [ ] **Step 2: Remove `Target` type from `internal/metric/metric.go`**

Remove:
```go
type Target int

const (
    File Target = iota
    Directory
)

func (t Target) String() string { ... }
```

- [ ] **Step 3: Fix compilation**

Run: `go build ./...`
Fix any remaining references (likely in test files).

- [ ] **Step 4: Run tests**

Run: `go test ./... -count=1 2>&1 | grep FAIL`
Expected: No failures

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "refactor: remove metric.Target type, replaced by MetricLevel

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 18: Remove Legacy Filesystem Provider Structs

**Files:**
- Modify: `internal/provider/filesystem/metrics.go`

- [ ] **Step 1: Remove provider Interface implementations**

Remove `FileSizeProvider`, `FileTypeProvider` struct method sets (Name, Kind, Target, Description, Dependencies, DefaultPalette — the Interface methods). Keep `Load` methods as they're used by the loader registrations.

Actually — the loader uses `FileSizeProvider{}.Load` etc. so the struct and `Load` method must stay. Only remove the `Interface`-satisfying methods that are no longer needed: `Name()`, `Kind()`, `Target()`, `Description()`, `Dependencies()`, `DefaultPalette()`.

- [ ] **Step 2: Fix compilation**

Run: `go build ./...`

- [ ] **Step 3: Run tests**

Run: `go test ./internal/provider/filesystem/ -count=1`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/provider/filesystem/
git commit -m "refactor: remove Interface methods from filesystem provider structs

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 19: Update Sample Configs to Expression Syntax

**Files:**
- Modify: `samples/codeviz-bubbletree.yml`
- Modify: `samples/codeviz-radial.yml`
- Modify: `samples/codeviz-treemap.yml`
- Modify: `samples/codeviz-scatter.yml`
- Modify: `samples/codeviz-spiral.yml`

- [ ] **Step 1: Check current sample configs for legacy metric names**

```bash
grep -r "metric\|size\|fill\|border" samples/*.yml
```

Replace any legacy metric names with expression equivalents per the migration table.

- [ ] **Step 2: Regenerate samples**

```bash
task samples
```

- [ ] **Step 3: Commit**

```bash
git add samples/
git commit -m "chore: update sample configs to expression syntax

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 20: Final CI Pass and Cleanup

**Files:**
- Remove any remaining dead code
- Delete `internal/provider/find_with_hint_test.go` if it exists and tests removed functions

- [ ] **Step 1: Run `task ci`**

Run: `task ci`
Expected: PASS with zero issues

- [ ] **Step 2: Fix any remaining lint/test issues**

Address any issues found.

- [ ] **Step 3: Push**

```bash
git push origin feature/metric-expressions-design
```

- [ ] **Step 4: Verify no remaining `provider.GetDescriptor` or `provider.Get` calls**

```bash
grep -rn "provider\.GetDescriptor\|provider\.Get\b\|provider\.All\b\|provider\.Register\b" --include="*.go" | grep -v "_test.go"
```

Expected: No matches (or only in the deleted files)
