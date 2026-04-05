# Implementation Plan: Add Devcontainer

**Branch**: `002-add-devcontainer` | **Date**: 2026-04-05 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/002-add-devcontainer/spec.md`

## Summary

Adapt the Azure Service Operator devcontainer for code-visualizer by stripping all ASO-specific tooling (Azure CLI, Kubernetes, Docker-in-Docker, etc.) and retaining only Go 1.26, golangci-lint, gofumpt, and go-task. Simplify the Dockerfile, install-dependencies.sh, devcontainer.json, and dockerignore to match the single-module Go CLI project structure. Pre-populate the Go module cache at image build time.

## Technical Context

**Language/Version**: Docker, Bash, JSON (devcontainer config); Go 1.26 (project language)  
**Primary Dependencies**: `mcr.microsoft.com/devcontainers/go:2-1.26` base image  
**Storage**: N/A  
**Testing**: Manual verification — `task ci` (build, test, lint) inside container  
**Target Platform**: Linux container (amd64/arm64), VS Code Dev Containers  
**Project Type**: CLI tool (devcontainer is infrastructure, not application code)  
**Performance Goals**: Container builds in reasonable time; first `go build` requires no network  
**Constraints**: Minimal image — no ASO-specific tools; no Docker-in-Docker  
**Scale/Scope**: 4 configuration files to modify

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle            | Applies? | Status | Notes                                                                    |
| -------------------- | -------- | ------ | ------------------------------------------------------------------------ |
| I. Test-First        | Partial  | PASS   | No production Go code — config files verified via `task ci` in container |
| II. API-First        | No       | PASS   | No public Go interfaces introduced                                       |
| III. Type Safety     | No       | PASS   | Configuration files only                                                 |
| IV. Simplicity/YAGNI | Yes      | PASS   | Spec explicitly requires stripping to minimum; no speculative tooling    |
| V. Performance       | Partial  | PASS   | Paring down reduces build time; module pre-download speeds first-run     |
| VI. Accessibility    | No       | PASS   | N/A for infrastructure                                                   |
| VII. Observability   | No       | PASS   | N/A for infrastructure                                                   |
| VIII. Documentation  | Yes      | PASS   | quickstart.md provides usage guidance                                    |

**Post-Phase 1 re-check**: All principles remain satisfied. No gate violations.

## Project Structure

### Documentation (this feature)

```text
specs/002-add-devcontainer/
├── plan.md                              # This file
├── research.md                          # Phase 0: ASO removal inventory, decisions
├── data-model.md                        # Phase 1: Configuration artifact relationships
├── quickstart.md                        # Phase 1: How to use the devcontainer
├── contracts/
│   └── devcontainer-contract.md         # Phase 1: Environment guarantees
└── tasks.md                             # Phase 2 output (not created by plan)
```

### Source Code (repository root)

```text
.devcontainer/
├── Dockerfile                 # Container image definition (to be simplified)
├── Dockerfile.dockerignore    # Build context filter (to be updated for root module)
├── devcontainer.json          # VS Code devcontainer config (to be updated)
└── install-dependencies.sh    # Tool installer script (to be simplified)
```

**Structure Decision**: All changes are within the existing `.devcontainer/` directory. No new directories or project structure changes needed. The four files are already present (copied from ASO) and need modification, not creation.

## Complexity Tracking

No constitution violations to justify. This feature is straightforward infrastructure configuration.
