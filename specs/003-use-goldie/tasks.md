# Tasks: Use Goldie for Golden File Testing

**Input**: Design documents from `/specs/003-use-goldie/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2)
- Include exact file paths in descriptions

---

## Phase 1: Setup

**Purpose**: Add Goldie v2 dependency to the project

- [x] T001 Add `github.com/sebdah/goldie/v2` dependency in go.mod

---

## Phase 2: User Story 1 - Replace handwritten golden file infrastructure with Goldie (Priority: P1) 🎯 MVP

**Goal**: Migrate the `goldenPaletteTest` helper and all 4 `TestGoldenFile_*` tests to use Goldie v2 for byte-level comparison, removing all handwritten golden file read/write/compare code.

**Independent Test**: Run `task test` — all 4 golden file tests pass using Goldie. Inspect `goldenPaletteTest` helper in `internal/render/renderer_test.go` — no manual file I/O, `image/png` decode, or pixel-comparison loops remain.

### Implementation for User Story 1

- [x] T002 [US1] Rewrite `goldenPaletteTest` helper to use Goldie v2 API (`goldie.New` with `WithFixtureDir("testdata")` and `WithNameSuffix(".png")`) in internal/render/renderer_test.go
- [x] T003 [US1] Remove unused imports (`image/png`) and add `goldie/v2` import in internal/render/renderer_test.go after Goldie migration
- [x] T004 [US1] Run `task test` and verify all 4 `TestGoldenFile_*` tests pass with Goldie

**Checkpoint**: All golden file tests pass using Goldie. No handwritten comparison code remains.

---

## Phase 3: User Story 2 - Consistent golden file update workflow (Priority: P2)

**Goal**: Update the Taskfile `update-golden-files` task to use Goldie's `GOLDIE_UPDATE` environment variable instead of the custom `UPDATE_GOLDEN`.

**Independent Test**: Run `task update-golden-files` — golden files in `internal/render/testdata/` are regenerated using Goldie's native update mechanism.

### Implementation for User Story 2

- [x] T005 [US2] Update `update-golden-files` task to use `GOLDIE_UPDATE=1` instead of `UPDATE_GOLDEN=1` in Taskfile.yml
- [x] T006 [US2] Run `task update-golden-files` and verify golden files are regenerated correctly

**Checkpoint**: Developers use `GOLDIE_UPDATE=1` or `-update` flag exclusively. Custom `UPDATE_GOLDEN` env var is retired.

---

## Phase 4: Polish & Cross-Cutting Concerns

**Purpose**: Final validation

- [x] T007 Run `task ci` (build, test, lint) to confirm no regressions

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **User Story 1 (Phase 2)**: Depends on Phase 1 (Goldie dependency must be available)
- **User Story 2 (Phase 3)**: Depends on Phase 2 (Goldie must be wired into tests before updating the task runner)
- **Polish (Phase 4)**: Depends on Phases 2 and 3

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Setup — no dependencies on US2
- **User Story 2 (P2)**: Depends on US1 completion (Goldie must be used in tests before updating the Taskfile to use `GOLDIE_UPDATE`)

### Within Each User Story

- T002 before T003 (rewrite helper before cleaning imports)
- T003 before T004 (clean code before verifying tests)
- T005 before T006 (update task before running it)

### Parallel Opportunities

- None in Phase 4 (single task remaining)

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Add Goldie dependency
2. Complete Phase 2: Migrate `goldenPaletteTest` to Goldie
3. **STOP and VALIDATE**: `task test` passes, no handwritten golden file code remains
4. This alone satisfies FR-001, FR-002, FR-004, FR-005, FR-006, FR-007

### Incremental Delivery

1. Add dependency → Foundation ready
2. Migrate tests → Core migration complete (MVP!)
3. Update Taskfile → Unified update workflow
4. Polish docs → Feature fully complete
