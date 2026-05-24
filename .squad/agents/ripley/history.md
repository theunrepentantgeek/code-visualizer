# Ripley — History

## Core Context

- **Project:** A Go CLI tool that scans file trees and renders treemap visualizations as PNG images with configurable metrics and colour palettes.
- **Role:** Lead
- **Joined:** 2026-04-14T09:49:33.769Z

## Learnings

<!-- Append learnings below -->

### Issue #284 — Radial Group Gaps (2026-05-23)

- **Layout pattern:** `internal/radialtree/layout.go` is the sole source of angular placement; rendering in `internal/radialtree/render.go` just consumes node angles/positions, so visual spacing changes belong in layout, not rendering.
- **Gap design:** Non-root directory sectors now reserve one child-sized blank slot at both the leading and trailing edges of the sector. This keeps sibling folder groups visually separated without adding CLI/config surface.
- **Allocation rule:** Angular budgeting must count empty directories too. `childAllocationUnits()` sums direct files plus `childWeight()` for each subdir, where empty subdirs still reserve one unit instead of collapsing onto the same angle.
- **Regression coverage:** `internal/radialtree/layout_test.go` now covers both grouped file gaps and empty sibling directories getting distinct sectors.
- **Key files:** `internal/radialtree/layout.go`, `internal/radialtree/layout_test.go`, `cmd/codeviz/radialtree_cmd.go`.
### Issue #282 — Bubbletree Legend Overlap (2026-05-23)

- **Root cause:** `internal/bubbletree/stages.go` explicitly left legend handling in overlay mode; unlike treemap, bubbletree ran layout against the full canvas and only attached the legend at render time.
- **Adopted pattern:** Use shared legend helpers from `internal/legend/reserve.go` — `ReserveAndLayout()` to shrink the layout area and `LayoutOffset()` to translate the finished layout into the non-legend region.
- **Bubble-specific detail:** Bubbletree needed its own recursive translation helper (`OffsetNodes` in `internal/bubbletree/layout.go`) because its layout stores absolute node centres, not rectangles.
- **Regression coverage:** `internal/bubbletree/layout_stage_test.go` verifies top and left legend positions reserve space and keep bubble content inside the remaining drawable bounds.
- **Key files:** `internal/bubbletree/stages.go`, `internal/bubbletree/layout.go`, `internal/bubbletree/layout_stage_test.go`, `internal/legend/reserve.go`, `internal/treemap/stages.go`.

### PR Review Etiquette (Team Directive, 2026-05-13)

- **PR review reply protocol:** After addressing a PR review comment, ALWAYS reply to the comment indicating what was done. Don't leave reviewers hanging. This closes the feedback loop and keeps communication clear for all stakeholders.

### PR #45 Review — Image Format Support (2026-04-18) — COMPLETED

- **Review comments addressed:** 6 issues across 6 files — all confirmed by repo owner.
- **Double-close pattern:** `defer f.Close()` + explicit `f.Close()` at return causes double-close. Fix: named return `(err error)` + deferred closure that conditionally assigns close error. Applied to `svg_radial.go`, `svg_treemap.go`, `save.go`.
- **XML unmarshal in tests:** `xml.Unmarshal(data, new(any))` fails because `encoding/xml` can't unmarshal into `*interface{}`. Use `var parsed struct{}` + `&parsed` instead.
- **CLI help text:** Both `treemap_cmd.go` and `radialtree_cmd.go` needed `jpeg` added to the supported formats list alongside `jpg`.
- **Key files:** `internal/render/svg_radial.go`, `internal/render/svg_treemap.go`, `internal/render/save.go`, `internal/render/renderer_test.go`, `cmd/codeviz/treemap_cmd.go`, `cmd/codeviz/radialtree_cmd.go`.
- **CI status:** All 8 check runs were green before push; build+tests pass locally after fixes.
- **Result:** 1 commit pushed to `feature/44-image-format-support`.
### PR #51 Review — Palette Documentation (2026-04-18) — COMPLETED

