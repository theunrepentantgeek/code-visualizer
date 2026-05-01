# Dallas — History

## Core Context

- **Project:** A Go CLI tool that scans file trees and renders treemap visualizations as PNG images with configurable metrics and colour palettes.
- **Role:** Go Dev
- **Joined:** 2026-04-14T09:49:33.771Z

## Recent Work

- **Issue #121 — Metric scanning logs fix (2026-04-27):** PR #122 (draft) standardized log messages, added metric name context, implemented progress tracking. Branch squad/121-fix-metric-scanning-logs.
- **Issue #114 — File freshness fix (2026-04-27):** PR #119 (draft) implements TREESAME check for go-git FileName bug.
- **Issue #107 — Export package (2026-04-26):** Created `internal/export/` with Export() function for JSON/YAML metrics export.
- **Issue #118 — MetricSpec integration (2026-04-27):** Noted Kane's MetricSpec type changes affecting metric validation.
- **PR #98 — Legend fixes (#89, #90) (2026-04-18)**: Fixed horizontal legend layout and orientation-aware carve-out. Branch squad/89-90-legend-fixes.

## Core Context

This is an accumulation of foundational learnings and architecture decisions from early project phases (2026-04-14 through 2026-04-20) that remain relevant for ongoing feature work:

- **Radialtree architecture (2026-04-15):** Package in `internal/radialtree` with `RadialNode` struct (X, Y, DiscRadius, Angle, Label, ShowLabel, IsDirectory, FillColour, BorderColour, Children), `LabelMode` constants (All, FoldersOnly, None), and `Layout()` function. Nodes positioned in rings at increasing depths; disc sizes scaled by metric value; children output files-first, then subdirectories.

- **Bubbletree layout engine (2026-04-19):** Package in `internal/bubbletree` with `BubbleNode` struct (X, Y, Radius, Label, ShowLabel, IsDirectory, FillColour, BorderColour, Path, Children) and `Layout()` function. Uses front-chain circle-packing with Welzl's enclosing-circle algorithm. Bottom-up packing, top-down coordinate assignment. Signature: `Layout(root *model.Directory, width, height int, sizeMetric metric.Name, labels LabelMode) BubbleNode`.

- **Bubbletree rendering (2026-04-19):** PNG/SVG rendering in `internal/render/bubbletree.go` and `internal/render/svg_bubble.go`. Three-pass z-order: directories (transparent fill), files, labels. Directory transparency ~18% alpha. Path field on BubbleNode (populated during layout) enables colour mapping after sorting.

- **Legend rendering (2026-04-18):** Code split across `legend.go` (types, constants), `legend_png.go` (PNG), `legend_svg.go` (SVG), `legend_test.go`. Supports both vertical and horizontal orientations with orientation-aware carve-out at corner positions. Horizontal layout increments X; vertical increments Y. Always verify outer loop handles orientation when adding new rendering features.

- **Folder metrics helpers (2026-04-15):** Five high-level helpers in `internal/metric/metrics.go` (`loadMaxQuantity`, `loadMinQuantity`, `loadSumQuantity`, `loadMeanMeasure`, `loadPositiveMeanMeasure`) encapsulate `WalkDirectories` loop for all folder provider Load() methods. Use integer duration arithmetic (not float hours/days) to avoid precision loss on long durations.

- **Kong + config defaults pattern (2026-04-15):** Never use `default:` tags on Kong fields — Kong silently overrides config file values. Instead, add `""` to enum lists and set actual defaults in `config.New()`. This ensures CLI and config file values can both be respected.

- **Index-based tree correlation bug (2026-04-20):** When two independent trees (BubbleNode + model.Directory) are sorted/reordered differently, index-based pairing fails. Use direct references (paths, pointers) instead. This applies whenever one tree is reordered for optimization (e.g., packing density) but must stay synchronized with colour mapping from another tree.

## Learnings

<!-- Append learnings below -->

### Metric scanning logs — Issue #121 (2026-04-28)

