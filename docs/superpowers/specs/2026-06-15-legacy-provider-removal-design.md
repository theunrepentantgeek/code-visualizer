# Legacy Provider Removal (Phase 4)

**Date:** 2026-06-15
**Status:** Approved
**Depends on:** [Metric Expression Computation](2026-06-14-metric-expression-computation-design.md) (Phases 1-3)

## Problem

The codebase currently has two parallel metric systems:

1. **Legacy** — `provider.Interface`-based registry with 45 flat metric definitions
   (e.g. `public-method-count`, `max-cyclomatic-complexity`)
2. **Expression** — `BaseMetricDescriptor` registry with composable expressions
   (e.g. `public.methods.count`, `max.cyclomatic-complexity`)

Both systems are running simultaneously. The expression system fully covers all
use cases that the legacy system handles. Maintaining both causes confusion,
code duplication, and blocks further metric development.

## Goal

Remove the legacy provider system entirely. All metric resolution, loading,
validation, and rendering flows through the expression/base-metric system.

## Scope

### Removed

| Component                                                                                | Location                                                   |
| ---------------------------------------------------------------------------------------- | ---------------------------------------------------------- |
| `provider.Interface` type                                                                | `internal/provider/provider.go`                            |
| Legacy registry (`globalRegistry`)                                                       | `internal/provider/registry.go`                            |
| `Register`, `Get`, `All`, `AllFor`, `GetDescriptor`, `FindWithHint`, `Names`, `NamesFor` | `internal/provider/registry.go`                            |
| `AllDescriptors`, `AllDescriptorsFor`                                                    | `internal/provider/registry.go`                            |
| `provider.Run` (legacy dep-expansion + topo-sort against Interface registry)             | `internal/provider/run.go`                                 |
| Go provider `providerDefs` (35 flat metrics)                                             | `internal/provider/golang/provider_defs.go`                |
| Go provider `goProvider` type + `walkGoFiles`                                            | `internal/provider/golang/go_provider.go`                  |
| Git provider `providerDefs` (7 flat metrics)                                             | `internal/provider/git/git_provider.go` (providerDefs map) |
| Filesystem legacy provider structs                                                       | `internal/provider/filesystem/metrics.go` (provider impls) |
| `metric.Target` type (`File`, `Directory`)                                               | `internal/metric/metric.go`                                |
| `MetricDescriptor` type                                                                  | `internal/provider/provider.go`                            |
| `provider.Descriptor()` helper                                                           | `internal/provider/provider.go`                            |
| `findLegacyMetrics` in help                                                              | `cmd/codeviz/help_metrics_cmd.go`                          |

### Kept (Adapted)

| Component                              | Changes                                    |
| -------------------------------------- | ------------------------------------------ |
| Base registry (`globalBaseRegistry`)   | Add loader registration                    |
| `BaseMetricDescriptor`                 | Unchanged                                  |
| `ProviderDescriptor`                   | Unchanged                                  |
| `ResolveExpression` / `ResolvedMetric` | Unchanged                                  |
| `provider.Run` (new version)           | Rewritten to use base registry loaders     |
| Provider `Load()` implementations      | Kept for filesystem/git file-level metrics |
| `PopulateDeclarations` stage           | Unchanged (handles declaration-level data) |

## Design

### 1. Loader Registration

Providers need a way to register their `Load()` function against the base
metrics they populate. New type:

```go
// BaseMetricLoader describes a unit of metric loading work.
type BaseMetricLoader struct {
    // Metrics lists the base metric names this loader populates.
    Metrics []metric.Name
    // Dependencies lists base metrics that must be loaded first.
    Dependencies []metric.Name
    // Load populates the directory tree with metric values.
    Load func(root *model.Directory) error
}
```

Registration:

```go
// RegisterLoader adds a loader to the global base registry.
func RegisterLoader(loader BaseMetricLoader)
```

The base registry gains a `loaders []BaseMetricLoader` field and a method to
find loaders needed for a set of requested base metric names.

### 2. Rewritten `provider.Run`

```go
func Run(root *model.Directory, requested []metric.Name, progress MetricProgress) error
```

Changes:
- Drops `target metric.Target` parameter (always file-level loading)
- Resolves requested names → finds covering loaders from base registry
- Topo-sorts loaders by their `Dependencies`
- Runs independent loaders in parallel (same errgroup pattern)
- Progress reporting via `MetricProgress` (simplified interface)

### 3. Provider Registration Rewiring

#### Filesystem

Keep the three `Load()` implementations (`FileSizeProvider`, `FileLinesProvider`,
`FileTypeProvider`) but register them as loaders instead of via `provider.Register`:

