# Metrics Framework Design

## Problem

The current implementation stores metrics as hardcoded fields on `scan.FileNode` and `scan.DirectoryNode`. Adding a new metric requires modifying these structs, their enrichment functions, the extract helpers in the metric package, and every consumer. This violates the open/closed principle and won't scale to the hundreds of metrics planned.

Performance is also a concern: all metrics are computed upfront with no parallelism, and expensive operations (git log traversals) block the entire pipeline.

## Approach

Replace the hardcoded metric fields with a pluggable provider framework. Each metric has a dedicated provider that knows how to load its values onto a shared model tree. A central scheduler resolves dependencies, runs independent providers in parallel, and surfaces errors.

**Scope:** Migrate the existing 6 metrics (`file-size`, `file-lines`, `file-type`, `file-age`, `file-freshness`, `author-count`) into the new framework. No new metrics are added in this change.

## Design

### Core Framework Types (`internal/metric/`)

#### Name and Kind

```go
// metric.go

type Name string

type Kind int

const (
    Quantity       Kind = iota // int values (file size, line count)
    Measure                    // float64 values (percentages, rates)
    Classification             // string values (file type, t-shirt size)
)
```

`Name` is defined in the metric package (renamed from the current `MetricName`). Metric name constants are defined by each provider package (e.g. `filesystem.FileSize`, `git.FileAge`).

#### Provider Interface

```go
type Provider interface {
    Name() Name
    Kind() Kind
    Dependencies() []Name
    DefaultPalette() palette.PaletteName
    Load(root *model.Directory) error
}
```

- `Name()` — unique identifier for this metric.
- `Kind()` — which value type this metric produces.
- `Dependencies()` — other metrics that must be loaded first. Returns nil if none.
- `DefaultPalette()` — the palette to use when the user doesn't specify one.
- `Load()` — populates metric values on the tree. Receives the root directory by pointer. Returns an error if loading fails (e.g. not in a git repo for git metrics). For metrics already set during the scan (file-size, file-type), `Load()` is a no-op that returns nil.

#### Registry

```go
// registry.go

func Register(p Provider)              // panics on duplicate name
func Get(name Name) (Provider, bool)   // lookup by name
func All() []Provider                  // all registered providers
```

Registration is unconditional and cannot fail. All providers are registered at startup regardless of whether they'll be used.

#### Scheduler

```go
// run.go

func Run(root *model.Directory, requested []Name) error
```

Execution steps:

1. Start with the explicitly requested metric names.
2. Recursively expand to include all transitive dependencies.
3. If any dependency has no registered provider, return an error (programming bug).
4. Topological sort the full set.
5. If a cycle is detected, return an error (programming bug).
6. Group providers into execution levels (providers whose dependencies are all satisfied).
7. Execute each level in parallel using `errgroup`.
8. If any provider's `Load()` returns an error, cancel remaining providers in that level and return the error.

### Model Package (`internal/model/`)

`scan.FileNode` and `scan.DirectoryNode` move to `internal/model/` and are renamed to `File` and `Directory`.

#### File

```go
// file.go

type File struct {
    Path      string
    Name      string
    Extension string
    IsBinary  bool // structural flag, not a metric

    mu              sync.RWMutex
    quantities      map[metric.Name]int
    measures        map[metric.Name]float64
    classifications map[metric.Name]string
}
```

Typed getters return `(value, bool)` where bool indicates whether the metric has been set:

```go
func (f *File) Quantity(name metric.Name) (int, bool)
func (f *File) Measure(name metric.Name) (float64, bool)
func (f *File) Classification(name metric.Name) (string, bool)
```

Typed setters take a `metric.Provider` (not a `metric.Name`) so the setter can verify `provider.Kind()` matches the method called — e.g. `SetQuantity` panics if the provider's Kind is not `Quantity`:

```go
func (f *File) SetQuantity(p metric.Provider, v int)
func (f *File) SetMeasure(p metric.Provider, v float64)
func (f *File) SetClassification(p metric.Provider, v string)
```

The three metric maps are lazily initialized on first `Set*` call. A single `sync.RWMutex` per File guards all three maps for concurrent access.

There are no special-cased metric fields. File size is stored via `file.SetQuantity(fileSizeProvider, value)` during the scan, not as a dedicated struct field.

#### Directory

```go
// directory.go

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
```

Same getter/setter methods as File. Uses pointer slices (`[]*File`, `[]*Directory`) so providers can mutate nodes in place.

### Providers

#### Filesystem Providers (`internal/provider/filesystem/`)

```go
const (
    FileSize  metric.Name = "file-size"
    FileLines metric.Name = "file-lines"
    FileType  metric.Name = "file-type"
)
```

**FileSizeProvider** — `Kind: Quantity`, no dependencies, no-op `Load()` (value set during scan), default palette: `Neutral`.

**FileLinesProvider** — `Kind: Quantity`, no dependencies, `Load()` walks the tree and counts lines for each non-binary file, sets `SetQuantity(self, count)`, default palette: `Neutral`.

**FileTypeProvider** — `Kind: Classification`, no dependencies, no-op `Load()` (value set during scan), default palette: `Categorization`.

#### Git Providers (`internal/provider/git/`)

```go
const (
    FileAge       metric.Name = "file-age"
    FileFreshness metric.Name = "file-freshness"
    AuthorCount   metric.Name = "author-count"
)
```

All three git providers share a `repoService` via `sync.Once`:

