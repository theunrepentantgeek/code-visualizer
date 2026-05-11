# Bishop — History

## Core Context

- **Project:** A Go CLI tool (`codeviz`) that scans file trees and renders treemap visualizations as PNG images with configurable metrics and colour palettes.
- **Role:** Artificer — abstractions, interfaces, types, design patterns, code smells, structural integrity
- **Joined:** 2026-04-15
- **Requested by:** Bevan Arps

## Project Knowledge

- Language: Go 1.26+
- Key packages: `cmd/codeviz/` (entry), `internal/metric/`, `internal/palette/`, `internal/render/`, `internal/scan/`, `internal/treemap/`
- Key packages: `cmd/codeviz/` (entry), `internal/metric/`, `internal/palette/`, `internal/render/`, `internal/scan/`, `internal/treemap/`, `internal/provider/`
- Build: `task build` → `bin/codeviz`
- Test: `task test` (`go test ./... -count=1`)
- Lint: `task lint` (golangci-lint v2 with nilaway, wsl_v5, revive, wrapcheck, gci)
- Format: `task fmt` (gofumpt); imports ordered by gci
- Error handling: eris wrapping throughout
- Test assertions: Gomega (not testify); golden files via Goldie v2
- Go is already a pattern-friendly language: interfaces are implicit, types are first-class, package boundaries enforce encapsulation
- Key structural seams to watch: metric/palette/render boundaries, how scan feeds treemap, how types flow from scan → treemap → render

## Learnings

<!-- Append learnings below -->

### PR #39 Review (2026-04-15)

**Reviewed:** Provider interface extension with `Scope()` and `Description()`, plus 9 new folder metrics.

**Structural decisions affirmed:**
- `Scope` as typed string (not iota) is correct for Open/Closed extensibility
- Interface remains cohesive at 7 methods — no segregation needed
- `Description` as `string` is appropriate (not primitive obsession)
- Dependencies as `[]metric.Name` with runtime validation is pragmatic
- New `internal/provider/folder/` package is well-bounded

**Pattern identified for future:**
- Folder aggregation pattern (sum/max/min/mean over tree) appears in 9 providers
- Currently WET — could extract `aggregateSum()`, `aggregateMax()`, `aggregateMean()` helpers
- Not blocking, but watch for more folder metrics tipping toward DRY

**Review output:** Orchestration log at `.squad/orchestration-log/2026-04-15T04:50:46Z-bishop.md`

### Full Structural Audit (2026-07-07)

**Scope:** Complete review of `cmd/codeviz/` and all `internal/` packages.

**Issues filed (11 total):**
- #152 — Extract shared command workflow (highest impact, ~1,500 lines duplicated across 4 commands)
- #153 — Extract shared base config struct (data clump across 4 viz config types)
- #154 — Extract shared LabelMode type (triplicated across radialtree/bubbletree/spiral)
- #155 — Replace 7 git provider wrappers with declarative registration (WET boilerplate)
- #156 — Extract MetricBag from File/Directory (duplicated metric storage, ~130 lines)
- #157 — Deduplicate luminance calculation (render vs palette)
- #158 — Unify raster/SVG rendering paths (duplicated across 8 renderer files + legend)
- #159 — Move git history loading out of spiral layout package (boundary violation)
- #160 — LegendEntry union type permits invalid states
- #161 — Extract shared progress ticker pattern (triplicated goroutine lifecycle)
- #162 — Consider splitting Provider interface (ISP)

**Key structural observations:**
- The cmd/ layer is the primary pain point: ~2,576 lines across 4 commands with ~60% duplication
- The render package is the second hotspot: ~2,300 lines with raster/SVG duplication throughout
- The model layer has clean semantics but duplicated implementation (metric storage)
- Provider/git has good centralised helpers (`loadGitMetric`) but 7 near-identical wrapper files
- Config package acknowledges its own duplication via `//nolint:dupl` comments
- The spiral package is the only layout package with a data-access dependency (boundary smell)
- The codebase is well-structured overall — issues are about duplication and missing abstractions, not about fundamental design problems

**Priority ranking:**
1. #152 (command workflow) — highest leverage, eliminates most duplication
2. #158 (render unification) — second highest, prevents duplication scaling with new viz types
3. #155 (git providers) — quick win, enables cleaner provider model
4. #153 (config base) — quick win, removes acknowledged duplication
5. #156 (MetricBag) — clean extraction, low risk
6. #154 (LabelMode) — trivial but prevents drift
7. #157 (luminance) — trivial fix
8. #159 (spiral boundary) — important for testability
9. #160 (LegendEntry) — moderate, prevents nil bugs
10. #161 (progress ticker) — small cleanup
11. #162 (provider ISP) — consider alongside #155

## Issue #157 — Deduplicate Luminance Calculation (2026-05-04)

