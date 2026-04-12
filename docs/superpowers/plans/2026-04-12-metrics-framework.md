# Metrics Framework Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace hardcoded metric fields on FileNode/DirectoryNode with a pluggable provider framework supporting parallel execution and dependency resolution.

**Architecture:** New `model` package holds File/Directory with typed metric maps. Metric package gains Provider interface, registry, and parallel scheduler. Provider packages (filesystem, git) each register one provider per metric. Scanner returns `*model.Directory`, setting cheap metrics during the walk. Consumers read metrics via typed getters.

**Tech Stack:** Go 1.26.1, sync.RWMutex (node-level locking), errgroup (parallel provider execution), eris (error wrapping), Gomega (test assertions)

**Import cycle avoidance:** `metric` defines Name, Kind, Provider (where `Load` takes `any`). `model` imports `metric` for Name/Kind/Provider. `metric` does NOT import `model`. The scheduler (`metric.Run`) also takes `any` and passes it through to providers, which type-assert to `*model.Directory` internally.

---

### Task 1: Metric Framework Types

Add Name (as alias for backward compat), Kind enum, and Provider interface to the metric package.

**Files:**
- Modify: `internal/metric/metric.go`
- Modify: `internal/metric/metric_test.go`

- [ ] **Step 1: Add Name, Kind, and Provider to metric.go**

At the top of `internal/metric/metric.go`, below the existing `MetricName` type and above the constants, add:

```go
// Name identifies a metric. Provider packages define their own Name constants.
type Name = MetricName

// Kind describes the value type of a metric.
type Kind int

const (
	Quantity       Kind = iota // int values (file sizes, line counts)
	Measure                    // float64 values (percentages, rates)
	Classification             // string values (file type, category)
)

// Provider is the interface every metric implements.
// Load receives the tree root (typically *model.Directory) as any to avoid import cycles.
type Provider interface {
	Name() Name
	Kind() Kind
	Dependencies() []Name
	DefaultPalette() palette.PaletteName
	Load(root any) error
}
```

Add `"github.com/bevan/code-visualizer/internal/palette"` to the imports.

**Note:** `type Name = MetricName` is a type alias — both names refer to the same underlying type. This allows all existing code that uses `MetricName` to continue working. In the final cleanup task, we'll flip the definition to `type Name string` and `type MetricName = Name`.

- [ ] **Step 2: Add Kind tests**

Add to `internal/metric/metric_test.go`:

```go
func TestKindConstants(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(Quantity).To(Equal(Kind(0)))
	g.Expect(Measure).To(Equal(Kind(1)))
	g.Expect(Classification).To(Equal(Kind(2)))
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/metric/ -count=1 -v`
Expected: All tests pass, including new TestKindConstants.

- [ ] **Step 4: Commit**

```bash
git add internal/metric/metric.go internal/metric/metric_test.go
git commit -m "feat(metric): add Name alias, Kind enum, and Provider interface

Introduces the core framework types for the pluggable metric system.
Name is a type alias for MetricName for backward compatibility.
Kind classifies metrics as Quantity, Measure, or Classification.
Provider is the interface every metric must implement.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 2: Metric Registry

Add provider registration (Register/Get/All) alongside the existing palette registry.

**Files:**
- Modify: `internal/metric/registry.go`
- Modify: `internal/metric/registry_test.go`

- [ ] **Step 1: Write failing tests for Register/Get/All**

Add to `internal/metric/registry_test.go`:

```go
func TestRegisterAndGet(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	p := &stubProvider{name: "test-metric", kind: Quantity}
	reg.register(p)

	got, ok := reg.get("test-metric")
	g.Expect(ok).To(BeTrue())
	g.Expect(got.Name()).To(Equal(Name("test-metric")))
	g.Expect(got.Kind()).To(Equal(Quantity))
}

func TestGetUnregistered(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	_, ok := reg.get("nonexistent")
	g.Expect(ok).To(BeFalse())
}

func TestAllProviders(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	reg.register(&stubProvider{name: "m1", kind: Quantity})
	reg.register(&stubProvider{name: "m2", kind: Classification})

	all := reg.all()
	g.Expect(all).To(HaveLen(2))
}

func TestRegisterDuplicatePanics(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	reg.register(&stubProvider{name: "dup", kind: Quantity})

	g.Expect(func() {
		reg.register(&stubProvider{name: "dup", kind: Quantity})
	}).To(Panic())
}

// stubProvider is a minimal Provider for testing.
type stubProvider struct {
	name Name
	kind Kind
}

func (s *stubProvider) Name() Name                    { return s.name }
func (s *stubProvider) Kind() Kind                    { return s.kind }
func (s *stubProvider) Dependencies() []Name          { return nil }
func (s *stubProvider) DefaultPalette() palette.PaletteName { return palette.Neutral }
func (s *stubProvider) Load(_ any) error              { return nil }
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/metric/ -count=1 -run TestRegisterAndGet -v`
Expected: FAIL — `newRegistry` not defined.

- [ ] **Step 3: Implement registry**

Add to `internal/metric/registry.go` (keep existing `DefaultPaletteFor` and palette map):

```go
import (
	"fmt"
	"sync"

	"github.com/bevan/code-visualizer/internal/palette"
)

// registry holds registered metric providers.
type registry struct {
	mu        sync.RWMutex
	providers map[Name]Provider
}

func newRegistry() *registry {
	return &registry{providers: make(map[Name]Provider)}
}

func (r *registry) register(p Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[p.Name()]; exists {
		panic(fmt.Sprintf("metric %q already registered", p.Name()))
	}

	r.providers[p.Name()] = p
}

func (r *registry) get(name Name) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.providers[name]
	return p, ok
}

func (r *registry) all() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		result = append(result, p)
	}

	return result
}

// globalRegistry is the process-wide provider registry.
var globalRegistry = newRegistry()

// Register adds a provider to the global registry. Panics on duplicate name.
func Register(p Provider) { globalRegistry.register(p) }

// Get retrieves a provider by name from the global registry.
func Get(name Name) (Provider, bool) { return globalRegistry.get(name) }

// All returns all registered providers.
func All() []Provider { return globalRegistry.all() }

// ResetRegistryForTesting clears the global registry. Test use only.
func ResetRegistryForTesting() {
	globalRegistry = newRegistry()
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/metric/ -count=1 -v`
Expected: All tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/metric/registry.go internal/metric/registry_test.go
git commit -m "feat(metric): add provider registry with Register/Get/All

Thread-safe registry backed by sync.RWMutex. Panics on duplicate
registration. Global registry used by Register/Get/All package functions.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 3: Model Package

Create `internal/model/` with File and Directory types. Typed getters return `(value, bool)`. Setters take `metric.Provider` and validate Kind.

**Files:**
- Create: `internal/model/file.go`
- Create: `internal/model/file_test.go`
- Create: `internal/model/directory.go`
- Create: `internal/model/directory_test.go`
- Create: `internal/model/walk.go`

- [ ] **Step 1: Write failing tests for File**

Create `internal/model/file_test.go`:

```go
package model

import (
	"sync"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/palette"
)

type fakeProvider struct {
	name metric.Name
	kind metric.Kind
}

func (f *fakeProvider) Name() metric.Name                    { return f.name }
func (f *fakeProvider) Kind() metric.Kind                    { return f.kind }
func (f *fakeProvider) Dependencies() []metric.Name          { return nil }
func (f *fakeProvider) DefaultPalette() palette.PaletteName  { return palette.Neutral }
func (f *fakeProvider) Load(_ any) error                     { return nil }

func TestFileSetAndGetQuantity(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := &File{Path: "/a.go", Name: "a.go"}
	p := &fakeProvider{name: "file-size", kind: metric.Quantity}

	f.SetQuantity(p, 1024)

	v, ok := f.Quantity("file-size")
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(Equal(1024))
}

func TestFileSetAndGetMeasure(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := &File{Path: "/a.go", Name: "a.go"}
	p := &fakeProvider{name: "complexity", kind: metric.Measure}

	f.SetMeasure(p, 3.14)

	v, ok := f.Measure("complexity")
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(Equal(3.14))
}

func TestFileSetAndGetClassification(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := &File{Path: "/a.go", Name: "a.go"}
	p := &fakeProvider{name: "file-type", kind: metric.Classification}

	f.SetClassification(p, "go")

	v, ok := f.Classification("file-type")
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(Equal("go"))
}

func TestFileGetUnsetMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := &File{Path: "/a.go", Name: "a.go"}

	_, ok := f.Quantity("unset")
	g.Expect(ok).To(BeFalse())

	_, ok = f.Measure("unset")
	g.Expect(ok).To(BeFalse())

	_, ok = f.Classification("unset")
	g.Expect(ok).To(BeFalse())
}

func TestFileSetQuantityPanicsOnWrongKind(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := &File{Path: "/a.go", Name: "a.go"}
	p := &fakeProvider{name: "file-type", kind: metric.Classification}

	g.Expect(func() { f.SetQuantity(p, 42) }).To(Panic())
}

func TestFileSetMeasurePanicsOnWrongKind(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := &File{Path: "/a.go", Name: "a.go"}
	p := &fakeProvider{name: "file-size", kind: metric.Quantity}

	g.Expect(func() { f.SetMeasure(p, 1.5) }).To(Panic())
}

