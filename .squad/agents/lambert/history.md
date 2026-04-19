# Lambert — History

## Core Context

- **Project:** A Go CLI tool that scans file trees and renders treemap visualizations as PNG images with configurable metrics and colour palettes.
- **Role:** Tester
- **Joined:** 2026-04-14T09:49:33.773Z

## Learnings

<!-- Append learnings below -->

### 2026-04-14 — radialtree layout tests

Wrote `internal/radialtree/layout_test.go` (white-box, `package radialtree`) with 12 test cases covering:
- Root always placed at centre (0,0)
- Children positioned in a ring at positive radius
- Single file child has positive DiscRadius
- Four equal-weight files produce four distinct angles (no duplicates)
- Nested depth: file radius > subdir radius > root radius (0)
- Larger metric value produces larger DiscRadius
- LabelAll: both root and file ShowLabel == true
- LabelFoldersOnly: root ShowLabel == true, file ShowLabel == false
- LabelNone: both ShowLabel == false
- Empty directory returns without panic
- Root.Label reflects directory Name
- Larger canvasSize produces larger child radii

Followed the exact style from `internal/treemap/layout_test.go`: `t.Parallel()`, `NewGomegaWithT(t)`, nilaway-safe nil guards, no testify.

### 2026-04-14 — PR review fixes: layout tests + render tests

**Changes to `internal/radialtree/layout_test.go`:**
- Added `"sort"` import
- Replaced `TestLayoutAnglesFullCircle` body: now sorts the 4 angles and verifies consecutive gaps are ~π/2 (within 5% tolerance) instead of just checking uniqueness
- Added `TestLayoutZeroMetricUsesMinDisc`: verifies file with no metric value gets `minFileDisc` radius (the floor)
- Added `TestLayoutUniformMetricUsesMidpoint`: verifies files with equal metric values all receive the `(fileMin+fileMax)/2` midpoint radius, and it's > minFileDisc
- Added `TestComputeLeafCountEmptyDir`: verifies `computeLeafCount` returns 0 for empty dir (actual behaviour, not the old misleading doc comment)
- Added `TestComputeLeafCountWithFiles`: verifies `computeLeafCount` returns 2 for a dir with 2 files

**New file `internal/render/radialtree_test.go`:**
- 4 tests: FlatDir, NestedDir, LabelModes (3 subtests), EmptyDir
- All use `&node` as per the pointer-receiver API Parker introduced
- Tests use `makeFile(name, ext, size)` helper from `renderer_test.go` (same package)
- Parker's `radialtree.go` had a pre-existing unused `sort` import (WIP) that blocks compilation of the render package; the render tests will compile once Parker resolves that

**Key learnings:**
- `computeLeafCount` returns actual 0 for empty dir; zero-guard happens at call site in `layoutDir`
- `buildDiscParams` sets `useEqual=true` when all non-zero metric values are equal (single-value or uniform case)
- Render test compilation depends on Parker completing their `sort`-usage addition to `radialtree.go`

### 2026-04-18 — Foliage palette tests

Added `TestFoliagePalette` to `internal/palette/palette_test.go` covering:
- 11 colour steps, ordered, correct name
- First step near-black (R, G, B all ≤ 30)
- Last step green-dominant (G > R and G > B)
- Foliage already included in `TestPaletteName_IsValid` and `TestWCAGContrastRatio` by Dallas

Pattern: palette tests follow a consistent shape — step count, ordered flag, name check, then endpoint colour assertions. WCAG contrast test covers all ordered palettes via a shared loop.

### 2026-04-19 — bubbletree layout tests