- **Status:** ✅ Complete
- **Work:** Deleted 16 lines of duplicated luminance calculation from `render/label.go`
- **Result:** Unified implementation now delegates to `palette.RelativeLuminance()` — single source of truth
- **Testing:** All tests pass, zero regressions
- **Committed:** `squad/157-dedup-luminance`
- **PR:** #165

### Canvas Abstraction Spec Review (2026-05-08)

- **Status:** ✅ Review complete
- **Spec:** `docs/superpowers/specs/2026-05-08-canvas-design.md`
- **Scope:** Reviewed abstractions, interfaces, type design, code smells, SOLID compliance

**Key findings (21 inline comments):**

1. **Ink type — best abstraction in the spec.** Correctly encapsulates the metric→color pipeline (Façade pattern). But dual `Dip`/`DipCategory` methods are a type-level smell — endorsed Bevan's `MetricValue` unification.
2. **RectangleSpec/DiscSpec data clump.** 6 of 7 fields identical — extract shared `ShapeStyle` base via embedding.
3. **Opacity belongs on Ink, not Spec** — endorsed Bevan's `WithOpacity()` suggestion.
4. **LegendEntry too thin.** Current legend rendering needs bucket boundaries, palette, and categories. Ink needs query methods or LegendEntry needs richer fields.
5. **LegendEntry.Role — primitive obsession.** Should be typed constant like existing `LegendPosition`.
6. **Backend subpackages — endorsed.** `internal/canvas/raster/` and `internal/canvas/svg/` for Ports & Adapters isolation.
7. **Canvas constructor should defer path/format to Render() time** — enables multi-format output and cleaner testing.
8. **Missing `AddText` method and `Text` shape** — `TextSpec` defined but no shape or drawing method.
9. **Backend long parameter lists** — flagged but tolerable for unexported interface with 2 impls.
10. **`drawArcText` breaks interface uniformity** — pragmatic, keep but document.
11. **Migration risk: sequence #152 (shared command workflow) before Stage 2** to avoid creating 4 bridge implementations that immediately need refactoring.

**Patterns identified:**
- Ink = Façade (multi-step pipeline behind single method)
- Specs = Flyweight (shared style template across many shapes)
- Canvas Add*/Render = Command pattern (retained then executed)
- Backend subpackages = Ports & Adapters
- Layer gaps = Sparse Namespace pattern

**Key file paths for implementation:**
- Spec: `docs/superpowers/specs/2026-05-08-canvas-design.md`
- New package: `internal/canvas/` with `raster/` and `svg/` subpackages
- Replaces: `internal/render/*.go` (8 renderer files) + cmd color application code
- Keeps: `internal/palette/`, `internal/metric/bucket.go`, layout packages (geometry-only)

### Team Orchestration (2026-05-09T02:59:06Z)

- **Cycle completed:** Three-agent Canvas spec review cycle finalized.
- **Orchestration:** Bishop review → Parker review → Dallas integration → Scribe logging.
- **Spec finalized:** All 5 key design decisions codified and approved. Ready for implementation kickoff.
- **Team log:** `.squad/log/2026-05-09T02:59:06Z-canvas-spec-review.md`
- **Decisions merged:** All inbox items → `decisions.md`. Specifications finalized.

### Issue #193 — Replace layeredShape Tagged Union with Go Interface (2026-07-16)

- **Status:** ✅ Complete
- **PR:** #212
- **Branch:** `squad/193-layered-shape-interface`

**What changed:**
- Deleted `shapeKind` type (iota enum) and its 6 constants from `canvas.go`
- Replaced 6-pointer tagged union in `layeredShape` with single `shape drawnShape` interface field
- Defined `drawnShape` interface: `drawTo(backend Backend)`
- Added `drawTo` method to each concrete shape type: `Rectangle`, `Disc`, `Text`, `Line`, `Path` (in `shape.go`), `ArcText` (in `text_spec.go`)
- Deleted `dispatchShape` switch and 6 `Canvas.draw*` helper methods
- Simplified all 6 `Add*` methods on Canvas — no more `kind` field or named pointer fields
- Net result: -37 lines (81 added, 118 deleted across 3 files)

**Key files:**
- `internal/canvas/canvas.go` — `drawnShape` interface, simplified `layeredShape`, cleaned `Add*` methods, removed dispatch
- `internal/canvas/shape.go` — `drawTo` on Rectangle, Disc, Text, Line, Path
- `internal/canvas/text_spec.go` — `drawTo` on ArcText

**Pattern observed:**
- The old `Canvas.draw*` methods had unused `*Canvas` receivers — pure functions masquerading as methods. Moving them to shape types was zero-friction because they were already stateless. This is the classic signal that logic belongs on the data it operates on, not on the coordinator that holds it.
- Adding a new shape is now safe by construction: implement `drawnShape`, and the compiler ensures it works end-to-end. No enum, no switch, no nullable pointer field to maintain.
