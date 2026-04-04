# Tasks: CLI Treemap Visualization

**Input**: Design documents from `/specs/001-cli-treemap-viz/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Included — constitution mandates TDD (Principle I, NON-NEGOTIABLE).

**Organization**: Tasks grouped by user story for independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup

**Purpose**: Go module, dependencies, build tooling, project skeleton

- [X] T001 Create project directory structure per plan: `cmd/codeviz/`, `internal/scan/`, `internal/metric/`, `internal/palette/`, `internal/treemap/`, `internal/render/`
- [X] T002 Initialize Go module (`go mod init`) and add dependencies to `go.mod`: kong, nikolaydubina/treemap, fogleman/gg, go-git/go-git/v5, onsi/gomega, sebdah/goldie/v2
- [X] T003 [P] Create `Taskfile.yml` with `build`, `test`, `lint` targets per constitution Development Workflow
- [X] T004 [P] Create `.golangci.yml` with linting rules per constitution Technology Constraints

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core types and infrastructure required by ALL user stories

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T005 Define `MetricName` custom string type with constants (`file-size`, `file-lines`, `file-type`, `file-age`, `file-freshness`, `author-count`) and `IsNumeric()`, `IsGitRequired()`, `IsValid()` methods in `internal/metric/metric.go`
- [X] T006 Write tests for `MetricName` methods (numeric check, git-required check, valid/invalid values) in `internal/metric/metric_test.go`
- [X] T007 [P] Define `PaletteName` custom string type with constants (`categorization`, `temperature`, `good-bad`, `neutral`) and `IsValid()` method in `internal/palette/palette.go`
- [X] T008 [P] Define `ColourPalette` struct (`Name PaletteName`, `Colours []color.RGBA`, `Ordered bool`) in `internal/palette/palette.go`
- [X] T009 [P] Define `FileNode` and `DirectoryNode` structs per data model in `internal/scan/scanner.go`
- [X] T010 [P] Define `TreemapRectangle` struct per data model in `internal/treemap/node.go`
- [X] T011 [P] Define `BucketBoundaries` struct per data model in `internal/metric/bucket.go`
- [X] T012 Define metric-to-default-palette mapping (`MetricDefaultPalette`) and `DefaultPaletteFor(MetricName)` function in `internal/metric/registry.go`
- [X] T013 Write tests for `DefaultPaletteFor()` — each metric returns correct palette — in `internal/metric/registry_test.go`
- [X] T014 Configure `slog` structured logger with `--verbose` level switching (INFO default, DEBUG when verbose) in `cmd/codeviz/main.go`

**Checkpoint**: All shared types defined — user story implementation can begin

---

## Phase 3: User Story 1 — Visualize a Directory by File Size (Priority: P1) 🎯 MVP

**Goal**: Scan a directory, compute file-size metric, produce squarified treemap layout, render to PNG with labels and directory headers

**Independent Test**: `codeviz ./sample -o out.png --size file-size` produces a valid PNG with proportional rectangles

### Tests for User Story 1 ⚠️

> **Write tests FIRST, ensure they FAIL before implementation**

- [X] T015 [P] [US1] Create test fixture directories in `internal/scan/testdata/`: `flat/` (3 files, known sizes), `nested/` (2 levels deep), `empty/`, `with-symlinks/` (file symlink + dir symlink)
- [X] T016 [P] [US1] Write scanner tests (flat scan, nested scan, empty dir returns error, file symlinks followed, dir symlinks skipped, permission-denied logs warning and continues) in `internal/scan/scanner_test.go`
- [X] T017 [P] [US1] Write tests for file-size metric extraction (regular file, zero-byte file, large file) in `internal/metric/metric_test.go`
- [X] T018 [P] [US1] Write tests for squarified layout (single file, multiple files proportional areas, nested dirs produce nested rectangles, zero-size file gets minimum rectangle per FR-013) in `internal/treemap/layout_test.go`
- [X] T019 [P] [US1] Write tests for directory header bar generation (header has directory name, padding separates groups) in `internal/treemap/node_test.go`
- [X] T020 [P] [US1] Write tests for label fitting (large rect → ShowLabel=true, small rect → ShowLabel=false, dark text on light fill, light text on dark fill) in `internal/render/label_test.go`
- [X] T021 [P] [US1] Write golden-file tests for PNG rendering (flat dir treemap, nested dir treemap) in `internal/render/renderer_test.go`

### Implementation for User Story 1

- [X] T022 [US1] Implement recursive directory scanner returning `DirectoryNode` tree in `internal/scan/scanner.go` — follow file symlinks, skip dir symlinks, log permission-denied via slog, error on empty dir
- [X] T023 [US1] Implement file-size metric populating `FileNode.Size` from `os.FileInfo` in `internal/metric/metric.go`
- [X] T024 [US1] Implement squarified treemap layout using `nikolaydubina/treemap` layout package, converting `DirectoryNode` tree to `TreemapRectangle` tree in `internal/treemap/layout.go` — handle zero-size files as minimum rectangles (FR-013), reserve space for directory header bars
- [X] T025 [US1] Implement `TreemapRectangle` tree construction with directory nesting, header bar nodes, and padding in `internal/treemap/node.go`
- [X] T026 [US1] Implement label fitting logic using `gg.MeasureString()` in `internal/render/label.go` — set `ShowLabel` based on available rect area, select dark/light text colour based on fill luminance
- [X] T027 [US1] Implement PNG renderer using `fogleman/gg` in `internal/render/renderer.go` — draw filled rectangles with structural dark borders (#333333), directory header bars, text labels where ShowLabel=true, write PNG to output path
- [X] T028 [US1] Define Kong CLI struct with `TargetPath` arg and `--output`, `--size`, `--verbose`, `--format`, `--width`, `--height` flags in `cmd/codeviz/main.go`
- [X] T029 [US1] Implement `Validate()` on CLI struct: target path exists and is a directory, output parent dir exists, --size must be numeric (FR-004) in `cmd/codeviz/main.go`
- [X] T030 [US1] Implement `Run()` method wiring scan → metric → layout → render pipeline in `cmd/codeviz/main.go`
- [X] T031 [US1] Implement text and JSON success/error output formatting per CLI contract in `cmd/codeviz/main.go`
- [X] T032 [US1] Implement exit codes 0–5 per CLI contract in `cmd/codeviz/main.go`
- [X] T033 [US1] Generate initial golden-file reference snapshots in `internal/render/testdata/`

**Checkpoint**: MVP complete — `codeviz ./dir -o out.png --size file-size` produces a valid treemap PNG

---

## Phase 4: User Story 2 — Colour Files by a Metric (Priority: P2)

**Goal**: Add fill colour driven by any metric mapped through a colour palette with quantile-based bucketing

**Independent Test**: `codeviz ./dir -o out.png --size file-size --fill file-size --fill-palette neutral` produces coloured rectangles

### Tests for User Story 2 ⚠️

> **Write tests FIRST, ensure they FAIL before implementation**

- [ ] T034 [P] [US2] Write tests for quantile bucketing (even distribution, skewed distribution, single value, all same value, boundary rounding to 2 sig figs, deduplication after rounding) in `internal/metric/bucket_test.go`
- [ ] T035 [P] [US2] Write tests for Neutral palette definition (exactly 9 steps, black→white, ordered=true) in `internal/palette/palette_test.go`
- [ ] T036 [P] [US2] Write tests for numeric metric-to-colour mapping (value at min→first colour, value at max→last colour, value at median→middle colour) in `internal/palette/mapper_test.go`
- [ ] T037 [P] [US2] Write tests for categorical metric-to-colour mapping (distinct values→distinct colours, wrap-around when >12 types logs warning) in `internal/palette/mapper_test.go`
- [ ] T038 [P] [US2] Write tests for file-lines metric (text file returns correct count, empty file returns 0, non-git dir treats all as text) in `internal/metric/metric_test.go`
- [ ] T039 [P] [US2] Write tests for file-type metric (`.go`→`go`, `.tar.gz`→`gz`, no extension→`no-extension`, mixed case preserved) in `internal/metric/metric_test.go`

### Implementation for User Story 2

- [ ] T040 [US2] Implement quantile-based bucketing with 2 sig-fig boundary rounding and deduplication in `internal/metric/bucket.go`
- [ ] T041 [US2] Implement file-lines metric (count newlines in file content, binary files via go-git report 0, non-git dirs treat all as text) in `internal/metric/metric.go`
- [ ] T042 [US2] Implement file-type metric extraction from file extension in `internal/metric/metric.go`
- [ ] T043 [US2] Define Neutral palette (9 monochromatic steps, black→white) with WCAG-compliant hex values in `internal/palette/palette.go`
- [ ] T044 [US2] Implement numeric metric-to-colour mapper (compute buckets → assign palette step via binary search) in `internal/palette/mapper.go`
- [ ] T045 [US2] Implement categorical metric-to-colour mapper (hash/index assignment, wrap-around with slog warning when values exceed palette capacity) in `internal/palette/mapper.go`
- [ ] T046 [US2] Add `--fill` and `--fill-palette` flags to Kong CLI struct (fill defaults to size metric, palette defaults to metric's default via registry) in `cmd/codeviz/main.go`
- [ ] T047 [US2] Integrate fill colour into pipeline: after metric computation, bucket fill metric values → map to palette colours → set `FillColour` on each `TreemapRectangle` in `cmd/codeviz/main.go`
- [ ] T048 [US2] Update renderer to use `TreemapRectangle.FillColour` instead of default fill in `internal/render/renderer.go`
- [ ] T049 [US2] Update golden-file snapshots for coloured output in `internal/render/testdata/`

**Checkpoint**: Fill colour working — any metric + palette combination colours treemap rectangles

---

## Phase 5: User Story 3 — Border Colour by a Second Metric (Priority: P3)

**Goal**: Add independent border colour driven by a separate metric + palette

**Independent Test**: `codeviz ./dir -o out.png --size file-size --fill file-lines --border file-type --border-palette categorization` shows coloured borders distinct from fills

### Tests for User Story 3 ⚠️

> **Write tests FIRST, ensure they FAIL before implementation**

- [ ] T050 [P] [US3] Write tests for border colour rendering (border visible when metric set, no border when metric absent, border colour independent of fill colour, same metric different palettes) in `internal/render/renderer_test.go`
- [ ] T051 [P] [US3] Write tests for CLI validation: `--border-palette` without `--border` returns exit code 1 with error message in `cmd/codeviz/main_test.go`

### Implementation for User Story 3

- [ ] T052 [US3] Add `--border` and `--border-palette` flags to Kong CLI struct in `cmd/codeviz/main.go`
- [ ] T053 [US3] Update `Validate()`: border-palette requires border metric; default palette applied when border specified without palette in `cmd/codeviz/main.go`
- [ ] T054 [US3] Integrate border colour into pipeline: bucket border metric values → map to border palette → set `BorderColour` on each `TreemapRectangle` in `cmd/codeviz/main.go`
- [ ] T055 [US3] Update renderer to draw coloured borders from `BorderColour` when present, replacing structural dark borders in `internal/render/renderer.go`
- [ ] T056 [US3] Update golden-file snapshots for bordered output in `internal/render/testdata/`

**Checkpoint**: Three data dimensions (size, fill, border) rendered independently in one treemap

---

## Phase 6: User Story 4 — Git-Aware Metrics (Priority: P4)

**Goal**: Extract file-age, file-freshness, author-count from git history; binary detection via go-git

**Independent Test**: `codeviz ./git-repo -o out.png --size file-lines --fill file-age` produces treemap with git-derived colours matching `git log` data

### Tests for User Story 4 ⚠️

> **Write tests FIRST, ensure they FAIL before implementation**

- [ ] T057 [P] [US4] Create test helper that initialises a temporary git repo with known commits (multiple files, multiple authors, known dates) for use in `internal/scan/gitinfo_test.go`
- [ ] T058 [P] [US4] Write tests for git repo detection (is git repo, is not git repo, nested inside git repo) in `internal/scan/gitinfo_test.go`
- [ ] T059 [P] [US4] Write tests for file-age computation (first commit date → correct duration) in `internal/scan/gitinfo_test.go`
- [ ] T060 [P] [US4] Write tests for file-freshness computation (most recent commit → correct duration) in `internal/scan/gitinfo_test.go`
- [ ] T061 [P] [US4] Write tests for author-count computation (distinct emails across commits) in `internal/scan/gitinfo_test.go`
- [ ] T062 [P] [US4] Write tests for binary file detection via go-git (binary file → IsBinary=true, text file → IsBinary=false) in `internal/scan/gitinfo_test.go`
- [ ] T063 [P] [US4] Write tests for untracked file handling (receives nil git metrics, sentinel "unknown" mapped to first palette step) in `internal/scan/gitinfo_test.go`
- [ ] T064 [P] [US4] Write tests for non-git directory with git metric producing exit code 3 with error message in `cmd/codeviz/main_test.go`

### Implementation for User Story 4

- [ ] T065 [US4] Implement git repository detection (open repo at target path via go-git) in `internal/scan/gitinfo.go`
- [ ] T066 [US4] Implement file-age extraction: iterate `Repository.Log` with `FileName` filter to last commit in `internal/scan/gitinfo.go`
- [ ] T067 [US4] Implement file-freshness extraction: take first commit from file-filtered log in `internal/scan/gitinfo.go`
- [ ] T068 [US4] Implement author-count extraction: collect unique `Author.Email` from file-filtered log in `internal/scan/gitinfo.go`
- [ ] T069 [US4] Implement binary file detection via go-git content inspection / `.gitattributes` in `internal/scan/gitinfo.go`
- [ ] T070 [US4] Handle untracked files: git metrics remain nil, binary defaults to false in `internal/scan/gitinfo.go`
- [ ] T071 [US4] Integrate git metadata into scan pipeline: after directory scan, enrich FileNodes with git fields when target is a git repo in `internal/scan/scanner.go`
- [ ] T072 [US4] Update file-lines metric to use `IsBinary` from go-git (binary → 0 lines) in `internal/metric/metric.go`
- [ ] T073 [US4] Implement exit code 3 and error message for git-required metric on non-git directory in `cmd/codeviz/main.go`

**Checkpoint**: Git metrics fully functional — file-age, file-freshness, author-count available for size/fill/border

---

## Phase 7: User Story 5 — Selecting Colour Palettes (Priority: P5)

**Goal**: Define and validate all four palettes; ensure end-to-end palette selection works correctly

**Independent Test**: Generate treemaps with each palette; verify correct step counts, colour values, and WCAG compliance

### Tests for User Story 5 ⚠️

> **Write tests FIRST, ensure they FAIL before implementation**

- [ ] T074 [P] [US5] Write tests verifying Categorization palette: exactly 12 colours, ordered=false in `internal/palette/palette_test.go`
- [ ] T075 [P] [US5] Write tests verifying Temperature palette: exactly 11 steps, first=dark blue, middle=white, last=bright red, ordered=true in `internal/palette/palette_test.go`
- [ ] T076 [P] [US5] Write tests verifying Good/Bad palette: exactly 13 steps, first=red, last=green, ordered=true in `internal/palette/palette_test.go`
- [ ] T077 [P] [US5] Write test verifying Neutral palette step count and ordering already defined in Phase 4 (regression guard) in `internal/palette/palette_test.go`
- [ ] T078 [P] [US5] Write WCAG contrast ratio validation tests: adjacent palette colours ≥3:1 contrast for all four palettes in `internal/palette/palette_test.go`
- [ ] T079 [P] [US5] Write golden-file rendering tests: one treemap per palette in `internal/render/renderer_test.go`

### Implementation for User Story 5

- [ ] T080 [US5] Define Categorization palette (12 visually distinct unordered colours, sourced from ColorBrewer Set3/Paired) in `internal/palette/palette.go`
- [ ] T081 [US5] Define Temperature palette (11 steps, dark blue→white→bright red, sourced from ColorBrewer RdBu diverging) in `internal/palette/palette.go`
- [ ] T082 [US5] Define Good/Bad palette (13 steps, red→orange→yellow→green, sourced from ColorBrewer RdYlGn) in `internal/palette/palette.go`
- [ ] T083 [US5] Implement WCAG relative luminance and contrast ratio utility functions for palette validation in `internal/palette/palette.go`
- [ ] T084 [US5] Generate golden-file reference snapshots for all four palette renderings in `internal/render/testdata/`

**Checkpoint**: All four palettes defined, validated for WCAG AA, and rendering correctly

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, benchmarks, final quality validation

- [ ] T085 [P] Add package-level doc comments to all `internal/` packages (`scan`, `metric`, `palette`, `treemap`, `render`)
- [ ] T086 [P] Create `docs/usage.md` with CLI usage examples from CLI contract
- [ ] T087 [P] Add benchmark tests for scan+render pipeline (1,000-file fixture) in `internal/render/renderer_test.go`
- [ ] T088 Run `task lint` (`golangci-lint`) across entire codebase — fix all warnings
- [ ] T089 Run `task test` — verify all tests pass
- [ ] T090 Run quickstart.md validation: build binary, run against a real directory, verify PNG output opens correctly
- [ ] T091 Verify all exit codes (0–5) against CLI contract with targeted test cases

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 — BLOCKS all user stories
- **US1 (Phase 3)**: Depends on Phase 2 — establishes core pipeline
- **US2 (Phase 4)**: Depends on US1 (needs rendering pipeline)
- **US3 (Phase 5)**: Depends on US2 (builds on fill colour infrastructure)
- **US4 (Phase 6)**: Depends on Phase 2 for types; depends on US1 for pipeline integration
- **US5 (Phase 7)**: Depends on US2 (needs palette mapping infrastructure)
- **Polish (Phase 8)**: Depends on all user stories

### User Story Dependencies

- **US1 (P1)**: Foundation only — no other story dependencies
- **US2 (P2)**: Depends on US1 (renderer must exist to add fill colour)
- **US3 (P3)**: Depends on US2 (extends fill colour to border colour)
- **US4 (P4)**: Depends on US1 for pipeline; git extraction itself is independent
- **US5 (P5)**: Depends on US2 (needs palette + mapper infrastructure)

### Parallel Opportunities

- Phase 1: T003 ∥ T004
- Phase 2: T007 ∥ T008 ∥ T009 ∥ T010 ∥ T011 (all type defs, different files)
- Each user story: all test tasks marked [P] can run in parallel
- US4 git extraction tests/impl (T057–T070) can run in parallel with US3 and US5 work
- US5 palette definition (T074–T084) can run in parallel with US4
- Phase 8: T085 ∥ T086 ∥ T087

### Within Each User Story

- Tests MUST be written and FAIL before implementation begins
- Type definitions before logic that uses them
- Internal packages before CLI integration
- Implementation before golden-file snapshot generation

---

## Implementation Strategy

### MVP Scope

User Story 1 alone is a viable MVP: scan a directory, lay out a squarified treemap sized by file-size, render to PNG with labels and directory headers.

### Incremental Delivery

1. **US1** (P1): Core scan → layout → render pipeline with file-size metric
2. **US2** (P2): Fill colour via palette + bucketing — doubles info density
3. **US3** (P3): Border colour — triples info density
4. **US4** (P4): Git-derived metrics — key differentiator
5. **US5** (P5): All four palettes defined and WCAG-validated
6. **Polish**: Docs, benchmarks, lint, final validation
