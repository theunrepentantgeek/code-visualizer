# Implementation Plan: Use Goldie for Golden File Testing

**Branch**: `003-use-goldie` | **Date**: 2026-04-05 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/003-use-goldie/spec.md`

## Summary

Replace handwritten golden file infrastructure in `internal/render/renderer_test.go` with the Goldie library (`github.com/sebdah/goldie/v2`). The current code manually reads/writes PNG files, uses a custom `UPDATE_GOLDEN` env var, and performs pixel-by-pixel image comparison. Goldie provides all of this out of the box: byte-level comparison, automatic update via `-update` flag or `GOLDIE_UPDATE` env var, and configurable fixture directories. The Taskfile `update-golden-files` task will be updated to use Goldie's mechanism.

## Technical Context

**Language/Version**: Go 1.26+  
**Primary Dependencies**: Goldie v2 (`github.com/sebdah/goldie/v2`), Gomega (assertions), fogleman/gg (PNG rendering)  
**Storage**: N/A  
**Testing**: Gomega (assertions) + Goldie (golden-file snapshots), standard `go test`  
**Target Platform**: Linux (devcontainer), macOS, Windows  
**Project Type**: CLI tool  
**Performance Goals**: N/A (test infrastructure change only)  
**Constraints**: N/A  
**Scale/Scope**: 4 golden file tests in `internal/render/`, 6 golden PNG files in `testdata/`

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Test-First | PASS | Constitution already mandates Goldie for golden-file snapshots — this feature brings the codebase into compliance |
| II. API-First Design | PASS | No public API changes; internal test infrastructure only |
| III. Type Safety | PASS | No new types introduced |
| IV. Simplicity / YAGNI | PASS | Removing custom infrastructure in favor of proven library reduces complexity. Goldie is justified as it replaces ~40 lines of handwritten golden file code |
| V. Performance | PASS | No performance impact; test-only change |
| VI. Accessibility | N/A | No UI changes |
| VII. Observability | N/A | No logging changes |
| VIII. Documentation | PASS | Taskfile task and DEVELOPMENT.md will be updated to reflect new update mechanism |

All gates pass. No violations to justify.

## Project Structure

### Documentation (this feature)

```text
specs/003-use-goldie/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
internal/
└── render/
    ├── renderer_test.go     # Migrated golden file tests (Goldie replaces handwritten code)
    └── testdata/             # Golden files (PNG snapshots, relocated to Goldie conventions)
        ├── categorization-palette.png
        ├── flat-treemap.png
        ├── goodbad-palette.png
        ├── nested-treemap.png
        ├── neutral-palette.png
        └── temperature-palette.png
go.mod                       # Goldie v2 dependency added
go.sum                       # Updated
Taskfile.yml                 # update-golden-files task updated
```

**Structure Decision**: No structural changes. Golden files remain in `internal/render/testdata/`. Goldie's `WithFixtureDir("testdata")` option preserves the existing directory layout. Goldie's default `.golden` suffix will be overridden with `.png` via `WithNameSuffix(".png")` to match existing file naming.
