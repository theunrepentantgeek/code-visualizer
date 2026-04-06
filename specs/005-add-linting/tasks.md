# Tasks: Configure Linting

**Input**: Design documents from `/specs/005-add-linting/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Not included — this feature is build/tooling infrastructure with no new production code. Existing tests must continue to pass.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup

**Purpose**: Ensure devcontainer has required tooling prerequisites

- [X] T001 Verify envsubst availability in devcontainer and add gettext-base to .devcontainer/Dockerfile if not already present
- [X] T002 Add SCRIPT_DIR variable to .devcontainer/install-dependencies.sh for locating template files

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Create the custom linter build template and install script changes that ALL user stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T003 Create custom linter build template at .devcontainer/.custom-gcl.template.yml with nilaway plugin definition (version v2.8.0, destination $TOOL_DEST)
- [X] T004 Add custom linter build step to .devcontainer/install-dependencies.sh after the existing golangci-lint installation block
- [X] T005 Build the custom golangci-lint-custom binary by running the updated install-dependencies.sh and verify it exists in the tools directory

**Checkpoint**: Custom linter binary available — configuration and task changes can now proceed

---

## Phase 3: User Story 1 - Developer Runs Comprehensive Linting (Priority: P1) 🎯 MVP

**Goal**: Developer can run `task lint` and get comprehensive linting including nilaway

**Independent Test**: Run `task lint` and verify the expanded linter set executes with nilaway included

- [X] T006 [P] [US1] Replace .golangci.yml with expanded configuration imported from go-vcr-tidy reference project, adapting module paths (nilaway include-pkgs, revive dot-imports, wrapcheck eris sigs) for this project
- [X] T007 [P] [US1] Update lint task in Taskfile.yml to invoke golangci-lint-custom run --verbose instead of golangci-lint run ./...
- [X] T008 [US1] Update tidy task in Taskfile.yml to invoke golangci-lint-custom run --fix --verbose instead of golangci-lint run --fix ./...
- [X] T009 [US1] Run task lint to verify the expanded linter set executes successfully and nilaway is included in the output

**Checkpoint**: `task lint` runs with expanded linter set including nilaway

---

## Phase 4: User Story 2 - Developer Sets Up Environment with Custom Linter (Priority: P1)

**Goal**: install-dependencies.sh automatically builds the custom linter binary during devcontainer setup

**Independent Test**: Run install-dependencies.sh and verify golangci-lint-custom binary is produced

- [X] T010 [US2] Verify install-dependencies.sh builds golangci-lint-custom in devcontainer mode (TOOL_DEST=/usr/local/bin)
- [X] T011 [US2] Verify install-dependencies.sh skips the custom build when using --skip-installed and binary already exists

**Checkpoint**: Devcontainer setup produces the custom linter binary without manual steps

---

## Phase 5: User Story 3 - Developer Fixes Existing Lint Issues (Priority: P2)

**Goal**: All existing code passes the new expanded linter configuration with zero errors

**Independent Test**: Run `task lint` on the full codebase and confirm zero exit code

- [X] T012 [US3] Run task lint and capture all lint issues reported by the expanded linter set
- [X] T013 [US3] Fix lint issues in cmd/codeviz/main.go — prefer root-cause fixes over suppressions
- [X] T014 [P] [US3] Fix lint issues in internal/metric/ package files — prefer root-cause fixes over suppressions
- [X] T015 [P] [US3] Fix lint issues in internal/palette/ package files — prefer root-cause fixes over suppressions
- [X] T016 [P] [US3] Fix lint issues in internal/render/ package files — prefer root-cause fixes over suppressions
- [X] T017 [P] [US3] Fix lint issues in internal/scan/ package files — prefer root-cause fixes over suppressions
- [X] T018 [P] [US3] Fix lint issues in internal/treemap/ package files — prefer root-cause fixes over suppressions
- [X] T019 [US3] Run task lint to confirm zero exit code across entire codebase

**Checkpoint**: All existing code passes the expanded linter configuration

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final validation and documentation

- [X] T020 Run task ci (build, test, lint) to confirm all checks pass end-to-end
- [X] T021 Run quickstart.md validation — verify golangci-lint-custom linters | grep nilaway shows nilaway in linter list

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion — BLOCKS all user stories
- **User Story 1 (Phase 3)**: Depends on Foundational phase — config and Taskfile changes
- **User Story 2 (Phase 4)**: Depends on Foundational phase — verification of install script
- **User Story 3 (Phase 5)**: Depends on User Story 1 completion (needs linter config in place to identify issues)
- **Polish (Phase 6)**: Depends on all user stories complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) — No dependencies on other stories
- **User Story 2 (P1)**: Can start after Foundational (Phase 2) — Independent of US1
- **User Story 3 (P2)**: Depends on User Story 1 (needs expanded config to identify issues)

### Within Each User Story

- T006 and T007 can run in parallel (different files)
- T008 depends on T007 (same file, sequential edits)
- T014–T018 can all run in parallel (different packages)

### Parallel Opportunities

- T006 and T007 can run in parallel (different files: .golangci.yml vs Taskfile.yml)
- T010 and T011 are sequential (same verification workflow)
- T014, T015, T016, T017, T018 can all run in parallel (independent packages)

---

## Parallel Example: User Story 3

```bash
# Fix lint issues across all packages in parallel:
Task: "Fix lint issues in internal/metric/ package files"
Task: "Fix lint issues in internal/palette/ package files"
Task: "Fix lint issues in internal/render/ package files"
Task: "Fix lint issues in internal/scan/ package files"
Task: "Fix lint issues in internal/treemap/ package files"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (envsubst availability)
2. Complete Phase 2: Foundational (custom build template + script)
3. Complete Phase 3: User Story 1 (config + Taskfile)
4. **STOP and VALIDATE**: Run `task lint` — expanded linters work
5. Functional MVP: comprehensive linting available

### Incremental Delivery

1. Complete Setup + Foundational → Custom binary built
2. Add User Story 1 → Linter config + task changes → MVP!
3. Add User Story 2 → Verify devcontainer automation
4. Add User Story 3 → Fix all existing lint issues → Clean baseline
5. Polish → Full CI validation

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- No test tasks included — this is infrastructure/tooling, not production code
- Existing tests (`task test`) must continue to pass after lint fixes
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