- **Review comments addressed:** Two items in `tools/swatches/main.go` — replaced hard-coded palette list with `palette.Names()` (new function), added directory existence guard in `writeSwatch`.
- **CI fix:** `revive` cognitive-complexity (11 > 10) and `wsl_v5` whitespace lint. Resolved by extracting `createSwatchImage` helper.
- **Key files:** `internal/palette/palette.go` (added `Names()` func), `tools/swatches/main.go`.
- **Pattern:** CI runs `task ci` inside devcontainer; local lint requires `golangci-lint-custom` with nilaway plugin. Use `go vet` locally as smoke check.
- **Palette package:** `palettes` map is source of truth. `Names()` returns sorted slice from it — single source of truth for tooling.
- **Result:** Build and tests pass. 2 commits pushed to `docs/palette-documentation`.

### Merge Conflict Resolution — PR #45 vs PR #51 (2026-04-18)

- **Cause:** PR #51 (palette documentation) merged into `main` while PR #45 (image format support) was still open. Both PRs touched `cmd/codeviz/treemap_cmd.go`. PR #51's `main` kept the old `render.RenderPNG` call; PR #45 had replaced it with the format-aware `render.Render`. Git couldn't auto-merge the overlapping hunk.
- **Conflict scope:** Single file (`cmd/codeviz/treemap_cmd.go`), single hunk — the render call inside `renderAndLog()`.
- **Resolution:** Kept PR #45's `render.Render` call (multi-format entrypoint) with its debug log line, discarding the stale `render.RenderPNG` reference from main. Both branches' intent preserved.
- **Verification:** `task build` and `task test` pass. All 14 packages green. Pushed merge commit to origin.
- **Process learning:** Always rebase PR branches onto latest main before pushing fixes, especially when multiple PRs are in flight touching related code. A quick `git fetch origin && git merge origin/main` (or rebase) before pushing would have caught this conflict locally instead of leaving the PR in a dirty state on GitHub.

### Issue #33 — Bubble Visualization Architecture (2026-07-15)

- **Task:** Research and architecture proposal for circle-packing bubble tree visualization.
- **Codebase pattern:** All visualizations follow the same structure: `internal/{viz}/node.go` (node type) + `layout.go` (Layout function) + `layout_test.go` (Gomega tests); renderer has `internal/render/{viz}.go` (PNG) + `svg_{viz}.go` (SVG); CLI has `cmd/codeviz/{viz}_cmd.go`; config has `internal/config/{viz}.go`.
- **Layout signatures:** Treemap takes `(root, width, height, sizeMetric)`, Radial takes `(root, canvasSize, discMetric, labels)`. Bubble proposed as `(root, width, height, sizeMetric, labels)`.
- **Node types:** Each viz has its own: `TreemapRectangle` (X/Y/W/H), `RadialNode` (X/Y/DiscRadius/Angle), proposed `BubbleNode` (X/Y/Radius).
- **Circle-packing algorithm:** Front-chain (Wang et al. 2006) + Welzl's enclosing circle. D3's `pack()` uses the same approach. No Go library exists; implement from scratch.
- **Reference implementation:** `githubocto/repo-visualizer` uses D3 `pack()` + custom force-simulation reflow (280 ticks). Force sim is polish, not essential for v1.
- **Renderer pattern:** Three-pass rendering (background → shapes → labels) for z-order correctness. Used by radial; bubble should follow same pattern.
- **Colour application:** Parallel walk of layout node tree + model.Directory tree, applying fill/border via bucket (numeric) or categorical mapping. Pattern duplicated per viz — acceptable for now.
- **Config pattern:** All pointer fields (nil = unset). `config.New()` sets defaults. CLI `applyOverrides()` only writes non-zero values.
- **Key files for implementation:** `render_cmd.go` (add `Bubbletree` subcommand), `config.go` (add `Bubbletree` field to `Config` struct and `New()`).
- **Proposal written to:** `.squad/decisions/inbox/ripley-bubble-architecture.md`
- **Result:** Architecture adopted. PR #64 created on branch `squad/33-bubble-visualization` with full implementation (layout engine, PNG+SVG rendering, CLI, config, 20 tests). CI green.

### PR #69 Review — Legend Feature (2026-04-19) — COMPLETED

