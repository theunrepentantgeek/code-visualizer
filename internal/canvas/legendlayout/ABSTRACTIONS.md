# Abstractions — `internal/canvas/legendlayout`

## StringMeasurer

**Purpose.**

- `StringMeasurer` is the text-measurement seam legend layout depends on: give it a string, get back rendered width and height ([measurer.go#L8](measurer.go#L8), [measurer.go#L10](measurer.go#L10)).
- It lets legend sizing be computed without a drawing context, so measurement and rendering are separable ([layout.go#L24](layout.go#L24), [layout.go#L67](layout.go#L67)).

**Boundary and invariants.**

- Implementations may use real font metrics or a fixed-width approximation — the interface deliberately does not promise pixel-exactness ([measurer.go#L9](measurer.go#L9)).
- The standard implementation uses the 7×13 bitmap font, chosen because it is the same default font `gg.NewContext` uses, so measurements match what the raster backend draws ([measurer.go#L14](measurer.go#L14), [measurer.go#L20](measurer.go#L20)).
- Every layout entry point takes the measurer as a parameter rather than reaching for a package-level default ([layout.go#L24](layout.go#L24), [layout.go#L89](layout.go#L89)).

**Related operations.**

- `NewBasicMeasurer` for the standard measurer; `MeasureLegend` and `ReserveSpace` are its principal consumers ([measurer.go#L23](measurer.go#L23), [layout.go#L24](layout.go#L24), [layout.go#L67](layout.go#L67)).

**Proper-use patterns.**

- Pass `NewBasicMeasurer()` at the call site that needs a size, so both space reservation and drawing measure identically ([../../legend/config.go#L73](../../legend/config.go#L73), [../../legend/render.go#L52](../../legend/render.go#L52)).

**Anti-patterns.**

- Do not estimate text width with a character count multiplier in layout code; the measurer exists so the estimate matches the font actually used ([measurer.go#L27](measurer.go#L27)).

**Source locations.**

- [measurer.go#L10](measurer.go#L10) — `StringMeasurer`, `NewBasicMeasurer`.
- [measurer_test.go#L9](measurer_test.go#L9) — construction and measurement behavior.

## Legend layout geometry

**Purpose.**

- The legend layout functions are the shared geometry contract between reservation and drawing: they measure a resolved `model.LegendData`, place its box, and expose the same entry/sample dimensions used while decomposing it into canvas primitives ([layout.go#L22](layout.go#L22), [layout.go#L36](layout.go#L36), [../../legend/render.go#L52](../../legend/render.go#L52)).

**Boundary and invariants.**

- Nil data or data without entries measures as zero; reservation additionally treats position `none` as zero ([layout.go#L24](layout.go#L24), [layout.go#L67](layout.go#L67)).
- Orientation determines the layout axis: horizontal legends accumulate entry widths, while vertical legends accumulate entry heights ([layout.go#L29](layout.go#L29), [layout.go#L89](layout.go#L89), [layout.go#L120](layout.go#L120)).
- Positioning applies the shared model margin, and label samples are always measured as square regions sized by their text or twice the swatch size ([layout.go#L42](layout.go#L42), [layout.go#L348](layout.go#L348)).
- Numeric breakpoint labels use integer formatting for exact integral values and one decimal place otherwise ([layout.go#L13](layout.go#L13), [layout_test.go#L13](layout_test.go#L13)).

**Related operations.**

- `MeasureLegend`, `LegendOrigin`, and `ReserveSpace` operate on the complete legend; `MeasureEntryHWidth`, `MeasureEntryVContentWidth`, `ContentOffsetV`, and `MeasureLabelSample` expose matching sub-layout geometry to the renderer ([layout.go#L24](layout.go#L24), [layout.go#L37](layout.go#L37), [layout.go#L67](layout.go#L67), [layout.go#L270](layout.go#L270), [layout.go#L307](layout.go#L307), [layout.go#L344](layout.go#L344)).

**Proper-use patterns.**

- Convert configuration to `LegendData` once, reserve with these functions before visualization layout, then use the same data and geometry helpers to draw the overlay ([../../legend/config.go#L70](../../legend/config.go#L70), [../../legend/render.go#L21](../../legend/render.go#L21)).
- Keep metric-to-display formatting here so legend construction and rendering share the same labels ([../../inks/legend_data.go#L42](../../inks/legend_data.go#L42)).

**Anti-patterns.**

- Do not duplicate swatch gaps, title heights, sample dimensions, or position offsets in a visualization package; those values are coupled to `model` constants and legend rendering ([layout.go#L191](layout.go#L191), [../model/legend.go#L88](../model/legend.go#L88)).

**Source locations.**

- [layout.go#L13](layout.go#L13) — breakpoint formatting and public legend geometry operations.
- [layout_test.go#L13](layout_test.go#L13) — breakpoint, placement, measurement, reservation, and centering tests.
- [helpers_test.go#L43](helpers_test.go#L43) — label-sample sizing tests.