```go
func Register() {
    RegisterBase()
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

#### Git

The git provider currently has 7 separate `Load()` calls (one per metric), each
independently walking git history. Consolidate into a single loader that
populates all 7 file-level metrics in one pass:

```go
func Register() {
    RegisterBase()
    provider.RegisterLoader(provider.BaseMetricLoader{
        Metrics: []metric.Name{
            FileAge, FileFreshness, AuthorCount, CommitCount,
            TotalLinesAdded, TotalLinesRemoved, CommitDensity,
        },
        Load: loadAllGitMetrics,
    })
}
```

This is a performance improvement — currently git spawns 7 independent walks.

#### Go

The Go provider's legacy `Load()` is fully replaced by:
1. `PopulateDeclarations` (populates per-declaration data from AST analysis)
2. `ComputeAggregations` (aggregates declarations into file/directory values)

The two remaining file-level Go metrics (`imports`, `comment-ratio`) need a
small dedicated loader:

```go
func Register() {
    RegisterBase()
    provider.RegisterLoader(provider.BaseMetricLoader{
        Metrics: []metric.Name{Imports, CommentRatio},
        Load:    loadFileMetrics,  // slim loader for imports + comment-ratio
    })
}
```

`loadFileMetrics` walks Go files and sets `imports` count and `comment-ratio`
directly on the file's MetricContainer. Declaration-level metrics (types,
functions, methods, etc.) are NOT populated by this loader — they come from
`PopulateDeclarations`.

### 4. `metric.Target` Removal

`metric.Target` (`File`, `Directory`) is replaced by `metric.MetricLevel`
(`LevelFile`, `LevelDeclaration`, `LevelCommit`, `LevelDirectory`).

All call sites that pass `metric.File` or `metric.Directory` are updated to use
the equivalent `MetricLevel` constant or removed entirely (most were for the
legacy registry lookup).

### 5. Validation (`MetricSpec.Validate`)

The current logic:
1. Try expression parse + resolve → pass
2. Fall back to legacy `provider.Get` → pass
3. Fall back to `provider.FindWithHint` → pass/fail

New logic:
1. Try expression parse + resolve → pass
2. If parse fails, check if name matches a known base metric at file level (bare
   metric without aggregation is valid as direct file-level usage)
3. On failure, produce helpful error with fuzzy match suggestions

Helpful error for old-style metric names:

```
unknown metric "cyclomatic-complexity-max"
  hint: try "cyclomatic-complexity.max" (use dots: [filter.]metric[.aggregation])
