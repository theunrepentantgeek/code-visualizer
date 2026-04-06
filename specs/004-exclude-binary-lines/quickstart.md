# Quickstart: Exclude Binary Files for Line-Count Metric

**Feature**: 004-exclude-binary-lines  
**Date**: 2026-04-06

## Overview

This feature automatically excludes binary files from treemap visualizations when using line count (`file-lines`) as the size metric. No new CLI flags are needed — the behaviour is driven by the existing `--size` flag.

## Implementation Summary

### 1. Add `FilterBinaryFiles` to `internal/scan/scanner.go`

```go
// FilterBinaryFiles returns a copy of the directory tree with binary files removed.
// Directories that become empty after removal are also pruned.
func FilterBinaryFiles(node DirectoryNode) DirectoryNode { ... }
```

- Recursively walks the tree
- Excludes `FileNode` entries where `IsBinary == true`
- Prunes `DirectoryNode` entries with no remaining files or subdirectories
- Logs each excluded file at `slog.Debug` level
- Returns a new tree (does not mutate the input)

### 2. Add `countFiles` check after filtering

Use the existing `countFiles` helper to verify the filtered tree has at least one file. If zero, return a `noFilesAfterFilterError`.

### 3. Add `noFilesAfterFilterError` to `cmd/codeviz/main.go`

```go
type noFilesAfterFilterError struct{ msg string }
```

Maps to exit code 6 in `classifyError()`.

### 4. Call the filter in the pipeline

In `main.go`, after `PopulateLineCounts(&root)` and before `treemap.Layout()`:

```go
if c.Size == metric.FileLines {
    root = scan.FilterBinaryFiles(root)
    if countFiles(root) == 0 {
        return &noFilesAfterFilterError{...}
    }
}
```

### 5. Update `docs/usage.md`

Add exit code 6 to the exit code table.

## Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| Filter after `PopulateLineCounts` | All binary detection complete at this point |
| Return new tree (don't mutate) | Enables before/after logging; avoids side effects |
| No new CLI flags | Automatic behaviour based on `--size` value |
| Exit code 6 (not reusing 2 or 5) | Semantically distinct from path errors and internal errors |
| Only filter for `file-lines` | All other metrics are meaningful for binary files |

## Files to Modify

| File | Change |
|------|--------|
| `internal/scan/scanner.go` | Add `FilterBinaryFiles()` function |
| `internal/scan/scanner_test.go` | Add tests for filtering and pruning |
| `cmd/codeviz/main.go` | Add `noFilesAfterFilterError`, call filter, update `classifyError` |
| `docs/usage.md` | Add exit code 6 |

## Test Strategy

- Unit tests in `internal/scan/scanner_test.go` for `FilterBinaryFiles`:
  - Mixed binary + text files → only text files remain
  - All binary files → empty tree returned
  - Nested directory with only binary files → directory pruned
  - No binary files → tree unchanged
  - Already empty directory → remains empty
- Integration-level verification via existing golden-file snapshot tests (update expected outputs if needed)
