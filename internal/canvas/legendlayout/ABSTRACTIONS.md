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
