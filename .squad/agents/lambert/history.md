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

### 2026-04-20 — issue #99 config-bypass validation tests

Added 6 tests to `cmd/codeviz/main_test.go` for issue #99 (config-supplied parameters bypass early validation):

- **TestTreemapCmd_Validate_EmptySize_Passes**: Validate() accepts empty Size (deferred to Run)
- **TestRadialCmd_Validate_EmptyDiscSize_Passes**: Validate() accepts empty DiscSize (deferred to Run)
- **TestBubbletreeCmd_Validate_EmptySize_Passes**: Validate() accepts empty Size (deferred to Run)
- **TestTreemapCmd_ConfigSuppliesSize**: config file's `treemap.size` survives applyOverrides when CLI omits `--size`
- **TestTreemapCmd_CLISizeOverridesConfig**: CLI `--size` overwrites config file value via applyOverrides
- **TestTreemapCmd_MissingSizeEverywhere_NilAfterMerge**: when neither CLI nor config provides size, `cfg.Treemap.Size` is nil after merge

Key learnings:
- `config.New()` initialises `Treemap: &Treemap{}` with all pointer fields nil — no default size
- `applyOverrides` guards each field with `if != ""` / `if != 0`, so empty CLI values are transparent
- Tests 1–3 won't pass until Kane removes size validation from Validate() (that's the fix)
- Tests 4–6 pass on current code — they exercise applyOverrides and config.Load independently of the Validate fix
- `go vet ./cmd/codeviz/` confirmed all 6 tests compile cleanly

### 2026-04-26 — export feature tests (issue #107)

Wrote `internal/export/export_test.go` (white-box, `package export`) with 9 test cases covering:
- **TestExport_JSON**: exports simple tree to JSON, unmarshals and verifies structure (root name, file count, metric values)
- **TestExport_YAML**: same tree to .yaml, verifies valid YAML output
- **TestExport_YML**: .yml extension accepted as YAML
- **TestExport_UnsupportedFormat**: .txt returns an error
- **TestExport_MetricFiltering**: sets fileSize + lineCount + fileType, requests only fileSize; verifies unrequested metrics are absent from both directory and file exports
- **TestExport_EmptyDirectory**: empty dir produces valid JSON with no files/directories
- **TestExport_NestedDirectories**: 3-level deep tree (root → mid → deep), verifies hierarchy preserved
- **TestExport_BinaryFileFlag**: binary file has isBinary=true, text file has isBinary=false
- **TestExport_AllMetricTypes**: quantity + measure + classification all present on both directory and file level

Dallas had already delivered `export.go` — all 9 tests pass on first run. Test patterns follow the project standard: `t.Parallel()`, `NewGomegaWithT(t)`, dot-imported gomega, nilaway-safe nil guards before dereferencing, `t.TempDir()` for output files. Used `go.yaml.in/yaml/v3` for YAML unmarshalling (same as config package).

### Issue #107 — Export Test Suite Complete (2026-04-26)

- **File:** `internal/export/export_test.go` with 9 comprehensive tests.
- **Test coverage:** JSON format, YAML format, unsupported format error, metric filtering (requested list respected), empty directories (omitted via omitempty), nested structures (3 levels deep), binary file flag preservation, all metric types (Quantity int64, Measure float64, Classification string).
- **Build status:** Pass. All 9 tests pass.
- **No regressions:** Tests validate Dallas's implementation was correct on first delivery.
- **Pattern compliance:** Gomega assertions, t.Parallel(), nil-safe guards, temp directories for file I/O, JSON/YAML round-trip validation.