- **Issue:** #68 — All visualizations should have a Legend
- **Branch:** `squad/68-legend-core` (consolidated from 5 phases)
- **Review finding:** One `unparam` lint issue — `writeSVGLegend` had an unused `x` parameter (always `0`). Fixed by removing the parameter from function signature and all 6 call sites (3 SVG renderers + 3 tests).
- **Architecture assessment:** Clean. Legend is a composable `*LegendInfo` struct — nil means no legend. Canvas height extension strategy (viz height + legend height) preserves all existing coordinate systems. Shared constants between PNG and SVG keep visual parity. `buildLegendRow` in cmd package correctly replicates the bucket/category computation from the colour-application functions.
- **Pattern:** Legend wiring follows the established pattern of building data structures in cmd, passing them to render functions. `--no-legend` uses `*bool` config field — consistent with the Kong pointer-field convention.
- **Key files:** `internal/render/legend.go`, `internal/render/svg_legend.go`, `cmd/codeviz/legend_builder.go`, all `*_cmd.go` files, all `config/*.go` files.
- **CI status:** Build, 15 test packages, and lint all green after fix.
- **Result:** PR #69 opened against main.

### Legend Phase 5 — Test Suite Complete (2026-04-19)

- **Status:** Lambert completed Phase 5 comprehensive test suite (47 tests across 3 files) for legend feature on squad/68-legend-core.
- **Validation:** All tests passing, build clean, lint clean. Validates renderer signatures (all accept `*LegendInfo`) and integration points from phases 2–4.
- **Readiness:** Test suite is comprehensive; ready for PR review and merge. No blockers identified.

### PR #98 Review — Legend Rendering Bugs #89, #90 (2026-04-26) — APPROVED

- **Issues:** #89 (horizontal legend too tall), #90 (orientation-aware margin carve-out)
- **Branch:** `squad/89-90-legend-fixes`, Author: Dallas
- **Correctness:** Both fixes verified correct. `measureLegendH` now sums entry widths (was stacking heights). `ReserveLegendSpace` corner positions now check orientation (vertical→carve width, horizontal→carve height). `legendLayoutOffset` mirrors the new carve-out logic.
- **Symmetry:** PNG (`drawLegendEntriesH`) and SVG (`writeSVGLegendEntriesH`) paths updated identically.
- **Architecture:** `measureSingleEntryH` helper cleanly shared between measurement and drawing. Fits existing legend patterns.
- **Minor suggestions:** 3 duplicate test pairs (updated old tests + new issue-specific tests test the same combos), stale "Currently fails" comments in new tests, TopRight corner not tested.
- **Key files:** `legend.go` (ReserveLegendSpace), `legend_png.go` (measureLegendH, measureSingleEntryH, drawLegendEntriesH), `legend_svg.go` (writeSVGLegendEntriesH), `treemap_cmd.go` (legendLayoutOffset), `legend_test.go`.
- **CI:** All tests pass, `go vet` clean. `golangci-lint-custom` only available in CI.
- **Result:** APPROVED with minor suggestions.

### Issue #107 — Design Review: Export Metrics Feature (2026-04-26)

- **Task:** Architectural decisions for `--export-data` CLI flag to export computed metrics (JSON/YAML).
- **Data structure:** Recursive `DirectoryExport` tree with flat `FileExport` leaves; metric maps use string keys (human-readable) to simplify JSON/YAML serialization. Preserves paths and binary flags for post-export analysis.
- **Package placement:** New `internal/export/` package (mirrors existing patterns: render, scan, config). Single `Export()` function independent of CLI, visualization type, and metric registry.
- **API signature:** `Export(root *model.Directory, requested []metric.Name, outputPath string) error`. Format inferred from file extension (like `render.FormatFromPath`).
- **Flag design:** `--export-data` added to `Flags` struct (not per-command), consistent with existing `--export-config` pattern. Enables cross-cutting export on any visualization command.
- **Metric visibility:** No new model methods. Export logic iterates through requested metric names and calls existing getters (`Quantity`, `Measure`, `Classification`). Only metrics actually requested are exported.
- **Integration point:** Export called after `provider.Run()` (metrics computed) but before render, following the established command flow in treemap_cmd.go.
- **Team ownership:** Dallas (export implementation), Kane (CLI wiring), Lambert (tests).
- **Output:** Design decisions written to `.squad/decisions/inbox/ripley-export-data-design.md`.

### Issue #107 — Export Feature Implementation Complete (2026-04-26)

