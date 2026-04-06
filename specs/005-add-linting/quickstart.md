# Quickstart: Configure Linting (005-add-linting)

**Date**: 2026-04-06

## Prerequisites

- Go 1.26.1+
- Task runner installed
- Devcontainer (recommended) or local tooling setup

## Setup

### Devcontainer (recommended)

Rebuild the devcontainer — `install-dependencies.sh` will automatically:
1. Install golangci-lint v2.8.0
2. Build `golangci-lint-custom` with the nilaway plugin
3. Place both binaries in `/usr/local/bin`

### Local

```bash
.devcontainer/install-dependencies.sh -v
```

This installs tools to `hack/tools/` (ensure this directory is on your `$PATH` or use `task` which references the binary directly).

## Usage

### Run linting

```bash
task lint
```

Runs the full expanded linter set including nilaway against all Go source files.

### Auto-fix lint issues

```bash
task tidy
```

Formats code, tidies modules, and auto-fixes lint issues where possible.

### CI pipeline

```bash
task ci
```

Builds, tests, and lints — same as before, now using expanded linter set.

## Verification

After setup, verify the custom binary includes nilaway:

```bash
golangci-lint-custom linters | grep nilaway
```

Expected output should show nilaway in the linter list.
