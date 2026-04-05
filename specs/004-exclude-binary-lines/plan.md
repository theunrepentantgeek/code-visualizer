# Implementation Plan: Exclude Binary Files for Line-Count Metric

**Branch**: `004-exclude-binary-lines` | **Date**: 2026-04-06 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/004-exclude-binary-lines/spec.md`

## Summary

When the size metric is `file-lines`, binary files must be completely excluded from the treemap visualization. The scanner already detects binary files (via git metadata and content heuristics) and sets `IsBinary = true` on `FileNode`. This feature adds a filtering pass after line-count population — removing binary `FileNode` entries and pruning empty `DirectoryNode` containers — before the tree is handed to the layout engine. A new exit code 6 is added for the "no files after filtering" scenario. All other size metrics (`file-size`, `file-age`, `file-freshness`, `author-count`) continue to include binary files.

## Technical Context

**Language/Version**: Go 1.26.1  
**Primary Dependencies**: Kong (CLI), go-git (git metadata), fogleman/gg (PNG rendering), eris (error wrapping), Gomega (test assertions), Goldie v2 (golden-file snapshots)  
**Storage**: N/A (stateless CLI)  
**Testing**: `go test` with Gomega + Goldie; co-located test files; `testdata/` directories for fixtures  
**Target Platform**: Linux, macOS (CLI)  
**Project Type**: CLI tool  
**Performance Goals**: <5s for repos <1,000 files; <30s for repos <10,000 files  
**Constraints**: <512 MB RSS for repos <10,000 files  
**Scale/Scope**: Single Go module, 6 internal packages

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Test-First | PASS | Tests written before filtering implementation; Gomega + Goldie used |
| II. API-First Design | PASS | New filter function has a clear typed signature; exit code 6 defined before implementation |
| III. Type Safety | PASS | Uses existing `FileNode.IsBinary` bool and `MetricName` type; no `any`/`interface{}` |
| IV. Simplicity / YAGNI | PASS | Single filter function on existing tree; no new abstractions; no new deps |
| V. Performance | PASS | Single tree traversal O(n); no impact on rendering budget |
| VI. Accessibility | N/A | No visual or interaction changes |
| VII. Observability | PASS | Verbose logging of excluded files (FR-005) via existing `--verbose` flag |
| VIII. Documentation | PASS | Exit code 6 added to docs/usage.md; no new CLI flags to document |

**Result**: All gates PASS. Proceeding to Phase 0.

## Project Structure

### Documentation (this feature)

```text
specs/004-exclude-binary-lines/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
│   └── cli-contract.md
└── tasks.md             # Phase 2 output (created by /speckit.tasks)
```

### Source Code (repository root)

```text
cmd/
  codeviz/
    main.go              # Pipeline orchestration, exit codes, new filter call site
internal/
  scan/
    scanner.go           # New FilterBinaryFiles() function + tree pruning
    scanner_test.go      # Tests for filtering/pruning
    testdata/             # Existing + new test fixtures
  metric/
    metric.go            # MetricName constants (existing FileLines)
docs/
  usage.md               # Updated exit code table
```

**Structure Decision**: No new packages or directories required. The filter function is added to the `scan` package (which owns the `DirectoryNode`/`FileNode` types) and called from `main.go` in the existing pipeline.

## Complexity Tracking

> No constitution violations. Table omitted.
