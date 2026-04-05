# Tasks: Exclude Binary Files for Line-Count Metric

**Input**: Design documents from `/specs/004-exclude-binary-lines/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/cli-contract.md, quickstart.md

**Tests**: Required by Constitution Principle I (Test-First, NON-NEGOTIABLE). Tests are written before implementation.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup

**Purpose**: No new project infrastructure needed — this feature adds to an existing codebase. Setup is limited to ensuring binary detection works end-to-end before filtering.

- [x] T001 Verify existing binary detection pipeline works by running `task test` in internal/scan/

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core filter function and error type that ALL user stories depend on.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

### Tests (write first, must FAIL)

- [x] T002 [P] Write test for FilterBinaryFiles with mixed binary and text files returning only text files in internal/scan/scanner_test.go
- [x] T003 [P] Write test for FilterBinaryFiles with all binary files returning empty tree in internal/scan/scanner_test.go
- [x] T004 [P] Write test for FilterBinaryFiles with no binary files returning tree unchanged in internal/scan/scanner_test.go
- [x] T005 [P] Write test for FilterBinaryFiles pruning directories that become empty after binary removal in internal/scan/scanner_test.go
- [x] T006 [P] Write test for FilterBinaryFiles with nested directories where only deepest dir has binary files in internal/scan/scanner_test.go
- [x] T007 [P] Write test for FilterBinaryFiles logging excluded files at Debug level in internal/scan/scanner_test.go

### Implementation

- [x] T008 Implement FilterBinaryFiles function in internal/scan/scanner.go
- [x] T009 Add noFilesAfterFilterError type to cmd/codeviz/main.go
- [x] T010 [P] Add exit code 6 mapping in classifyError function in cmd/codeviz/main.go
- [x] T011 Run `task test` to verify all foundational tests pass

**Checkpoint**: FilterBinaryFiles works correctly in isolation; error type and exit code defined.

---

## Phase 3: User Story 1 — Binary Files Omitted from Line-Count Treemap (Priority: P1) 🎯 MVP

**Goal**: When `--size file-lines` is specified, binary files are completely excluded from the treemap visualization.

**Independent Test**: Run `codeviz` against a directory with binary and text files using `--size file-lines` and verify binary files are absent from the output.

### Tests for User Story 1 (write first, must FAIL)

- [x] T012 [P] [US1] Write test verifying filter is called when size metric is file-lines in cmd/codeviz/main.go (or internal test)
- [x] T013 [P] [US1] Write test verifying noFilesAfterFilterError with exit code 6 when all files are binary in cmd/codeviz/main.go

### Implementation for User Story 1

- [x] T014 [US1] Add FilterBinaryFiles call in Run() pipeline after PopulateLineCounts and before treemap.Layout when size metric is file-lines in cmd/codeviz/main.go
- [x] T015 [US1] Add zero-file check after filtering that returns noFilesAfterFilterError in cmd/codeviz/main.go
- [x] T016 [US1] Add verbose logging summary (excluded count, remaining count) after filter call in cmd/codeviz/main.go
- [x] T017 [US1] Run `task test` to verify all US1 tests pass

**Checkpoint**: User Story 1 is fully functional — `codeviz --size file-lines` excludes binary files from the treemap.

---

## Phase 4: User Story 2 — Binary Files Still Included for File-Size Treemap (Priority: P2)

**Goal**: Confirm that `--size file-size` (and all other non-line-count metrics) continues to include binary files — no regression.

**Independent Test**: Run `codeviz` against a directory with binary and text files using `--size file-size` and verify all files appear.

### Tests for User Story 2 (write first, must FAIL)

- [ ] T018 [P] [US2] Write test verifying filter is NOT called when size metric is file-size in cmd/codeviz/main.go (or internal test)
- [ ] T019 [P] [US2] Write test verifying filter is NOT called when size metric is file-age in cmd/codeviz/main.go (or internal test)

### Verification for User Story 2

- [ ] T020 [US2] Review existing tests for file-size mode to confirm no regressions from Phase 2/3 changes
- [ ] T021 [US2] Run `task test` to verify all existing tests pass — no regressions

**Checkpoint**: User Story 2 confirmed — file-size and other non-line-count metrics still include binary files.

---

## Phase 5: User Story 3 — Fill/Border Metric Interaction (Priority: P3)

**Goal**: Confirm that fill and border metric choices do not affect which files appear in the treemap — exclusion is driven by the size metric only.

**Independent Test**: Run `codeviz --size file-lines --fill file-type` and verify binary files are still excluded despite file-type being defined for them.

### Tests for User Story 3 (write first, must FAIL)

- [ ] T022 [P] [US3] Write test verifying binary files excluded when size=file-lines and fill=file-type in cmd/codeviz/main.go (or internal test)
- [ ] T023 [P] [US3] Write test verifying binary files included when size=file-size and fill=file-type in cmd/codeviz/main.go (or internal test)

### Verification for User Story 3

- [ ] T024 [US3] Review that filter call site in Run() is conditioned on c.Size only, not fillMetric or borderMetric in cmd/codeviz/main.go
- [ ] T025 [US3] Run `task test` to verify all US3 tests pass

**Checkpoint**: User Story 3 confirmed — fill/border metrics do not affect file inclusion.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, final validation, cleanup.

- [ ] T026 [P] Add exit code 6 row to exit code table in docs/usage.md
- [ ] T027 [P] Run `task lint` to verify no linting issues
- [ ] T028 Run `task ci` to verify full CI pipeline passes (build + test + lint)
- [ ] T029 Run quickstart.md validation — verify implementation matches all design decisions

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 — BLOCKS all user stories
- **User Story 1 (Phase 3)**: Depends on Phase 2 completion
- **User Story 2 (Phase 4)**: Depends on Phase 2 completion; can run in parallel with Phase 3
- **User Story 3 (Phase 5)**: Depends on Phase 2 completion; can run in parallel with Phase 3/4
- **Polish (Phase 6)**: Depends on Phase 3 completion (minimum); ideally after all stories

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) — no dependencies on other stories
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) — independent regression verification
- **User Story 3 (P3)**: Can start after Foundational (Phase 2) — independent interaction verification

### Within Each User Story

- Tests MUST be written and FAIL before implementation (Constitution Principle I)
- Implementation follows Red → Green → Refactor cycle
- Run `task test` after implementation to verify Green
- Story complete before checkpoint

### Parallel Opportunities

- T002–T007: All foundational tests can be written in parallel (different test cases, same file)
- T009–T010: Error type and exit code mapping can be written in parallel (same file but independent sections)
- T012–T013: US1 tests can be written in parallel
- T018–T019: US2 tests can be written in parallel
- T022–T023: US3 tests can be written in parallel
- T026–T027: Polish tasks can run in parallel (different files)
- Phases 3, 4, 5 can run in parallel once Phase 2 is complete

---

## Parallel Example: Foundational Phase

```bash
# Write all filter tests in parallel:
Task T002: "Test FilterBinaryFiles mixed files"
Task T003: "Test FilterBinaryFiles all binary"
Task T004: "Test FilterBinaryFiles no binary"
Task T005: "Test FilterBinaryFiles directory pruning"
Task T006: "Test FilterBinaryFiles nested directory pruning"
Task T007: "Test FilterBinaryFiles debug logging"

# Then implement sequentially:
Task T008: "Implement FilterBinaryFiles"
Task T009: "Add noFilesAfterFilterError"
Task T010: "Add exit code 6 mapping"
Task T011: "Run tests"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (verify existing tests)
2. Complete Phase 2: Foundational (filter function + error type)
3. Complete Phase 3: User Story 1 (wire filter into pipeline)
4. **STOP and VALIDATE**: Test with `codeviz --size file-lines` on real repo
5. Ready for use

### Incremental Delivery

1. Setup + Foundational → Filter function ready
2. User Story 1 → Binary files excluded for line count → MVP!
3. User Story 2 → Regression confirmed for file size
4. User Story 3 → Interaction confirmed for fill/border
5. Polish → Documentation and CI green

---

## Notes

- [P] tasks = different files or independent sections, no dependencies
- [Story] label maps task to specific user story for traceability
- Constitution Principle I (Test-First) is NON-NEGOTIABLE — all tests written before implementation
- The filter function is the single new code artifact; everything else is wiring and verification
- Commit after each phase checkpoint
