# Abstractions — `internal/canvas/textlayout`

## Text layout

**Purpose.**

- `textlayout` supplies the shared Go Regular font and measurements used to fit labels before the raster backend renders them. It keeps font selection and sizing consistent across independently implemented labelers ([textlayout.go](textlayout.go), [../raster/backend.go](../raster/backend.go)).

**Boundary and invariants.**

- `FontFace(points)` constructs a face at the requested point size; `MeasureString` and `MeasureStrings` use that same font. Width is glyph advance and height is the font's line height, converted from fixed-point metrics to whole-pixel values, not a tight visible-ink bounding box ([textlayout.go](textlayout.go)).
- `MeasureStrings` returns one width per input line, in order, and one common line height, using a single face. It agrees with measuring each line separately ([textlayout_test.go](textlayout_test.go)).
- This is the scalable Go Regular text contract, not the bitmap `legendlayout.StringMeasurer` contract used for legend sizing ([textlayout.go](textlayout.go), [../legendlayout/measurer.go](../legendlayout/measurer.go)).

**Related operations.**

- `FontFace` for drawing; `MeasureString` for one string; `MeasureStrings` for a batch at one font size. Measurement helpers release their temporary face after use ([textlayout.go](textlayout.go)).

**Proper-use patterns.**

- Tree-map chrome measures truncation candidates together; donut labels measure their lines at the chosen size; the raster backend selects the same font for explicit-size text ([../../treemap/directory_chrome.go#L111](../../treemap/directory_chrome.go#L111), [../../donuttree/labels.go#L154](../../donuttree/labels.go#L154), [../raster/backend.go#L312](../raster/backend.go#L312)).

**Anti-patterns.**

- Do not fit Go Regular labels using character counts or the legend's bitmap measurer: those do not describe the font used to draw them. Do not create a new face for each line when all lines share a size ([../block_label.go#L96](../block_label.go#L96), [../../alluvial/render.go#L131](../../alluvial/render.go#L131)).

**Source locations.**

- [textlayout.go](textlayout.go) and [textlayout_test.go](textlayout_test.go) — font, measurement operations, and batch equivalence.
- [../../treemap/directory_chrome.go](../../treemap/directory_chrome.go), [../../donuttree/labels.go](../../donuttree/labels.go), [../../alluvial/render.go](../../alluvial/render.go), and [../raster/backend.go](../raster/backend.go) — independent labelers and drawing.
