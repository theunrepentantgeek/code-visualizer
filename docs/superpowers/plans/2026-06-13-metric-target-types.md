# Metric Target Types Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Classify each metric with a target type (File or Directory) and restructure the registry to support same-name metrics for different targets, with helpful error messages.

**Architecture:** Add `metric.Target` enum, extend `provider.Interface` with `Target()`, restructure registry to `map[Target]map[Name]Interface`, update all public API functions to accept a target parameter, add `FindWithHint` for actionable errors.

**Tech Stack:** Go 1.26.1, Gomega (test assertions), eris (error wrapping)

---

### Task 1: Define metric.Target type

**Files:**
- Modify: `internal/metric/metric.go`
- Create: `internal/metric/target_test.go`

- [ ] **Step 1: Write the test for Target.String()**

```go
// internal/metric/target_test.go
package metric

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestTargetString(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(File.String()).To(Equal("file"))
	g.Expect(Directory.String()).To(Equal("directory"))
}

func TestTargetStringUnknown(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(Target(99).String()).To(Equal("unknown"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `task test`
Expected: FAIL — `File`, `Directory`, `Target` not defined.

- [ ] **Step 3: Implement metric.Target**

Add to `internal/metric/metric.go` after the `Kind` constants:

```go
// Target classifies what a metric applies to.
type Target int

const (
	File      Target = iota // metric applies to individual files
	Directory               // metric applies to directories (aggregates)
)

