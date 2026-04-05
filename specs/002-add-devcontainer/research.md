# Research: Add Devcontainer

**Feature**: 002-add-devcontainer  
**Date**: 2026-04-05

## R1: ASO Components to Remove

### Decision: Remove all ASO-specific tooling and configuration

### Rationale

The ASO devcontainer was designed for a large Kubernetes operator project. code-visualizer is a Go CLI tool with no Azure, Kubernetes, or Docker dependencies. Every ASO-specific component adds build time, image size, and maintenance burden without any benefit.

### Inventory of ASO-specific components

#### Dockerfile — APT packages to REMOVE

| Package               | ASO Purpose                          | code-visualizer Need |
| --------------------- | ------------------------------------ | -------------------- |
| docker-ce-cli         | Docker-from-docker pattern for CI    | None                 |
| docker-compose-plugin | Docker compose for integration tests | None                 |
| azure-cli             | Azure resource management            | None                 |
| graphviz              | DOT graph generation in ASO          | None                 |
| nodejs, npm           | Hugo docs site tooling (PostCSS)     | None                 |
| python3-pip           | Python virtualenv for tooling        | None                 |
| gnuplot               | Performance graphing                 | None                 |

#### Dockerfile — APT packages to KEEP

| Package         | Reason                                                           |
| --------------- | ---------------------------------------------------------------- |
| bash-completion | Developer convenience in terminal                                |
| lsb-release     | Used during APT setup (can be dropped if Docker repo is removed) |

**Note**: `lsb-release` is only needed for the Docker APT repo setup. Since we're removing Docker CLI, the entire Docker APT repo block can be removed, and `lsb-release` is no longer needed either. Only `bash-completion` is worth keeping, and it's likely already in the base image.

#### Dockerfile — Sections to REMOVE entirely

- Docker APT repository setup (GPG key, sources list, docker-ce-cli install)
- Azure CLI installation
- `install-dependencies.sh` invocation (script will be replaced or eliminated)
- ASO multi-module `go mod download` block (v2/, v2/cmd/asoctl/, etc.)
- envtest setup (`setup-envtest use`, `KUBEBUILDER_ASSETS` env)
- kubectl completions (alias, completion, source)
- `KIND_CLUSTER_NAME=aso` env var
- Docker group setup (`groupadd docker`, `usermod -aG docker vscode`)
- `CMD ["sleep", "infinity"]`

#### Dockerfile — Sections to ADD or MODIFY

- Simple `go mod download` for the single root module (COPY go.mod go.sum, RUN go mod download)
- go-task completion setup (keep from ASO)
- gofumpt installation (keep, via `go install`)
- golangci-lint installation (keep, via curl installer)
- go-task installation (keep, via curl)

#### install-dependencies.sh — Tools to REMOVE

| Tool           | ASO Purpose                       |
| -------------- | --------------------------------- |
| conversion-gen | Kubernetes CRD code generation    |
| controller-gen | Kubernetes controller scaffolding |
| kind           | Local Kubernetes clusters         |
| kustomize      | Kubernetes manifest management    |
| hugo           | ASO documentation site            |
| htmltest       | HTML link checking for docs       |
| crddoc         | CRD documentation generator       |
| go-vcr-tidy    | VCR test recording cleanup        |
| setup-envtest  | Kubernetes envtest binary setup   |
| helm           | Kubernetes package manager        |
| yq             | YAML processing                   |
| cmctl          | cert-manager CLI                  |
| docker-buildx  | Docker multi-arch builds          |
| azwi           | Azure Workload Identity CLI       |
| postcss        | Hugo CSS processing               |
| Trivy          | Container security scanning       |

#### install-dependencies.sh — Tools to KEEP

| Tool          | Version | Reason                                     |
| ------------- | ------- | ------------------------------------------ |
| gofumpt       | latest  | Go formatting (required by Taskfile)       |
| golangci-lint | v2.8.0  | Go linting (required by Taskfile)          |
| go-task       | v3.49.1 | Build orchestration (required by Taskfile) |

#### install-dependencies.sh — Sections to REMOVE

- Azure CLI check (`command -v az`)
- Pip3 check (`command -v pip3`)
- `KUBEBUILDER_DEST` and `BUILDX_DEST` variables
- Webhook certs setup
- Python virtualenv installation
- All ASO-specific go-install calls

