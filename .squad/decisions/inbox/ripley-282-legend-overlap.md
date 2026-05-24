# Issue #282 — Bubbletree Legend Reservation

**Author:** Ripley  
**Date:** 2026-05-23  
**Status:** Proposed  
**Issue:** #282 — Bubbletree legend overlaps visualisation

## Decision

Bubbletree should reserve legend space using the same shared legend-reservation flow as treemap instead of drawing the legend as an overlay on the full-layout canvas.

## Implementation Pattern

1. Build `LegendConfig` as today.
2. Call `legend.ReserveAndLayout()` before bubble layout to reduce the layout width/height.
3. Run `bubbletree.Layout()` within the reduced dimensions.
4. If space was reserved, call `LegendConfig.ReserveSpace()` plus `legend.LayoutOffset()` and translate the whole bubble node tree into the remaining drawable region.

## Rationale

The existing overlay behaviour causes the legend to sit on top of bubble content whenever the visualisation fills the full canvas. Treemap already solved this with shared helpers in `internal/legend/reserve.go`; reusing that path keeps legend behaviour consistent across visualisations and preserves the fallback to overlay mode when reserving space would make the drawable area too small.

## Code Impact

- `internal/bubbletree/stages.go`
- `internal/bubbletree/layout.go`
- `internal/bubbletree/layout_stage_test.go`
- Reference pattern: `internal/treemap/stages.go`, `internal/legend/reserve.go`
