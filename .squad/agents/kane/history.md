# Kane — History

## Core Context

- **Project:** A Go CLI tool that scans file trees and renders treemap visualizations as PNG images with configurable metrics and colour palettes.
- **Role:** CLI Dev
- **Joined:** 2026-04-14T09:49:33.772Z

## Learnings

<!-- Append learnings below -->

### RadialCmd structure and flags

- `RadialCmd` mirrors `TreemapCmd` but uses `DiscSize metric.Name` (short flag `-d`) instead of `Size` for the sizing metric.
- Additional `Labels string` flag with `default:"all"` and `enum:"all,folders,none"` controls label rendering via `radialtree.LabelMode`.
- `Width` and `Height` both default to 1920 (square canvas); canvas size is `min(width, height)`.
- All colour flags (Fill, FillPalette, Border, BorderPalette) are identical to `TreemapCmd`.

### How applyOverrides works for Radial config

- `applyOverrides` writes non-zero CLI flag values into `*config.Config`.
- Width/Height go to `cfg.Width`/`cfg.Height` (top-level); colour and labels fields go to `cfg.Radial.*`.
- A nil-guard `if cfg.Radial == nil { cfg.Radial = &config.Radial{} }` is needed because config may be loaded from file without a `radial:` section.
- Zero-valued CLI strings are transparent (config file values pass through unchanged).

### Bubble Tree — Architecture Proposal Ready (2026-04-19)

- **Issue #33:** Ripley completed architecture research for circle-packing bubble tree visualization.
- **Your role in Phase 4 (CLI + Config wiring):** Implement `BubbletreeCmd` struct in `cmd/codeviz/bubbletree_cmd.go` and update `RenderCmd` to register `Bubbletree` subcommand. Follow the radial/treemap pattern.
- **BubbletreeCmd fields:**
  - `TargetPath`, `Output` (required, format-aware like radial)
  - `Size metric.Name` (short `-s`, required, numeric metrics) — primary sizing metric
  - `Fill, FillPalette, Border, BorderPalette` (optional, identical to TreemapCmd/RadialCmd)
  - `Labels string` (enum: `all,folders,none`, default empty for config.New() to set)
  - `Width int` (default 1920), `Height int` (default 1080) — **non-square canvas, unlike radial** 
  - `Filter []string` (repeatable glob rules)
- **Default dimensions:** 1920×1080 (like treemap, not square like radial) — bubble layout adapts to non-square
- **Config defaults:** Handled in `config.New()` → `Labels = "folders"` (file dots unlabelled by design)
- **Colour application:** Parallel walk of BubbleNode tree + model.Directory tree (like radial). Separate implementation v1, consider extraction later.
- **Key files to create/update:** `cmd/codeviz/bubbletree_cmd.go` (new), `cmd/codeviz/render_cmd.go` (register subcommand), `internal/config/bubbletree.go` (new, with Bubbletree struct), `internal/config/config.go` (add Bubbletree field, update New()).
- **Dependency:** Awaits Phase 1 (Dallas) — `internal/bubbletree/layout.go` must exist before you can call Layout() in Run().

### Bubble Tree — Phase 4 CLI + Config Complete (2026-04-19)

- **Created `cmd/codeviz/bubbletree_cmd.go`:** Full BubbletreeCmd with Validate, Run, applyOverrides, validatePaths, buildFilterRules, checkGitRequirement, filterBinaryFiles, resolveLabels, and all colour application functions (numeric + categorical for both fill and border).
- **Created `internal/config/bubbletree.go`:** Bubbletree config struct with pointer fields (Fill, FillPalette, Border, BorderPalette, Labels).
- **Updated `internal/config/config.go`:** Added Bubbletree field to Config struct and initialized in New() with Labels default "folders".
- **Updated `cmd/codeviz/render_cmd.go`:** Registered BubbletreeCmd as `bubbletree` subcommand.
- **Key pattern differences from radial:** Size flag is `--size/-s` (not `--disc-size/-d`); default 1920×1080 (not square); Layout takes width+height (not canvasSize); LabelMode defaults to "folders" (not "all").
- **Won't compile yet:** Imports `internal/bubbletree` (Layout, BubbleNode, LabelMode) and `render.RenderBubble` — Dallas is building these in parallel.

