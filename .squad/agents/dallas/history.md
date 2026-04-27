# Dallas — History

## Core Context

- **Project:** A Go CLI tool that scans file trees and renders treemap visualizations as PNG images with configurable metrics and colour palettes.
- **Role:** Go Dev
- **Joined:** 2026-04-14T09:49:33.771Z

## Recent Work

- **Issue #114 — File freshness fix (2026-04-27):** PR #119 (draft) implements TREESAME check for go-git FileName bug.
- **Issue #107 — Export package (2026-04-26):** Created `internal/export/` with Export() function for JSON/YAML metrics export.
- **Issue #118 — MetricSpec integration (2026-04-27):** Noted Kane's MetricSpec type changes affecting metric validation.
- **PR #98 — Legend fixes (#89, #90) (2026-04-18)**: Fixed horizontal legend layout and orientation-aware carve-out. Branch squad/89-90-legend-fixes.
- **PR #39 resolution (2026-04-15)**: Fixed FolderAuthorCountProvider dependencies, mean-file-lines description, git error logging. Branch squad/38-extend-provider-capabilities.

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