func TestFileSetClassificationPanicsOnWrongKind(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := &File{Path: "/a.go", Name: "a.go"}
	p := &fakeProvider{name: "file-size", kind: metric.Quantity}

	g.Expect(func() { f.SetClassification(p, "go") }).To(Panic())
}

func TestFileConcurrentAccess(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := &File{Path: "/a.go", Name: "a.go"}
	pSize := &fakeProvider{name: "size", kind: metric.Quantity}
	pType := &fakeProvider{name: "type", kind: metric.Classification}

	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()

		for i := range 100 {
			f.SetQuantity(pSize, i)
		}
	}()

	go func() {
		defer wg.Done()

		for range 100 {
			f.SetClassification(pType, "go")
		}
	}()

	wg.Wait()

	v, ok := f.Quantity("size")
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(Equal(99))

	s, ok := f.Classification("type")
	g.Expect(ok).To(BeTrue())
	g.Expect(s).To(Equal("go"))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/model/ -count=1 -v`
Expected: FAIL — package does not exist yet.

- [ ] **Step 3: Implement File**

Create `internal/model/file.go`:

```go
// Package model defines the tree data structure used by the metric framework.
package model

import (
	"fmt"
	"sync"

	"github.com/bevan/code-visualizer/internal/metric"
)

// File represents a single file in the scanned tree.
type File struct {
	Path      string
	Name      string
	Extension string
	IsBinary  bool

	mu              sync.RWMutex
	quantities      map[metric.Name]int
	measures        map[metric.Name]float64
	classifications map[metric.Name]string
}

// Quantity returns the int value for the named metric and whether it was set.
func (f *File) Quantity(name metric.Name) (int, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	v, ok := f.quantities[name]

	return v, ok
}

// Measure returns the float64 value for the named metric and whether it was set.
func (f *File) Measure(name metric.Name) (float64, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	v, ok := f.measures[name]

	return v, ok
}

// Classification returns the string value for the named metric and whether it was set.
func (f *File) Classification(name metric.Name) (string, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	v, ok := f.classifications[name]

	return v, ok
}

