# Metric Expressions

**Date:** 2026-06-14
**Status:** Approved

## Problem

The metric system suffers from combinatorial explosion. The Go provider manually declares 34 flat metrics to cover every combination of base concept × visibility filter × aggregation function. This approach does not scale: adding new model levels (declarations, commits) would multiply the number of required registrations further.

We need a structured system that:

1. Lets providers declare base metrics with their valid filters and aggregations.
2. Gives users a composable expression syntax to request any valid combination.
3. Computes aggregated values on demand — only what the user actually requests.
4. Keeps the help listing terse by showing base metrics with their options, not every permutation.

## Syntax

### Grammar

```
metric-expr  = [filter "."] base-name ["." aggregation]
filter       = kebab-word
base-name    = kebab-word
aggregation  = kebab-word
kebab-word   = [a-z][a-z0-9-]*
```

The `.` character separates structural roles. Hyphens `-` separate words within a component.

### Constraints

- Shell-safe: no characters requiring quoting (`(){}[]|*?<>!$&;`)
- YAML-safe: no `:`, `#`, or leading `*`/`&`/`!`
- Unambiguous: aggregation verbs are a small, fixed set checked first; filters validated against per-metric declarations

### Examples

| Expression                    | Filter | Base Metric           | Aggregation |
| ----------------------------- | ------ | --------------------- | ----------- |
| `file-size`                   | —      | file-size             | —           |
| `file-size.sum`               | —      | file-size             | sum         |
| `public.types.count`          | public | types                 | count       |
| `cyclomatic-complexity.max`   | —      | cyclomatic-complexity | max         |
| `public.function-length.mean` | public | function-length       | mean        |
| `file-type.mode`              | —      | file-type             | mode        |

### Parsing Algorithm

1. Split input on `.` into segments.
2. If the last segment matches a known aggregation verb → it is the aggregation.
3. The remaining segments are candidate base metric + optional filter.
4. Look up the remaining segment(s) in the base metric registry. If a single segment matches → it is the base name. If two segments remain, the first is checked as a filter against the resolved metric's declared filter list.
5. Validation: confirm filter and aggregation are declared valid for the resolved base metric.

## Core Types

### MetricExpression

```go
package metric

// FilterName identifies a filter/qualifier (e.g., "public", "stdlib").
type FilterName string

// AggregationName identifies an aggregation function (e.g., "sum", "max").
type AggregationName string

// MetricExpression is the parsed form of a user-provided metric string.
type MetricExpression struct {
    Filter      FilterName
    Base        Name
    Aggregation AggregationName
}
```

### MetricLevel

```go
package metric

// MetricLevel identifies where raw data lives in the model hierarchy.
type MetricLevel int

const (
    LevelFile        MetricLevel = iota // native to files (file-size, file-lines)
    LevelDeclaration                    // native to declarations (cyclomatic-complexity)
    LevelCommit                         // native to commits (commit-date)
    LevelDirectory                      // native to directories (computed aggregates)
)
```

## Base Metric Registration

### Provider Descriptor

Each provider declares its filter vocabulary once:

```go
package provider

// ProviderDescriptor declares shared metadata for a metric provider package.
type ProviderDescriptor struct {
    Name    string
    Filters map[metric.FilterName]string // filter name → human description
}
```

### Base Metric Descriptor

Each base metric references filters from its provider's vocabulary:

```go
package provider

// BaseMetricDescriptor is the static metadata for a composable base metric.
type BaseMetricDescriptor struct {
    Name           metric.Name
    Kind           metric.Kind
    Level          metric.MetricLevel
    Description    string
    Filters        []metric.FilterName      // valid adjectives (from provider vocabulary)
    Aggregations   []metric.AggregationName // valid verbs
    Dependencies   []metric.Name
    DefaultPalette palette.PaletteName
}
```

### Example: Go Provider

Provider-level:
```go
ProviderDescriptor{
    Name: "go",
    Filters: map[metric.FilterName]string{
        "public":  "Exported declarations only",
        "private": "Unexported declarations only",
    },
}
```

