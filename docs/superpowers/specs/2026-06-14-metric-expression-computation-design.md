# Metric Expression Computation

**Date:** 2026-06-14
**Status:** Approved
**Depends on:** [Metric Expressions Design](2026-06-14-metric-expressions-design.md)

## Problem

The metric expressions system (PR #413) adds parsing, validation, and registration
of composable metric expressions (`[filter.]base-metric[.aggregation]`), but does
not compute aggregated values. If a user writes `mean.file-bytes` in their config,
validation passes but the pipeline crashes at runtime because no provider exists
with that literal name.

This spec covers all remaining work to make metric expressions produce computed
values end-to-end.

## Phasing

| Phase | Scope | Model changes? |
|-------|-------|----------------|
| 1 | Pipeline integration + file→directory aggregation | No |
| 2 | Declaration model + Go provider per-declaration data | Yes |
| 3 | Commit model + git provider per-commit data | Yes |
| 4 | Legacy flat provider removal | No (deletion) |

Each phase is independently shippable. Legacy providers remain running in parallel
until Phase 4 explicitly removes them.

---

## Phase 1: Pipeline Integration + File→Directory Aggregation

### Goal

Make expressions like `sum.file-bytes`, `max.file-lines`, `mode.file-type` compute
values at directory level by aggregating file-level data from all descendants.

### Design

#### 1.1 Split Requested Metrics

`stages.CollectRequestedMetrics` currently returns a flat list of metric names.
Change it to return a struct containing:

```go
type RequestedMetrics struct {
    // Base metric names to pass to provider.Run (deduplicated)
    BaseMetrics []metric.Name
    // Full expressions that need aggregation post-provider-run
    Expressions []metric.MetricExpression
    // Expressions that need no aggregation (bare metric at native level)
    PassThrough []metric.MetricExpression
}
```

For each requested metric string:
- Parse with `metric.ParseExpression`
- If parse fails, treat as legacy metric name (pass directly to `provider.Run`)
- If parse succeeds, resolve with `provider.ResolveExpression`:
  - If `NeedsAggregation: true` → add base metric to `BaseMetrics`, add expression to `Expressions`
  - If `NeedsAggregation: false` → add base metric to `BaseMetrics`, add to `PassThrough`

#### 1.2 New Stage: `stages.ComputeAggregations`

Runs after `stages.RunProviders`, before rendering stages.

```go
func ComputeAggregations(root *model.Directory, expressions []metric.MetricExpression) error
```

For each expression:
1. Determine the source level from the base metric's `BaseMetricDescriptor.Level`
2. Walk the directory tree recursively
3. At each directory node, collect values from all descendant source-level nodes:
   - `LevelFile` → collect from all descendant files
   - `LevelDeclaration` → collect from all declarations in all descendant files (Phase 2)
   - `LevelCommit` → collect from all commits in all descendant files (Phase 3)
4. Apply the filter (if present) to exclude non-matching source nodes
5. Apply the aggregation function to the collected values
6. Store the result on the directory's `MetricContainer` under `expression.ResultName()`

#### 1.3 Recursive Collection

Aggregation collects from **all descendants**, not just immediate children. For
example, `mean.file-bytes` on `/src/` includes files in `/src/pkg/foo/bar/`.

Each directory gets its own independently-computed aggregate (computed from its
own descendants, not from child directory aggregates). This avoids the
averaging-averages problem.

#### 1.4 Pipeline Wiring

The viz-specific stages (treemap, etc.) need minimal changes:
- `c.Requested` changes from `[]metric.Name` to `RequestedMetrics`
- `stages.RunProviders` receives `c.Requested.BaseMetrics` plus any legacy names
- New `stages.ComputeAggregations(c.Root, c.Requested.Expressions)` call added
  after `RunProviders`
- Rendering stages look up values by `expression.ResultName()` which is already
  the key used in config/MetricSpec

#### 1.5 Scope for Phase 1

Only `LevelFile` base metrics are aggregatable in Phase 1:
- `file-bytes` (filesystem) — sum, min, max, mean, count, range
- `file-lines` (filesystem) — sum, min, max, mean, count, range
- `file-type` (filesystem) — mode, distinct
- All git file-level metrics — appropriate aggregations per descriptor

If an expression references a `LevelDeclaration` or `LevelCommit` metric,
`ComputeAggregations` returns a clear error:
`"aggregation of declaration-level metrics requires Phase 2 (declaration model)"`

No file-level metrics currently declare filters (filesystem and git providers
have `Filters: nil`), so Phase 1 does not implement filter predicate evaluation.
If a filter is present on a file-level expression, validation will already reject
it (the base metric's descriptor doesn't list any valid filters).

---

## Phase 2: Declaration Model + Go Provider

### Goal

Enable `mean.cyclomatic-complexity`, `max.function-length`, `public.types.count`
to aggregate across all declarations in a directory's descendant files.

### Design

#### 2.1 New Model Type

```go
// internal/model/declaration.go
type Declaration struct {
    Name       string // e.g., "HandleRequest", "UserService"
    Kind       string // e.g., "function", "method", "interface", "struct"
    Visibility string // "public" or "private"
    MetricContainer
}
```

#### 2.2 Extend model.File

```go
type File struct {
    Path         string
    Declarations []*Declaration
    Commits      []*Commit // Phase 3
    MetricContainer
}
```

#### 2.3 Go Provider Changes

The Go provider currently iterates declarations in `analyzeFile()` but discards
per-declaration data after aggregating into `fileStats`. Changes:

1. For each declaration encountered during analysis, create a `*model.Declaration`
2. Set metrics on the declaration's `MetricContainer`:
   - `cyclomatic-complexity` (Measure)
   - `function-length` (Quantity — lines of code)
   - `parameter-count` (Quantity)
   - `return-count` (Quantity)
3. Set `Kind` and `Visibility` fields for filter matching
4. Append to `file.Declarations`
5. Continue computing file-level aggregates for legacy compatibility

#### 2.4 Filter Predicate Evaluation

`ComputeAggregations` needs to evaluate filters against declarations. The filter
vocabulary is defined per provider in `ProviderDescriptor.Filters`. For the Go
provider:

| Filter | Matches |
|--------|---------|
| `public` | `decl.Visibility == "public"` |
| `private` | `decl.Visibility == "private"` |
| `stdlib` | declaration imports from stdlib (N/A for declarations — skip) |
| `external` | N/A for declarations |
| `internal` | N/A for declarations |

Implementation: each provider registers a `FilterFunc` that takes a filter name
and a model node, returning whether the node passes:

```go
type FilterFunc func(filter metric.FilterName, node any) bool
```

This is registered on `ProviderDescriptor` or `BaseMetricDescriptor` and used by
`ComputeAggregations` during collection.

#### 2.5 Walk Extension

`model.WalkFiles` exists but there is no `WalkDeclarations`. Add:

```go
func WalkDeclarations(dir *Directory, fn func(*Declaration, *File)) {
    WalkFiles(dir, func(f *File) {
        for _, d := range f.Declarations {
            fn(d, f)
        }
    })
}
```

---

## Phase 3: Commit Model + Git Provider

### Goal

Enable `max.lines-changed`, `sum.lines-added`, etc. to aggregate per-commit data.

### Design

#### 3.1 New Model Type

```go
// internal/model/commit.go
type Commit struct {
    Hash   string
    Author string
    Date   time.Time
    MetricContainer
}
```

#### 3.2 Extend model.File

`File.Commits []*Commit` — each commit that touched this file gets a record.

#### 3.3 Git Provider Changes

The git provider currently walks commit history and produces per-file rollups.
Changes:

1. For each commit that touches a file, create a `*model.Commit`
2. Set metrics on the commit's `MetricContainer`:
   - `lines-added` (Quantity)
   - `lines-removed` (Quantity)
   - `lines-changed` (Quantity)
3. Append to `file.Commits`
4. Continue computing file-level aggregates for legacy compatibility

#### 3.4 Walk Extension

```go
func WalkCommits(dir *Directory, fn func(*Commit, *File)) {
    WalkFiles(dir, func(f *File) {
        for _, c := range f.Commits {
            fn(c, f)
        }
    })
}
```

#### 3.5 Commit Filters (Future)

The initial implementation supports no commit-level filters. Future candidates:
- `recent` — commits within N days
- Author-based filters

These are NOT part of this spec; they would be a separate enhancement.

---

## Phase 4: Legacy Flat Provider Removal

### Goal

Remove the old `Interface`-based flat providers and their registry once the
expression system fully covers all use cases.

### Design

#### 4.1 Coverage Audit

Before removal, verify every legacy metric has an expression equivalent:

| Legacy metric | Expression equivalent |
|--------------|----------------------|
| `file-bytes` | `file-bytes` (bare, unchanged) |
| `total-type-count` | `types.count` |
| `public-type-count` | `public.types.count` |
| `max-cyclomatic-complexity` | `max.cyclomatic-complexity` |
| ... | ... (full audit at implementation time) |

#### 4.2 Removal Steps

1. Remove all `provider_defs.go` / provider definition files (flat metric definitions)
2. Remove `internal/provider/registry.go` (the `Interface`-based registry)
3. Remove `provider.Get`, `provider.All`, `provider.Register` (legacy API)
4. Remove `provider.Run`'s legacy metric resolution path
5. Remove `MetricSpec.Validate`'s legacy fallback
6. Update `provider.Run` to work exclusively with expressions
7. Remove `metric.Target` type (replaced by `MetricLevel`)

#### 4.3 Error Messages for Old Syntax

Since this is pre-1.0, old config files will break. Provide actionable errors:

```
unknown metric "max-cyclomatic-complexity"
  hint: did you mean "max.cyclomatic-complexity"?
  (metric expressions use dots to separate filter, metric, and aggregation)
```

Implementation: when a metric name fails expression parsing AND legacy lookup,
attempt fuzzy matching against known base metrics + aggregation combinations.

---

## Cross-Cutting Concerns

### Empty Collections

If aggregation collects zero values (empty directory, or filter excludes
everything), the metric is **not set** on the directory's `MetricContainer`.
This is distinct from zero — it means "no data available".

The rendering pipeline already handles missing metrics (leaves cells uncolored).

### Result Kind

Aggregation result kind follows the rules from the original design spec:

| Aggregation | Input Kind | Result Kind |
|-------------|-----------|-------------|
| sum, min, max, range | Quantity | Quantity |
| sum, min, max, range | Measure | Measure |
| mean | Quantity | Measure |
| mean | Measure | Measure |
| count, distinct | any | Quantity |
| mode | Classification | Classification |

### Result Naming

Aggregated values are stored under `expression.ResultName()`:
- `sum.file-bytes` → stored as `"sum.file-bytes"` on the directory
- `public.types.count` → stored as `"public.types.count"` on the directory
- Bare `file-bytes` at file level → stored as `"file-bytes"` (unchanged)

### Performance

Recursive collection is O(n) where n = total source nodes in subtree. For large
repositories this could be significant. Mitigations:

- Compute only requested aggregations (not all possible ones)
- Each directory's aggregation is independent → parallelizable
- Cache collected values if the same base metric is aggregated multiple ways
  (e.g., both `min.file-bytes` and `max.file-bytes` collect the same values)

### Dependencies Between Metrics

Some base metrics depend on others (declared in `BaseMetricDescriptor.Dependencies`).
The aggregation stage doesn't need to handle this — dependencies are resolved by
`provider.Run` which ensures base providers execute in dependency order.

---

## Testing Strategy

### Phase 1
- Unit tests for `ComputeAggregations` with hand-built model trees
- Integration test: config with `sum.file-bytes` → verify directory gets correct value
- Golden file test: treemap rendered with aggregated metric
- Error test: declaration-level metric in Phase 1 → clear error message

### Phase 2
- Unit tests for declaration model construction
- Unit tests for filter predicate evaluation
- Integration test: Go source tree → `mean.cyclomatic-complexity` at directory level
- Verify legacy Go metrics still work unchanged

### Phase 3
- Unit tests for commit model construction
- Integration test: git repo → `max.lines-changed` at directory level
- Verify legacy git metrics still work unchanged

### Phase 4
- Verify all previously-working configs work with expression equivalents
- Verify error messages for old syntax are helpful
- Verify no dead code remains

---

## Success Criteria

- **Phase 1:** `sum.file-bytes` in a treemap config produces correct directory coloring
- **Phase 2:** `mean.cyclomatic-complexity` correctly averages across all functions in a directory tree
- **Phase 3:** `max.lines-changed` correctly finds the largest single-commit change across a directory tree
- **Phase 4:** Legacy `provider_defs.go` files deleted, all tests pass with expression syntax only
