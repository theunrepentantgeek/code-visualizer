# Research: Exclude Binary Files for Line-Count Metric

**Feature**: 004-exclude-binary-lines  
**Date**: 2026-04-06

## Research Topics

### 1. Where to Insert the Filter in the Pipeline

**Decision**: Add a `FilterBinaryFiles` function in the `scan` package, called in `main.go` after `PopulateLineCounts()` and before `treemap.Layout()`.

**Rationale**: At this point in the pipeline, all binary detection is complete (both git-based and content-heuristic-based via `PopulateLineCounts`). The `DirectoryNode` tree is fully enriched and ready to be pruned. Filtering before layout avoids wasted computation and keeps the layout engine unaware of filtering concerns.

**Alternatives considered**:
- *Filter during scan*: Rejected — binary status is not yet known during the initial `scanDir()` walk.
- *Filter inside `PopulateLineCounts`*: Rejected — conflates line counting with tree pruning; violates single responsibility.
- *Filter inside `treemap.Layout`*: Rejected — layout should not need to know about binary detection; couples layout to scan semantics.
- *Filter at the metric extraction level*: Rejected — would still produce empty rectangles in the layout, violating FR-001.

### 2. Tree Pruning Strategy

**Decision**: In-place filter that returns a new `DirectoryNode` tree (not mutating the original). The filter:
1. Removes `FileNode` entries where `IsBinary == true` from each directory's `Files` slice.
2. Recursively filters child directories.
3. Removes child `DirectoryNode` entries that are empty after filtering (no files and no non-empty subdirectories).
4. Returns the filtered tree.

**Rationale**: Returning a new tree avoids side effects on the original scan data, which may be useful for logging or diagnostics. The recursive approach naturally handles arbitrarily nested empty directories.

**Alternatives considered**:
- *Mutate in place*: Simpler but prevents logging of "before vs. after" file counts. Rejected for observability reasons.
- *Mark-and-skip in layout*: Would require threading binary awareness through the layout engine. Rejected for coupling.

### 3. Exit Code 6 — No Files After Filtering

**Decision**: Add a new `noFilesAfterFilterError` type to `cmd/codeviz/main.go` with exit code 6. The error is raised when `FilterBinaryFiles` returns a tree with zero files.

**Rationale**: The existing exit codes (1–5) have clearly defined meanings. "No files available after filtering" is semantically distinct from:
- Exit 2 (target path error — the path exists and is accessible)
- Exit 5 (internal error — nothing has gone wrong internally)

A new exit code allows scripts to distinguish "no suitable files" from other errors.

**Alternatives considered**:
- *Reuse exit code 2*: The path is valid but empty after filtering — semantically different. Rejected.
- *Reuse exit code 5*: Not an internal error. Rejected.

### 4. Verbose Logging of Excluded Files

**Decision**: Use `slog.Debug` to log each excluded binary file during the filter pass. This integrates with the existing `--verbose` flag (which sets the log level to `DEBUG`).

**Rationale**: Consistent with existing observability patterns (e.g., git metadata warnings, symlink skipping). No new logging mechanism needed.

**Alternatives considered**:
- *`slog.Info` level*: Too noisy for normal operation. Rejected — binaries are common in real repos.
- *Summary-only logging*: Loses per-file visibility. Rejected — users need to verify which specific files were excluded.

### 5. Conditional Calling: When to Filter

**Decision**: Only call `FilterBinaryFiles` when the size metric is `file-lines`. For all other size metrics, skip the filter entirely.

**Rationale**: FR-002 requires binary files to be included for all non-line-count metrics. Conditionally calling the filter avoids unnecessary work and keeps the behaviour explicit in the pipeline.

**Alternatives considered**:
- *Always filter, with metric-aware logic inside the filter*: Adds unnecessary complexity. Rejected — YAGNI.

### 6. Binary Detection in Non-Git Directories

**Decision**: No changes needed. In non-git directories, `PopulateLineCounts` already detects binaries via the 64KB line-length heuristic and sets `IsBinary = true`. The filter function works solely on the `IsBinary` flag, regardless of how it was set.

**Rationale**: The existing detection is sufficient (FR-006). Decoupling detection from filtering means the filter function is simple and testable.

**Alternatives considered**:
- *Add explicit null-byte scanning for non-git dirs*: Would improve detection accuracy but is out of scope (YAGNI). Can be added later if needed.
