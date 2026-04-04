# Devcontainer Contract

**Feature**: 002-add-devcontainer  
**Date**: 2026-04-05

## Overview

The devcontainer exposes a development environment contract: given a repository clone, opening in the devcontainer provides a fully configured workspace. This contract defines what the environment guarantees.

## Environment Contract

### Tools Available on PATH

| Tool            | Minimum Version   | Purpose                        |
| --------------- | ----------------- | ------------------------------ |
| `go`            | 1.26.x            | Go compiler and tools          |
| `gofumpt`       | latest            | Strict Go formatting           |
| `golangci-lint` | v2.8.0            | Go linting                     |
| `task`          | v3.49.1           | Build orchestration (Taskfile) |
| `git`           | (from base image) | Version control                |

### Pre-conditions Met at Container Start

- Go module cache is populated with all dependencies from `go.mod`/`go.sum`
- Go workspace is at `/workspace` (default devcontainer mount)
- User is `vscode` with sudo access (from base image)
- Git safe.directory is configured for `/workspace`

### VS Code Extensions Installed

| Extension ID                        | Purpose                                                |
| ----------------------------------- | ------------------------------------------------------ |
| `ms-vscode.go`                      | Go language support (IntelliSense, debugging, testing) |
| `task.vscode-task`                  | Taskfile integration                                   |
| `redhat.vscode-yaml`                | YAML language support                                  |
| `github.vscode-pull-request-github` | GitHub PR integration                                  |

### VS Code Settings Applied

| Setting                                                | Value             | Purpose                       |
| ------------------------------------------------------ | ----------------- | ----------------------------- |
| `editor.formatOnSave`                                  | `true`            | Auto-format on save           |
| `go.useLanguageServer`                                 | `true`            | Enable gopls                  |
| `go.lintTool`                                          | `"golangci-lint"` | Use golangci-lint for linting |
| `go.gopath`                                            | `"/go"`           | Standard Go path              |
| `[go].editor.codeActionsOnSave.source.organizeImports` | `"explicit"`      | Auto-organize imports         |

### Taskfile Commands That Must Work

All existing Taskfile commands must work identically inside the devcontainer:

| Command      | Expected Behavior                     |
| ------------ | ------------------------------------- |
| `task build` | Compiles `bin/codeviz` without errors |
| `task test`  | All tests pass                        |
| `task lint`  | Zero lint warnings                    |
| `task fmt`   | Formats all Go files                  |
| `task ci`    | Build + test + lint all pass          |

### What Is NOT Provided

The devcontainer explicitly does NOT include:
- Docker CLI or Docker-in-Docker
- Azure CLI or any Azure tooling
- Kubernetes tools (kubectl, kind, kustomize, helm, envtest)
- Node.js, npm, or Python
- Any ASO-specific tooling