Base metrics (11 replace the current 34):

| Base Metric             | Level       | Kind     | Filters                    | Aggregations        |
| ----------------------- | ----------- | -------- | -------------------------- | ------------------- |
| `types`                 | Declaration | Quantity | public, private            | count, sum          |
| `interfaces`            | Declaration | Quantity | public, private            | count, sum          |
| `structs`               | Declaration | Quantity | public, private            | count, sum          |
| `functions`             | Declaration | Quantity | public, private            | count, sum          |
| `methods`               | Declaration | Quantity | public, private            | count, sum          |
| `constants`             | Declaration | Quantity | public, private            | count, sum          |
| `variables`             | Declaration | Quantity | public, private            | count, sum          |
| `imports`               | File        | Quantity | stdlib, external, internal | sum, min, max, mean |
| `cyclomatic-complexity` | Declaration | Quantity | —                          | sum, min, max, mean |
| `function-length`       | Declaration | Quantity | —                          | sum, min, max, mean |
| `comment-ratio`         | File        | Measure  | —                          | min, max, mean      |

## Aggregation Functions

A finite set of verbs with generic implementations:

| Verb       | Applies to Kind   | Result Kind    | Semantics                         |
| ---------- | ----------------- | -------------- | --------------------------------- |
| `sum`      | Quantity          | Quantity       | Sum of all values                 |
| `min`      | Quantity, Measure | same           | Minimum value                     |
| `max`      | Quantity, Measure | same           | Maximum value                     |
| `mean`     | Quantity, Measure | Measure        | Arithmetic mean                   |
| `count`    | any               | Quantity       | Number of items (after filtering) |
| `mode`     | Classification    | Classification | Most common category              |
| `distinct` | Classification    | Quantity       | Number of distinct categories     |
| `range`    | Quantity, Measure | same           | max − min                         |

Each aggregation function is a single generic implementation (~10-20 lines). Adding a new verb works for all metrics automatically.

### Valid Aggregation Rule

Each base metric declares exactly which aggregation verbs it supports. Not all verbs make sense for all metrics (e.g., `sum` is meaningless for `commit-density`). Kind provides a rough guide but the per-metric list is authoritative.

## Resolution & Computation Pipeline

### Resolution Phase

After CLI/config parsing, before provider execution:

1. **Parse** each metric string into a `MetricExpression`.
2. **Resolve** the base metric name against the registry → get `BaseMetricDescriptor`.
3. **Validate:**
   - If aggregation specified: is it in the metric's declared `Aggregations` list?
   - If filter specified: is it in the metric's declared `Filters` list?
   - If no aggregation: is the metric's native level the same as the request context? If not → error: aggregation required.
4. **Plan** → produce a `ResolvedMetric`.

```go
package provider

// ResolvedMetric is a fully validated metric ready for computation.
type ResolvedMetric struct {
    Expression  metric.MetricExpression
    Descriptor  BaseMetricDescriptor
    SourceLevel metric.MetricLevel
    TargetLevel metric.MetricLevel
    ResultKind  metric.Kind  // may differ from source (e.g., mean of Quantity → Measure)
    ResultName  metric.Name  // full expression as stored key in MetricContainer
}
```

### Computation Phase

After source providers have populated raw data:

1. For each `ResolvedMetric` requiring aggregation:
   - Walk the tree at the target level.
   - For each target node, collect all source-level nodes within it.
   - Apply filter (discard nodes not matching).
   - Extract raw values from surviving nodes.
   - Apply aggregation function.
   - Store result in target node's `MetricContainer` under `ResultName`.
2. Metrics with no aggregation (bare names at native level): source providers already stored them — no extra work.

### Critical Constraint: No Intermediate Re-aggregation

Aggregation always operates on raw source-level data, never on intermediate roll-ups. For example, `mean.cyclomatic-complexity` at directory level computes the mean across ALL declarations in all files in that directory — NOT the mean of per-file means. This avoids the statistical error of averaging averages.