### Validate() vs Run() ordering with Kong (Issue #99)

- **Problem:** Kong calls `Validate()` before `Run()`, but config file loading happens inside `Run()`. Size fields that come from config are empty at `Validate()` time, causing false "unknown metric" errors.
- **Fix pattern:** `Validate()` now only checks CLI-only concerns (filter glob syntax). A new `validateEffective()` method runs inside `Run()` after `TryAutoLoad()` + `applyOverrides()` + config backfill, handling all config-dependent validation (size metric, fill/border metric-palette, border-palette-requires-border).
- **Kong struct tag requirement:** Enum fields without `required:"true"` need both `default:""` and a leading comma in the enum list (e.g., `enum:",file-size,..."`) to accept empty values.
- **Applied to:** `TreemapCmd`, `RadialCmd`, `BubbletreeCmd` — all three have the same structural pattern.
- **Key files:** `cmd/codeviz/treemap_cmd.go`, `cmd/codeviz/radialtree_cmd.go`, `cmd/codeviz/bubbletree_cmd.go`.

### Export Data CLI Flag — Issue #107 (2026-04-26)

- **Added `--export-data` flag** to `CLI` struct and `Flags` struct in `cmd/codeviz/main.go`, following the `--export-config` pattern.
- **Wired `export.Export()` call** into all three visualization commands (`TreemapCmd`, `RadialCmd`, `BubbletreeCmd`).
- **Placement:** After `filterBinaryFiles()` and before render/layout, matching the design spec (after metrics computed, before rendering).
- **Import:** `github.com/bevan/code-visualizer/internal/export` added to all three command files.
- **Won't compile yet:** Depends on Dallas's `internal/export/` package (parallel work).
- **Error message:** Uses `"failed to export data"` consistently across all three commands.

### Issue #107 — CLI Integration Complete (2026-04-26)

- **Added to Flags struct:** ExportData string field (consistent with ExportConfig pattern).
- **Added to CLI struct:** ExportData string field with Kong tag `help:"Write computed metrics to file (.json or .yaml/.yml)." name:"export-data" optional:""`.
- **Updated all 3 commands:** treemap_cmd.go, radial_cmd.go, bubbletree_cmd.go each check flags.ExportData after provider.Run() and call export.Export() with requested metrics.
- **Integration pattern:** Consistent across all commands — collect requested metrics, call export after metric computation, before render.
- **Build status:** Passes. All three commands wired correctly.
- **Flag design rationale:** Cross-cutting flag on Flags struct allows any visualization command to export metrics without duplication.

### MetricSpec — Combined metric+palette CLI parameter (Issue #118, 2026-07-06)

- **New type `config.MetricSpec`** (`internal/config/metric_spec.go`): Bundles metric name and palette name. Parsed from "metric,palette" or just "metric" format.
- **Kong integration:** Implements `encoding.TextUnmarshaler` — Kong automatically calls `UnmarshalText` for CLI parsing. No custom mapper needed.
- **Config serialization:** Custom `MarshalYAML`/`UnmarshalYAML` and `MarshalJSON`/`UnmarshalJSON` for config file support.
- **CLI struct changes:** `Fill` and `Border` fields changed from `string` to `config.MetricSpec`. Removed separate `FillPalette` and `BorderPalette` fields from all three commands.
- **Config struct changes:** `Treemap`, `Radial`, `Bubbletree` now use `*MetricSpec` for Fill and Border instead of separate `*string` fields.
- **Helper functions:** `specMetric(s *MetricSpec) string` and `specPalette(s *MetricSpec) string` replace `ptrString` for MetricSpec access.
- **Validation:** `--border-palette requires --border` check removed (palette is always bundled with metric). Validation uses `validateMetricPalette()` with extracted metric/palette strings.
- **Lint:** Used `//nolint:recvcheck` on MetricSpec struct because marshal methods need value receivers while unmarshal methods need pointer receivers.
- **Key files:** `internal/config/metric_spec.go`, `internal/config/metric_spec_test.go`, all three `*_cmd.go` files, config structs.
- **PR:** #120

