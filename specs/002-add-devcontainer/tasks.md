# Tasks: Add Devcontainer

**Input**: Design documents from `/specs/002-add-devcontainer/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/devcontainer-contract.md, quickstart.md

**Tests**: Not required — this feature involves configuration files only. Verification is manual (`task ci` inside the container).

**Organization**: US1 (working devcontainer) and US2 (strip ASO) are both P1 and deeply intertwined — stripping ASO tooling IS the work of making the devcontainer correct. They are combined into a single phase. US3 (module cache) is a separate P2 phase.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup

**Purpose**: No setup needed — the `.devcontainer/` directory and all four files already exist (copied from ASO). This feature is entirely about modifying existing files.

_(No tasks — proceed directly to Phase 2)_

---

## Phase 2: User Stories 1 & 2 — Working Devcontainer with ASO Stripped (Priority: P1)

**Goal**: Simplify the ASO devcontainer so it provides a minimal, working Go development environment for code-visualizer with no ASO-specific tooling.

**Independent Test**: Open the project in the devcontainer, run `task ci`, and confirm build/test/lint all pass. Verify no ASO-specific tools are present.

### Implementation

- [ ] T001 [P] [US1] [US2] Update devcontainer name, extensions, mounts, and remove ASO-specific settings in `.devcontainer/devcontainer.json`
- [ ] T002 [P] [US1] [US2] Strip ASO APT packages (Azure CLI, Docker CLI, nodejs, npm, python3-pip, graphviz, gnuplot) and remove envtest, kubectl, kind, and Docker group setup from `.devcontainer/Dockerfile`
- [ ] T003 [P] [US1] [US2] Simplify install-dependencies.sh to only install gofumpt, golangci-lint, and go-task; remove all ASO tools, az/pip3 checks, kubebuilder/buildx dest, webhook certs, and python virtualenv from `.devcontainer/install-dependencies.sh`
- [ ] T004 [P] [US1] [US2] Update Dockerfile.dockerignore to reference root-level go.mod/go.sum instead of v2/ paths in `.devcontainer/Dockerfile.dockerignore`
- [ ] T005 [US1] Configure git safe.directory for /workspace in postCreateCommand in `.devcontainer/devcontainer.json`

- [ ] T006 [US1] [US2] Run `docker build -f .devcontainer/Dockerfile .` to verify the container image builds successfully after ASO stripping

**Checkpoint**: At this point, the devcontainer configuration should be clean of all ASO content and properly configured for code-visualizer. Container should build successfully.

---

## Phase 3: User Story 3 — Pre-populated Go Module Cache (Priority: P2)

**Goal**: Pre-download Go modules during container build so the first `go build` or `go test` requires no network access.

**Independent Test**: Open the devcontainer, run `go build ./...`, and confirm no module downloads occur.

### Implementation

- [ ] T007 [US3] Replace ASO multi-module COPY/download block with single-module `COPY go.mod go.sum` + `go mod download` in `.devcontainer/Dockerfile`

**Checkpoint**: First `go build` inside the container completes without downloading modules.

---

## Phase 4: Polish & Cross-Cutting Concerns

**Purpose**: Final validation and documentation alignment

- [ ] T008 Verify `task ci` passes both inside the built devcontainer and outside it (manual validation, covers FR-009)
- [ ] T009 [P] Run quickstart.md validation — confirm the steps in `specs/002-add-devcontainer/quickstart.md` match the actual devcontainer experience

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: Skipped — files already exist
- **Phase 2 (US1 & US2)**: No dependencies — can start immediately
- **Phase 3 (US3)**: Can run in parallel with Phase 2 (different Dockerfile section) but logically depends on the Dockerfile being cleaned up in T002 first
- **Phase 4 (Polish)**: Depends on Phases 2 and 3 being complete

### Within Phase 2

- T001, T002, T003, T004 are all [P] — they modify different files and can be done in parallel
- T005 depends on T001 (modifies the same file: devcontainer.json)
- T006 depends on T001–T005 (builds the image to verify all edits)

### Task File Mapping

| Task       | File                                       |
| ---------- | ------------------------------------------ |
| T001, T005 | `.devcontainer/devcontainer.json`          |
| T002, T007 | `.devcontainer/Dockerfile`                 |
| T003       | `.devcontainer/install-dependencies.sh`    |
| T004       | `.devcontainer/Dockerfile.dockerignore`    |
| T006       | `docker build` verification (no file edit) |
| T008       | Manual verification (no file)              |
| T009       | `specs/002-add-devcontainer/quickstart.md` |

### Parallel Opportunities

```
T001 ─┐
T002 ─┤ (all in parallel — different files)
T003 ─┤
T004 ─┘
  │
T005 (after T001 — same file)
  │
T006 (docker build — verifies T001–T005)
T007 (after T002 — same file, module cache)
  │
T008 (after all implementation tasks)
T009 (in parallel with T008)
```

---

## Implementation Strategy

### MVP First (Phase 2 Only)

1. Complete T001–T005 (strip ASO, configure for code-visualizer)
2. Run T006 (`docker build`) to verify the image builds
3. **STOP and VALIDATE**: Open in devcontainer, run `task ci`
4. If passing, the devcontainer is usable — module pre-caching (Phase 3) is a bonus

### Full Delivery

1. Complete Phase 2 (T001–T006) → Core devcontainer building and working
2. Complete Phase 3 (T007) → Module cache pre-populated
3. Complete Phase 4 (T008–T009) → Validated and documented