## Help & Discovery UX

### `codeviz help metrics` Output Format

```
Syntax: [filter.]metric[.aggregation]
Examples: file-size.sum, public.types.count, cyclomatic-complexity.max

Filesystem Metrics
──────────────────
  file-size         Quantity     Size of each file in bytes.
                    Aggregations: sum, min, max, mean

  file-lines        Quantity     Number of lines in each text file.
                    Aggregations: sum, min, max, mean

  file-type         Classification  File extension category (e.g. go, md, png).
                    Aggregations: mode, distinct

Go Metrics
──────────
  Filters: public (exported only), private (unexported only)

  types             Quantity     Count of type declarations.
                    Aggregations: count, sum
                    Filters: public, private

  cyclomatic-complexity  Quantity  Cyclomatic complexity per function.
                    Aggregations: sum, min, max, mean

  function-length   Quantity     Function length in lines.
                    Aggregations: sum, min, max, mean
```

Provider-level filters are shown once at the group header. Per-metric filters listed only when a subset applies.

### Error Messages

Actionable, hint-rich:

```
Error: metric "file-size.mode" is invalid
  "mode" is not a valid aggregation for "file-size" (Quantity).
  Valid aggregations: sum, min, max, mean

Error: metric "cyclomatic-complexity" requires aggregation at directory level
  "cyclomatic-complexity" is native to declarations.
  Try: cyclomatic-complexity.sum, cyclomatic-complexity.max, cyclomatic-complexity.mean

Error: metric "stdlib.file-size.sum" is invalid
  "stdlib" is not a valid filter for "file-size".
  "file-size" has no filters.
```

## Implementation Impact

### New Files

- `internal/metric/expression.go` — `MetricExpression`, `FilterName`, `AggregationName`, parsing
- `internal/metric/level.go` — `MetricLevel` type and constants
- `internal/metric/aggregation.go` — generic aggregation function implementations
- `internal/provider/base_descriptor.go` — `BaseMetricDescriptor`, `ProviderDescriptor`
- `internal/provider/resolution.go` — `ResolvedMetric`, resolution and validation logic
- `internal/stages/aggregate.go` — pipeline stage computing aggregated metrics

### Modified Packages

- `internal/provider/registry.go` — stores `BaseMetricDescriptor` alongside (or replacing) the current `Interface` registrations
- `internal/config/metric_spec.go` — validation uses expression parser + resolution
- `cmd/codeviz/help_metrics_cmd.go` — rewritten for grouped display
- `internal/provider/golang/` — simplified from 34 flat providers to 11 base metric declarations

### Renderer Compatibility

Renderers read from `MetricContainer` by `metric.Name`. The `ResultName` stored is the full expression string (e.g., `"public.types.count"`), which is a valid `metric.Name`. Renderers do not need changes — they receive the resolved metric name through config and read it from the container.

### Model Changes (Future, Not This Work)

This design anticipates `File` gaining:
- A slice of `Declaration` structs (visibility, kind, associated metrics)
- A slice of `Commit` structs (date, author, associated metrics)

These are not required for the initial implementation. The Go provider can continue populating file-level metrics directly. The aggregation framework is ready to consume them once they exist.

## Testing Strategy

### Unit Tests

- **Expression parsing:** bare names, with aggregation, full expressions, edge cases, ambiguity guards
- **Resolution/validation:** valid combinations, invalid aggregation/filter errors, missing aggregation at wrong level
- **Aggregation functions:** known input → known output, empty slices, single elements, edge cases

### Integration Tests (Golden Files)

- End-to-end CLI with expression syntax → rendered output matches golden files
- `help metrics` output → golden file
- Error messages → golden files

### Existing Test Patterns

- Goldie v2 for snapshots, Gomega for assertions
- Provider tests shift from testing individual flat providers to testing base metrics + aggregation combinations