### Spiral Visualization — Phase 3 CLI + Config Ready (2026-04-29)

- **Ripley (Architect) delivered comprehensive proposal** for Issue #127 spiral visualization. See `.squad/decisions.md` → "Spiral Visualization — Architecture Proposal" for full details.
- **Lambert (Tester) delivered 50 test specs** in `.squad/agents/lambert/spiral-test-spec.md` covering time-series input validation, bucket aggregation, angular spacing, geometry, disc sizing, label modes, empty buckets, colour mapping, rendering, and CLI integration.
- **Your Phase 3 task (CLI + Config):** Build spiral command and config integration:
  - `cmd/codeviz/spiral_cmd.go` (new): `SpiralCmd` struct with fields TargetPath, Output, Resolution (enum: "", "hourly", "daily"), Size (metric.Name), Fill/Border (MetricSpec), Labels (enum: "", "all", "laps", "none"), Width/Height (default 1920×1920), Filter. Implement Validate(), Run(), applyOverrides(), resolveLabels(), and colour application (numeric + categorical for both fill and border). Run() flow: scan → load config → apply overrides → load metrics → **build time buckets from git history** → call spiral.Layout() → apply disc sizes from metrics → apply fill/border colours → render.
  - `internal/config/spiral.go` (new): `Spiral` struct with pointer fields Resolution, Size, Fill, Border, Labels, Legend, LegendOrientation (matching existing config pattern).
  - `cmd/codeviz/render_cmd.go`: Add `Spiral SpiralCmd` field to `RenderCmd`, register as `spiral` subcommand.
  - `internal/config/config.go`: Add `Spiral *Spiral` field to Config struct, set defaults in New(): `Resolution: "daily"`, `Labels: "laps"`, `Legend: true`.
