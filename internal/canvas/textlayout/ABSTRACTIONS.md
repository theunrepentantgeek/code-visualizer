# Abstractions — `internal/canvas/textlayout`

## Go Regular text metrics

**Purpose.**

- The package’s text-metrics abstraction gives layout code and the raster backend one shared Go Regular font face and measurement model, so fitted or truncated text is measured with the font that is actually drawn ([textlayout.go#L10](textlayout.go#L10), [textlayout.go#L22](textlayout.go#L22), [../raster/backend.go#L312](../raster/backend.go#L312)).

**Boundary and invariants.**

- Point size is the only font choice exposed; the embedded Go Regular face is parsed once and reused as immutable font data ([textlayout.go#L9](textlayout.go#L9), [textlayout.go#L22](textlayout.go#L22)).
- Width is glyph advance and height is the face’s common line height, both converted from 26.6 fixed-point units to pixels ([textlayout.go#L27](textlayout.go#L27), [textlayout.go#L36](textlayout.go#L36)).
- `MeasureStrings` creates one face for a batch and returns one width per input line plus their common line height ([textlayout.go#L42](textlayout.go#L42), [textlayout.go#L45](textlayout.go#L45)).

**Related operations.**

- `FontFace` supplies raster drawing; `MeasureString` handles one label; `MeasureStrings` supports block fitting and repeated-label layout ([textlayout.go#L22](textlayout.go#L22), [textlayout.go#L27](textlayout.go#L27), [textlayout.go#L45](textlayout.go#L45)).

**Proper-use patterns.**

- Batch lines of one size through `MeasureStrings`, as block, donut, treemap, and alluvial label layout do ([../block_label.go#L95](../block_label.go#L95), [../../donuttree/labels.go#L141](../../donuttree/labels.go#L141), [../../treemap/directory_chrome.go#L111](../../treemap/directory_chrome.go#L111), [../../alluvial/render.go#L125](../../alluvial/render.go#L125)).

**Anti-patterns.**

- Do not estimate widths from rune counts or create a different font solely for measurement; that makes layout disagree with raster rendering ([textlayout.go#L27](textlayout.go#L27), [../raster/backend.go#L312](../raster/backend.go#L312)).
- Do not retain or mutate a returned face across unrelated work; callers own the face and measurement helpers close their temporary faces after use ([textlayout.go#L22](textlayout.go#L22), [textlayout.go#L28](textlayout.go#L28)).

**Source locations.**

- [textlayout.go#L10](textlayout.go#L10) — shared font data and all public operations.
- [textlayout_test.go#L9](textlayout_test.go#L9) — positive, scale-sensitive, and batch-equivalence tests.