Wrote `internal/bubbletree/layout_test.go` (white-box, `package bubbletree`) with 16 test cases covering:
- Root enclosure (radius > 0, IsDirectory true, all children geometrically contained)
- No overlap (sibling circles don't overlap within 1px tolerance)
- Radius scaling (larger metric → larger radius)
- Nesting depth (nested dirs produce nested circles, containment holds at every level)
- Label modes (LabelAll, LabelFoldersOnly, LabelNone each set ShowLabel correctly)
- Empty directory (non-panic, positive radius, no children)
- Single file (centred in parent, contained)
- Large flat directory (20 files pack without overlap)
- Zero metric (missing value gets positive radius floor)
- Uniform metric (equal values → equal radii)
- Canvas bounds (root circle fits within width × height)
- Root label (matches directory name)
- Root IsDirectory (root true, file child false)
- Deep nesting (3-level tree, containment at every level)
- Mixed files and dirs (file + subdir siblings, no overlap, containment)

Helper functions: `assertContainment` (recursive parent-child geometric check), `assertNoOverlap` (recursive sibling pair distance check), `allChildren` (depth-first collector).

Tests follow exact style from radialtree/treemap: `t.Parallel()`, `NewGomegaWithT(t)`, nilaway-safe nil guards, dot-imported gomega matchers. Tests won't compile until Dallas delivers the layout engine — that's expected.

### 2026-04-19 — bubbletree render smoke tests

Wrote `internal/render/bubbletree_test.go` with 4 smoke tests:
- **TestRenderBubble_PNG**: renders sample tree to .png, decodes with `image.DecodeConfig`, asserts format == "png"
- **TestRenderBubble_JPG**: renders to .jpg, asserts format == "jpeg"
- **TestRenderBubble_SVG**: renders to .svg, XML-parses to find `<svg>` root element
- **TestRenderBubble_GoldenFile**: renders to .png, compares against golden file via `goldie.New(t, WithFixtureDir("testdata"), WithNameSuffix(".png"))` with fixture name "bubble-tree"

Shared helper `sampleBubbleTree()` builds a deterministic `BubbleNode` tree directly (root dir with nested "src" subdir + 2 file children + 1 sibling file). No Layout call — these are pure render tests.

Pattern follows `radialtree_test.go` and `renderer_test.go` exactly: `t.Parallel()`, `NewGomegaWithT(t)`, dot-imported gomega, `t.TempDir()` for output. `RenderBubble` signature: `func RenderBubble(root *bubbletree.BubbleNode, width, height int, outputPath string) error`. Tests won't compile until Dallas delivers the render implementation — that's expected.

### 2026-04-19 — Legend feature tests (issue #68)

Wrote 47 new tests across 3 files for the legend feature on `squad/68-legend-core`:

**`cmd/codeviz/legend_builder_test.go` (19 tests):**
- `BuildNumericLegendRow`: Quantity, Measure, single-bucket (all-same values), 4 palette variants (Temperature, Neutral, GoodBad, Foliage)
- `BuildCategoricalLegendRow`: basic (4 categories), single category, many categories (20, triggers wrap warning), colours match `CategoricalMapper.Map()`
- `buildLegendRow`: empty metric → nil, unknown metric → nil, Quantity metric with files, Classification metric with files, no files → nil
- `buildLegendInfo`: NoLegend=true → nil, NoLegend=false → builds, nil flag → builds, all-nil rows → nil, mixed nil/real → filters, two rows preserved in order

**`internal/render/svg_legend_test.go` (14 tests):**
- nil/empty info writes nothing
- Numeric row: contains `<g>`, 3 `<rect>`, metric name, breakpoint values
- Categorical row: contains category labels
- Multiple rows (Quantity+Measure+Classification): 9 rects, all metric names
- HTML escaping: `<`, `&`, `>` properly escaped
- Single colour row, empty colours row (skips swatches)
- Many categories (20), narrow width (no swatches when < legendLabelWidth)
- Swatch stroke colour `#808080` present

**`internal/render/legend_test.go` (14 new, added to Dallas's 11):**
- `ComputeLegendHeight` 3-row formula check (116px)
- `DrawLegendBand` 3-row, single-colour, very long metric name, narrow width
- Integration: treemap/radial/bubble PNG with legend taller than without
- Integration: nil legend preserves original 300px height
- SVG integration: treemap/radial/bubble with legend contain `<g>` group + metric name
- SVG integration: treemap/bubble without legend have no `translate` group
- White-box: `BuildNumericLegendRow`/`BuildCategoricalLegendRow` bucket count

**Key learnings:**
- All renderers (Render, RenderRadial, RenderBubble) accept `*LegendInfo` and increase image height by `ComputeLegendHeight(legend)` for raster output
- SVG legend uses a `<g transform="translate(x,y)">` wrapper — its presence/absence is a clean integration test signal
- `writeSVGLegend` takes `*os.File` directly (not a buffer), so SVG legend tests create temp files in the working directory
- Pre-existing `unparam` lint on `svg_legend.go:20` (`x always receives 0`) — not introduced by tests
- `palette.NewCategoricalMapper` logs a warning when categories > palette capacity but doesn't error
