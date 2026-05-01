# Spiral Visualization — Test Specification

**Issue:** #127
**Author:** Lambert (Tester)
**Status:** Pre-implementation (spec only — no Go code yet)

This document specifies test cases for the spiral visualization. Tests are organized by category. Each test name follows project convention: `TestLayout{Description}` for layout tests, `TestRender{Description}` for render tests. All tests will use `t.Parallel()`, `NewGomegaWithT(t)`, dot-imported Gomega, and nilaway-safe nil guards.

> **Note:** The spiral is fundamentally different from treemap/radialtree/bubbletree — it visualizes **time-series data**, not directory trees. The Layout function signature will differ accordingly. Test helpers and input builders will be defined once the architecture is finalized.

---

## 1. Layout — Spiral Geometry

These tests verify that spots are placed on a spiral path with the correct geometric properties.

### `TestLayoutSpiralClockwise`
**Verifies:** Spots proceed in clockwise angular order as time advances.
**How:** Create 4+ consecutive timestamps. Compute the angle of each spot from the spiral centre. Assert angles increase clockwise (in standard screen coordinates where Y increases downward, clockwise means increasing angle in the positive direction).

### `TestLayoutSpiralOutward`
**Verifies:** Later timestamps are farther from the centre than earlier timestamps (spiral moves outward).
**How:** Create timestamps spanning more than one lap. Compute radial distance from centre for first and last spot. Assert last > first.

### `TestLayoutInnerDiameterRatio`
**Verifies:** Inner (minimum) diameter ≈ 1/3 of outer diameter.
**How:** Create timestamps spanning several laps. Measure the radial distance of the innermost spot and the outermost spot. Assert `innerRadius / outerRadius` is approximately `1/3` (within ±10% tolerance).

### `TestLayoutConsistentAngularSpacing`
**Verifies:** All spots are placed at a consistent angular distance from their neighbours.
**How:** Create 10+ timestamps at a given resolution. Compute angles of all spots. Assert that consecutive angular differences are equal (within ±5% tolerance).

### `TestLayoutCentreOfSpiral`
**Verifies:** The spiral is centred within the canvas.
**How:** Create a multi-lap spiral. Assert the spiral centre is at approximately (width/2, height/2) or (0, 0) depending on the coordinate system chosen.

### `TestLayoutFitsWithinCanvas`
**Verifies:** All spots (including their disc radii) fit within the canvas bounds.
**How:** Create a multi-lap spiral. For every spot, assert `spot.X ± spot.DiscRadius` and `spot.Y ± spot.DiscRadius` are within `[0, width]` and `[0, height]` (or equivalent bounds with tolerance).

### `TestLayoutScalesWithCanvasSize`
**Verifies:** Larger canvas produces proportionally larger spiral.
**How:** Layout the same data on a 800×800 canvas and a 1600×1600 canvas. Assert the outer radius of the larger layout is approximately 2× the smaller layout.

---

## 2. Time Resolution — Hourly