- **Key differences from Radial/Treemap:**
  - Time-series input instead of file tree (will use Dallas's `spiral.Layout()` and time-bucketing infrastructure)
  - No square-canvas constraint (1920×1920 default, but flexible)
  - Three metric destinations (size/fill/border) all already supported by existing metric pipeline
  - Default size metric = commit count (not required to specify)
- **Metric aggregation:** In Run(), after building time buckets from git history, apply metric aggregation: sum for numeric metrics (Quantity/Measure), mode for categorical. This happens before colour application.
- **Test integration:** Lambert will write 50 Go tests once your CLI command is wired. Tests will use Gomega assertions and Goldie snapshots.
- **Dependency:** Awaits Phase 1 (Dallas) — `internal/spiral/` package must exist before you can call Layout() and time-bucketing functions.
- **Next phases:** Phase 2 (Dallas) renders to PNG/SVG. Phase 4 (Lambert) writes Go test suite. Phase 5 (Bishop) polishes visuals.

### Spiral Visualization — Phase 3 Implementation Complete

- **Created `internal/config/spiral.go`:** Spiral config struct with pointer fields: Resolution, Size (`*string`), Fill/Border (`*MetricSpec` — consistent with post-#118 convention, not separate FillPalette/BorderPalette), Labels, Legend, LegendOrientation.
- **Updated `internal/config/config.go`:** Added `Spiral *Spiral` field to Config struct. Initialized in `New()` with `Resolution: new("daily")`, `Labels: new("laps")`.
- **Created `cmd/codeviz/spiral_cmd.go`:** Full SpiralCmd with Kong struct, Validate, Run, mergeConfigAndValidate, applyOverrides, validatePaths, buildFilterRules, checkGitRepo, collectSpiralMetrics, resolveResolution, resolveLabels, resolveFillMetric, resolveFillPalette, filterBinaryFiles, scanAndRunProviders, buildTimeBuckets, aggregateBucketMetrics, layoutAndRender, logRendered, applyFill, applyBorder, and all helper functions.
- **Updated `cmd/codeviz/render_cmd.go`:** Registered `SpiralCmd` as `spiral` subcommand.
- **Key architectural differences from tree-based visualizations:**
  - Spiral always requires git (checkGitRepo replaces per-metric git check).
  - Size metric is optional — default disc size = commit count per bucket (`len(b.Files)`).
  - Run() has extra steps: build time buckets from git history → aggregate per-file metrics into buckets → layout and render.
  - Colour application iterates flat slices (not parallel tree walk). Much simpler.
  - New functions: `aggregateBucketMetrics`, `aggregateColourMetric`, `sumNumericMetric`, `modeCategory`, `commitTimeRange`, `assignFilesToBuckets`, `applySpiralDiscSizes`, `collectBucketCategories`.
  - `applySpiralDiscSizes` scales layout-assigned disc radii by sqrt(sizeValue/maxSize) for area-proportional discs.
- **API adjustment from architecture doc:** `spiral.BuildTimeBuckets` actual signature is `(resolution, startTime, endTime)` without `root` param. The file assignment is handled separately by `assignFilesToBuckets` using CommitRecords.
- **Expected compile errors (4):** `spiral.LoadCommitHistory`, `spiral.CommitRecord` (Dallas Phase 1 — githistory.go), `render.RenderSpiral` (Dallas Phase 2). All config and non-cmd packages compile and pass tests.
- **Default canvas:** 1920×1920 (square, not 1920×1080) per architecture spec for spiral geometry.


## 2026-04-29 — Spiral Phase 1 CLI

**Completed:** Spiral config and CLI command scaffold
- `internal/config/spiral.go` with *MetricSpec for Fill/Border
- `cmd/codeviz/spiral_cmd.go` with CLI flags and command struct
- Updated command registration and wiring

**Key decision:**
- *MetricSpec for Fill/Border (not separate fields) — consistent with Treemap/Radial/Bubbletree post-issue #118

**Expected compile errors:** 4 (layout, provider, result type definitions)

**Next:** Bind Dallas's layout output to CLI result. Provider integration for file → bucket → node → position flow.


### Bug #141 — Treemap Borders Too Narrow (2026-05-02)

- **Status:** Research complete, ready for implementation.
- **Scope:** Render-layer only. Four locations across two files: `internal/render/renderer.go` (drawFileRect, drawDirectoryHeader) and `internal/render/svg_treemap.go` (writeSVGFileRect, writeSVGDirectoryHeader).
- **Root cause:** Hardcoded `SetLineWidth(1)` and `stroke-width="1"` for all rectangles.
- **Solution:** Implement `treemapBorderWidth(w, h, hasMetric)` helper. Logic: no-metric→0.5px hairline, small (<20px)→1.0px, large (>100px)→3.0px, else→2.0px. Thresholds are suggestions, tune visually.
- **No layout changes needed** — purely render-time decision based on existing rectangle geometry.
- **Implementation note:** Update both PNG and SVG renderers in parallel for parity.

### Bug #139 — Spiral Visualization Issues (2026-05-02)

- **Status:** Research complete. Three sub-issues with distinct root causes.
- **Sub-issue 1 (Empty dots):** `applySpiralDiscSizes()` sets `emptyBucketRadius=2.0` (cmd/codeviz/spiral_cmd.go:555-556). Fix: Set radius to 0 for empty buckets.
- **Sub-issue 2 (Disc size constraints):** `computeMaxDisc()` exists (internal/spiral/layout.go:89-106) but not exposed. Fix: Expose maxDisc via API, apply min/max clamping in CLI scaling (cmd/codeviz/spiral_cmd.go).
- **Sub-issue 3 (Border width):** `drawSingleSpot()` hardcodes 1px (internal/render/spiral.go:175). Fix: Implement `spiralBorderWidth(discRadius)` helper — threshold 8px: return 2.0, else 3.0.
- **Routing:** Can split across two PRs: PR1 (sub-issues 1&3, render+CLI), PR2 (sub-issue 2, layout API+CLI).