```go
// service.go (unexported)

var (
    svcOnce sync.Once
    svc     *repoService
    svcErr  error
)

func getService(repoPath string) (*repoService, error) {
    svcOnce.Do(func() {
        svc, svcErr = newRepoService(repoPath)
    })
    return svc, svcErr
}
```

The first provider to call `getService()` initializes the shared git repository handle. If the path is not a git repository, all three providers receive the same error from their `Load()` calls.

**FileAgeProvider** — `Kind: Quantity`, no dependencies, `Load()` calls `getService(root.Path)`, walks tree, sets `SetQuantity(self, durationDays)`, default palette: `Temperature`.

**FileFreshnessProvider** — `Kind: Quantity`, no dependencies, `Load()` calls `getService(root.Path)`, walks tree, sets `SetQuantity(self, durationDays)`, default palette: `Temperature`.

**AuthorCountProvider** — `Kind: Quantity`, no dependencies, `Load()` calls `getService(root.Path)`, walks tree, sets `SetQuantity(self, count)`, default palette: `GoodBad`.

#### Registration

Each provider package exports a `Register()` function:

```go
// filesystem/register.go
func Register() {
    metric.Register(FileSizeProvider{})
    metric.Register(FileLinesProvider{})
    metric.Register(FileTypeProvider{})
}

// git/register.go
func Register() {
    metric.Register(&FileAgeProvider{})
    metric.Register(&FileFreshnessProvider{})
    metric.Register(&AuthorCountProvider{})
}
```

Registration always succeeds. Errors are deferred to `Load()`.

### Scanner Changes (`internal/scan/`)

`scan.Scan()` returns `*model.Directory` instead of `scan.DirectoryNode`. During the directory walk:

1. Constructs `model.Directory` and `model.File` nodes with pointer slices.
2. Sets `file-size` via `file.SetQuantity(fileSizeProvider, info.Size())`.
3. Sets `file-type` via `file.SetClassification(fileTypeProvider, fileType)`.
4. Sets `IsBinary` as a direct field.
5. Does **not** count lines or fetch git metadata.

This means the scan package imports `provider/filesystem` for the provider instances used in `Set*` calls. This is an intentional compile-time dependency — the scanner is the code that populates these cheap filesystem metrics.

`scan.FileNode` and `scan.DirectoryNode` are deleted.

`FilterBinaryFiles` stays in the scan package — it operates on tree structure, not metrics.

`EnrichWithGitMetadata` and `PopulateLineCounts` are deleted — their logic moves into the git and filesystem providers respectively.

### Consumer-Side Changes

#### CLI Flow (`cmd/codeviz/treemap.go`)

```
1. filesystem.Register()
2. git.Register()
3. root := scan.Scan(path)
4. requested := collectRequestedMetrics(size, fill, border)
5. metric.Run(root, requested)
6. rect := treemap.Layout(root, width, height, sizeMetric)
7. applyColours(rect, root, ...)
8. render.RenderPNG(rect, ...)
```

#### Treemap Package

`treemap.Layout()` takes `*model.Directory` and a `metric.Name` for sizing. Reads size values via `file.Quantity(sizeMetric)` instead of accessing `node.Size` directly.

#### Metric Package Deletions

- `ExtractFileSize()`, `ExtractFileLines()`, `ExtractFileType()` — deleted. Consumers read from model nodes.
- `IsNumeric()`, `IsGitRequired()` on `MetricName` — replaced by querying the registry: `metric.Get(name).Kind()`.
- `metricDefaultPalette` map — deleted. Default palettes are on the Provider interface.
- `validMetrics` map — replaced by `metric.Get(name)` returning `(Provider, bool)`.
- `MetricName` type — renamed to `Name` (see Core Framework Types).

#### Metric Package Retained

- `ComputeBuckets()`, `BucketBoundaries`, `BucketIndex()` — these quantile-based bucketing utilities are used by the color mapping pipeline and remain in the metric package unchanged.

### Error Handling

**Circular dependencies:** `Run()` detects cycles during topological sort and returns: `"circular dependency detected: X → Y → X"`.

**Unknown dependency:** If a provider declares a dependency on a metric with no registered provider, `Run()` returns: `"metric X depends on unknown metric Y — no provider registered"`. This is a programming bug.

**Provider Load() failure:** The errgroup cancels remaining providers in the current execution level. `Run()` returns the first error. The CLI reports it and exits.

**Metric not set after successful Load():** The getter returns `(zero, false)`. Consumers handle this — the color mapper can assign a "no data" color for nodes missing a metric value.

### Testing Strategy

**Provider tests (per provider):**
- Build a `*model.Directory` tree in memory with known file structure.
- Call `provider.Load(tree)`.
- Assert metric values on nodes using typed getters.
- For git providers: use a temp git repo fixture (matches existing `gitinfo_test.go` patterns).

**Framework tests (`metric` package):**
- Registration: duplicate name panics, `Get()` and `All()` work correctly.
- Dependency resolution: auto-expands transitive deps, detects cycles, detects unknown deps.
- Scheduler: mock providers that record call order; verify topological ordering and parallel execution.
- Error propagation: mock provider returns error; verify `Run()` surfaces it.

**Integration tests (CLI level):**
- Existing golden-file tests (Goldie) continue to work — rendered PNG output should not change.
- Update test setup to use the new registration + `metric.Run()` flow.

**Concurrency tests:**
- Multiple providers writing to the same `model.File` concurrently — verify no data races via `go test -race`.