- **Status:** Feature fully implemented and integrated. All team members completed their assigned work.
- **Dallas (Go Dev):** Implemented `internal/export/` package with recursive tree walking. Export() function handles JSON/YAML format inference, lazy-init metric maps, proper error handling with eris. Dependency added: gopkg.in/yaml.v3.
- **Kane (CLI Dev):** Wired `--export-data` flag into Flags struct and CLI struct. Updated all 3 command Run() methods (treemap, radial, bubbletree). Export called after provider.Run(), before render. Consistent integration pattern across all commands.
- **Lambert (QA):** Comprehensive test suite created: 9 tests covering JSON export, YAML export, format error handling, metric filtering, empty directories, nested structures, binary flags, and all metric types. All tests pass. Build green.
- **Integration:** Feature ready for deployment. Design decisions merged into decisions.md.

### Issue #127 — Spiral Visualization Architecture (2026-07-19)

- **Task:** Architecture proposal for time-based spiral visualization — a fundamentally different viz type from the existing tree-based ones.
- **Key departure:** Spiral visualizes a time series, not a file tree. Returns `[]SpiralNode` (flat slice), not a tree. Requires new time-bucketing infrastructure in `internal/spiral/` that doesn't extend the core model.
- **New infrastructure:** `TimeBucket` struct aggregates per-file metrics into time windows. `LoadCommitHistory()` reuses existing `repoService` from `internal/provider/git/` to fetch commit timestamps without adding a new metric provider.
- **Spiral geometry:** Archimedean spiral `r = a + b*θ` with inner radius = outer/3. Clockwise from north. Hourly = 24 spots/lap, Daily = 28 spots/lap.
- **Three metric destinations:** Disc size (numeric, bucket-mapped), fill (any kind), border (any kind) — all reuse existing colour pipeline. Values come from time-bucket aggregation instead of direct file metrics.
- **Aggregation semantics:** Numeric = sum, Categorical = mode. Empty buckets rendered as small grey dots to preserve temporal fidelity.
- **Label strategy:** Default LabelLaps (only at lap boundaries) because dense spiral makes full labelling unreadable.
- **Risks:** Git history performance on large repos (mitigated by --since/--until), very long time ranges producing dense spirals (mitigated by auto-resolution selection), aggregation semantics may need user feedback.
- **Proposal written to:** `.squad/decisions/inbox/ripley-spiral-architecture.md`
- **Key files for implementation:** `internal/spiral/` (new package), `internal/config/spiral.go`, `internal/render/spiral.go`, `internal/render/svg_spiral.go`, `cmd/codeviz/spiral_cmd.go`, `cmd/codeviz/render_cmd.go` (add subcommand), `internal/config/config.go` (add Spiral field).

### Issue #158 — Renderer Unification Architecture (2026-07-20)

- **Task:** Architecture proposal to unify raster/SVG rendering paths across 8 renderer files + 2 legend files.
- **Approach chosen:** Draw list (Option 2), not Strategy/Visitor. gg.Context is stateful (push/pop/stroke); SVG is declarative. An interface-based Canvas abstraction creates impedance mismatch. Draw list separates prepare (shared) from emit (backend-specific) cleanly.
- **Key insight:** Codebase already has proto-draw-lists — `collectDiscs`/`collectSVGDiscs` in radial, `collectBubbleDirs`/`collectBubbleFiles` in bubble. Unifying means making these shared, not inventing a new pattern.
- **Op types (8):** FillCircle, StrokeCircle, FillRect, StrokeRect, DrawLine, DrawText, DrawPath, DrawArcText. Closed set, type-switch in each emitter.
- **Legend system:** Deepest duplication (12 parallel function pairs, 682 lines). PrepareLegend replaces both legend_png.go and legend_svg.go. Text measurement stays on gg.NewContext(1,1) — already the case for SVG legend.
- **Ordering vs #152:** Do #158 first. It's a pure internal/render refactor with no cmd-layer changes. #152 (command workflow) benefits from unified render dispatch. Reverse order would require changing both layers.
- **Migration strategy:** 6 phases (draw list types → treemap → radial → bubble → spiral → legend), each a self-contained PR with golden test verification.
- **Estimated outcome:** ~600–800 lines eliminated, zero duplicated logic between PNG/SVG paths.
- **Proposal written to:** `.squad/decisions/inbox/ripley-renderer-unification-design.md`

