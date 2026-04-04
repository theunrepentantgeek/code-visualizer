# Feature Specification: Add Devcontainer

**Feature Branch**: `002-add-devcontainer`  
**Created**: 2026-04-05  
**Status**: Draft  
**Input**: User description: "Add a devcontainer as described in GH issue #5."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Open Project in Devcontainer (Priority: P1)

A contributor clones the repository and opens it in VS Code. VS Code detects the devcontainer configuration and prompts them to reopen in the container. After the container builds, the contributor has a fully working development environment with all required tools preinstalled and the Go module cache populated, ready to build, test, and lint without any manual setup.

**Why this priority**: This is the core value of the feature — eliminating manual setup and ensuring every contributor has an identical, working environment from the start.

**Independent Test**: Can be fully tested by opening the repository in a devcontainer and running `task ci` (build, test, lint) successfully without any additional setup steps.

**Acceptance Scenarios**:

1. **Given** a fresh clone of the repository, **When** a contributor opens the project in VS Code and reopens in the devcontainer, **Then** the container builds successfully and all required tools are available.
2. **Given** a running devcontainer, **When** the contributor runs `task ci`, **Then** the build, test, and lint steps all pass without errors.
3. **Given** a running devcontainer, **When** the contributor opens a Go file, **Then** the Go language server, linting, and formatting work correctly in the editor.

---

### User Story 2 - Pare Down from ASO Devcontainer (Priority: P1)

The devcontainer is based on the Azure Service Operator devcontainer structure but stripped down to only include the tools and configuration relevant to this project. ASO-specific dependencies (Azure CLI, Kubernetes tools, Docker-in-Docker, envtest, kind, controller-gen, etc.) are removed, leaving only what code-visualizer needs: Go, golangci-lint, gofumpt, go-task, and relevant VS Code extensions.

**Why this priority**: Equally critical to P1 — an overly bloated container increases build time, image size, and maintenance burden, directly undermining the value of having a devcontainer.

**Independent Test**: Can be tested by verifying the Dockerfile does not install ASO-specific tools, and that the resulting container image is significantly smaller than the ASO devcontainer.

**Acceptance Scenarios**:

1. **Given** the devcontainer configuration, **When** the container is built, **Then** it does not contain Azure CLI, Kubernetes tools, Docker CLI, envtest, kind, controller-gen, or other ASO-specific dependencies.
2. **Given** the devcontainer configuration, **When** the container is built, **Then** it contains Go, golangci-lint, gofumpt, and go-task.
3. **Given** the devcontainer configuration, **When** reviewed, **Then** the devcontainer.json name, extensions, settings, and mounts are appropriate for code-visualizer (not ASO).

---

### User Story 3 - Pre-populated Go Module Cache (Priority: P2)

When the devcontainer finishes building, the Go module cache is already populated with the project's dependencies. This means the first `go build` or `go test` does not need to download modules, making the initial development experience faster.

**Why this priority**: A nice-to-have optimization that improves the first-run experience but is not essential for the devcontainer to be functional.

**Independent Test**: Can be tested by opening the devcontainer and running `go build ./...` and confirming no modules are downloaded during the build.

**Acceptance Scenarios**:

1. **Given** a freshly built devcontainer, **When** the contributor runs `go build ./...`, **Then** no module downloads occur because they are already cached.
2. **Given** the Dockerfile, **When** the project's `go.mod` or `go.sum` changes, **Then** rebuilding the devcontainer picks up the updated dependencies.

---

### Edge Cases

- What happens when the contributor's machine does not have Docker installed? The devcontainer cannot be used; the project must remain buildable without the devcontainer using standard Go tooling.
- What happens when the container image base is updated (e.g., new Go version)? The Dockerfile should use a versioned base image tag so updates are intentional and trackable (e.g., via Dependabot).
- What happens when the Dockerfile.dockerignore is not aligned with the project structure? Only the files needed for the container build (devcontainer scripts, go.mod, go.sum) should be allowed through; everything else should be ignored for build performance.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The devcontainer MUST provide a working Go development environment with the same Go version specified in the project's `go.mod`.
- **FR-002**: The devcontainer MUST include golangci-lint, gofumpt, and go-task preinstalled and available on the PATH.
- **FR-003**: The devcontainer MUST configure VS Code with the Go extension, Task extension, and appropriate editor settings (format on save, organize imports on save, golangci-lint as the lint tool).
- **FR-004**: The devcontainer MUST NOT include ASO-specific tools or configuration (Azure CLI, Kubernetes tools, Docker-in-Docker, envtest, kind, controller-gen, etc.).
- **FR-005**: The devcontainer MUST pre-download Go modules so the first build does not require a network fetch.
- **FR-006**: The `Dockerfile.dockerignore` MUST be updated to match the project's structure (single `go.mod`/`go.sum` at root, no `v2/` subtree).
- **FR-007**: The `devcontainer.json` MUST use a project-appropriate name (not "Azure Service Operator").
- **FR-008**: The devcontainer MUST NOT require Docker-in-Docker or mount the host Docker socket.
- **FR-009**: The project MUST remain fully buildable and testable outside the devcontainer using standard Go tooling.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A new contributor can go from cloning the repository to a passing `task ci` run in under 5 minutes (excluding container image download time).
- **SC-002**: The devcontainer image contains only tools relevant to this project — no ASO-specific dependencies remain.
- **SC-003**: All existing tests pass inside the devcontainer (`task test` exits with 0).
- **SC-004**: VS Code language features (IntelliSense, formatting, linting) work correctly inside the devcontainer without additional manual configuration.
- **SC-005**: The first `go build` inside a freshly built container completes without downloading any modules.

## Assumptions

- Contributors use VS Code (or a compatible editor with devcontainer support such as GitHub Codespaces) as their primary development environment.
- Contributors have Docker (or a compatible container runtime) installed on their machine.
- The devcontainer is not required for CI — CI has its own environment setup. The devcontainer is purely for local development convenience.
- The existing Taskfile.yml commands (`task build`, `task test`, `task lint`, etc.) are the standard way to interact with the project and should work identically inside and outside the devcontainer.
- The Go devcontainer base image from Microsoft (`mcr.microsoft.com/devcontainers/go`) is used as the foundation, consistent with the ASO model.
