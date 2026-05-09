# Canvas Spec Revision — Feedback Integration

**Author:** Dallas
**Date:** 2026-05-08
**Status:** Complete

## Summary

Revised the Canvas abstraction design spec to integrate all review feedback from Bevan, Bishop, and Parker. The spec is now a clean, final design document with no review artifacts.

## Key Decisions Codified

1. **MetricValue type** — Unifies numeric/categorical metric data into one struct. Shapes carry `Fill MetricValue` and `Border MetricValue` instead of four separate fields. Ink has a single `Dip(MetricValue)` method.

2. **Opacity on Ink** — Moved from Spec-level `Opacity` field to `WithOpacity()` InkOption. Resolved at `Dip()` time (alpha channel). Backend methods receive pre-resolved RGBA.

3. **ShapeStyle embedding** — Extracted common fields into `ShapeStyle` struct, embedded by `RectangleSpec` and `DiscSpec`.

4. **Backend subpackages** — Bevan decided: `internal/canvas/raster/` and `internal/canvas/svg/`. Exported `Backend` interface in parent `canvas` package. Ports & Adapters pattern.

5. **Canvas constructor** — `NewCanvas(width, height int)`. Output path deferred to `Render(outputPath string)`. Backend selection at render time.

6. **Position/Size helper structs** — Reduce backend method parameter counts and prevent swap errors.

7. **Migration approach** — Each viz migration is a single atomic PR. Colour field stripping + render replacement must be in the same change.
