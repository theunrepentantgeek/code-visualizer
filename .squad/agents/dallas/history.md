# Dallas — History

## Core Context

- **Project:** A Go CLI tool that scans file trees and renders treemap visualizations as PNG images with configurable metrics and colour palettes.
- **Role:** Go Dev
- **Joined:** 2026-04-14T09:49:33.771Z

## Recent Work

- **PR #39 resolution (2026-04-15)**: Fixed FolderAuthorCountProvider dependencies, mean-file-lines description, git error logging. Extracted folder load helpers. Fixed float-to-days conversion and test error handling. Branch squad/38-extend-provider-capabilities task ci passed and pushed.

## Learnings

<!-- Append learnings below -->

### Curved bubble labels — issue #65 (2026-04-20)

- **New file:** `internal/render/bubble_font.go` — shared TrueType font helpers for bubble tree arc labels.
- **Font dependency:** `golang.org/x/image/font/gofont/goregular` provides embedded TrueType font; `github.com/golang/freetype/truetype` parses it. Both were already indirect deps, now promoted to direct.
- **Arc sizing algorithm:** `computeArcFontSize` iteratively shrinks font until text width ≤ `arcRadius × π/2` (90° arc constraint). Returns 0 to hide labels below 7px minimum.
- **Glyph positioning:** `computeGlyphPositions` uses `font.Face.GlyphAdvance` + `Kern` for per-character widths, placing each glyph at the midpoint of its angular span on the arc. Arc centred at top of circle (-π/2 radians).
- **PNG rendering:** Each glyph drawn individually with `dc.Push()/RotateAbout(angle + π/2, gx, gy)/DrawStringAnchored/Pop()`. Must call `dc.SetFontFace(face)` before drawing.
- **SVG rendering:** Uses `<defs>` block with `<path>` arcs (top semicircle, left to right), then `<textPath href="#arc-{idx}" startOffset="50%" text-anchor="middle">`. Traversal indices for path IDs, not node paths.
- **Lint note:** `fixed.Int26_6` is the return type for `font.Face.GlyphAdvance` and `Kern`, not `font.MeasureUnit`. Accumulate in fixed-point then convert to float64 at the end.
- **Cognitive complexity:** Split arc computation into small helpers (`clampFontToArc`, `measureStringWidth`, `collectAdvances`, `sumAdvances`, `placeGlyphs`) to stay under revive's max-10 limit.

### Foliage palette — issue #46 (2026-04-18)

- **New palette added:** `Foliage` (`"foliage"`) in `internal/palette/palette.go` — 11-step ordered palette progressing black → brown → orange → yellow → green for plant-health visualisation.
- **Pattern:** Adding a palette requires changes in four places: const block, `validPalettes` map, `palettes` map, and the `ColourPalette` var definition. Tests need updating in `palette_test.go` for both `IsValid` and the WCAG contrast loop.
- **WCAG constraint:** All ordered palettes must pass adjacent-step contrast ratio >= 1.0 (checked in `TestWCAGContrastRatio`). The foliage colours were chosen with sufficient luminance steps to satisfy this.
- **Environment note:** `gofumpt` and `golangci-lint-custom` aren't available in this shell; rely on `task test` for validation.

### PR review fixes — radialtree_cmd.go + config (2026-04-15)

- **Kong defaults silently override config**: `default:"all"` on a Kong string field causes Kong to always set the value, so `applyOverrides` always writes over config file values. Fix: remove `default:` tags from Kong fields, add `""` to enum, set defaults in `config.New()` only.
- **`defaultStr` helper**: Added `func defaultStr(s string) *string { return &s }` to `internal/config/config.go` for use in `New()`.
- **`resolveLabels` simplification**: After config.New() guarantees `Labels` is always set, the `c.Labels != ""` fallback branch in `resolveLabels` is dead code and was removed.
- **RenderRadialPNG pointer call site**: Changed `render.RenderRadialPNG(nodes, ...)` → `render.RenderRadialPNG(&nodes, ...)` to align with Parker's upcoming signature change. Build fails until Parker's change lands (expected).
- **Colour-apply function invariant**: Documented in all four `applyRadial*` functions that `node.Children` must be files-first, then subdirectories — matching `layoutDir` output order.

### radialtree package (2026-04-15)

