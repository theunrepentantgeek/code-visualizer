# Data Model: Add Devcontainer

**Feature**: 002-add-devcontainer  
**Date**: 2026-04-05

## Overview

This feature involves only configuration files (Dockerfile, devcontainer.json, shell script, dockerignore). There are no application data entities, database schemas, or domain model changes.

## Configuration Artifacts

### Dockerfile

Defines the container image build. Key layers:

1. **Base image**: `mcr.microsoft.com/devcontainers/go:2-1.26` — provides Go, git, devcontainer infrastructure
2. **Tool installation**: `install-dependencies.sh devcontainer` — installs gofumpt, golangci-lint, go-task
3. **Module cache**: `COPY go.mod go.sum` + `go mod download` — pre-populates Go module cache
4. **Shell config**: go-task bash completions

### devcontainer.json

Defines the VS Code devcontainer experience:

- **Build context**: Points to Dockerfile with repo root as context
- **Extensions**: Go, Task, YAML, GitHub PR extensions
- **Settings**: Format on save, golangci-lint, organize imports
- **Runtime**: Debug-friendly `runArgs`, `vscode` user

### install-dependencies.sh

Dual-mode script (devcontainer vs local install) that installs:

- gofumpt (via `go install`)
- golangci-lint (via curl installer script)
- go-task (via curl tarball)

### Dockerfile.dockerignore

Restricts Docker build context to only necessary files:

- `.devcontainer/install-dependencies.sh`
- `go.mod`, `go.sum` (root level)

## Relationships

```
devcontainer.json
  └── references → Dockerfile (build.dockerfile)
        ├── invokes → install-dependencies.sh
        └── copies → go.mod, go.sum (for module pre-download)

Dockerfile.dockerignore
  └── filters → Docker build context (allow-list pattern)
```

## State / Lifecycle

No runtime state. The devcontainer is either:
- **Not built**: Container image doesn't exist yet
- **Building**: Docker is executing Dockerfile layers
- **Running**: Container is active, all tools available
- **Stale**: `go.mod`/`go.sum` changed since last build; needs rebuild to refresh module cache