- **Progress architecture:** `cmd/codeviz/progress.go` contains all verbose/debug progress tracking. Two phases: filesystem scan (`scanCounter` + `startScanTicker`) and metric calculation (`metricProgressTracker` + `startMetricTicker`). Both use 1-second ticker goroutines with channel-based stop.
- **Bug root cause:** The scan ticker was deferred until function return, so it kept firing during metric loading with frozen file/dir counts. Fix: explicit `stopScanTicker()` call after `scan.Scan()` returns, before metric phase.
- **Concurrency note:** `metricProgressTracker` uses `sync.Mutex` for the active-metric slice (not atomic — slice ops aren't atomic) plus `atomic.Int64` for the completed counter. Providers run concurrently via `errgroup` in `internal/provider/run.go`, so `OnMetricStarted`/`OnMetricFinished` are called from multiple goroutines.
- **Name collision:** `formatMetricNames` already exists in `treemap_cmd.go` (returns all available metrics). Named the progress helper `joinMetricNames` to avoid collision within the `main` package.
- **PR:** #122 (draft), branch `squad/121-fix-metric-scanning-logs`.

### Per-file progress tracking — Issue #121 follow-up (2026-04-28)

- **Optional interface pattern:** Used `FileProgressReporter` interface (with `SetOnFileProcessed(fn func())`) to thread per-file callbacks into providers without changing the core `Interface.Load()` signature. `runProvider` checks for this via type assertion before calling Load.
- **sync.Map for hot-path counters:** Per-metric file counts use `sync.Map` (metric.Name → `*atomic.Int64`) because the write pattern is few stores in `OnMetricStarted` with many concurrent loads in `OnFileProcessed`. Cheaper than mutex for this read-heavy pattern.
- **Callback placement:** `defer onFile()` at the top of each WalkFiles callback ensures every file increments progress regardless of early returns (errors, binary skips).
- **FileLinesProvider receiver change:** Changed from value receiver to pointer receiver for `Load` and added `SetOnFileProcessed`. Registration changed from `FileLinesProvider{}` to `&FileLinesProvider{}`. Value receiver methods still satisfy the interface through the pointer.
- **Removed `joinMetricNames`:** The ticker now logs one line per active metric instead of joining all names into one line. This made `joinMetricNames` unused — lint caught it.

### MetricSpec type — Issue #118 (2026-04-27)

- **CLI integration with Kane:** Kane's new `config.MetricSpec` type bundles metric+palette into single parameters (`--fill metric,palette`). This affects all metric validation in provider work — when checking metric names against `metric.Provider`, extract via `specMetric()` helper.
- **Config struct impact:** All visualization config structs (`Treemap`, `Radial`, `Bubbletree`) now use `*MetricSpec` for Fill/Border fields instead of separate palette pointers. Check any metric validation code that reads these config fields.
- **Helper functions:** Use `specMetric(spec)` and `specPalette(spec)` instead of `ptrString()` to safely extract metric/palette values.

### go-git FileName filter bug — issue #114 (2026-04-27)

- **Root cause:** go-git's `LogOptions{FileName: &path}` includes merge commits that didn't modify the file, inflating the commit set. This made `data.newest` reflect the repo's most recent commit, not the file's last change.
- **Fix location:** `internal/provider/git/service.go` — added `commitModifiedFile()` and `blobHash()` to verify each commit by comparing blob hashes with the first parent's tree.
- **Impact:** Affected all three git metrics (file-age, file-freshness, author-count) since they share `fetchCommitData`. file-freshness was most visibly broken because `newest` was always near-now, truncating to 0 days.
- **Key insight:** `file-age` appeared correct only by coincidence — for files present since the initial commit, `oldest` happened to match the repo's oldest commit.
- **PR:** #119 (draft), branch `squad/114-fix-file-freshness`.

### Export package — issue #107 (2026-04-26)

- **New package:** `internal/export/` — serializes model tree + computed metrics to JSON or YAML files.
- **Public API:** `Export(root *model.Directory, requested []metric.Name, outputPath string) error` — walks tree recursively, collects only requested metrics, infers format from file extension.
- **Data structures:** `ExportData` (wrapper with `Root`), `DirectoryExport` (recursive), `FileExport` (leaf) — all carry `json` + `yaml` struct tags with `omitempty` on collection fields.
- **Format inference:** `.json` → `json.MarshalIndent` (2-space indent + trailing newline), `.yaml`/`.yml` → `gopkg.in/yaml.v3`. Unsupported/missing extensions return descriptive eris errors.
- **Metric collection pattern:** Lazy-init maps — only allocate `Quantities`/`Measures`/`Classifications` maps when a metric value is actually present. Combined with `omitempty`, this keeps output clean.
- **Dependency added:** `gopkg.in/yaml.v3` (direct dependency in go.mod).
- **No tests created** — Lambert owns the test suite. No CLI wiring — Kane handles that.

### Foliage palette — issue #46 (2026-04-18)

- **New palette added:** `Foliage` (`"foliage"`) in `internal/palette/palette.go` — 11-step ordered palette progressing black → brown → orange → yellow → green for plant-health visualisation.
- **Pattern:** Adding a palette requires changes in four places: const block, `validPalettes` map, `palettes` map, and the `ColourPalette` var definition. Tests need updating in `palette_test.go` for both `IsValid` and the WCAG contrast loop.
- **WCAG constraint:** All ordered palettes must pass adjacent-step contrast ratio >= 1.0 (checked in `TestWCAGContrastRatio`). The foliage colours were chosen with sufficient luminance steps to satisfy this.
- **Environment note:** `gofumpt` and `golangci-lint-custom` aren't available in this shell; rely on `task test` for validation.

### Curved bubble labels — issue #65 (2026-04-20)

- **New file:** `internal/render/bubble_font.go` — shared TrueType font helpers for bubble tree arc labels.
- **Font dependency:** `golang.org/x/image/font/gofont/goregular` provides embedded TrueType font; `github.com/golang/freetype/truetype` parses it. Both were already indirect deps, now promoted to direct.
- **Arc sizing algorithm:** `computeArcFontSize` iteratively shrinks font until text width ≤ `arcRadius × π/2` (90° arc constraint). Returns 0 to hide labels below 7px minimum.
- **Glyph positioning:** `computeGlyphPositions` uses `font.Face.GlyphAdvance` + `Kern` for per-character widths, placing each glyph at the midpoint of its angular span on the arc. Arc centred at top of circle (-π/2 radians).
- **PNG rendering:** Each glyph drawn individually with `dc.Push()/RotateAbout(angle + π/2, gx, gy)/DrawStringAnchored/Pop()`. Must call `dc.SetFontFace(face)` before drawing.
- **SVG rendering:** Uses `<defs>` block with `<path>` arcs (top semicircle, left to right), then `<textPath href="#arc-{idx}" startOffset="50%" text-anchor="middle">`. Traversal indices for path IDs, not node paths.
- **Lint note:** `fixed.Int26_6` is the return type for `font.Face.GlyphAdvance` and `Kern`, not `font.MeasureUnit`. Accumulate in fixed-point then convert to float64 at the end.
- **Cognitive complexity:** Split arc computation into small helpers (`clampFontToArc`, `measureStringWidth`, `collectAdvances`, `sumAdvances`, `placeGlyphs`) to stay under revive's max-10 limit.

### Funlen limits in cmd Run methods (2026-04-28)

- **funlen config:** `lines: 65, ignore-comments: true` in `.golangci.yml`. The line count includes blanks but excludes the func signature and closing brace. Comments are free.
- **Pattern:** The three `Run()` methods in `bubbletree_cmd.go`, `radialtree_cmd.go`, and `treemap_cmd.go` follow an identical scan-and-load pipeline. Use inline `if err := ...; err != nil` (not separate `err = ...` / `if err != nil`) to stay under the limit — each combined check saves one line.
- **dupl interaction:** Matching the inline style across cmd types makes their Run methods token-identical, triggering the `dupl` linter. Suppressed with `//nolint:dupl` since the methods genuinely operate on different config types and can't easily be unified without adding interface complexity.

### Spiral Visualization — Architecture Ready for Phase 1 (2026-04-29)

- **Ripley (Architect) delivered comprehensive proposal** for Issue #127 spiral visualization. See `.squad/decisions.md` → "Spiral Visualization — Architecture Proposal" for full details.
- **Your Phase 1 task:** Build `internal/spiral/` package:
  - `node.go`: `SpiralNode` struct with X, Y, DiscRadius, Angle, SpiralRadius, TimeStart, TimeEnd, Label, ShowLabel, FillColour, BorderColour fields. Fundamentally different from tree nodes — no Children field.
  - `layout.go`: `Resolution` enum (Hourly/Daily), `LabelMode` enum (All/Laps/None), and `Layout(buckets []TimeBucket, width, height, resolution, labels) []SpiralNode` function. Implements Archimedean spiral with inner diameter = 1/3 outer. Returns flat list, not tree. Angular spacing uniform.
  - `timebucket.go`: `TimeBucket` struct (Start, End, Files, SizeValue, FillValue, FillLabel, BorderValue, BorderLabel) and `BuildTimeBuckets(root, resolution, startTime, endTime) []TimeBucket`. Aggregates file metrics into time buckets.
  - `githistory.go`: `CommitRecord` struct and `LoadCommitHistory(root) []CommitRecord` function. Fetches commit timestamps from git service for time-bucket assignment.
  - `layout_test.go`: 8+ tests for spiral geometry, angular distribution, radius progression, edge cases, boundary handling.
- **Key design decisions (Ripley D1–D8):**
  - D1: Flat node list (no tree) — Layout returns `[]SpiralNode`
  - D2: Clockwise from north angle (matches clock reading)
  - D3: Inner diameter = 1/3 outer (from spec)
  - D4: Spots-per-lap fixed by resolution (24 hourly, 28 daily)
  - D5: Three metric destinations (size/fill/border) reuse existing metric pipeline
  - D6: Default size metric = commit count
  - D7: Empty buckets render as grey dots (preserve time axis)
  - D8: LabelLaps is v1 default (LabelAll too crowded)
- **Test coverage (Lambert):** 50 test specs ready in `.squad/agents/lambert/spiral-test-spec.md`. Once your layout signature is final, Lambert will convert to Go tests with Gomega assertions.
- **Risks & mitigations:** Git history perf (caching), time zone boundaries (careful interval logic), dense spirals (auto-resolution hints), aggregation semantics (pragmatic sum/mode, historic metrics v2+).
- **Next:** Phase 2 is rendering (`internal/render/spiral.go` + SVG). Phase 3 (Kane) is CLI command and config. Phase 4 (Lambert) is tests. Phase 5 (Bishop) is polish.

### Spiral Phase 1 — Core package built (2026-07-19)

- **Package delivered:** `internal/spiral/` with four files:
  - `node.go`: `SpiralNode` struct, `LabelMode` type (string-based: `"all"`, `"laps"`, `"none"`).
  - `timebucket.go`: `Resolution` type (Hourly=24/lap, Daily=28/lap), `TimeBucket` struct with aggregated metric fields, `BuildTimeBuckets()` function with start-time truncation.
  - `layout.go`: `Layout()` function implementing Archimedean spiral (`r = a + b*θ`), clockwise from north, inner/outer ratio 1:3, uniform angular spacing. Functions kept small (all under funlen 65).
  - `layout_test.go` + `timebucket_test.go`: 28 tests total — covers node count, monotonic radius, inner/outer ratio, spots per lap, uniform spacing, clockwise-from-north, all three label modes, edge cases (0/1/exact/partial lap), rectangular canvas, time field preservation, label formatting, bucketing logic.
- **Design choices:**
  - `LabelMode` is `string`-based (matching radialtree/bubbletree pattern), not `int`-based as in Ripley's proposal. Easier for Kong enum tags.
  - `Resolution` is `int`-based (not exported as string) since it's internal to the layout engine.
  - `BuildTimeBuckets` does NOT take `*model.Directory` — it only needs start/end times and resolution. File assignment to buckets is the CLI layer's job (per the architecture: layout only positions).
  - `computeTotalAngle` uses `n-1` (not `n`) since angles are 0-indexed; bucket 0 is at θ=0.
  - Disc radius defaults to a small fixed value; the CLI layer overrides from the size metric.
- **Key file paths:** `internal/spiral/node.go`, `internal/spiral/timebucket.go`, `internal/spiral/layout.go`, `internal/spiral/layout_test.go`, `internal/spiral/timebucket_test.go`.


## 2026-04-29 — Spiral Phase 1

**Completed:** Spiral layout engine (node.go, timebucket.go, layout.go)
- SpiralNode struct with position and label mode
- TimeBucket for grouping files by time ranges
- BuildTimeBuckets for creating n buckets over time range
- Layout function computing spiral positions (angle, radius, depth)
- 28 tests passing (19 layout, 9 bucket tests)

**Key decisions:**
- LabelMode string-typed ("all", "laps", "none") for Kong integration
- BuildTimeBuckets drops *model.Directory param — CLI layer handles binding
- n-1 angle spacing for 0-indexed bucket positioning
- Resolution internal-only (Hourly/Daily iota, CLI maps strings)

**Next:** Await Kane's CLI integration for full pipeline.

### Spiral Phase 1b + Phase 2 — git history & renderer (2026-07-08)

- **Git history loader:** `internal/spiral/githistory.go` — `CommitRecord` struct (FilePath, Timestamp, File pointer) and `LoadCommitHistory(root)` function. Walks model tree, calls into git package for per-file commit timestamps.
- **Git package extension:** Added `FileCommitTimestamps()` and `RepoRootFor()` exports to `internal/provider/git/service.go`. Uses the same TREESAME filtering as metric providers via `fetchCommitTimestamps()` on `repoService`.
- **Spiral PNG renderer:** `internal/render/spiral.go` — three-pass rendering: guide track curve (Archimedean spiral reconstruction), discs (fill + border), labels (tangent-oriented with upright flipping). Uses `inferTrackParams()` to reconstruct spiral geometry from positioned nodes.
- **Spiral SVG renderer:** `internal/render/svg_spiral.go` — SVG `<path>` for guide curve, `<circle>` elements for spots, rotated `<text>` for labels. Reuses `colourToHex()`, `writeSVGTextRotated()` from shared helpers.
- **Key pattern:** For flat-sequence visualizations, the renderer takes a `[]SpiralNode` slice (not a tree root). No recursive traversal needed — simple range loops over the slice for all three passes.
- **Label orientation:** Spiral labels use clockwise-from-north angle convention (matching layout). Half-plane check at π (not π/2) because the spiral coordinate system differs from the radial tree's east-based system.