### `TestLayoutHourly_24SpotsPerLap`
**Verifies:** With hourly resolution, one complete lap contains exactly 24 spots.
**How:** Create timestamps spanning exactly 24 hours. Assert there are 24 spots, and the first and last spots are approximately one full lap apart (angular positions near-identical, radii differ by one lap's growth).

### `TestLayoutHourly_AngularStep`
**Verifies:** Angular distance between consecutive hourly spots is `2π/24` (15°).
**How:** Create 3+ consecutive hourly timestamps. Compute angular differences. Assert each is approximately `2π/24`.

### `TestLayoutHourly_MultiLap`
**Verifies:** 48 hours of data produces exactly 2 complete laps.
**How:** Create timestamps spanning 48 hours. Assert 48 spots. Verify the spot at index 24 is roughly one full rotation ahead of spot at index 0 (same angle, larger radius).

### `TestLayoutHourly_AggregationWindow`
**Verifies:** Each hourly spot aggregates events in the half-open interval `[H:00, H+1:00)`.
**How:** Create events at 1:00, 1:30, 1:59, and 2:00. The spot for hour 1 should contain the first three; the spot for hour 2 should contain the last one.

---

## 3. Time Resolution — Daily

### `TestLayoutDaily_28SpotsPerLap`
**Verifies:** With daily resolution, one complete lap contains exactly 28 spots (four weeks).
**How:** Create timestamps spanning exactly 28 days. Assert 28 spots, first and last approximately one full lap apart.

### `TestLayoutDaily_AngularStep`
**Verifies:** Angular distance between consecutive daily spots is `2π/28`.
**How:** Create 3+ consecutive daily timestamps. Compute angular differences. Assert each is approximately `2π/28`.

### `TestLayoutDaily_MultiLap`
**Verifies:** 56 days of data produces exactly 2 complete laps.
**How:** Create timestamps spanning 56 days. Assert 56 spots.

### `TestLayoutDaily_AggregationWindow`
**Verifies:** Each daily spot aggregates events from `[00:00 day D, 00:00 day D+1)`.
**How:** Create events at midnight start-of-day, noon, 23:59:59, and midnight end-of-day. First three aggregate into day D; the last aggregates into day D+1.

---

## 4. Time Aggregation Boundary Tests

### `TestAggregation_HalfOpenInterval`
**Verifies:** Aggregation intervals are half-open `[start, end)` — start-inclusive, end-exclusive.
**How:** For hourly resolution, create an event at exactly 2:00:00. Assert it falls into the 2:00 bucket, not the 1:00 bucket.

### `TestAggregation_HourlyEdge_MidnightWrap`
**Verifies:** 23:00–00:00 boundary works correctly (midnight wraps to next day's first hour).
**How:** Create events at 23:30 and 00:00 the next day. Assert they fall into separate hourly spots on separate laps.

### `TestAggregation_DailyEdge_MidnightBoundary`
**Verifies:** Events at exactly midnight belong to the new day.
**How:** Create an event at midnight (00:00:00.000). Assert it aggregates into the day that starts at that midnight, not the preceding day.

### `TestAggregation_EmptyBuckets`
**Verifies:** Time periods with no events still produce a spot (with zero/default metrics).
**How:** Create events at hour 1 and hour 5 (hourly resolution). Assert spots exist for hours 2, 3, 4 with zero/default metric values.

---

## 5. Metric Destination Tests

### `TestLayout_DiscSizeNumeric`
**Verifies:** Disc size is driven by a numeric metric and scales proportionally.
**How:** Create two time slots, one with metric value 100 and one with 400. Assert the larger-metric spot has a larger disc radius.

### `TestLayout_DiscSizeZero`
**Verifies:** A spot with zero numeric metric gets a positive minimum disc radius (floor), not zero.
**How:** Create a spot with no metric value. Assert disc radius > 0.

### `TestLayout_DiscSizeUniform`
**Verifies:** Spots with equal metric values produce equal disc radii.
**How:** Create three spots with identical metric values. Assert all disc radii are equal (within tolerance).

### `TestLayout_FillColourApplied`
**Verifies:** Fill colour field is populated on each spot node.
**How:** After layout + colour application, assert each spot's `FillColour` is non-zero-value (not the zero `color.RGBA{}`). (This test runs after the colour-application pass, not the layout itself.)

### `TestLayout_BorderColourApplied`
**Verifies:** Border colour field is populated when a border metric is specified.
**How:** After colour application with a border metric, assert `BorderColour` is non-nil on each spot.

### `TestLayout_BorderColourNilWhenUnset`
**Verifies:** Border colour is nil when no border metric is specified.
**How:** After colour application without a border metric, assert `BorderColour` is nil on each spot.

### `TestLayout_MetricDestinationsIndependent`
**Verifies:** Disc size, fill, and border are driven by independent metrics.
**How:** Create spots where size metric differs from fill metric. Assert disc radii vary by size metric while fill colours vary by fill metric (they don't correlate).

---

## 6. Edge Cases

### `TestLayout_ZeroTimestamps`
**Verifies:** Empty input (no timestamps) produces a valid result without panic.
**How:** Pass empty time-series data. Assert result is valid (empty spots list or minimal structure), no panic.

### `TestLayout_SingleTimestamp`
**Verifies:** A single timestamp produces exactly one spot.
**How:** Pass one event. Assert one spot is produced. Assert it has a positive disc radius and is positioned on the spiral.

### `TestLayout_ExactlyOneLap_Hourly`
**Verifies:** Exactly 24 hourly timestamps produce one complete lap.
**How:** Create 24 consecutive hourly timestamps. Assert 24 spots. Assert the angular range spans approximately 2π.

### `TestLayout_ExactlyOneLap_Daily`
**Verifies:** Exactly 28 daily timestamps produce one complete lap.
**How:** Create 28 consecutive daily timestamps. Assert 28 spots.

### `TestLayout_PartialLap_Hourly`
**Verifies:** Fewer than 24 hourly timestamps produce a partial arc (less than one full rotation).
**How:** Create 6 hourly timestamps. Assert 6 spots. Assert the angular span is approximately `6 × 2π/24 = π/2`.

### `TestLayout_PartialLap_Daily`
**Verifies:** Fewer than 28 daily timestamps produce a partial arc.
**How:** Create 7 daily timestamps. Assert 7 spots. Assert angular span is approximately `7 × 2π/28 = π/2`.

### `TestLayout_ManyLaps`
**Verifies:** Data spanning many laps renders correctly with monotonically increasing radius.
**How:** Create 100 hourly timestamps (4+ laps). Assert 100 spots. Assert radial distance from centre increases monotonically across the full sequence.

### `TestLayout_ManyLaps_NoOverlap`
**Verifies:** Spots on adjacent laps don't overlap.
**How:** Create timestamps for 3+ complete laps. For every pair of spots that are angularly adjacent but on different laps, assert the gap between their radial positions minus their disc radii is positive.

### `TestLayout_GapInTimeSeries`
**Verifies:** Non-contiguous timestamps still produce spots for intermediate empty periods.
**How:** Create hourly events at 01:00 and 05:00 only. Assert spots exist for 01:00 through 05:00 (5 spots), with intermediate spots having default/zero metrics.

---

## 7. Empty Input Handling

### `TestLayout_NilInput`
**Verifies:** Nil input doesn't panic and returns a valid (empty) result.
**How:** Pass nil as the time-series input. Assert no panic, valid return.

### `TestLayout_EmptySlice`
**Verifies:** Empty slice input produces valid empty result.
**How:** Pass an empty slice. Assert result has zero spots.

---

## 8. Label Mode Tests

*(If the spiral supports label modes like other visualizations.)*

### `TestLayout_LabelAll`
**Verifies:** When label mode is "all", every spot has `ShowLabel == true`.
**How:** Create multi-spot spiral. Assert all spots have `ShowLabel == true`.

### `TestLayout_LabelNone`
**Verifies:** When label mode is "none", no spot has `ShowLabel == true`.
**How:** Create multi-spot spiral. Assert all spots have `ShowLabel == false`.

### `TestLayout_SpotLabelsContainTimestamp`
**Verifies:** Spot labels contain a meaningful time representation (hour or date).
**How:** Create an hourly spot for 14:00. Assert its label contains "14" or "2pm" or the appropriate formatted timestamp. Create a daily spot for 2024-04-29. Assert its label contains "Apr 29" or "2024-04-29" or similar.

---

## 9. Render Smoke Tests

*(These follow the exact pattern from `radialtree_test.go` and `bubbletree_test.go` — pure render tests using a pre-built node tree, no Layout call.)*

### `TestRenderSpiral_PNG`
**Verifies:** Rendering a spiral to PNG produces a valid PNG file.
**How:** Build a deterministic `SpiralNode` tree directly. Render to `.png`. Decode with `image.DecodeConfig`. Assert format == "png".

### `TestRenderSpiral_JPG`
**Verifies:** Rendering to JPG produces a valid JPEG file.
**How:** Same tree, render to `.jpg`. Assert format == "jpeg".

### `TestRenderSpiral_SVG`
**Verifies:** Rendering to SVG produces valid SVG XML.
**How:** Same tree, render to `.svg`. XML-parse to find `<svg>` root element.

### `TestRenderSpiral_GoldenFile`
**Verifies:** Rendered output matches approved golden file.
**How:** Render to `.png`. Compare via `goldie.New(t)` with fixture name "spiral".

---

## 10. Config & CLI Tests

*(To be detailed after CLI design is decided. Placeholder specs based on the established pattern.)*

### `TestSpiralCmd_Validate_EmptySize_Passes`
**Verifies:** `Validate()` accepts empty disc-size (deferred to Run after config merge).

### `TestSpiralCmd_ConfigSuppliesSize`
**Verifies:** Config file's `spiral.size` populates when CLI omits `--disc-size`.

### `TestSpiralCmd_CLISizeOverridesConfig`
**Verifies:** CLI `--disc-size` overwrites config file value.

### `TestSpiralCmd_TimeResolutionDefault`
**Verifies:** Default time resolution is applied when not specified.

### `TestSpiralCmd_TimeResolutionHourly`
**Verifies:** `--resolution hourly` sets hourly time resolution.

### `TestSpiralCmd_TimeResolutionDaily`
**Verifies:** `--resolution daily` sets daily time resolution.

---

## Summary

| Category                      | Test Count |
|-------------------------------|-----------|
| Layout — Spiral Geometry      | 7         |
| Time Resolution — Hourly      | 4         |
| Time Resolution — Daily       | 4         |
| Time Aggregation Boundaries   | 4         |
| Metric Destinations           | 7         |
| Edge Cases                    | 9         |
| Empty Input Handling          | 2         |
| Label Modes                   | 3         |
| Render Smoke Tests            | 4         |
| Config & CLI                  | 6         |
| **Total**                     | **50**    |