// SetQuantity stores an int metric value. Panics if the provider's Kind is not Quantity.
func (f *File) SetQuantity(p metric.Provider, v int) {
	if p.Kind() != metric.Quantity {
		panic(fmt.Sprintf("SetQuantity called with %s provider %q (expected Quantity)", kindName(p.Kind()), p.Name()))
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.quantities == nil {
		f.quantities = make(map[metric.Name]int)
	}

	f.quantities[p.Name()] = v
}

// SetMeasure stores a float64 metric value. Panics if the provider's Kind is not Measure.
func (f *File) SetMeasure(p metric.Provider, v float64) {
	if p.Kind() != metric.Measure {
		panic(fmt.Sprintf("SetMeasure called with %s provider %q (expected Measure)", kindName(p.Kind()), p.Name()))
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.measures == nil {
		f.measures = make(map[metric.Name]float64)
	}

	f.measures[p.Name()] = v
}

// SetClassification stores a string metric value. Panics if the provider's Kind is not Classification.
func (f *File) SetClassification(p metric.Provider, v string) {
	if p.Kind() != metric.Classification {
		panic(fmt.Sprintf("SetClassification called with %s provider %q (expected Classification)", kindName(p.Kind()), p.Name()))
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.classifications == nil {
		f.classifications = make(map[metric.Name]string)
	}

	f.classifications[p.Name()] = v
}

func kindName(k metric.Kind) string {
	switch k {
	case metric.Quantity:
		return "Quantity"
	case metric.Measure:
		return "Measure"
	case metric.Classification:
		return "Classification"
	default:
		return "Unknown"
	}
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/model/ -count=1 -v`
Expected: All tests pass.

- [ ] **Step 5: Run with race detector**

Run: `go test ./internal/model/ -count=1 -race`
Expected: No races detected.

- [ ] **Step 6: Write failing tests for Directory**

Create `internal/model/directory_test.go`:

```go
package model

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/bevan/code-visualizer/internal/metric"
)

func TestDirectorySetAndGetQuantity(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	d := &Directory{Path: "/src", Name: "src"}
	p := &fakeProvider{name: "folder-size", kind: metric.Quantity}

	d.SetQuantity(p, 9999)

	v, ok := d.Quantity("folder-size")
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(Equal(9999))
}

func TestDirectoryGetUnsetMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	d := &Directory{Path: "/src", Name: "src"}

	_, ok := d.Quantity("unset")
	g.Expect(ok).To(BeFalse())
}

func TestDirectoryPointerSlices(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	child := &File{Path: "/src/a.go", Name: "a.go"}
	subdir := &Directory{Path: "/src/sub", Name: "sub"}
	d := &Directory{
		Path:  "/src",
		Name:  "src",
		Files: []*File{child},
		Dirs:  []*Directory{subdir},
	}

	g.Expect(d.Files).To(HaveLen(1))
	g.Expect(d.Dirs).To(HaveLen(1))
	g.Expect(d.Files[0].Name).To(Equal("a.go"))
	g.Expect(d.Dirs[0].Name).To(Equal("sub"))
}
```

- [ ] **Step 7: Implement Directory**

Create `internal/model/directory.go`:

```go
package model

import (
	"fmt"
	"sync"

	"github.com/bevan/code-visualizer/internal/metric"
)

// Directory represents a directory in the scanned tree.
type Directory struct {
	Path  string
	Name  string
	Files []*File
	Dirs  []*Directory

	mu              sync.RWMutex
	quantities      map[metric.Name]int
	measures        map[metric.Name]float64
	classifications map[metric.Name]string
}

// Quantity returns the int value for the named metric and whether it was set.
func (d *Directory) Quantity(name metric.Name) (int, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	v, ok := d.quantities[name]

	return v, ok
}

// Measure returns the float64 value for the named metric and whether it was set.
func (d *Directory) Measure(name metric.Name) (float64, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	v, ok := d.measures[name]

	return v, ok
}

// Classification returns the string value for the named metric and whether it was set.
func (d *Directory) Classification(name metric.Name) (string, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	v, ok := d.classifications[name]

	return v, ok
}

// SetQuantity stores an int metric value. Panics if the provider's Kind is not Quantity.
func (d *Directory) SetQuantity(p metric.Provider, v int) {
	if p.Kind() != metric.Quantity {
		panic(fmt.Sprintf("SetQuantity called with %s provider %q (expected Quantity)", kindName(p.Kind()), p.Name()))
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if d.quantities == nil {
		d.quantities = make(map[metric.Name]int)
	}

	d.quantities[p.Name()] = v
}

// SetMeasure stores a float64 metric value. Panics if the provider's Kind is not Measure.
func (d *Directory) SetMeasure(p metric.Provider, v float64) {
	if p.Kind() != metric.Measure {
		panic(fmt.Sprintf("SetMeasure called with %s provider %q (expected Measure)", kindName(p.Kind()), p.Name()))
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if d.measures == nil {
		d.measures = make(map[metric.Name]float64)
	}

	d.measures[p.Name()] = v
}

// SetClassification stores a string metric value. Panics if the provider's Kind is not Classification.
func (d *Directory) SetClassification(p metric.Provider, v string) {
	if p.Kind() != metric.Classification {
		panic(fmt.Sprintf("SetClassification called with %s provider %q (expected Classification)", kindName(p.Kind()), p.Name()))
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if d.classifications == nil {
		d.classifications = make(map[metric.Name]string)
	}

	d.classifications[p.Name()] = v
}
```

- [ ] **Step 8: Implement WalkFiles helper**

Create `internal/model/walk.go`:

```go
package model

// WalkFiles calls fn for every file in the tree, depth-first.
func WalkFiles(dir *Directory, fn func(*File)) {
	for _, f := range dir.Files {
		fn(f)
	}

	for _, d := range dir.Dirs {
		WalkFiles(d, fn)
	}
}
```

- [ ] **Step 9: Run all tests**

Run: `go test ./internal/model/ -count=1 -race -v`
Expected: All tests pass, no races.

- [ ] **Step 10: Commit**

```bash
git add internal/model/
git commit -m "feat(model): add File and Directory with typed metric storage

Typed getters return (value, bool). Setters take metric.Provider and
panic if Kind mismatches. Lazy map init and sync.RWMutex per node.
WalkFiles utility for tree traversal.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 4: Metric Scheduler

Implement `metric.Run()` with dependency resolution, topological sort, and parallel execution.

**Files:**
- Create: `internal/metric/run.go`
- Create: `internal/metric/run_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/metric/run_test.go`:

```go
package metric

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/bevan/code-visualizer/internal/palette"
)

// orderTracker records which providers ran and in what order.
type orderTracker struct {
	mu    sync.Mutex
	calls []Name
}

func (o *orderTracker) record(name Name) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.calls = append(o.calls, name)
}

// mockProvider records load calls and optionally returns an error.
type mockProvider struct {
	name    Name
	kind    Kind
	deps    []Name
	loadErr error
	tracker *orderTracker
}

func (m *mockProvider) Name() Name                    { return m.name }
func (m *mockProvider) Kind() Kind                    { return m.kind }
func (m *mockProvider) Dependencies() []Name          { return m.deps }
func (m *mockProvider) DefaultPalette() palette.PaletteName { return palette.Neutral }

func (m *mockProvider) Load(_ any) error {
	if m.tracker != nil {
		m.tracker.record(m.name)
	}

	return m.loadErr
}

func TestRunBasicExecution(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	tracker := &orderTracker{}
	reg.register(&mockProvider{name: "m1", kind: Quantity, tracker: tracker})

	err := runWithRegistry(reg, nil, []Name{"m1"})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(tracker.calls).To(Equal([]Name{"m1"}))
}

func TestRunTransitiveDependencies(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	tracker := &orderTracker{}
	reg.register(&mockProvider{name: "base", kind: Quantity, tracker: tracker})
	reg.register(&mockProvider{name: "derived", kind: Quantity, deps: []Name{"base"}, tracker: tracker})

	err := runWithRegistry(reg, nil, []Name{"derived"})
	g.Expect(err).NotTo(HaveOccurred())

	// "base" must run before "derived"
	g.Expect(tracker.calls).To(HaveLen(2))

	baseIdx := -1
	derivedIdx := -1

	for i, n := range tracker.calls {
		if n == "base" {
			baseIdx = i
		}

		if n == "derived" {
			derivedIdx = i
		}
	}

	g.Expect(baseIdx).To(BeNumerically("<", derivedIdx))
}

func TestRunCycleDetection(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	reg.register(&mockProvider{name: "a", kind: Quantity, deps: []Name{"b"}})
	reg.register(&mockProvider{name: "b", kind: Quantity, deps: []Name{"a"}})

	err := runWithRegistry(reg, nil, []Name{"a"})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("circular dependency"))
}

func TestRunUnknownDependency(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	reg.register(&mockProvider{name: "a", kind: Quantity, deps: []Name{"missing"}})

	err := runWithRegistry(reg, nil, []Name{"a"})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("unknown metric"))
}

func TestRunUnknownRequestedMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()

	err := runWithRegistry(reg, nil, []Name{"nonexistent"})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("unknown metric"))
}

func TestRunErrorPropagation(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	reg.register(&mockProvider{name: "fail", kind: Quantity, loadErr: errors.New("load failed")})

	err := runWithRegistry(reg, nil, []Name{"fail"})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("load failed"))
}

// concurrentProvider tracks concurrent Load calls via shared atomics.
type concurrentProvider struct {
	name          Name
	counter       *atomic.Int32
	maxConcurrent *atomic.Int32
}

func (c *concurrentProvider) Name() Name                    { return c.name }
func (c *concurrentProvider) Kind() Kind                    { return Quantity }
func (c *concurrentProvider) Dependencies() []Name          { return nil }
func (c *concurrentProvider) DefaultPalette() palette.PaletteName { return palette.Neutral }

func (c *concurrentProvider) Load(_ any) error {
	cur := c.counter.Add(1)

	for {
		old := c.maxConcurrent.Load()
		if cur <= old || c.maxConcurrent.CompareAndSwap(old, cur) {
			break
		}
	}

	// Small sleep to keep goroutines alive long enough to overlap
	time.Sleep(10 * time.Millisecond)
	c.counter.Add(-1)

	return nil
}

func TestRunParallelExecution(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	var counter atomic.Int32
	var maxConcurrent atomic.Int32

	// Register 3 independent providers — they should run concurrently
	reg.register(&concurrentProvider{name: "p1", counter: &counter, maxConcurrent: &maxConcurrent})
	reg.register(&concurrentProvider{name: "p2", counter: &counter, maxConcurrent: &maxConcurrent})
	reg.register(&concurrentProvider{name: "p3", counter: &counter, maxConcurrent: &maxConcurrent})

	err := runWithRegistry(reg, nil, []Name{"p1", "p2", "p3"})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(maxConcurrent.Load()).To(BeNumerically(">", 1), "expected concurrent execution")
}

func TestRunEmptyRequest(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()

	err := runWithRegistry(reg, nil, []Name{})
	g.Expect(err).NotTo(HaveOccurred())
}

func TestRunAutoExpandsDependencies(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	tracker := &orderTracker{}
	reg.register(&mockProvider{name: "base", kind: Quantity, tracker: tracker})
	reg.register(&mockProvider{name: "mid", kind: Quantity, deps: []Name{"base"}, tracker: tracker})
	reg.register(&mockProvider{name: "top", kind: Quantity, deps: []Name{"mid"}, tracker: tracker})

	// Only request "top" — "mid" and "base" should be auto-included
	err := runWithRegistry(reg, nil, []Name{"top"})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(tracker.calls).To(HaveLen(3))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/metric/ -count=1 -run TestRun -v`
Expected: FAIL — `runWithRegistry` not defined.

- [ ] **Step 3: Implement Run**

Create `internal/metric/run.go`:

```go
package metric

import (
	"github.com/rotisserie/eris"
	"golang.org/x/sync/errgroup"
)

// Run loads the requested metrics (plus transitive dependencies) onto the tree.
// Providers run in parallel where dependency ordering allows.
func Run(root any, requested []Name) error {
	return runWithRegistry(globalRegistry, root, requested)
}

func runWithRegistry(reg *registry, root any, requested []Name) error {
	if len(requested) == 0 {
		return nil
	}

	expanded, err := expandDeps(reg, requested)
	if err != nil {
		return err
	}

	levels, err := topoSort(reg, expanded)
	if err != nil {
		return err
	}

	for _, level := range levels {
		g := new(errgroup.Group)

		for _, name := range level {
			p, _ := reg.get(name)

			g.Go(func() error {
				return p.Load(root)
			})
		}

		if err := g.Wait(); err != nil {
			return eris.Wrap(err, "provider load failed")
		}
	}

	return nil
}

// expandDeps returns the transitive closure of requested metric names.
func expandDeps(reg *registry, requested []Name) ([]Name, error) {
	seen := make(map[Name]bool)
	var result []Name

	var visit func(Name) error
	visit = func(name Name) error {
		if seen[name] {
			return nil
		}

		p, ok := reg.get(name)
		if !ok {
			return eris.Errorf("unknown metric %q — no provider registered", name)
		}

		seen[name] = true
		result = append(result, name)

		for _, dep := range p.Dependencies() {
			if err := visit(dep); err != nil {
				return err
			}
		}

		return nil
	}

	for _, name := range requested {
		if err := visit(name); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// topoSort groups metrics into execution levels. Each level's metrics have
// all dependencies satisfied by previous levels.
func topoSort(reg *registry, names []Name) ([][]Name, error) {
	nameSet := make(map[Name]bool, len(names))
	for _, n := range names {
		nameSet[n] = true
	}

	inDegree := make(map[Name]int, len(names))
	dependents := make(map[Name][]Name)

	for _, n := range names {
		inDegree[n] = 0
	}

	for _, n := range names {
		p, _ := reg.get(n)

		for _, dep := range p.Dependencies() {
			if nameSet[dep] {
				inDegree[n]++
				dependents[dep] = append(dependents[dep], n)
			}
		}
	}

	var levels [][]Name
	processed := 0

	for processed < len(names) {
		var level []Name

		for _, n := range names {
			if inDegree[n] == 0 {
				level = append(level, n)
			}
		}

		if len(level) == 0 {
			return nil, eris.New("circular dependency detected among metric providers")
		}

		for _, n := range level {
			inDegree[n] = -1
			processed++

			for _, dep := range dependents[n] {
				inDegree[dep]--
			}
		}

		levels = append(levels, level)
	}

	return levels, nil
}
```

- [ ] **Step 4: Add errgroup dependency**

Run: `go get golang.org/x/sync && go mod tidy`

If `golang.org/x/sync` is already present transitively, `go mod tidy` alone suffices.

- [ ] **Step 5: Run tests**

Run: `go test ./internal/metric/ -count=1 -v`
Expected: All tests pass.

- [ ] **Step 6: Run full test suite**

Run: `go test ./... -count=1`
Expected: All packages pass.

- [ ] **Step 7: Commit**

```bash
git add internal/metric/run.go internal/metric/run_test.go go.mod go.sum
git commit -m "feat(metric): add scheduler with dependency resolution and parallel execution

Run() expands transitive dependencies, topologically sorts into execution
levels, and runs each level in parallel via errgroup. Detects cycles and
unknown dependencies. Error from any provider cancels remaining work.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 5: Filesystem Providers

Create providers for file-size, file-lines, and file-type.

**Files:**
- Create: `internal/provider/filesystem/metrics.go`
- Create: `internal/provider/filesystem/register.go`
- Create: `internal/provider/filesystem/metrics_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/provider/filesystem/metrics_test.go`:

```go
package filesystem

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
)

func TestFileSizeProvider(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	p := FileSizeProvider{}
	g.Expect(p.Name()).To(Equal(FileSize))
	g.Expect(p.Kind()).To(Equal(metric.Quantity))
	g.Expect(p.Dependencies()).To(BeNil())

	root := &model.Directory{Path: "/root", Name: "root"}
	g.Expect(p.Load(root)).NotTo(HaveOccurred()) // no-op
}

func TestFileTypeProvider(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	p := FileTypeProvider{}
	g.Expect(p.Name()).To(Equal(FileType))
	g.Expect(p.Kind()).To(Equal(metric.Classification))
	g.Expect(p.Dependencies()).To(BeNil())

	root := &model.Directory{Path: "/root", Name: "root"}
	g.Expect(p.Load(root)).NotTo(HaveOccurred()) // no-op
}

func TestFileLinesProvider(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "three.go"), []byte("a\nb\nc\n"), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "one.txt"), []byte("single\n"), 0o600)

	f1 := &model.File{Path: filepath.Join(dir, "three.go"), Name: "three.go", Extension: "go"}
	f2 := &model.File{Path: filepath.Join(dir, "one.txt"), Name: "one.txt", Extension: "txt"}
	root := &model.Directory{
		Path:  dir,
		Name:  "root",
		Files: []*model.File{f1, f2},
	}

	p := FileLinesProvider{}
	err := p.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	v1, ok := f1.Quantity(FileLines)
	g.Expect(ok).To(BeTrue())
	g.Expect(v1).To(Equal(3))

	v2, ok := f2.Quantity(FileLines)
	g.Expect(ok).To(BeTrue())
	g.Expect(v2).To(Equal(1))
}

func TestFileLinesProviderSkipsBinaryFiles(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	// Write a single line longer than bufio.MaxScanTokenSize (65536) to trigger binary detection
	_ = os.WriteFile(filepath.Join(dir, "bin.dat"), append([]byte("hello\x00world"), make([]byte, 66000)...), 0o600)

	f := &model.File{Path: filepath.Join(dir, "bin.dat"), Name: "bin.dat"}
	root := &model.Directory{Path: dir, Name: "root", Files: []*model.File{f}}

	p := FileLinesProvider{}
	err := p.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	_, ok := f.Quantity(FileLines)
	g.Expect(ok).To(BeFalse())
	g.Expect(f.IsBinary).To(BeTrue())
}

func TestFileLinesProviderNestedDirs(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	_ = os.MkdirAll(sub, 0o755)
	_ = os.WriteFile(filepath.Join(sub, "deep.go"), []byte("a\nb\n"), 0o600)

	f := &model.File{Path: filepath.Join(sub, "deep.go"), Name: "deep.go", Extension: "go"}
	root := &model.Directory{
		Path: dir,
		Name: "root",
		Dirs: []*model.Directory{
			{Path: sub, Name: "sub", Files: []*model.File{f}},
		},
	}

	p := FileLinesProvider{}
	err := p.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	v, ok := f.Quantity(FileLines)
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(Equal(2))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/provider/filesystem/ -count=1 -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Implement providers**

Create `internal/provider/filesystem/metrics.go`:

```go
// Package filesystem provides metric providers for filesystem-derived metrics.
package filesystem

import (
	"bufio"
	"errors"
	"log/slog"
	"os"

	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/palette"
)

// Metric name constants for filesystem metrics.
const (
	FileSize  metric.Name = "file-size"
	FileLines metric.Name = "file-lines"
	FileType  metric.Name = "file-type"
)

// FileSizeProvider reports file size in bytes. Value is set during scan; Load is a no-op.
type FileSizeProvider struct{}

func (FileSizeProvider) Name() metric.Name                    { return FileSize }
func (FileSizeProvider) Kind() metric.Kind                    { return metric.Quantity }
func (FileSizeProvider) Dependencies() []metric.Name          { return nil }
func (FileSizeProvider) DefaultPalette() palette.PaletteName  { return palette.Neutral }
func (FileSizeProvider) Load(_ any) error                     { return nil }

// FileTypeProvider reports the file type classification. Value is set during scan; Load is a no-op.
type FileTypeProvider struct{}

func (FileTypeProvider) Name() metric.Name                    { return FileType }
func (FileTypeProvider) Kind() metric.Kind                    { return metric.Classification }
func (FileTypeProvider) Dependencies() []metric.Name          { return nil }
func (FileTypeProvider) DefaultPalette() palette.PaletteName  { return palette.Categorization }
func (FileTypeProvider) Load(_ any) error                     { return nil }

// FileLinesProvider counts lines in each text file.
type FileLinesProvider struct{}

func (FileLinesProvider) Name() metric.Name                    { return FileLines }
func (FileLinesProvider) Kind() metric.Kind                    { return metric.Quantity }
func (FileLinesProvider) Dependencies() []metric.Name          { return nil }
func (FileLinesProvider) DefaultPalette() palette.PaletteName  { return palette.Neutral }

func (p FileLinesProvider) Load(root any) error {
	dir := root.(*model.Directory)
	model.WalkFiles(dir, func(f *model.File) {
		if f.IsBinary {
			return
		}

		count, err := countLines(f.Path)
		if err != nil {
			if errors.Is(err, errBinaryFile) {
				f.IsBinary = true

				return
			}

			slog.Warn("could not count lines", "path", f.Path, "error", err)

			return
		}

		f.SetQuantity(p, count)
	})

	return nil
}

var errBinaryFile = errors.New("file appears to be binary (line exceeds 64KB)")

func countLines(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	count := 0
	for scanner.Scan() {
		count++
	}

	if err := scanner.Err(); err != nil {
		return 0, errBinaryFile
	}

	return count, nil
}
```

Create `internal/provider/filesystem/register.go`:

```go
package filesystem

import "github.com/bevan/code-visualizer/internal/metric"

// Register adds all filesystem metric providers to the global registry.
func Register() {
	metric.Register(FileSizeProvider{})
	metric.Register(FileLinesProvider{})
	metric.Register(FileTypeProvider{})
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/provider/filesystem/ -count=1 -v`
Expected: All tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/provider/filesystem/
git commit -m "feat(provider): add filesystem providers (file-size, file-lines, file-type)

FileSizeProvider and FileTypeProvider are no-op (values set during scan).
FileLinesProvider counts lines per file and detects binary files.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 6: Git Providers

Create providers for file-age, file-freshness, author-count, with shared repoService.

**Files:**
- Create: `internal/provider/git/service.go`
- Create: `internal/provider/git/metrics.go`
- Create: `internal/provider/git/register.go`
- Create: `internal/provider/git/metrics_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/provider/git/metrics_test.go`:

```go
package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
)

func setupTestGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		t.Helper()

		cmd := exec.Command(args[0], args[1:]...) //nolint:gosec // test helper
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Alice",
			"GIT_AUTHOR_EMAIL=alice@example.com",
			"GIT_COMMITTER_NAME=Alice",
			"GIT_COMMITTER_EMAIL=alice@example.com",
		)

		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command %v failed: %s\n%s", args, err, out)
		}
	}

	runAs := func(name, email string, args ...string) {
		t.Helper()

		cmd := exec.Command(args[0], args[1:]...) //nolint:gosec // test helper
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME="+name,
			"GIT_AUTHOR_EMAIL="+email,
			"GIT_COMMITTER_NAME="+name,
			"GIT_COMMITTER_EMAIL="+email,
		)

		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command %v failed: %s\n%s", args, err, out)
		}
	}

	run("git", "init")
	run("git", "config", "user.name", "Alice")
	run("git", "config", "user.email", "alice@example.com")

	_ = os.WriteFile(filepath.Join(dir, "old.go"), []byte("package main\n"), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "shared.go"), []byte("package shared\n"), 0o600)
	run("git", "add", ".")
	run("git", "commit", "-m", "initial commit", "--date=2024-01-01T00:00:00+00:00")

	_ = os.WriteFile(filepath.Join(dir, "shared.go"), []byte("package shared\n// updated by bob\n"), 0o600)
	runAs("Bob", "bob@example.com", "git", "add", "shared.go")
	runAs("Bob", "bob@example.com", "git", "commit", "-m", "bob update", "--date=2025-06-15T00:00:00+00:00")

	_ = os.WriteFile(filepath.Join(dir, "new.go"), []byte("package new\n"), 0o600)
	run("git", "add", "new.go")
	run("git", "commit", "-m", "add new.go")

	return dir
}

func buildTree(dir string, files ...string) *model.Directory {
	root := &model.Directory{Path: dir, Name: filepath.Base(dir)}

	for _, name := range files {
		root.Files = append(root.Files, &model.File{
			Path: filepath.Join(dir, name),
			Name: name,
		})
	}

	return root
}

func TestFileAgeProvider(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := setupTestGitRepo(t)
	root := buildTree(dir, "old.go", "new.go")

	resetService()

	p := &FileAgeProvider{}
	g.Expect(p.Name()).To(Equal(FileAge))
	g.Expect(p.Kind()).To(Equal(metric.Quantity))

	err := p.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	// old.go has age > 0
	ageOld, ok := root.Files[0].Quantity(FileAge)
	g.Expect(ok).To(BeTrue())
	g.Expect(ageOld).To(BeNumerically(">", 0))

	// new.go has age >= 0 (just committed)
	ageNew, ok := root.Files[1].Quantity(FileAge)
	g.Expect(ok).To(BeTrue())
	g.Expect(ageNew).To(BeNumerically(">=", 0))

	// old.go should be older than new.go
	g.Expect(ageOld).To(BeNumerically(">", ageNew))
}

func TestFileFreshnessProvider(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := setupTestGitRepo(t)
	root := buildTree(dir, "old.go", "new.go")

	resetService()

	p := &FileFreshnessProvider{}
	err := p.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	// new.go was just committed — should be very fresh (small number)
	freshNew, ok := root.Files[1].Quantity(FileFreshness)
	g.Expect(ok).To(BeTrue())
	g.Expect(freshNew).To(BeNumerically(">=", 0))
}

func TestAuthorCountProvider(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := setupTestGitRepo(t)
	root := buildTree(dir, "shared.go", "old.go")

	resetService()

	p := &AuthorCountProvider{}
	err := p.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	// shared.go: 2 authors (Alice + Bob)
	count, ok := root.Files[0].Quantity(AuthorCount)
	g.Expect(ok).To(BeTrue())
	g.Expect(count).To(Equal(2))

	// old.go: 1 author (Alice)
	count, ok = root.Files[1].Quantity(AuthorCount)
	g.Expect(ok).To(BeTrue())
	g.Expect(count).To(Equal(1))
}

func TestGitProviderNotAGitRepo(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	root := buildTree(dir, "file.go")

	resetService()

	p := &FileAgeProvider{}
	err := p.Load(root)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("git"))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/provider/git/ -count=1 -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Implement service.go**

Create `internal/provider/git/service.go`:

```go
// Package git provides metric providers for git-derived metrics.
package git

import (
	"errors"
	"sync"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/rotisserie/eris"
)

type repoService struct {
	repo *gogit.Repository
}

var (
	svcOnce sync.Once
	svc     *repoService
	svcErr  error
)

func getService(repoPath string) (*repoService, error) {
	svcOnce.Do(func() {
		repo, err := gogit.PlainOpenWithOptions(repoPath, &gogit.PlainOpenOptions{DetectDotGit: true})
		if err != nil {
			svcErr = eris.Wrap(err, "failed to open git repository")

			return
		}

		svc = &repoService{repo: repo}
	})

	return svc, svcErr
}

// resetService clears the cached service. Test use only.
func resetService() {
	svcOnce = sync.Once{}
	svc = nil
	svcErr = nil
}

var errUntracked = errors.New("file has no git history")

func (s *repoService) fileAge(relPath string) (int, error) {
	commits, err := s.fileCommitTimes(relPath)
	if err != nil {
		return 0, err
	}

	if len(commits) == 0 {
		return 0, errUntracked
	}

	oldest := commits[len(commits)-1]
	age := time.Since(oldest)

	return int(age.Seconds()), nil
}

func (s *repoService) fileFreshness(relPath string) (int, error) {
	commits, err := s.fileCommitTimes(relPath)
	if err != nil {
		return 0, err
	}

	if len(commits) == 0 {
		return 0, errUntracked
	}

	newest := commits[0]
	freshness := time.Since(newest)

	return int(freshness.Seconds()), nil
}

func (s *repoService) authorCount(relPath string) (int, error) {
	log, err := s.repo.Log(&gogit.LogOptions{FileName: &relPath})
	if err != nil {
		return 0, eris.Wrap(err, "failed to get git log")
	}
	defer log.Close()

	authors := map[string]bool{}

	err = log.ForEach(func(c *object.Commit) error {
		authors[c.Author.Email] = true

		return nil
	})
	if err != nil {
		return 0, eris.Wrap(err, "failed to iterate commits")
	}

	if len(authors) == 0 {
		return 0, errUntracked
	}

	return len(authors), nil
}

func (s *repoService) fileCommitTimes(relPath string) ([]time.Time, error) {
	log, err := s.repo.Log(&gogit.LogOptions{FileName: &relPath})
	if err != nil {
		return nil, eris.Wrap(err, "failed to get git log")
	}
	defer log.Close()

	var times []time.Time

	err = log.ForEach(func(c *object.Commit) error {
		times = append(times, c.Author.When)

		return nil
	})
	if err != nil {
		return nil, eris.Wrap(err, "failed to iterate commits")
	}

	return times, nil
}
```

- [ ] **Step 4: Implement metrics.go**

Create `internal/provider/git/metrics.go`:

```go
package git

import (
	"errors"
	"log/slog"
	"path/filepath"

	"github.com/rotisserie/eris"

	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/palette"
)

const (
	FileAge       metric.Name = "file-age"
	FileFreshness metric.Name = "file-freshness"
	AuthorCount   metric.Name = "author-count"
)

// FileAgeProvider reports time since first commit in seconds.
type FileAgeProvider struct{}

func (*FileAgeProvider) Name() metric.Name                    { return FileAge }
func (*FileAgeProvider) Kind() metric.Kind                    { return metric.Quantity }
func (*FileAgeProvider) Dependencies() []metric.Name          { return nil }
func (*FileAgeProvider) DefaultPalette() palette.PaletteName  { return palette.Temperature }

func (p *FileAgeProvider) Load(root any) error {
	dir := root.(*model.Directory)

	s, err := getService(dir.Path)
	if err != nil {
		return eris.Wrap(err, "file-age requires a git repository")
	}

	model.WalkFiles(dir, func(f *model.File) {
		relPath, err := filepath.Rel(dir.Path, f.Path)
		if err != nil {
			slog.Warn("could not compute relative path", "path", f.Path, "error", err)

			return
		}

		age, err := s.fileAge(relPath)
		if err != nil {
			if !errors.Is(err, errUntracked) {
				slog.Debug("could not get file age", "path", relPath, "error", err)
			}

			return
		}

		f.SetQuantity(p, age)
	})

	return nil
}

// FileFreshnessProvider reports time since most recent commit in seconds.
type FileFreshnessProvider struct{}

func (*FileFreshnessProvider) Name() metric.Name                    { return FileFreshness }
func (*FileFreshnessProvider) Kind() metric.Kind                    { return metric.Quantity }
func (*FileFreshnessProvider) Dependencies() []metric.Name          { return nil }
func (*FileFreshnessProvider) DefaultPalette() palette.PaletteName  { return palette.Temperature }

func (p *FileFreshnessProvider) Load(root any) error {
	dir := root.(*model.Directory)

	s, err := getService(dir.Path)
	if err != nil {
		return eris.Wrap(err, "file-freshness requires a git repository")
	}

	model.WalkFiles(dir, func(f *model.File) {
		relPath, err := filepath.Rel(dir.Path, f.Path)
		if err != nil {
			slog.Warn("could not compute relative path", "path", f.Path, "error", err)

			return
		}

		freshness, err := s.fileFreshness(relPath)
		if err != nil {
			if !errors.Is(err, errUntracked) {
				slog.Debug("could not get file freshness", "path", relPath, "error", err)
			}

			return
		}

		f.SetQuantity(p, freshness)
	})

	return nil
}

// AuthorCountProvider reports the number of distinct commit authors.
type AuthorCountProvider struct{}

func (*AuthorCountProvider) Name() metric.Name                    { return AuthorCount }
func (*AuthorCountProvider) Kind() metric.Kind                    { return metric.Quantity }
func (*AuthorCountProvider) Dependencies() []metric.Name          { return nil }
func (*AuthorCountProvider) DefaultPalette() palette.PaletteName  { return palette.GoodBad }

func (p *AuthorCountProvider) Load(root any) error {
	dir := root.(*model.Directory)

	s, err := getService(dir.Path)
	if err != nil {
		return eris.Wrap(err, "author-count requires a git repository")
	}

	model.WalkFiles(dir, func(f *model.File) {
		relPath, err := filepath.Rel(dir.Path, f.Path)
		if err != nil {
			slog.Warn("could not compute relative path", "path", f.Path, "error", err)

			return
		}

		count, err := s.authorCount(relPath)
		if err != nil {
			if !errors.Is(err, errUntracked) {
				slog.Debug("could not get author count", "path", relPath, "error", err)
			}

			return
		}

		f.SetQuantity(p, count)
	})

	return nil
}
```

Create `internal/provider/git/register.go`:

```go
package git

import "github.com/bevan/code-visualizer/internal/metric"

// Register adds all git metric providers to the global registry.
func Register() {
	metric.Register(&FileAgeProvider{})
	metric.Register(&FileFreshnessProvider{})
	metric.Register(&AuthorCountProvider{})
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/provider/git/ -count=1 -v`
Expected: All tests pass.

- [ ] **Step 6: Run full test suite**

Run: `go test ./... -count=1`
Expected: All packages pass.

- [ ] **Step 7: Commit**

```bash
git add internal/provider/git/
git commit -m "feat(provider): add git providers (file-age, file-freshness, author-count)

Shared repoService via sync.Once. Each provider walks the tree and
sets Quantity values in seconds (age/freshness) or count (authors).
Untracked files are silently skipped.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 7: Migrate Scanner

Change `scan.Scan()` to return `*model.Directory`. Set file-size and file-type during the walk. Update FilterBinaryFiles. Delete old types.

**Files:**
- Modify: `internal/scan/scanner.go`
- Modify: `internal/scan/scanner_test.go`
- Modify: `internal/scan/scanner_unix_test.go`
- Delete contents of: `internal/scan/gitinfo.go` (keep `IsGitRepo` only)
- Delete: `internal/scan/gitinfo_test.go` (git tests moved to provider/git)

- [ ] **Step 1: Rewrite scanner.go**

Replace the full contents of `internal/scan/scanner.go` with:

```go
// Package scan provides recursive directory scanning with symlink handling.
package scan

import (
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/rotisserie/eris"

	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/provider/filesystem"
)

// fileSizeProvider and fileTypeProvider are used to set cheap metrics during scanning.
var (
	fileSizeProvider = filesystem.FileSizeProvider{}
	fileTypeProvider = filesystem.FileTypeProvider{}
)

// Scan recursively scans the directory at path and returns a model.Directory tree.
// File symlinks are followed; directory symlinks are skipped.
// Permission-denied errors are logged and scanning continues.
// Returns an error if the directory contains no files.
func Scan(path string) (*model.Directory, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, eris.Wrap(err, "failed to resolve absolute path")
	}

	root, err := scanDir(absPath)
	if err != nil {
		return nil, err
	}

	if countFiles(root) == 0 {
		return nil, errors.New("no files found in directory")
	}

	return root, nil
}

func scanDir(dirPath string) (*model.Directory, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, eris.Wrapf(err, "failed to read directory %s", dirPath)
	}

	node := &model.Directory{
		Path: dirPath,
		Name: filepath.Base(dirPath),
	}

	for _, entry := range entries {
		entryPath := filepath.Join(dirPath, entry.Name())

		if err := processEntry(node, entry, entryPath); err != nil {
			return nil, err
		}
	}

	return node, nil
}

func processEntry(node *model.Directory, entry os.DirEntry, entryPath string) error {
	info, err := os.Stat(entryPath)
	if err != nil {
		if errors.Is(err, fs.ErrPermission) {
			slog.Warn("skipping file: permission denied", "path", entryPath)

			return nil
		}

		slog.Warn("skipping file", "path", entryPath, "error", err)

		return nil
	}

	if info.IsDir() {
		return processDir(node, entry, entryPath)
	}

	if info.Mode().IsRegular() || isSymlink(entry) {
		processFile(node, entry, info, entryPath)
	}

	return nil
}

func processDir(node *model.Directory, entry os.DirEntry, entryPath string) error {
	if isSymlink(entry) {
		slog.Debug("skipping directory symlink", "path", entryPath)

		return nil
	}

	child, err := scanDir(entryPath)
	if err != nil {
		if errors.Is(err, fs.ErrPermission) {
			slog.Warn("skipping directory: permission denied", "path", entryPath)

			return nil
		}

		return err
	}

	node.Dirs = append(node.Dirs, child)

	return nil
}

func processFile(node *model.Directory, entry os.DirEntry, info os.FileInfo, entryPath string) {
	ext := strings.TrimPrefix(filepath.Ext(entry.Name()), ".")

	fileType := ext
	if fileType == "" {
		fileType = "no-extension"
	}

	f := &model.File{
		Path:      entryPath,
		Name:      entry.Name(),
		Extension: ext,
	}

	f.SetQuantity(fileSizeProvider, int(info.Size()))
	f.SetClassification(fileTypeProvider, fileType)

	node.Files = append(node.Files, f)
}

func isSymlink(entry os.DirEntry) bool {
	return entry.Type()&os.ModeSymlink != 0
}

func countFiles(node *model.Directory) int {
	count := len(node.Files)
	for _, d := range node.Dirs {
		count += countFiles(d)
	}

	return count
}

// FilterBinaryFiles returns a copy of the directory tree with binary files removed.
// Directories that become empty after removal are also pruned.
func FilterBinaryFiles(node *model.Directory) *model.Directory {
	result := &model.Directory{
		Path: node.Path,
		Name: node.Name,
	}

	for _, f := range node.Files {
		if f.IsBinary {
			slog.Debug("excluding binary file", "path", f.Path)

			continue
		}

		result.Files = append(result.Files, f)
	}

	for _, d := range node.Dirs {
		filtered := FilterBinaryFiles(d)
		if len(filtered.Files) > 0 || len(filtered.Dirs) > 0 {
			result.Dirs = append(result.Dirs, filtered)
		}
	}

	return result
}
```

- [ ] **Step 2: Trim gitinfo.go to keep only IsGitRepo**

Replace `internal/scan/gitinfo.go` with:

```go
package scan

import (
	"errors"

	"github.com/go-git/go-git/v5"
	"github.com/rotisserie/eris"
)

// IsGitRepo checks if the given path is inside a git repository.
func IsGitRepo(path string) (bool, error) {
	_, err := git.PlainOpenWithOptions(path, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		if errors.Is(err, git.ErrRepositoryNotExists) {
			return false, nil
		}

		return false, eris.Wrap(err, "failed to check git repository")
	}

	return true, nil
}
```

- [ ] **Step 3: Rewrite scanner_test.go**

Replace `internal/scan/scanner_test.go` with tests that use `*model.Directory` and `*model.File`:

```go
package scan

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/provider/filesystem"
)

func TestScanFlat(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := filepath.Join("testdata", "flat")

	root, err := Scan(dir)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(root.Name).To(Equal("flat"))
	g.Expect(root.Files).To(HaveLen(3))
	g.Expect(root.Dirs).To(BeEmpty())

	sizes := map[string]int{}
	for _, f := range root.Files {
		v, ok := f.Quantity(filesystem.FileSize)
		g.Expect(ok).To(BeTrue())
		sizes[f.Name] = v
	}

	g.Expect(sizes["small.txt"]).To(Equal(5))
	g.Expect(sizes["medium.go"]).To(Equal(100))
	g.Expect(sizes["large.rs"]).To(Equal(1000))
}

func TestScanNested(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := filepath.Join("testdata", "nested")

	root, err := Scan(dir)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(root.Name).To(Equal("nested"))
	g.Expect(root.Files).To(HaveLen(1))
	g.Expect(root.Dirs).To(HaveLen(1))

	sub := root.Dirs[0]
	g.Expect(sub.Name).To(Equal("sub"))
	g.Expect(sub.Files).To(HaveLen(1))
	g.Expect(sub.Dirs).To(HaveLen(1))

	deep := sub.Dirs[0]
	g.Expect(deep.Name).To(Equal("deep"))
	g.Expect(deep.Files).To(HaveLen(1))
	g.Expect(deep.Files[0].Name).To(Equal("leaf.md"))
}

func TestScanEmptyDir(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := filepath.Join("testdata", "empty")

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	_, err := Scan(dir)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("no files"))
}

func TestScanFollowsFileSymlinks(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := filepath.Join("testdata", "with-symlinks")

	root, err := Scan(dir)
	g.Expect(err).NotTo(HaveOccurred())

	fileNames := map[string]bool{}
	for _, f := range root.Files {
		fileNames[f.Name] = true
	}

	g.Expect(fileNames).To(HaveKey("real.txt"))
	g.Expect(fileNames).To(HaveKey("link-to-file.txt"))
}

func TestScanSkipsDirSymlinks(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := filepath.Join("testdata", "with-symlinks")

	root, err := Scan(dir)
	g.Expect(err).NotTo(HaveOccurred())

	dirNames := map[string]bool{}
	for _, d := range root.Dirs {
		dirNames[d.Name] = true
	}

	g.Expect(dirNames).To(HaveKey("target-dir"))
	g.Expect(dirNames).NotTo(HaveKey("link-to-dir"))
}

func TestScanFileExtension(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := filepath.Join("testdata", "flat")

	root, err := Scan(dir)
	g.Expect(err).NotTo(HaveOccurred())

	exts := map[string]string{}
	for _, f := range root.Files {
		exts[f.Name] = f.Extension
	}

	g.Expect(exts["small.txt"]).To(Equal("txt"))
	g.Expect(exts["medium.go"]).To(Equal("go"))
	g.Expect(exts["large.rs"]).To(Equal("rs"))
}

func TestScanSetsFileType(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := filepath.Join("testdata", "flat")

	root, err := Scan(dir)
	g.Expect(err).NotTo(HaveOccurred())

	for _, f := range root.Files {
		ft, ok := f.Classification(filesystem.FileType)
		g.Expect(ok).To(BeTrue())
		g.Expect(ft).NotTo(BeEmpty())
	}
}

func TestFilterBinaryFilesMixed(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Path: "/project",
		Name: "project",
		Files: []*model.File{
			{Path: "/project/main.go", Name: "main.go", IsBinary: false},
			{Path: "/project/image.png", Name: "image.png", IsBinary: true},
			{Path: "/project/util.go", Name: "util.go", IsBinary: false},
		},
	}

	filtered := FilterBinaryFiles(root)
	g.Expect(filtered.Files).To(HaveLen(2))
	g.Expect(filtered.Files[0].Name).To(Equal("main.go"))
	g.Expect(filtered.Files[1].Name).To(Equal("util.go"))
}

func TestFilterBinaryFilesPrunesEmptyDirs(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Path: "/project",
		Name: "project",
		Files: []*model.File{
			{Path: "/project/main.go", Name: "main.go", IsBinary: false},
		},
		Dirs: []*model.Directory{
			{
				Path: "/project/assets",
				Name: "assets",
				Files: []*model.File{
					{Path: "/project/assets/logo.png", Name: "logo.png", IsBinary: true},
				},
			},
		},
	}

	filtered := FilterBinaryFiles(root)
	g.Expect(filtered.Files).To(HaveLen(1))
	g.Expect(filtered.Dirs).To(BeEmpty())
}

//nolint:paralleltest // mutates global slog default logger
func TestFilterBinaryFilesLogsExcluded(t *testing.T) {
	g := NewGomegaWithT(t)

	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	oldDefault := slog.Default()
	slog.SetDefault(slog.New(handler))
	defer slog.SetDefault(oldDefault)

	root := &model.Directory{
		Path: "/project",
		Name: "project",
		Files: []*model.File{
			{Path: "/project/main.go", Name: "main.go", IsBinary: false},
			{Path: "/project/image.png", Name: "image.png", IsBinary: true},
		},
	}

	_ = FilterBinaryFiles(root)
	g.Expect(buf.String()).To(ContainSubstring("excluding binary file"))
}
```

- [ ] **Step 4: Rewrite scanner_unix_test.go**

Replace `internal/scan/scanner_unix_test.go`:

```go
//go:build linux || darwin

package scan

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
)

func TestScanPermissionDenied(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	tmp := t.TempDir()
	f, err := os.Create(filepath.Join(tmp, "readable.txt"))
	g.Expect(err).NotTo(HaveOccurred())
	f.WriteString("hello") //nolint:errcheck // test data
	f.Close()

	unreadable := filepath.Join(tmp, "unreadable.txt")
	err = os.WriteFile(unreadable, []byte("secret"), 0o000)
	g.Expect(err).NotTo(HaveOccurred())

	root, err := Scan(tmp)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(len(root.Files)).To(BeNumerically(">=", 1))
}
```

- [ ] **Step 5: Delete gitinfo_test.go**

Delete `internal/scan/gitinfo_test.go` — these tests are superseded by `internal/provider/git/metrics_test.go`.

- [ ] **Step 6: Run scanner tests**

Run: `go test ./internal/scan/ -count=1 -v`
Expected: All tests pass.

- [ ] **Step 7: Commit**

```bash
git add internal/scan/
git commit -m "refactor(scan): return *model.Directory, set file-size and file-type during walk

Scanner now builds model.File/Directory pointers. Sets file-size (Quantity)
and file-type (Classification) during the walk. FilterBinaryFiles works
with pointer types. Removed FileNode, DirectoryNode, EnrichWithGitMetadata,
PopulateLineCounts. Trimmed gitinfo.go to IsGitRepo only.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 8: Migrate Treemap

Update Layout to accept `*model.Directory` and `metric.Name` for sizing.

**Files:**
- Modify: `internal/treemap/layout.go`
- Modify: `internal/treemap/layout_test.go`
- Modify: `internal/treemap/node_test.go`

- [ ] **Step 1: Rewrite layout.go**

Replace the contents of `internal/treemap/layout.go`:

```go
// Package treemap implements squarified treemap layout using the
// nikolaydubina/treemap library.
package treemap

import (
	"github.com/nikolaydubina/treemap/layout"

	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
)

const (
	HeaderHeight = 20.0
	padding      = 4.0
	siblingGap   = 2.0
	minFileSize  = 1.0
)

// Layout computes a squarified treemap layout from a Directory tree.
func Layout(root *model.Directory, width, height int, sizeMetric metric.Name) TreemapRectangle {
	box := layout.Box{X: 0, Y: 0, W: float64(width), H: float64(height)}

	return layoutDir(root, box, sizeMetric)
}

func layoutDir(dir *model.Directory, box layout.Box, sizeMetric metric.Name) TreemapRectangle {
	rect := TreemapRectangle{
		X: box.X, Y: box.Y, W: box.W, H: box.H,
		Label: dir.Name, IsDirectory: true,
	}

	children := collectChildren(dir, sizeMetric)
	if len(children) == 0 {
		return rect
	}

	contentBox := contentArea(box)
	if contentBox.W <= 0 || contentBox.H <= 0 {
		return rect
	}

	areas := make([]float64, len(children))
	for i, c := range children {
		areas[i] = c.area
	}

	boxes := layout.Squarify(contentBox, areas)

	for i, c := range children {
		b := insetBox(boxes[i], siblingGap/2)
		rect.Children = append(rect.Children, layoutChild(dir, c, b, sizeMetric))
	}

	return rect
}

type child struct {
	isDir   bool
	fileIdx int
	dirIdx  int
	area    float64
}

func collectChildren(dir *model.Directory, sizeMetric metric.Name) []child {
	children := make([]child, 0, len(dir.Files)+len(dir.Dirs))

	for i, f := range dir.Files {
		area := fileSize(f, sizeMetric)
		if area <= 0 {
			area = minFileSize
		}

		children = append(children, child{isDir: false, fileIdx: i, area: area})
	}

	for i, d := range dir.Dirs {
		area := dirTotalSize(d, sizeMetric)
		if area <= 0 {
			area = minFileSize
		}

		children = append(children, child{isDir: true, dirIdx: i, area: area})
	}

	return children
}

func fileSize(f *model.File, sizeMetric metric.Name) float64 {
	v, ok := f.Quantity(sizeMetric)
	if !ok {
		return 0
	}

	return float64(v)
}

func contentArea(box layout.Box) layout.Box {
	return layout.Box{
		X: box.X + padding,
		Y: box.Y + HeaderHeight,
		W: box.W - 2*padding,
		H: box.H - HeaderHeight - padding,
	}
}

func layoutChild(dir *model.Directory, c child, b layout.Box, sizeMetric metric.Name) TreemapRectangle {
	if c.isDir {
		return layoutDir(dir.Dirs[c.dirIdx], b, sizeMetric)
	}

	f := dir.Files[c.fileIdx]

	return TreemapRectangle{
		X: b.X, Y: b.Y, W: b.W, H: b.H,
		Label: f.Name,
	}
}

func insetBox(b layout.Box, inset float64) layout.Box {
	if b.W <= 2*inset || b.H <= 2*inset {
		return b
	}

	return layout.Box{
		X: b.X + inset, Y: b.Y + inset,
		W: b.W - 2*inset, H: b.H - 2*inset,
	}
}

func dirTotalSize(dir *model.Directory, sizeMetric metric.Name) float64 {
	var total float64

	for _, f := range dir.Files {
		s := fileSize(f, sizeMetric)
		if s <= 0 {
			s = minFileSize
		}

		total += s
	}

	for _, d := range dir.Dirs {
		total += dirTotalSize(d, sizeMetric)
	}

	return total
}
```

- [ ] **Step 2: Rewrite layout_test.go and node_test.go**

Replace both test files to construct `*model.Directory` trees. Use the `filesystem.FileSize` constant for the size metric. Each test builds trees with `model.File` nodes that have `file-size` set via `SetQuantity`. See the full test code below.

For `internal/treemap/layout_test.go` — replace the file, constructing trees like:

```go
package treemap

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/provider/filesystem"
)

func makeFile(name string, size int) *model.File {
	f := &model.File{Name: name}
	f.SetQuantity(filesystem.FileSizeProvider{}, size)

	return f
}

func TestLayoutSingleFile(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name:  "root",
		Files: []*model.File{makeFile("only.go", 100)},
	}

	rects := Layout(root, 1920, 1080, filesystem.FileSize)
	g.Expect(rects.Children).To(HaveLen(1))
	g.Expect(rects.Children[0].W).To(BeNumerically(">", 0))
	g.Expect(rects.Children[0].H).To(BeNumerically(">", 0))
}

func TestLayoutProportionalAreas(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			makeFile("big.go", 900),
			makeFile("small.go", 100),
		},
	}

	rects := Layout(root, 1000, 1000, filesystem.FileSize)

	var bigRect, smallRect TreemapRectangle
	for _, c := range rects.Children {
		switch c.Label {
		case "big.go":
			bigRect = c
		case "small.go":
			smallRect = c
		}
	}

	ratio := (bigRect.W * bigRect.H) / (smallRect.W * smallRect.H)
	g.Expect(ratio).To(BeNumerically("~", 9.0, 2.0))
}

