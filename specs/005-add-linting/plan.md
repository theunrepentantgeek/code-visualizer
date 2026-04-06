# Implementation Plan: Configure Linting

**Branch**: `005-add-linting` | **Date**: 2026-04-06 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/005-add-linting/spec.md`

## Summary

Configure comprehensive linting for the project by building a custom golangci-lint binary with the nilaway plugin, importing the linter configuration from the go-vcr-tidy reference project, updating build tasks to use the custom binary, and fixing all existing code to pass the expanded linter set.

## Technical Context

**Language/Version**: Go 1.26.1  
**Primary Dependencies**: golangci-lint v2.8.0, nilaway (Uber nil-safety analyzer), envsubst (gettext-base)  
**Storage**: N/A  
**Testing**: Gomega (assertions) + Goldie v2 (golden-file snapshots), standard `go test`  
**Target Platform**: Linux (devcontainer), macOS (local development)  
**Project Type**: CLI tool  
**Performance Goals**: N/A (build tooling, not runtime)  
**Constraints**: Custom linter binary must be buildable in both devcontainer and local modes  
**Scale/Scope**: ~15 Go source files across 6 packages

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Test-First | PASS | Linting configuration is infrastructure — no production code added. Existing tests must continue to pass. |
| II. API-First Design | PASS | No new APIs introduced. |
| III. Type Safety | PASS | No new types introduced. |
| IV. Simplicity / YAGNI | PASS | Importing proven config from reference project; no custom abstraction. |
| V. Performance | PASS | No runtime performance impact. |
| VI. Accessibility | PASS | No UI changes. |
| VII. Observability | PASS | Lint output provides structured feedback via `--verbose` flag. |
| VIII. Documentation | PASS | Taskfile tasks are self-documenting. |
| Formatting/Linting constraint | PASS | This feature directly enhances the linting mandate. |

**Gate result**: PASS — no violations.

## Project Structure

### Documentation (this feature)

```text
specs/005-add-linting/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
└── tasks.md             # Phase 2 output (NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
.devcontainer/
├── install-dependencies.sh     # MODIFY: add custom linter build step
├── .custom-gcl.template.yml    # NEW: template for custom linter build config
├── Dockerfile                  # REVIEW: ensure envsubst/gettext-base available
└── Dockerfile.dockerignore
.golangci.yml                   # MODIFY: expand with go-vcr-tidy configuration
Taskfile.yml                    # MODIFY: lint/tidy tasks use golangci-lint-custom
cmd/                            # REVIEW: fix any lint issues
internal/                       # REVIEW: fix any lint issues
```

**Structure Decision**: No new directories or packages. Changes are limited to build/tooling configuration files and fixes to existing Go source files.
