# Implementation Plan: CLI Treemap Visualization

**Branch**: `001-cli-treemap-viz` | **Date**: 2026-04-04 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-cli-treemap-viz/spec.md`

## Summary

Build a CLI tool in Go that scans a directory (optionally a git repo), computes per-file metrics (size, line count, file type, and git-derived age/freshness/author count), and renders a squarified treemap visualization to a PNG image. Rectangle size is driven by a user-chosen numeric metric, fill and border colours by independently selected metrics mapped through discrete colour palettes (Categorization, Temperature, Good/Bad, Neutral) with quantile-based bucketing. Directory hierarchy is reflected through nested groupings with labelled header bars.

## Technical Context

**Language/Version**: Go (latest stable, currently 1.22+)
**Primary Dependencies**: Kong (CLI parsing), image/png + image/draw (rendering), os/exec for git commands
**Storage**: N/A — stateless CLI tool, reads filesystem and git history, writes PNG
**Testing**: Gomega (assertions) + Goldie (golden-file snapshots), standard `go test`
**Target Platform**: Linux, macOS, Windows (cross-platform CLI)
**Project Type**: CLI tool
**Performance Goals**: <5s for 1,000 files (no git), <15s for 1,000 files (with git), <500ms cold start
**Constraints**: <512 MB RSS for repos under 10,000 files; default output 1920×1080
**Scale/Scope**: Repositories up to 10,000+ files; single-user CLI invocation

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle            | Status | Evidence                                                                                        |****
| -------------------- | ------ | ----------------------------------------------------------------------------------------------- |
| I. Test-First        | ✅ PASS | Plan includes test phases before implementation in task ordering                                |
| II. API-First        | ✅ PASS | CLI contract and internal interfaces defined in contracts/ before implementation                |
| III. Type Safety     | ✅ PASS | Custom types for MetricName, PaletteName, FileNode, etc. specified in data model                |
| IV. Simplicity/YAGNI | ✅ PASS | MVP scope bounded; future features explicitly deferred (custom palettes, image config, filters) |
| V. Performance       | ✅ PASS | Performance budgets defined in spec (SC-001, SC-002); benchmark tests planned                   |
| VI. Accessibility    | ✅ PASS | WCAG AA contrast for palettes; labels on rectangles; JSON output for machine parsing            |
| VII. Observability   | ✅ PASS | slog structured logging; --verbose flag; contextual error messages                              |
| VIII. Documentation  | ✅ PASS | Kong auto-generates help; package docs required; usage examples in docs/                        |

**GATE RESULT: PASS** — No violations. Proceeding to Phase 0.

## Project Structure

### Documentation (this feature)

```text
specs/001-cli-treemap-viz/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
│   └── cli-contract.md  # CLI command interface specification
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
cmd/
└── codeviz/
    └── main.go              # CLI entrypoint (Kong)

internal/
├── scan/
│   ├── scanner.go           # Directory tree scanning
│   ├── scanner_test.go      # Scanner tests
│   ├── gitinfo.go           # Git metadata collection
│   ├── gitinfo_test.go      # Git metadata tests
│   └── testdata/            # Sample directory fixtures
├── metric/
│   ├── metric.go            # Metric types and computation
│   ├── metric_test.go       # Metric computation tests
│   ├── bucket.go            # Quantile bucketing with sig-fig rounding
│   ├── bucket_test.go       # Bucketing tests
│   ├── registry.go          # Metric registry and validation
│   └── registry_test.go     # Registry tests
├── palette/
│   ├── palette.go           # Palette type and colour definitions
│   ├── palette_test.go      # Palette definition tests
│   ├── mapper.go            # Metric-to-colour mapping
│   └── mapper_test.go       # Mapper tests
├── treemap/
│   ├── layout.go            # Squarified treemap layout algorithm
│   ├── layout_test.go       # Layout tests
│   ├── node.go              # Layout tree node types
│   └── node_test.go         # Node tests
└── render/
    ├── renderer.go          # PNG image rendering
    ├── renderer_test.go     # Renderer tests (golden-file snapshots)
    ├── label.go             # Text label fitting and rendering
    ├── label_test.go        # Label fitting tests
    └── testdata/            # Golden-file snapshots
```

**Structure Decision**: Single-project Go layout with `cmd/` for the CLI entrypoint and `internal/` for all library packages. This follows Go conventions and the constitution's module structure guidance. Shared `pkg/` is not needed since this feature is CLI-only; when the Fyne UI feature is added later, shared packages can be promoted to `pkg/` at that time (YAGNI).

## Complexity Tracking

> No constitution violations to justify — all gates passed.