func TestLayoutNestedDirs(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name:  "root",
		Files: []*model.File{makeFile("top.go", 100)},
		Dirs: []*model.Directory{
			{
				Name:  "sub",
				Files: []*model.File{makeFile("inner.go", 200)},
			},
		},
	}

	rects := Layout(root, 1920, 1080, filesystem.FileSize)
	g.Expect(len(rects.Children)).To(BeNumerically(">=", 2))

	var dirRect *TreemapRectangle
	for i, c := range rects.Children {
		if c.IsDirectory {
			dirRect = &rects.Children[i]

			break
		}
	}

	g.Expect(dirRect).NotTo(BeNil())
	g.Expect(dirRect.Label).To(Equal("sub"))
	g.Expect(dirRect.Children).NotTo(BeEmpty())
}

func TestLayoutZeroSizeFile(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			makeFile("normal.go", 1000),
			makeFile("empty.go", 0),
		},
	}

	rects := Layout(root, 1920, 1080, filesystem.FileSize)

	var emptyRect *TreemapRectangle
	for i, c := range rects.Children {
		if c.Label == "empty.go" {
			emptyRect = &rects.Children[i]

			break
		}
	}

	g.Expect(emptyRect).NotTo(BeNil())
	g.Expect(emptyRect.W).To(BeNumerically(">", 0))
	g.Expect(emptyRect.H).To(BeNumerically(">", 0))
}
```

For `internal/treemap/node_test.go` — similarly replace with `*model.Directory` construction using `makeFile` helper (move the helper to a shared `_test.go` or define it locally in each file).

- [ ] **Step 3: Run treemap tests**

Run: `go test ./internal/treemap/ -count=1 -v`
Expected: All tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/treemap/
git commit -m "refactor(treemap): accept *model.Directory and metric.Name for sizing

Layout now reads file sizes via Quantity(sizeMetric) instead of accessing
struct fields. All tests updated to construct model.Directory trees.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 9: Migrate Render Tests

Update render test helpers to construct model trees.

**Files:**
- Modify: `internal/render/renderer_test.go`

- [ ] **Step 1: Update renderer_test.go**

Replace all `scan.DirectoryNode` / `scan.FileNode` references with `model.Directory` / `model.File`. The render package itself (`renderer.go`, `label.go`) only depends on `treemap.TreemapRectangle` — no changes needed there.

The tests that call `treemap.Layout(root, w, h)` must change to `treemap.Layout(root, w, h, filesystem.FileSize)`. The test trees must use `*model.Directory` with `makeFile` helpers. The golden-file palette tests that don't use Layout (they construct TreemapRectangles directly) need no changes.

Update imports to include `"github.com/bevan/code-visualizer/internal/model"` and `"github.com/bevan/code-visualizer/internal/provider/filesystem"`. Remove `"github.com/bevan/code-visualizer/internal/scan"`.

- [ ] **Step 2: Run render tests**

Run: `go test ./internal/render/ -count=1 -v`
Expected: All tests pass, golden files match.

- [ ] **Step 3: Commit**

```bash
git add internal/render/renderer_test.go
git commit -m "refactor(render): update tests to use model.Directory trees

