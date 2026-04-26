# Ripley — History

## Core Context

- **Project:** A Go CLI tool that scans file trees and renders treemap visualizations as PNG images with configurable metrics and colour palettes.
- **Role:** Lead
- **Joined:** 2026-04-14T09:49:33.769Z

## Learnings

<!-- Append learnings below -->

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