- **Package path:** `github.com/bevan/code-visualizer/internal/radialtree` — files `node.go` and `layout.go`
- **RadialNode fields:** `X, Y float64` (pixel offset from canvas centre), `DiscRadius float64` (disc pixel radius), `Angle float64` (radians, 0=east, π/2=down), `Label string`, `ShowLabel bool`, `IsDirectory bool`, `FillColour color.RGBA`, `BorderColour *color.RGBA`, `Children []RadialNode`
- **Layout() signature:** `Layout(root *model.Directory, canvasSize int, discMetric metric.Name, labels LabelMode) RadialNode`
- **LabelMode constants:** `LabelAll = "all"`, `LabelFoldersOnly = "folders"`, `LabelNone = "none"`
- **Algorithm:** ring spacing = (canvasSize/2 - 40) / (maxDepth+1); angular sectors proportional to leaf counts; files placed before subdirs; disc sizes scaled by discMetric quantity/measure value
- Removed pre-existing `radialtree.go` stub (replaced by `node.go`)

### PR #39 review fixes (2026-04-15)

- **FolderAuthorCountProvider.Dependencies()**: Added `gitprovider.AuthorCount` dependency for correct scheduler ordering. Note: this provider queries git directly (not via file metrics) to compute author union sets — simple count summation wouldn't give correct cross-file deduplication.
- **mean-file-lines description**: Updated to explicitly say "skips binary files" per the issue spec.
- **Git debug logging**: Removed the `!errors.Is(err, errUntracked)` guard so untracked-file events are now logged at `slog.Debug` rather than silently swallowed.
- **Folder load helpers**: Added five higher-level helpers to `metrics.go` (`loadMaxQuantity`, `loadMinQuantity`, `loadSumQuantity`, `loadMeanMeasure`, `loadPositiveMeanMeasure`) that encapsulate the full WalkDirectories loop — all 8 folder Load() methods now delegate to a single helper call.
- **Float conversion fix**: Changed `int64(age.Hours()/24)` to `int64(age/(24*time.Hour))` to use integer duration arithmetic, avoiding float precision loss on long durations.
- **Test error handling**: Fixed ignored `os.WriteFile` errors in folder test setup.

### Bubble Tree — Architecture Proposal Ready (2026-04-19)