Tests that call treemap.Layout now pass *model.Directory and a size metric.
No changes to renderer.go itself — it only depends on TreemapRectangle.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 10: Migrate CLI

Rewrite the CLI flow to use provider registration and `metric.Run()`.

**Files:**
- Modify: `cmd/codeviz/main.go`
- Modify: `cmd/codeviz/treemap_cmd.go`
- Modify: `cmd/codeviz/main_test.go`

- [ ] **Step 1: Rewrite main.go**

Update `main.go`:
- Register providers at startup: `filesystem.Register()` and `git.Register()` before parsing args.
- Replace `countAll(scan.DirectoryNode)` with `countAll(*model.Directory)`.
- Update `gitRequiredError` to use `metric.Name` instead of `metric.MetricName`.
- Update imports.

Key changes:

```go
import (
	// ...
	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/provider/filesystem"
	"github.com/bevan/code-visualizer/internal/provider/git"
)

func main() {
	filesystem.Register()
	git.Register()

	// ... rest of main unchanged
}

func countAll(node *model.Directory) (files int, dirs int) {
	files = len(node.Files)
	for _, d := range node.Dirs {
		dirs++
		f, d2 := countAll(d)
		files += f
		dirs += d2
	}

	return files, dirs
}

type gitRequiredError struct {
	metric metric.Name
	target string
}
```