```

### 6. CLI Command Size/Axis Validation

CLI commands currently validate `--size`, `--fill`, `--border` using
`provider.GetDescriptor`. These change to:

1. Parse as expression
2. If bare metric name, look up in base registry via `provider.GetBase()`
3. Use `BaseMetricDescriptor.Kind` for validation (must be Quantity for size)

### 7. `help metrics` Command

Remove the "Other metrics" (legacy) section entirely. The command already
displays base metrics with their aggregations and filters. Users seeing old
metric names get the helpful validation error pointing them to the new syntax.

### 8. Scatter and Spiral Migration

These two viz packages still use `provider.GetDescriptor` directly:
- `internal/scatter/inks.go`
- `internal/scatter/stages.go`
- `internal/spiral/inks.go`
- `internal/spiral/aggregation.go`

Migrate to the same `requested.DescriptorFor()` pattern used by bubbletree,
radialtree, and treemap. Their `BuildInks` / ink-related functions gain a
`stages.RequestedMetrics` parameter.

### 9. `stages.RunProviders` Update

```go
func RunProviders(c *CommonState) error {
    return provider.Run(c.Root, c.Requested.BaseMetrics, metricProg)
}
```

The `LegacyNames()` method is removed from `RequestedMetrics`. The `Legacy`
field is removed — all metrics are now either expressions (with base metrics
extracted) or validation errors.

### 10. `ClassifyRequestedMetrics` Simplification

The "legacy fallback" path is removed:
- If expression parse fails → validation error (not silent fallback)
- If expression resolves with `NeedsAggregation: false` → it's a bare base
  metric, add to `BaseMetrics` list (still needs `Load()`)
- No more `Legacy []metric.Name` field

## Migration Table

Grammar reminder: `[filter.]base-metric[.aggregation]` — aggregation is SUFFIX.

| Legacy metric                | Expression equivalent                                        |
| ---------------------------- | ------------------------------------------------------------ |
| `file-size`                  | `file-size` (bare, unchanged)                                |
| `file-lines`                 | `file-lines` (bare, unchanged)                               |
| `file-type`                  | `file-type` (bare, unchanged)                                |
| `type-count`                 | `types.count`                                                |
| `public-type-count`          | `public.types.count`                                         |
| `private-type-count`         | `private.types.count`                                        |
| `interface-count`            | `interfaces.count`                                           |
| `public-interface-count`     | `public.interfaces.count`                                    |
| `private-interface-count`    | `private.interfaces.count`                                   |
| `struct-count`               | `structs.count`                                              |
| `public-struct-count`        | `public.structs.count`                                       |
| `private-struct-count`       | `private.structs.count`                                      |
| `function-count`             | `functions.count`                                            |
| `public-function-count`      | `public.functions.count`                                     |
| `private-function-count`     | `private.functions.count`                                    |
| `method-count`               | `methods.count`                                              |
| `public-method-count`        | `public.methods.count`                                       |
| `private-method-count`       | `private.methods.count`                                      |
| `constant-count`             | `constants.count`                                            |
| `public-constant-count`      | `public.constants.count`                                     |
| `private-constant-count`     | `private.constants.count`                                    |
| `variable-count`             | `variables.count`                                            |
| `public-variable-count`      | `public.variables.count`                                     |
| `private-variable-count`     | `private.variables.count`                                    |
| `import-count`               | `imports` (bare)                                             |
| `stdlib-import-count`        | `stdlib.imports` (filtered file-level, loader-populated)     |
| `external-import-count`      | `external.imports` (filtered file-level, loader-populated)   |
| `internal-import-count`      | `internal.imports` (filtered file-level, loader-populated)   |
| `declaration-count`          | `declarations.count` (**requires new base metric**)          |
| `public-declaration-count`   | `public.declarations.count` (**requires new base metric**)   |
| `private-declaration-count`  | `private.declarations.count` (**requires new base metric**)  |
| `cyclomatic-complexity-sum`  | `cyclomatic-complexity.sum`                                  |
| `cyclomatic-complexity-max`  | `cyclomatic-complexity.max`                                  |
| `cyclomatic-complexity-mean` | `cyclomatic-complexity.mean`                                 |
| `function-length-sum`        | `function-length.sum`                                        |
| `function-length-max`        | `function-length.max`                                        |
| `function-length-mean`       | `function-length.mean`                                       |
| `comment-ratio`              | `comment-ratio` (bare, unchanged)                            |
| `file-age`                   | `file-age` (bare, unchanged)                                 |
| `file-freshness`             | `file-freshness` (bare, unchanged)                           |
| `author-count`               | `author-count` (bare, unchanged)                             |
| `commit-count`               | `commit-count` (bare, unchanged)                             |
| `total-lines-added`          | `total-lines-added` (bare, unchanged)                        |
| `total-lines-removed`        | `total-lines-removed` (bare, unchanged)                      |
| `commit-density`             | `commit-density` (bare, unchanged)                           |

### Filtered File-Level Metrics

`stdlib.imports`, `external.imports`, `internal.imports` resolve as file-level
metrics with a filter but no aggregation (`NeedsAggregation=false`). This means:
- They are NOT processed by `ComputeAggregations`
- The file-level loader must populate them directly under their result name
  (e.g. `f.SetQuantity("stdlib.imports", count)`)
- At directory level, they can be further aggregated: `stdlib.imports.sum`

## New Base Metric: `declarations`

The legacy `declaration-count` aggregates types+functions+methods+constants+variables.
To support `declarations.count`, add a new `declarations` base metric at declaration
level — it represents "all declarations regardless of kind". Its filter function
always returns true. Supports `public`/`private` visibility filters.

## Naming Note

The filesystem base metric is `file-size` (not `file-bytes`). The original
computation spec incorrectly used `file-bytes` in examples. All references
should use `file-size`.

## Error Messages

When a metric name fails validation, attempt fuzzy matching:

1. Strip hyphens, split into words
2. Check if words match known base metrics + aggregation combos
3. Suggest the dot-separated equivalent

Example:
```
unknown metric "public-method-count"
  hint: try "public.methods.count"
  (metric expressions use dots: [filter.]metric[.aggregation])
```

## Testing Strategy

- Unit tests for `provider.Run` with new loader system
- Unit tests for validation error messages (fuzzy suggestions)
- Integration: verify all sample configs render correctly with expression syntax
- Golden file tests should not change (same computed values, different path)
- CI gate: `task ci` passes with zero legacy API usage

## Ordering

Implementation order:

1. Add `BaseMetricLoader` + `RegisterLoader` + rewrite `provider.Run`
2. Rewire filesystem provider registration (simplest, proves the pattern)
3. Rewire git provider (consolidate to single loader)
4. Rewire Go provider (remove 35 flat metrics, keep slim file-level loader)
5. Migrate scatter + spiral to `DescriptorFor` pattern
6. Remove `metric.Target` type
7. Update `MetricSpec.Validate` (remove legacy fallback)
8. Update CLI commands (size/axis validation)
9. Update `help metrics` command
10. Remove `RequestedMetrics.Legacy` field + `LegacyNames()` method
11. Delete `registry.go`, `provider.Interface`, `MetricDescriptor`
12. Update sample configs to expression syntax
13. Clean up tests