### Issue #152 — Command Workflow Extraction Architecture (2026-07-20)

- **Task:** Architecture proposal to extract shared orchestration from 4 viz commands (2,576 lines total).
- **Analysis:** Categorized all duplication into 4 buckets: (1) identical functions (validatePaths, buildFilterRules, filterBinaryFiles, etc.), (2) same-pattern-different-config (mergeConfigAndValidate, applyOverrides, validateConfig, Run body), (3) structurally identical but different node types (colour tree-walks), (4) truly viz-specific (layout, render, spiral time-bucketing).
- **Key constraint:** Kong struct tags can't be shared via embedding when field names differ (Size vs DiscSize) or defaults differ (Height 1080 vs 1920). CLI structs must stay per-viz.
- **Proposed approach:** Three new files in cmd/codeviz/ — `workflow.go` (shared functions), `pipeline.go` (RunPipeline orchestration), `colour.go` (ColourDispatch for border/fill resolution). No new packages.
- **Spiral is the outlier:** Time-series, not tree-based. Shares scan+metrics pipeline but adds its own time-bucketing phase. Pipeline returns after export, spiral adds its own steps.
- **Colour application stays per-viz:** Treemap/radial use positional tree-walking (identical, can share). Bubble uses path-indexed maps. Spiral uses flat slices. Abstracting all four into one interface is over-engineering.
- **Estimated savings:** ~700-800 lines (2,576 → ~1,800). Remaining lines are genuinely viz-specific.
- **Phase ordering:** Phase 1 (workflow.go) → Phase 2 (pipeline.go) → Phase 3 (colour.go). Phase 4 (VizConfig interface) optional.
- **Relationship to #158:** Renderer unification (#158) should land first — it's a pure internal/render refactor. This proposal (#152) benefits from unified render dispatch.
- **Lint compliance:** All proposed files respect funlen ≤ 65 and max-public-structs ≤ 5.
- **Proposal written to:** `.squad/decisions/inbox/ripley-command-workflow-design.md`

### Canvas Stage 1 — Implementation Plan (2026-05-09)

- **Task:** Write comprehensive 20-task implementation plan for `internal/canvas/` package from the finalized design spec.
- **Spec:** `docs/superpowers/specs/2026-05-08-canvas-design.md` — retained-then-render Canvas with Ink colour resolution, Spec style templates, layered z-ordering, pluggable backends.
- **Plan structure:** 20 sequential tasks building from foundational types (MetricValue, InkOption, Ink) through geometry and specs, to Canvas core, backends (raster, SVG, mock), and integration tests.
- **Key design constraints respected:** `max-public-structs ≤ 5` per file (split types across files), `funlen ≤ 65` (small focused functions), `cognitive-complexity ≤ 10`.
- **Test approach:** Every implementation task writes tests first (red-green), using Gomega assertions, `t.Parallel()`, table-driven patterns. Mock backend enables Canvas unit tests without rendering. Golden-file snapshots for raster/SVG backend tests.
- **Stage 1 scope:** Dark-shipped package — zero changes to existing code. Full test coverage in isolation.
- **Plan saved to:** `docs/superpowers/plans/2026-05-09-canvas-stage1.md`

### Issue #196 — Directory Border Metric Color Fix (2026-07-21)

- **Task:** Fix directory nodes always rendering with wrong border metric color in bubbletree and radial.
- **Root cause:** Directory entries had no file to resolve metric values from, so `MetricValue{}` was passed to metric-based border inks, producing the minimum-value color instead of the default border color.
- **Fix pattern:** Use `canvas.FixedInk(defaultBorder)` for directory border ink, matching the existing pattern for directory fill ink. Both visualizations already had default border color constants.
- **Bubble (`bubble_canvas.go`):** Replaced `inks.border` with `canvas.FixedInk(bubbleDefaultBorder)` in `dirSpec`. Removed unused `inks` parameter from `addBubbleDirDiscs`.
- **Radial (`radial_canvas.go`):** Added `border` variable alongside `fill`, overridden to `canvas.FixedInk(radialDefaultBorder)` when `e.isDir`. Used `border` in `discSpec` instead of `inks.border`.
- **Key files:** `cmd/codeviz/bubble_canvas.go`, `cmd/codeviz/radial_canvas.go`.
- **Lint fixes:** Removed unused parameter (`revive`), added blank line before `if` block (`wsl_v5`).
- **Result:** PR #211 opened. All tests pass, zero lint issues.

