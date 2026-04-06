# CLI Contract: Lint Tasks (005-add-linting)

**Date**: 2026-04-06

## Task: `lint`

**Command**: `task lint`  
**Underlying**: `golangci-lint-custom run --verbose`  
**Exit code 0**: All linters pass with no issues  
**Exit code 1**: One or more lint issues found  
**stdout**: Lint issue details (file, line, linter name, message)

## Task: `tidy`

**Command**: `task tidy`  
**Steps** (sequential):
1. `gofumpt -w .` — format all Go source  
2. `go mod tidy` — clean module dependencies  
3. `golangci-lint-custom run --fix --verbose` — auto-fix lint issues where possible  

**Exit code 0**: All steps succeed  
**Exit code non-zero**: One or more steps failed

## Task: `ci`

**Command**: `task ci`  
**Steps**: build → test → lint  
**No change to interface** — still invokes `lint` task, which now uses the custom binary.

## Binary: `golangci-lint-custom`

**Location**: `hack/tools/golangci-lint-custom` (local) or `/usr/local/bin/golangci-lint-custom` (devcontainer)  
**Interface**: Drop-in replacement for `golangci-lint` with identical CLI flags  
**Additional capability**: nilaway linter plugin embedded