- [ ] **Step 2: Rewrite treemap_cmd.go**

Major changes:
- `Size` field type: `metric.Name` (was `metric.MetricName`)
- `Validate()`: use `metric.Get(name)` to check validity and Kind instead of `IsValid()`/`IsNumeric()`/`IsGitRequired()`
- `Run()`: call `scan.Scan()`, then `metric.Run(root, requested)`, then filter/layout/color/render
- Delete `enrichGitMetadata`, `needsLineCounts`, `resolveGitMetric` — replaced by `metric.Run`
- Color mapping: `extractNumeric` reads from `f.Quantity(m)` / `f.Measure(m)`, `extractClassification` reads from `f.Classification(m)`
- All recursive color functions take `*model.Directory` instead of `scan.DirectoryNode`
- `collectRequestedMetrics(size, fill, border)` gathers the unique set of metrics to pass to `metric.Run`
- `resolveFillPalette` and `applyBorderColours` use `metric.Get(name)` to look up `DefaultPalette()`

Layout call: `treemap.Layout(root, width, height, c.Size)`

- [ ] **Step 3: Update main_test.go**

Replace all `scan.DirectoryNode` / `scan.FileNode` with `model.Directory` / `model.File`. Replace `countFilesInTree` with the model-based version. Tests that reference `metric.FileSize` etc. should use the provider constants (`filesystem.FileSize`, etc.) or `metric.Name(...)`.

