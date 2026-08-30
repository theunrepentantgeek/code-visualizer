# Scatter Directory Grain

**Date:** 2026-08-30
**Status:** Approved

## Summary

Add directory-only granularity to the scatter visualization by reusing the
radial visualization's `--grain file|directory` command-line flag. Persist the
same choice as `scatter.grain` in YAML and JSON configuration.

The default remains `file`, preserving current scatter output. Directory grain
plots the scan root and every descendant directory using directory-level
metrics.

## Command and Configuration Surface

`ScatterCmd` gains the same flag shape and help text as `RadialCmd`:

```go
Grain string `enum:",file,directory" default:"" help:"Granularity of nodes shown: file (default) or directory."`
```

The persistent scatter configuration gains:

```go
Grain *string `yaml:"grain,omitempty" json:"grain,omitempty"`
```

`Scatter.OverrideGrain` follows the existing override convention: an empty CLI
value leaves configuration unchanged, while `file` or `directory` overrides
the configured value.

Examples:

```shell
codeviz scatter . -o scatter.png \
  --grain directory \
  --x-axis file-lines.sum \
  --y-axis file-size.sum \
  --size file-size.sum
```

```yaml
scatter:
  grain: directory
  xAxis: file-lines.sum
  yAxis: file-size.sum
  size: file-size.sum
```

## Metric Resolution

Scatter resolves grain before resolving axes, size, fill, and border metrics.
Omitted grain resolves to `file`.

- File grain keeps the existing file-level resolution behavior.
- Directory grain resolves selected metrics at directory level.
- Directory-native metrics may be used directly.
- Metrics native to finer levels require a supported aggregation expression,
  such as `file-size.sum` or `file-type.mode`.

The requested-metric pipeline continues to load base data and compute
aggregations before scatter dataset collection.

## Dataset and Rendering

Generalize scatter's point data around a small internal metric-bearing node
abstraction implemented by both `model.File` and `model.Directory`. Each point
also retains its display name and concrete node kind so rendering can obtain
fill and border values through the existing file or directory ink helpers.

Dataset collection selects one traversal based on grain:

- File grain uses the current depth-first file traversal.
- Directory grain uses the existing directory traversal and includes the scan
  root as well as every descendant directory.

Both modes use the same missing-value checks, axis resolution, radius scaling,
sorting, labels, legend construction, and canvas rendering. Log-scale errors
identify the offending node using grain-neutral wording. Result logging records
the grain and selected node count without reporting directory points as files.

## Compatibility and Error Handling

Existing commands and configuration without `grain` remain file-based and
produce the same scatter behavior.

Kong rejects unknown grain values at CLI parsing. Configuration validation
rejects unsupported values and reports metric expressions that cannot resolve
at the selected grain. As today, nodes missing an axis or size value are
skipped and included in the corresponding skip counts.

## Testing

Add focused coverage for:

1. CLI parsing of `scatter --grain directory`.
2. Scatter config deserialization, export, and CLI-over-config precedence.
3. Default file grain and explicit directory grain resolution.
4. Directory-level metric validation and aggregation requests.
5. Directory dataset collection, including the scan root and missing-value
   skip counts.
6. File-mode regression behavior.
7. Directory metric colour lookup, layout labels, and rendering.
8. Grain-aware log-scale errors and result metadata where directly testable.

Run the existing targeted Go tests for `cmd/codeviz` and `internal/scatter`,
then the repository CI task.

## Out of Scope

- Mixed file-and-directory scatter points.
- A third granularity or automatic metric aggregation.
- Changes to radial grain behavior.
- Directory hierarchy connectors or other scatter layout changes.