- **Issue #33:** Ripley completed architecture research for circle-packing bubble tree visualization.
- **Your role in Phase 1 (Layout engine):** Implement `internal/bubbletree/` package — node type, front-chain circle-packing algorithm, enclosing circle (Welzl's).
- **BubbleNode struct:** `X, Y float64` (pixel offset from canvas centre), `Radius float64` (circle radius in px), `Label string`, `ShowLabel bool`, `IsDirectory bool`, `FillColour color.RGBA`, `BorderColour *color.RGBA`, `Children []BubbleNode`
- **LabelMode constants:** `LabelAll = "all"`, `LabelFoldersOnly = "folders"`, `LabelNone = "none"` (parallel to RadialNode)
- **Layout() signature:** `func Layout(root *model.Directory, width, height int, sizeMetric metric.Name, labels LabelMode) BubbleNode`
  - Note: takes width+height (like treemap, allows non-square canvas), unlike radial's square canvasSize
  - Bottom-up recursive packing with front-chain algorithm (Wang et al. 2006)
  - Leaf sizing: radius ∝ √(metricValue), with minimum floor
  - Sort children by radius descending (improves packing density)
  - Front-chain: maintain doubly-linked circular list of outermost circles; place each new circle tangent to best adjacent pair
  - Enclosing circle: Welzl's algorithm O(n) expected — compute parent radius as enclosing circle + padding
  - Top-down: assign absolute pixel coordinates, scale to fit width×height
- **Geometric primitives needed:** Tangent placement, enclosing circle test, circle-circle overlap test (all straightforward)
- **Padding:** Sibling gap (2–4px), parent inset (4–8px for labels)
- **Complexity:** O(n²) per level, acceptable for typical codebases (hundreds–low thousands files/directory)
- **Files to create:** `internal/bubbletree/node.go`, `internal/bubbletree/layout.go`, `internal/bubbletree/layout_test.go` (unit tests: root enclosure, no overlap, radius scaling, nesting depth, label modes, edge cases)
- **Dependencies:** Already available (model, metric packages)

### Bubble Tree — Phase 1 Layout Engine (2026-04-19)

- **Files created:** `internal/bubbletree/node.go` (BubbleNode type, LabelMode constants) and `internal/bubbletree/layout.go` (Layout function + packing algorithm).
- **Algorithm implemented:** Front-chain circle packing with Welzl's enclosing circle. Bottom-up recursive packing then top-down coordinate assignment with scaling.
- **Key constants:** `minFileRadius=2`, `siblingPadding=3`, `parentPadding=6`. Leaf radius = `sqrt(metricValue)` with floor.
- **Layout signature:** `Layout(root *model.Directory, width, height int, sizeMetric metric.Name, labels LabelMode) BubbleNode` — matches treemap pattern (width+height, not square canvas).
- **Geometric primitives:** `tangentPositions` (two-circle tangent placement), `computeEnclosing` (Welzl's adapted for circles not points), `anyOverlap` (circle-circle gap test with padding).
- **Front chain:** Doubly-linked circular list; no pruning (chain only grows). O(n³) per level worst case, acceptable for typical directory sizes.
- **Welzl adaptation:** `enclosingTwo` handles containment and diametrically-opposite cases. `enclosingThree` uses algebraic elimination (subtract equation pairs → linear in u,v,R → quadratic in R). Falls back to pairwise when degenerate (collinear centres, det≈0).
- **`goldenAngle` computed at runtime** (`math.Sqrt` is not const-eligible in Go); used in `placeFallback` for even angular distribution when front-chain tangent fails.
- **Pre-existing `bubbletree_cmd.go`** references `render.RenderBubble` which doesn't exist yet (Phase 2). Full project `go build ./...` fails on that file, but `go build ./internal/bubbletree/...` passes cleanly.

### Bubble Tree — Phases 2+3: PNG & SVG Rendering (2026-04-19)

- **Files created:** `internal/render/bubbletree.go` (PNG/JPG entry point + image rendering) and `internal/render/svg_bubble.go` (SVG rendering).
- **Signature:** `RenderBubble(root *bubbletree.BubbleNode, width, height int, outputPath string) error` — matches Kane's call site in `bubbletree_cmd.go`.
- **Three-pass z-order:** (1) Directory circles sorted by radius descending (outermost first, semi-transparent fills at ~18% alpha), (2) File circles with solid fills, (3) Labels drawn last on top.
- **Coordinate system:** BubbleNode X/Y are absolute pixel coordinates after `Layout()` calls `scaleToFit`. Renderer draws at node coordinates directly (no cx/cy translation).
- **Directory transparency:** PNG uses `color.RGBA` with `A=0x30` (~18%). SVG uses `fill-opacity="0.19"` on `<circle>` elements.
- **Labels:** Straight centred text; directory labels positioned inside circle near top edge (Y - Radius + 14px inset). File labels centred on circle. Colour is constant dark `#222222`.
- **SVG structure:** Flat three-pass approach matching `svg_radial.go` pattern — not nested `<g>` groups. Three passes ensure consistent z-order with PNG.
- **Shared helpers:** `collectBubblesByType`, `resolveDirFill`, `resolveFileFill`, `resolveBorder` used by both PNG and SVG renderers.
- **Golden file:** Created `internal/render/testdata/bubble-tree.png` via `-update` flag.
- **Full build + all tests pass** (`go build ./...` and `go test ./...` clean).

### Legend Wiring — Issue #68 (Phase 2)

- **New file:** `cmd/codeviz/legend_builder.go` — shared helpers `buildLegendRow` and `buildLegendInfo` used by all three viz commands.
- **New file:** `internal/render/svg_legend.go` — SVG legend renderer (fixed from broken untracked stub; uses `metric.Kind`, `color.RGBA`).
- **Legend builder API:** `render.BuildNumericLegendRow(name, kind, buckets, numBuckets, palette)` and `render.BuildCategoricalLegendRow(name, categories, palette)` in `legend.go`.
- **Config:** Added `NoLegend *bool` to `config.Treemap`, `config.Radial`, `config.Bubbletree`.
- **CLI flags:** Added `--no-legend` (`NoLegend bool`) to `TreemapCmd`, `RadialCmd`, `BubbletreeCmd` with corresponding `applyOverrides` wiring.
- **Render function signatures:** All three renderers already accepted `*LegendInfo` param (pre-existing partial work). Replaced `nil` with actual legend info built from fill and border metrics.
- **Legend flow:** CLI builds legend rows from the same metric/palette/buckets used for colour application, assembles `LegendInfo`, passes to renderer. Renderer extends canvas height by `ComputeLegendHeight`, draws viz in original area, draws legend band below.
- **NoLegend semantics:** `nil` or `false` → legend shown (default). `true` → skip legend entirely, full canvas for viz.
- **Gotcha:** `svg_legend.go` was a broken untracked file using wrong type names (`metricKind`, `colourRGBA`). Fixed to use `metric.Kind`, `color.RGBA`.
- **No golden file changes needed:** Existing tests pass `nil` legend (no fill/border metric specified in tests), so canvas dimensions unchanged.