- [ ] **Step 4: Run CLI tests**

Run: `go test ./cmd/codeviz/ -count=1 -v`
Expected: All tests pass.

- [ ] **Step 5: Run full test suite**

Run: `go test ./... -count=1`
Expected: All packages pass.

- [ ] **Step 6: Build**

Run: `go build -o bin/codeviz ./cmd/codeviz`
Expected: Build succeeds.

- [ ] **Step 7: Commit**

```bash
git add cmd/codeviz/
git commit -m "refactor(cli): use provider registration and metric.Run for all metrics

Registers filesystem and git providers at startup. Scan returns
*model.Directory. metric.Run loads only requested metrics with
parallel execution. Color mapping reads from typed getters.
Removed bespoke git/line-count enrichment pipeline.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 11: Cleanup

Delete deprecated code and flip the Name/MetricName alias direction.

**Files:**
- Modify: `internal/metric/metric.go`
- Modify: `internal/metric/metric_test.go`
- Modify: `internal/metric/registry.go`
- Modify: `internal/metric/registry_test.go`

- [ ] **Step 1: Clean metric.go**

In `internal/metric/metric.go`:
- Change `type MetricName string` to `type Name string`
- Change `type Name = MetricName` to `type MetricName = Name` (flip the alias direction — MetricName is now the alias)
- Delete `validMetrics` map
- Delete `IsValid()`, `IsNumeric()`, `IsGitRequired()` methods
- Delete `ExtractFileSize()`, `ExtractFileLines()`, `ExtractFileType()` functions
- Delete the `"github.com/bevan/code-visualizer/internal/scan"` import
- Delete the old metric name constants (FileSize, FileLines, etc.) — they now live in provider packages

- [ ] **Step 2: Clean registry.go**

- Delete `metricDefaultPalette` map
- Delete `DefaultPaletteFor()` function (replaced by `Provider.DefaultPalette()`)

- [ ] **Step 3: Update metric tests**

In `internal/metric/metric_test.go`:
- Delete `TestMetricName_IsValid`, `TestMetricName_IsNumeric`, `TestMetricName_IsGitRequired`
- Delete all `TestExtract*` tests
- Remove the `"github.com/bevan/code-visualizer/internal/scan"` import

In `internal/metric/registry_test.go`:
- Delete `TestDefaultPaletteFor` and `TestDefaultPaletteFor_InvalidMetric`

- [ ] **Step 4: Update CLI references**

In `cmd/codeviz/treemap_cmd.go`: replace any remaining `metric.MetricName(...)` casts with `metric.Name(...)`.

- [ ] **Step 5: Run full test suite**

Run: `go test ./... -count=1`
Expected: All packages pass.

- [ ] **Step 6: Build**

Run: `go build -o bin/codeviz ./cmd/codeviz`
Expected: Build succeeds.

- [ ] **Step 7: Commit**

```bash
git add internal/metric/ cmd/codeviz/
git commit -m "refactor(metric): delete deprecated types, extractors, and palette registry