#### devcontainer.json — Changes needed

| Field                                                     | Current (ASO)                  | Target (code-visualizer) |
| --------------------------------------------------------- | ------------------------------ | ------------------------ |
| `name`                                                    | "Azure Service Operator"       | "Code Visualizer"        |
| Extensions: `ms-azuretools.vscode-docker`                 | Remove                         | —                        |
| Extensions: `ms-kubernetes-tools.vscode-kubernetes-tools` | Remove                         | —                        |
| Extensions: `redhat.vscode-yaml`                          | Keep (useful for YAML editing) | —                        |
| `mounts` (Docker socket)                                  | Remove                         | —                        |
| `overrideCommand`                                         | Remove (not needed)            | —                        |

#### Dockerfile.dockerignore — Changes needed

- Remove all `v2/` paths
- Add `!go.mod` and `!go.sum` at root level

### Alternatives considered

1. **Keep install-dependencies.sh as-is, just remove tools**: Rejected — the script has complex ASO-specific logic (kubebuilder dest, buildx dest, az/pip3 checks) that would be dead code.
2. **Eliminate install-dependencies.sh entirely, inline in Dockerfile**: Preferred for simplicity — the remaining tools (gofumpt, golangci-lint, go-task) can be installed directly in the Dockerfile with fewer layers. However, keeping the script preserves the dual-mode pattern (devcontainer vs local install) which may be useful later.
3. **Keep install-dependencies.sh but simplify**: Best balance — retains the useful shell structure and dual-mode capability while removing all ASO content.

**Decision**: Simplify `install-dependencies.sh` to only install gofumpt, golangci-lint, and go-task. Keep the dual-mode structure (devcontainer vs local) and the helper functions (`should-install`, `go-install`, `write-info`, etc.) as they are well-written and useful.

## R2: Go Module Cache Pre-population

### Decision: Use simple COPY + `go mod download` pattern

### Rationale

code-visualizer has a single Go module at the repository root with no replace directives. The ASO pattern of copying multiple `go.mod`/`go.sum` files from nested modules is unnecessary.

### Approach

```dockerfile
# Pre-download Go modules
COPY go.mod go.sum /tmp/mod-download/
RUN cd /tmp/mod-download && go mod download && rm -rf /tmp/mod-download
```

This leverages Docker layer caching: the module download layer only rebuilds when `go.mod` or `go.sum` change, not on every source code change.

### Alternatives considered

1. **Mount go module cache as a volume**: Rejected — module cache wouldn't persist across container rebuilds, defeating the purpose.
2. **Use go build cache instead**: Rejected — module download is the bottleneck for first-run, not compilation.
3. **Skip pre-population entirely**: Rejected — spec FR-005 requires it.

## R3: Devcontainer Base Image

### Decision: Keep `mcr.microsoft.com/devcontainers/go:2-1.26`

### Rationale

The Microsoft Go devcontainer base image provides:
- Go 1.26 pre-installed (matching `go.mod`)
- Standard devcontainer infrastructure (user `vscode`, sudo, git, etc.)
- Multi-arch support (amd64/arm64)
- Dependabot-trackable version tag

The `2-` prefix is the devcontainer image version (v2), and `1.26` is the Go version. This matches the project's `go 1.26.1` in `go.mod`.

### Alternatives considered

1. **Official Go image (`golang:1.26`)**: Rejected — lacks devcontainer infrastructure (vscode user, SSH, etc.).
2. **Build from scratch**: Rejected — unnecessary complexity for no benefit.

## R4: Devcontainer runArgs

### Decision: Keep `--cap-add=SYS_PTRACE`, `--security-opt seccomp=unconfined`, `--init`

### Rationale

- `SYS_PTRACE`: Required for Go's `dlv` debugger to attach to processes. Standard for Go devcontainers.
- `seccomp=unconfined`: Required for `dlv` debugger. Without this, ptrace-based debugging fails.
- `--init`: Ensures proper signal handling and zombie process cleanup. Good practice for any container.

These are not ASO-specific; they're standard Go devcontainer settings.

### Alternatives considered

1. **Remove all runArgs**: Rejected — would break Go debugging in the container.
2. **Add only `--init`**: Rejected — debugging is a core developer workflow.