// String returns the human-readable label for the target.
func (t Target) String() string {
	switch t {
	case File:
		return "file"
	case Directory:
		return "directory"
	default:
		return "unknown"
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `task test`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/metric/metric.go internal/metric/target_test.go
git commit -m "feat(metric): add Target type with File and Directory values

Part of #405 — metric target types."
```

---

### Task 2: Add Target() to provider.Interface and MetricDescriptor

**Files:**
- Modify: `internal/provider/provider.go`

- [ ] **Step 1: Add Target to MetricDescriptor**

In `internal/provider/provider.go`, add `Target metric.Target` field to `MetricDescriptor`:

```go
type MetricDescriptor struct {
	Name           metric.Name
	Kind           metric.Kind
	Target         metric.Target
	Description    string
	Dependencies   []metric.Name
	DefaultPalette palette.PaletteName
}
```

- [ ] **Step 2: Add Target() to Interface**

Add `Target() metric.Target` to the `Interface`:

```go
type Interface interface {
	Name() metric.Name
	Kind() metric.Kind
	Target() metric.Target
	Description() string
	Dependencies() []metric.Name
	DefaultPalette() palette.PaletteName
	Loader
}
```

- [ ] **Step 3: Update Descriptor() to copy Target**

```go
func Descriptor(p Interface) MetricDescriptor {
	return MetricDescriptor{
		Name:           p.Name(),
		Kind:           p.Kind(),
		Target:         p.Target(),
		Description:    p.Description(),
		Dependencies:   p.Dependencies(),
		DefaultPalette: p.DefaultPalette(),
	}
}
```

- [ ] **Step 4: Build to verify compilation (expect failures in providers)**

Run: `go build ./...`
Expected: Compile errors in provider packages (they don't implement `Target()` yet). This is expected and will be fixed in Task 3.

- [ ] **Step 5: Commit**

```bash
git add internal/provider/provider.go
git commit -m "feat(provider): add Target() to Interface and MetricDescriptor

Part of #405 — metric target types. Providers will be updated next."
```

---

### Task 3: Implement Target() on all providers

**Files:**
- Modify: `internal/provider/filesystem/metrics.go`
- Modify: `internal/provider/git/git_provider.go`
- Modify: `internal/provider/golang/go_provider.go`
- Modify: `internal/provider/registry_test.go` (stubProvider)
- Modify: `internal/provider/run_test.go` (mockProvider)

- [ ] **Step 1: Add Target() to filesystem providers**

In `internal/provider/filesystem/metrics.go`, add to each provider struct:

```go
func (FileSizeProvider) Target() metric.Target  { return metric.File }
func (FileTypeProvider) Target() metric.Target  { return metric.File }
```

For `FileLinesProvider`, add the same method (find where it's defined — it's a pointer receiver provider):

```go
func (*FileLinesProvider) Target() metric.Target { return metric.File }
```

- [ ] **Step 2: Add Target() to git provider**

In `internal/provider/git/git_provider.go`, add:

```go
func (*gitProvider) Target() metric.Target { return metric.File }
```

- [ ] **Step 3: Add Target() to golang provider**

In `internal/provider/golang/go_provider.go`, add:

```go
func (*goProvider) Target() metric.Target { return metric.File }
```

- [ ] **Step 4: Add Target() to test stubs**

In `internal/provider/registry_test.go`, add to `stubProvider`:

```go
func (*stubProvider) Target() metric.Target { return metric.File }
```

In `internal/provider/run_test.go`, add to `mockProvider`:

```go
func (*mockProvider) Target() metric.Target { return metric.File }
```

- [ ] **Step 5: Build and test**

Run: `task test`
Expected: All tests pass, compilation succeeds.

- [ ] **Step 6: Commit**

```bash
git add internal/provider/filesystem/metrics.go internal/provider/git/git_provider.go \
  internal/provider/golang/go_provider.go internal/provider/registry_test.go \
  internal/provider/run_test.go
git commit -m "feat(provider): implement Target() on all providers

All existing providers return metric.File. Part of #405."
```

---

### Task 4: Restructure registry backing store

**Files:**
- Modify: `internal/provider/registry.go`
- Modify: `internal/provider/registry_test.go`

- [ ] **Step 1: Write tests for the new registry behaviour**

Replace the existing tests in `internal/provider/registry_test.go` with updated versions that pass target:

```go
package provider

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// stubProvider is a minimal Interface implementation for testing.
type stubProvider struct {
	name   metric.Name
	kind   metric.Kind
	target metric.Target
}

func (s *stubProvider) Name() metric.Name                 { return s.name }
func (s *stubProvider) Kind() metric.Kind                 { return s.kind }
func (s *stubProvider) Target() metric.Target             { return s.target }
func (*stubProvider) Description() string                 { return "" }
func (*stubProvider) Dependencies() []metric.Name         { return nil }
func (*stubProvider) DefaultPalette() palette.PaletteName { return palette.Neutral }
func (*stubProvider) Load(_ *model.Directory) error       { return nil }

func TestRegisterAndGet(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	p := &stubProvider{name: "test-metric", kind: metric.Quantity, target: metric.File}
	reg.register(p)

	got, ok := reg.get("test-metric", metric.File)
	g.Expect(ok).To(BeTrue())
	g.Expect(got).ToNot(BeNil())
	g.Expect(got.Name()).To(Equal(metric.Name("test-metric")))
}

func TestGetWithWrongTarget(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	reg.register(&stubProvider{name: "test-metric", kind: metric.Quantity, target: metric.File})

	_, ok := reg.get("test-metric", metric.Directory)
	g.Expect(ok).To(BeFalse())
}

func TestGetUnregistered(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	_, ok := reg.get("nonexistent", metric.File)
	g.Expect(ok).To(BeFalse())
}

func TestSameNameDifferentTargets(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	fileP := &stubProvider{name: "size", kind: metric.Quantity, target: metric.File}
	dirP := &stubProvider{name: "size", kind: metric.Quantity, target: metric.Directory}
	reg.register(fileP)
	reg.register(dirP)

	gotFile, ok := reg.get("size", metric.File)
	g.Expect(ok).To(BeTrue())
	g.Expect(gotFile.Target()).To(Equal(metric.File))

	gotDir, ok := reg.get("size", metric.Directory)
	g.Expect(ok).To(BeTrue())
	g.Expect(gotDir.Target()).To(Equal(metric.Directory))
}

func TestAllProviders(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	reg.register(&stubProvider{name: "m1", kind: metric.Quantity, target: metric.File})
	reg.register(&stubProvider{name: "m2", kind: metric.Classification, target: metric.File})
	reg.register(&stubProvider{name: "m3", kind: metric.Quantity, target: metric.Directory})

	all := reg.all(metric.File)
	g.Expect(all).To(HaveLen(2))

	allDir := reg.all(metric.Directory)
	g.Expect(allDir).To(HaveLen(1))
}

func TestRegisterDuplicatePanics(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	reg.register(&stubProvider{name: "dup", kind: metric.Quantity, target: metric.File})

	g.Expect(func() {
		reg.register(&stubProvider{name: "dup", kind: metric.Quantity, target: metric.File})
	}).To(Panic())
}

func TestDuplicateNameDifferentTargetDoesNotPanic(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	reg.register(&stubProvider{name: "dup", kind: metric.Quantity, target: metric.File})

	g.Expect(func() {
		reg.register(&stubProvider{name: "dup", kind: metric.Quantity, target: metric.Directory})
	}).ToNot(Panic())
}

func TestNamesSorted(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	reg.register(&stubProvider{name: "zebra", kind: metric.Quantity, target: metric.File})
	reg.register(&stubProvider{name: "alpha", kind: metric.Quantity, target: metric.File})
	reg.register(&stubProvider{name: "mid", kind: metric.Quantity, target: metric.File})

	names := reg.names(metric.File)
	g.Expect(names).To(Equal([]metric.Name{"alpha", "mid", "zebra"}))
}

func TestHasName(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	reg.register(&stubProvider{name: "exists", kind: metric.Quantity, target: metric.File})

	g.Expect(reg.hasName("exists")).To(BeTrue())
	g.Expect(reg.hasName("missing")).To(BeFalse())
}

func TestTargetsForName(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	reg.register(&stubProvider{name: "size", kind: metric.Quantity, target: metric.File})
	reg.register(&stubProvider{name: "size", kind: metric.Quantity, target: metric.Directory})

	targets := reg.targetsForName("size")
	g.Expect(targets).To(ConsistOf(metric.File, metric.Directory))

	targets = reg.targetsForName("missing")
	g.Expect(targets).To(BeEmpty())
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `task test`
Expected: FAIL — `get` takes wrong number of arguments, `all` takes wrong number, etc.

- [ ] **Step 3: Rewrite registry.go**

Replace `internal/provider/registry.go` with:

```go
package provider

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"sync"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
)

// registry holds registered metric providers, grouped by target type.
type registry struct {
	mu        sync.RWMutex
	providers map[metric.Target]map[metric.Name]Interface
}

func newRegistry() *registry {
	return &registry{
		providers: make(map[metric.Target]map[metric.Name]Interface),
	}
}

func (r *registry) register(p Interface) {
	r.mu.Lock()
	defer r.mu.Unlock()

	target := p.Target()
	if r.providers[target] == nil {
		r.providers[target] = make(map[metric.Name]Interface)
	}

	if _, exists := r.providers[target][p.Name()]; exists {
		panic(fmt.Sprintf("provider %q already registered for target %q", p.Name(), target))
	}

	r.providers[target][p.Name()] = p
}

func (r *registry) get(name metric.Name, target metric.Target) (Interface, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	inner := r.providers[target]
	if inner == nil {
		return nil, false
	}

	p, ok := inner[name]
	if !ok || p == nil {
		return nil, false
	}

	return p, true
}

func (r *registry) all(target metric.Target) []Interface {
	r.mu.RLock()
	defer r.mu.RUnlock()

	inner := r.providers[target]
	if inner == nil {
		return nil
	}

	result := slices.Collect(maps.Values(inner))
	slices.SortFunc(
		result,
		func(left Interface, right Interface) int {
			return cmp.Compare(left.Name(), right.Name())
		},
	)

	return result
}

func (r *registry) names(target metric.Target) []metric.Name {
	r.mu.RLock()
	defer r.mu.RUnlock()

	inner := r.providers[target]
	if inner == nil {
		return nil
	}

	names := slices.Collect(maps.Keys(inner))
	slices.SortFunc(names, cmp.Compare)

	return names
}

// hasName reports whether any target has a provider with the given name.
func (r *registry) hasName(name metric.Name) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, inner := range r.providers {
		if _, ok := inner[name]; ok {
			return true
		}
	}

	return false
}

// targetsForName returns all targets that have a provider with the given name.
func (r *registry) targetsForName(name metric.Name) []metric.Target {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var targets []metric.Target

	for target, inner := range r.providers {
		if _, ok := inner[name]; ok {
			targets = append(targets, target)
		}
	}

	return targets
}

// globalRegistry is the process-wide provider registry.
var globalRegistry = newRegistry()

// Register adds a provider to the global registry.
// Panics on duplicate (name, target) pair.
func Register(p Interface) { globalRegistry.register(p) }

// Get retrieves a provider by name and target from the global registry.
func Get(name metric.Name, target metric.Target) (Interface, bool) {
	return globalRegistry.get(name, target)
}

// GetDescriptor retrieves only the metadata for a provider by name and target.
func GetDescriptor(name metric.Name, target metric.Target) (MetricDescriptor, bool) {
	p, ok := globalRegistry.get(name, target)
	if !ok {
		return MetricDescriptor{}, false
	}

	return Descriptor(p), true
}

// All returns all registered providers for the given target.
func All(target metric.Target) []Interface { return globalRegistry.all(target) }

// AllDescriptors returns metadata for all registered providers for the given target.
func AllDescriptors(target metric.Target) []MetricDescriptor {
	providers := globalRegistry.all(target)

	descriptors := make([]MetricDescriptor, len(providers))
	for i, p := range providers {
		descriptors[i] = Descriptor(p)
	}

	return descriptors
}

// Names returns the sorted names of all registered providers for the given target.
func Names(target metric.Target) []metric.Name { return globalRegistry.names(target) }

// FindWithHint looks up a provider by name and target. On failure, it checks
// whether the metric exists for a different target and includes that as a hint.
func FindWithHint(name metric.Name, target metric.Target) (Interface, error) {
	p, ok := globalRegistry.get(name, target)
	if ok {
		return p, nil
	}

	targets := globalRegistry.targetsForName(name)
	if len(targets) > 0 {
		return nil, fmt.Errorf(
			"unknown %s metric %q; metric %q exists for target %q",
			target, name, name, targets[0])
	}

	return nil, fmt.Errorf("unknown %s metric %q", target, name)
}

// ResetRegistryForTesting clears the global registry. Test use only.
func ResetRegistryForTesting() {
	globalRegistry = newRegistry()
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `task test`
Expected: Registry tests pass, but other packages will fail due to API change. That's expected.

- [ ] **Step 5: Commit**

```bash
git add internal/provider/registry.go internal/provider/registry_test.go
git commit -m "feat(provider): restructure registry with target-keyed map

Registry now uses map[metric.Target]map[metric.Name]Interface.
Get, All, Names all require a target parameter. FindWithHint provides
helpful errors when a metric exists for a different target.

Part of #405."
```

---

### Task 5: Update run.go (provider scheduler)

**Files:**
- Modify: `internal/provider/run.go`
- Modify: `internal/provider/run_test.go`

- [ ] **Step 1: Update mockProvider in run_test.go**

The `mockProvider` already has `Target()` from Task 3. Update calls to `reg.get(name)` → `reg.get(name, metric.File)` in the test file (or verify there are none — they call through `runWithRegistry` which internally calls `reg.get`).

Actually, the `run.go` functions call `reg.get(name)` internally. Update `expandDeps`, `addEdges`, and the run loop:

- [ ] **Step 2: Update run.go internal calls**

In `internal/provider/run.go`, update all `reg.get(name)` calls to `reg.get(name, target)`. The `Run` function needs to accept a target parameter:

Change `Run` signature and its helper:

```go
// Run loads the requested metrics (plus transitive dependencies) onto the tree.
// Providers run in parallel where dependency ordering allows.
func Run(root *model.Directory, requested []metric.Name, target metric.Target, progress MetricProgress) error {
	return runWithRegistry(globalRegistry, root, requested, target, progress)
}

func runWithRegistry(reg *registry, root *model.Directory, requested []metric.Name, target metric.Target, progress MetricProgress) error {
	if len(requested) == 0 {
		return nil
	}

	expanded, err := expandDeps(reg, requested, target)
	if err != nil {
		return err
	}

	levels, err := topoSort(reg, expanded, target)
	if err != nil {
		return err
	}

	for _, level := range levels {
		g := new(errgroup.Group)

		for _, name := range level {
			p, _ := reg.get(name, target)

			g.Go(func() error {
				return runProvider(p, root, name, progress)
			})
		}

		if err := g.Wait(); err != nil {
			return err //nolint:wrapcheck // error is wrapped inside runProvider
		}
	}

	return nil
}
```

Update `expandDeps` and `visitDep`:

```go
func expandDeps(reg *registry, requested []metric.Name, target metric.Target) ([]metric.Name, error) {
	seen := make(map[metric.Name]bool)

	var result []metric.Name

	for _, name := range requested {
		if err := visitDep(reg, name, target, seen, &result); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func visitDep(reg *registry, name metric.Name, target metric.Target, seen map[metric.Name]bool, result *[]metric.Name) error {
	if seen[name] {
		return nil
	}

	p, ok := reg.get(name, target)
	if !ok || p == nil {
		return eris.Errorf("unknown metric %q; available metrics: %s", name, formatNames(reg.names(target)))
	}

	seen[name] = true
	*result = append(*result, name)

	for _, dep := range p.Dependencies() {
		if err := visitDep(reg, dep, target, seen, result); err != nil {
			return err
		}
	}

	return nil
}
```

Update `topoSort`, `buildDepGraph`, `addEdges`:

```go
func topoSort(reg *registry, names []metric.Name, target metric.Target) ([][]metric.Name, error) {
	inDegree, dependents := buildDepGraph(reg, names, target)

	return computeLevels(names, inDegree, dependents)
}

func buildDepGraph(reg *registry, names []metric.Name, target metric.Target) (map[metric.Name]int, map[metric.Name][]metric.Name) {
	nameSet := make(map[metric.Name]bool, len(names))
	for _, n := range names {
		nameSet[n] = true
	}

	inDegree := make(map[metric.Name]int, len(names))
	dependents := make(map[metric.Name][]metric.Name)

	for _, n := range names {
		inDegree[n] = 0
	}

	for _, n := range names {
		addEdges(reg, n, target, nameSet, inDegree, dependents)
	}

	return inDegree, dependents
}

func addEdges(
	reg *registry,
	n metric.Name,
	target metric.Target,
	nameSet map[metric.Name]bool,
	inDegree map[metric.Name]int,
	dependents map[metric.Name][]metric.Name,
) {
	p, ok := reg.get(n, target)
	if !ok || p == nil {
		return
	}

	for _, dep := range p.Dependencies() {
		if nameSet[dep] {
			inDegree[n]++
			dependents[dep] = append(dependents[dep], n)
		}
	}
}
```

- [ ] **Step 3: Update run_test.go calls**

All calls to `runWithRegistry(reg, root, requested, progress)` become `runWithRegistry(reg, root, requested, metric.File, progress)`.

- [ ] **Step 4: Run tests for the provider package**

Run: `go test ./internal/provider/...`
Expected: All provider tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/provider/run.go internal/provider/run_test.go
git commit -m "feat(provider): add target parameter to Run and internal helpers

Part of #405."
```

---

### Task 6: Update caller sites — stages and inks packages

**Files:**
- Modify: `internal/stages/metrics.go`
- Modify: `internal/inks/inks.go`
- Modify: `internal/scatter/stages.go`
- Modify: `internal/scatter/inks.go`
- Modify: `internal/spiral/aggregation.go`
- Modify: `internal/spiral/inks.go`

- [ ] **Step 1: Update internal/stages/metrics.go**

Change `provider.GetDescriptor(fillMetric)` → `provider.GetDescriptor(fillMetric, metric.File)`:

```go
func ResolveFillPalette(fill *config.MetricSpec, fillMetric metric.Name) palette.PaletteName {
	if fp := fill.PaletteName(); fp != "" {
		return fp
	}

	if d, ok := provider.GetDescriptor(fillMetric, metric.File); ok {
		return d.DefaultPalette
	}

	return palette.Neutral
}

func ResolveBorderMetricAndPalette(border *config.MetricSpec) (metric.Name, palette.PaletteName) {
	borderMetric := border.MetricName()
	if borderMetric == "" {
		return "", ""
	}

	borderPaletteName := border.PaletteName()
	if borderPaletteName == "" {
		if d, ok := provider.GetDescriptor(borderMetric, metric.File); ok {
			borderPaletteName = d.DefaultPalette
		} else {
			borderPaletteName = palette.Neutral
		}
	}

	return borderMetric, borderPaletteName
}
```

Add `"github.com/theunrepentantgeek/code-visualizer/internal/metric"` to the import if not already there.

- [ ] **Step 2: Update internal/inks/inks.go**

Change `provider.GetDescriptor(m)` → `provider.GetDescriptor(m, metric.File)`.

- [ ] **Step 3: Update internal/scatter/stages.go and internal/scatter/inks.go**

Change all `provider.GetDescriptor(metricName)` → `provider.GetDescriptor(metricName, metric.File)`.

- [ ] **Step 4: Update internal/spiral/aggregation.go and internal/spiral/inks.go**

Change all `provider.GetDescriptor(m)` → `provider.GetDescriptor(m, metric.File)`.

- [ ] **Step 5: Build to verify**

Run: `go build ./internal/...`
Expected: Internal packages compile. Cmd packages may still fail (next task).

- [ ] **Step 6: Commit**

```bash
git add internal/stages/metrics.go internal/inks/inks.go internal/scatter/stages.go \
  internal/scatter/inks.go internal/spiral/aggregation.go internal/spiral/inks.go
git commit -m "fix: update internal packages to pass metric.File to provider lookups

Part of #405."
```

---

### Task 7: Update caller sites — config and cmd packages

**Files:**
- Modify: `internal/config/metric_spec.go`
- Modify: `cmd/codeviz/treemap_cmd.go`
- Modify: `cmd/codeviz/bubbletree_cmd.go`
- Modify: `cmd/codeviz/spiral_cmd.go`
- Modify: `cmd/codeviz/radialtree_cmd.go`
- Modify: `cmd/codeviz/scatter_cmd.go`
- Modify: `cmd/codeviz/help_metrics_cmd.go`

- [ ] **Step 1: Update internal/config/metric_spec.go**

Change `Validate` method. Replace `provider.Get(m.Metric)` with `provider.Get(m.Metric, metric.File)` and `provider.Names()` with `provider.Names(metric.File)`:

```go
func (m *MetricSpec) Validate(label string) error {
	if m == nil {
		return nil
	}

	if m.Metric != "" {
		if _, ok := provider.Get(m.Metric, metric.File); !ok {
			names := provider.Names(metric.File)
			strs := make([]string, len(names))

			for i, n := range names {
				strs[i] = string(n)
			}

			return eris.Errorf("invalid %s metric %q; available metrics: %s", label, m.Metric, strings.Join(strs, ", "))
		}
	}

	if m.Palette != "" {
		if !m.Palette.IsValid() {
			return eris.Errorf("invalid %s palette %q", label, m.Palette)
		}
	}

	return nil
}
```

Add `"github.com/theunrepentantgeek/code-visualizer/internal/metric"` to imports.

- [ ] **Step 2: Update treemap_cmd.go**

Change `provider.GetDescriptor(metric.Name(size))` → `provider.GetDescriptor(metric.Name(size), metric.File)`.
Change `provider.Names()` → `provider.Names(metric.File)`.

- [ ] **Step 3: Update bubbletree_cmd.go, spiral_cmd.go, radialtree_cmd.go, scatter_cmd.go**

Same pattern: add `metric.File` argument to all `provider.GetDescriptor()` calls.

- [ ] **Step 4: Update help_metrics_cmd.go**

Change `provider.AllDescriptors()` → `provider.AllDescriptors(metric.File)`.

- [ ] **Step 5: Build and test the full project**

Run: `task test`
Expected: All tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/config/metric_spec.go cmd/codeviz/treemap_cmd.go \
  cmd/codeviz/bubbletree_cmd.go cmd/codeviz/spiral_cmd.go \
  cmd/codeviz/radialtree_cmd.go cmd/codeviz/scatter_cmd.go \
  cmd/codeviz/help_metrics_cmd.go
git commit -m "fix: update config and cmd packages to pass metric.File target

Part of #405."
```

---

### Task 8: Update pipeline stages that call provider.Run

**Files:**
- Modify: `internal/stages/progress.go` (or wherever `provider.Run` is called from pipeline stages)

- [ ] **Step 1: Find all callers of provider.Run**

Search for `provider.Run(` in `.go` files to locate the call sites.

- [ ] **Step 2: Add metric.File target parameter**

Change each `provider.Run(root, requested, progress)` to `provider.Run(root, requested, metric.File, progress)`.

- [ ] **Step 3: Build and test**

Run: `task test`
Expected: All tests pass.

- [ ] **Step 4: Commit**

```bash
git add -u
git commit -m "fix: pass metric.File target to provider.Run in pipeline stages

Part of #405."
```

---

### Task 9: Add FindWithHint test and integrate into validation

**Files:**
- Create: `internal/provider/find_with_hint_test.go`
- Modify: `cmd/codeviz/treemap_cmd.go` (use FindWithHint for better errors)

- [ ] **Step 1: Write FindWithHint tests**

```go
// internal/provider/find_with_hint_test.go
package provider

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
)

func TestFindWithHint_Found(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	oldReg := globalRegistry
	defer func() { globalRegistry = oldReg }()

	globalRegistry = newRegistry()
	globalRegistry.register(&stubProvider{name: "file-size", kind: metric.Quantity, target: metric.File})

	p, err := FindWithHint("file-size", metric.File)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(p.Name()).To(Equal(metric.Name("file-size")))
}

func TestFindWithHint_WrongTarget(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	oldReg := globalRegistry
	defer func() { globalRegistry = oldReg }()

	globalRegistry = newRegistry()
	globalRegistry.register(&stubProvider{name: "dir-count", kind: metric.Quantity, target: metric.Directory})

	_, err := FindWithHint("dir-count", metric.File)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("exists for target"))
	g.Expect(err.Error()).To(ContainSubstring("directory"))
}

func TestFindWithHint_NotFound(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	oldReg := globalRegistry
	defer func() { globalRegistry = oldReg }()

	globalRegistry = newRegistry()

	_, err := FindWithHint("nonexistent", metric.File)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("unknown file metric"))
	g.Expect(err.Error()).ToNot(ContainSubstring("exists for target"))
}
```

- [ ] **Step 2: Run FindWithHint tests**

Run: `go test ./internal/provider/ -run TestFindWithHint -v`
Expected: PASS

- [ ] **Step 3: Integrate FindWithHint into treemap validateConfig**

Update `treemap_cmd.go` `validateConfig`:

```go
func (*TreemapCmd) validateConfig(cfg *config.Treemap) error {
	size := ptrString(cfg.Size)

	p, err := provider.FindWithHint(metric.Name(size), metric.File)
	if err != nil {
		return eris.Wrap(err, "invalid size metric")
	}

	d := provider.Descriptor(p)
	if d.Kind != metric.Quantity && d.Kind != metric.Measure {
		return eris.Errorf("size metric must be numeric, got %q (kind: %d)", size, d.Kind)
	}

	if err := cfg.Fill.Validate("fill"); err != nil {
		return eris.Wrap(err, "invalid fill spec")
	}

	if err := cfg.Border.Validate("border"); err != nil {
		return eris.Wrap(err, "invalid border spec")
	}

	return nil
}
```

Note: `provider.Descriptor(p)` accepts a `provider.Interface` (the `Descriptor` helper function), which is already defined.

- [ ] **Step 4: Run full test suite**

Run: `task test`
Expected: All tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/provider/find_with_hint_test.go cmd/codeviz/treemap_cmd.go
git commit -m "feat: add FindWithHint tests and integrate into treemap validation

Provides actionable error messages when a metric exists for a different
target type. Part of #405."
```

---

### Task 10: Update help metrics table with Target column

**Files:**
- Modify: `cmd/codeviz/help_metrics_cmd.go`

- [ ] **Step 1: Add Target column to the metrics table**

Update `writeProviderGroupTable`:

```go
func writeProviderGroupTable(content *strings.Builder, group []provider.MetricDescriptor) {
	tbl := table.New("Metric", "Target", "Kind", "Default Palette", "Description")
	tbl.SetMaxWidth(consoleWidth())

	for _, d := range group {
		tbl.AddRow(string(d.Name), d.Target.String(), kindLabel(d.Kind), string(d.DefaultPalette), d.Description)
	}

	tbl.WriteTo(content)
}
```

- [ ] **Step 2: Build and test**

Run: `task test`
Expected: All tests pass.

- [ ] **Step 3: Commit**

```bash
git add cmd/codeviz/help_metrics_cmd.go
git commit -m "feat: add Target column to help metrics output

Shows 'file' or 'directory' for each metric. Part of #405."
```

---

### Task 11: Run full CI and lint

**Files:** None (validation only)

- [ ] **Step 1: Run full CI**

Run: `task ci`
Expected: Build, test, and lint all pass.

- [ ] **Step 2: Fix any lint issues**

Address any linting issues that arise from the changes.

- [ ] **Step 3: Commit any lint fixes**

```bash
git add -u
git commit -m "fix: resolve lint issues from metric target implementation"
```

---

### Task 12: Create pull request

**Files:** None

- [ ] **Step 1: Push branch**

```bash
git push -u origin feature/metric-targeting
```

- [ ] **Step 2: Create PR**

```bash
gh pr create --title "Feature: Metric target types (#405)" \
  --body "Implements #405 — metric target types.

## Changes

- Added \`metric.Target\` enumeration with \`File\` and \`Directory\` values
- Extended \`provider.Interface\` with \`Target()\` method
- Restructured registry to \`map[Target]map[Name]Interface\`
- Updated all public API functions to accept a target parameter
- Added \`FindWithHint\` for actionable error messages when a metric exists for a wrong target
- All existing providers classified as \`metric.File\`
- Added Target column to \`help metrics\` output

Closes #405" \
  --base main
```