### Issue #191 — Delete Zombie internal/render Package (2026-05-11)

- **Task:** Remove dead `internal/render/` package — orphaned after Canvas refactoring moved functionality to `internal/canvas/`.
- **Verification:** Confirmed zero Go imports of `internal/render` outside the package itself (grep across all `.go` files).
- **Deleted:** 5 source files (`renderer.go`, `format.go`, `label.go`, `format_test.go`, `label_test.go`) + 6 golden-file PNGs in `testdata/`. Total: 219 lines removed.
- **Dead code details:** `format.go` duplicated `canvas.ImageFormat`/`canvas.FormatFromPath`. `label.go` duplicated `canvas.TextColourFor` and had unused `ShouldShowLabel`. `renderer.go` was just a package comment.
- **Build/test/lint:** All clean. No breakage. Pre-existing lint issues in other packages unrelated.
- **Result:** PR #215 opened on branch `squad/191-delete-render-pkg`. Pure deletion, no behaviour change.

### Issue #195 — Remove Dead Canvas API Fields and Stubs (2026-05-11)

- **Task:** Remove dead exported fields, types, and initialisation code from `internal/canvas/` and related packages.
- **Removed 7 items:** (1) ShapeStyle.ShowLabel/.LabelInk/.LabelStyle fields, (2) LabelStyle type and 3 constants, (3) MappingStrategy type + Linear/Logarithmic constants + WithMapping() option, (4) Rectangle.Label and Disc.Label fields, (5) dead fill init in 3 buildInks functions, (6) TreemapRectangle.ShowLabel, (7) bubbleDirEntry.mv field.
- **Verification approach:** grep-verified each item has no callers outside its own definition (or tests for that dead code). MappingStrategy.strategy field was stored but never read — ink.go always uses quantile via metric.ComputeBuckets. Rectangle.Label/Disc.Label were set but never read by canvas backends. TreemapRectangle.ShowLabel was never assigned. bubbleDirEntry.mv was never populated despite being used as Border value.
- **PR #211 check (item 7):** PR #211 removed the `inks` parameter from `addBubbleDirDiscs` but did NOT remove the `mv` field from `bubbleDirEntry`. Confirmed the field is still dead and removed it.
- **Files changed:** 9 files, net -49 lines. All 22 test packages pass. Build clean.
- **Result:** PR #217 opened on branch `squad/195-remove-dead-canvas-api`.


### GitHub Issue #289 — Additional Go Metrics Analysis (2026-01-06)

- **Task:** Suggest additional Go-specific source code metrics beyond issue #289's base proposal (type/function/method/constant/variable counts).
- **Existing metrics:** filesystem: `file-size`, `file-lines`, `file-type`; git: `file-age`, `file-freshness`, `author-count`, `commit-count`, `total-lines-added`, `total-lines-removed`, `commit-density`.
- **Metric system:** Provider interface with Name/Kind/Description/Dependencies/DefaultPalette. Kinds: Quantity (int), Measure (float64), Classification (string).
- **Key insight:** Go complexity stems from coupling (imports), concurrency (goroutines), error handling, and structural patterns (interfaces/structs). Metrics should highlight maintenance burden and architectural boundaries.
- **High-priority suggestions:** `interface-count`, `struct-count` (complete type taxonomy), `import-count` (coupling), `cyclomatic-complexity` (maintenance burden), `test-coverage-indicator` (gap identification).
- **Go-specific idioms:** `goroutine-spawn-count`, `error-return-count`, `defer-count`, `dot-import-count`, `init-function-count`.
- **Feasibility:** Most metrics are straightforward AST/DST node counts. `github.com/dave/dst` preserves comments, enabling `comment-ratio`. Reference resolution (for cohesion) is v2 scope.
- **Visualization fit:** All suggested metrics are file-level aggregates suitable for treemap/radial rendering to highlight hotspots.
