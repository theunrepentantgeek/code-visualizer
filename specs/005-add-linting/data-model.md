# Data Model: Configure Linting (005-add-linting)

**Date**: 2026-04-06

## Overview

This feature is build/tooling infrastructure — it introduces no new runtime entities, data structures, or state. The "data model" consists of configuration file formats.

## Configuration Entities

### Custom Linter Build Template (`.custom-gcl.template.yml`)

A YAML file consumed by `golangci-lint custom` to build the custom binary.

| Field | Type | Description |
|-------|------|-------------|
| version | string | golangci-lint version to target (e.g., `v2.8.0`) |
| name | string | Output binary name (`golangci-lint-custom`) |
| destination | string | Output directory (uses `$TOOL_DEST` via envsubst) |
| plugins | list | Plugin definitions to embed |
| plugins[].module | string | Go module path of the plugin |
| plugins[].import | string | Go import path for the plugin entrypoint |
| plugins[].version | string | Version constraint (`latest` or pinned) |

### Linter Configuration (`.golangci.yml`)

A YAML file controlling which linters run, their settings, and exclusion rules. Key sections:

| Section | Purpose |
|---------|---------|
| `linters.enable` | List of enabled linters (~60+ from reference config) |
| `linters.disable` | Explicitly disabled linters with justification |
| `linters.settings` | Per-linter configuration (thresholds, rules, custom types) |
| `linters.settings.custom.nilaway` | nilaway plugin definition (type: module) |
| `linters.exclusions` | Global exclusion rules (generated code, test files, paths) |
| `formatters.enable` | Formatter linters (gci, gofmt) |
| `formatters.settings` | Formatter configuration (import ordering) |
| `issues` | Issue reporting limits |

## Relationships

```text
.custom-gcl.template.yml  ──envsubst──▶  .custom-gcl.yml (temp)  ──golangci-lint custom──▶  golangci-lint-custom (binary)
                                                                                                       │
.golangci.yml  ────────────────────────────────────────────────────────────────────────────────reads─────┘
```

## Validation Rules

- The `version` in `.custom-gcl.template.yml` MUST match the installed golangci-lint version.
- The `nilaway` plugin module/import paths MUST be valid Go module paths.
- All linters listed in `.golangci.yml` MUST be recognized by the golangci-lint version in use.
- The `include-pkgs` setting for nilaway MUST reference the correct module path for this project.