Name is now the primary type (MetricName kept as alias for compatibility).
Removed IsValid/IsNumeric/IsGitRequired, Extract* functions, validMetrics
map, DefaultPaletteFor, and metric name constants (moved to providers).

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 12: Final Verification

Verify the full pipeline works end-to-end.

**Files:** None modified.

- [ ] **Step 1: Run full test suite with race detector**

Run: `go test ./... -count=1 -race`
Expected: All packages pass, no data races.

- [ ] **Step 2: Build**

Run: `go build -o bin/codeviz ./cmd/codeviz`
Expected: Build succeeds.

- [ ] **Step 3: Run the binary against this repository**

Run: `./bin/codeviz render treemap . -o /tmp/test-codeviz.png -s file-size`
Expected: Produces a valid PNG file.

Run: `./bin/codeviz render treemap . -o /tmp/test-codeviz2.png -s file-lines -f file-type`
Expected: Produces a valid PNG file.

- [ ] **Step 4: Verify golden files still match**

Run: `go test ./internal/render/ -count=1 -v -run Golden`
Expected: All golden file tests pass (no visual regression).

- [ ] **Step 5: Commit any remaining changes**

If `go mod tidy` changed `go.mod`/`go.sum`:

```bash
go mod tidy
git add go.mod go.sum
git commit -m "chore: tidy go modules

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```
