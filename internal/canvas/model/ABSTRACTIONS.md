# Abstractions — `internal/canvas/model`

## Backend

**Purpose.**

- `Backend` is the rendering interface every output format adapter implements; its methods take *resolved* colours and fills plus primitive geometry ([backend.go#L12](backend.go#L12), [backend.go#L14](backend.go#L14)).
- It lives in this package specifically to break the import cycle between the canvas package and the backend packages ([backend.go#L1](backend.go#L1)).

**Boundary and invariants.**

- The backend never sees metrics, inks, or palettes: everything is already resolved to geometry, `Fill`, and `color.RGBA` by the time it is called ([backend.go#L13](backend.go#L13), [../rectangle.go#L18](../rectangle.go#L18)).
- Text alignment is expressed with the shared `TextAnchor` vocabulary rather than backend-specific flags ([backend.go#L26](backend.go#L26), [backend.go#L21](backend.go#L21)).
- `Finish(outputPath)` is where the accumulated drawing is written, so backends may buffer, and layout-relevant constants callers must agree with — `DefaultFontSize`, `ArcTextInset` — are defined alongside the interface ([backend.go#L23](backend.go#L23), [backend.go#L42](backend.go#L42)).

**Related operations.**

- `Canvas.RenderTo` drives a backend; each shape's `drawTo` is the adapter from retained shape to backend call ([../canvas.go#L264](../canvas.go#L264), [../rectangle.go#L18](../rectangle.go#L18)).

**Proper-use patterns.**

- Add a new output format by implementing `Backend`, not by special-casing formats inside the canvas ([../raster/backend.go#L46](../raster/backend.go#L46), [../svg/backend.go#L58](../svg/backend.go#L58)).

**Anti-patterns.**

- Do not import the canvas package from a backend; the shared types are here precisely to avoid that dependency ([backend.go#L1](backend.go#L1)).

**Source locations.**

- [backend.go#L14](backend.go#L14) — `Backend`, `TextAnchor`, and the shared rendering constants.

## Fill

**Purpose.**

- `Fill` is a sealed description of how a shape's interior is painted — currently a uniform `SolidFill` or a `RadialGradientFill` from a focus-point centre colour to an edge colour ([fill.go#L11](fill.go#L11), [fill.go#L16](fill.go#L16), [fill.go#L21](fill.go#L21)).

**Boundary and invariants.**

- The interface is sealed by an unexported method, so the set of fills is closed and backends can switch on it exhaustively ([fill.go#L13](fill.go#L13), [fill.go#L29](fill.go#L29)).
- `GradientPoint` coordinates are normalised but explicitly allowed to fall outside [0,1] ([fill.go#L5](fill.go#L5), [fill.go#L6](fill.go#L6)).
- `SolidColor` gives every fill a single representative colour — the fill colour, the gradient centre, or opaque black for anything unknown — so a backend that cannot do gradients still renders something sensible ([fill.go#L32](fill.go#L32), [fill.go#L41](fill.go#L41)).

**Related operations.**

- `inks.Ink.Fill` produces fills from a metric value and focus point; backends consume them ([../../inks/ink.go#L127](../../inks/ink.go#L127), [../raster/backend.go#L46](../raster/backend.go#L46)).

**Proper-use patterns.**

- Type-switch for the gradient case and fall back to `SolidColor` for everything else, as both backends do ([../raster/backend.go#L46](../raster/backend.go#L46), [../svg/backend.go#L315](../svg/backend.go#L315)).

**Anti-patterns.**

- Do not pass `color.RGBA` where a `Fill` is expected in order to "keep it simple": the gradient case is what makes pincushion shading possible across backends ([fill.go#L21](fill.go#L21)).

**Source locations.**

- [fill.go#L11](fill.go#L11) — `Fill`, `SolidFill`, `RadialGradientFill`, `GradientPoint`, `SolidColor`.

## LegendData

**Purpose.**

- `LegendData` is the fully resolved, backend-facing description of a legend overlay: position, orientation, an optional label sample, and one entry per metric role ([legend.go#L43](legend.go#L43), [legend.go#L44](legend.go#L44)).

**Boundary and invariants.**

- It is resolved data, not configuration: entries carry colours and labels, never metrics or inks ([legend.go#L72](legend.go#L72), [legend.go#L84](legend.go#L84)).
- The numeric/categorical distinction is explicit, and for numeric entries the swatch label is the breakpoint at the divider, with the last swatch label empty ([legend.go#L32](legend.go#L32), [legend.go#L80](legend.go#L80)).
- A label sample is optional and identifies its backing shape separately from its compacted text lines; square is the zero-value shape for compatibility ([legend.go#L47](legend.go#L47), [legend.go#L51](legend.go#L51), [legend.go#L63](legend.go#L63)).
- The shared sizing constants live alongside the data so measurement and rendering cannot disagree ([legend.go#L89](legend.go#L89), [../legendlayout/layout.go#L24](../legendlayout/layout.go#L24)).

**Related operations.**

- `legend.Config.toLegendData` builds it; `legendlayout.MeasureLegend`/`ReserveSpace` size it; the legend renderer draws it ([../../legend/config.go#L78](../../legend/config.go#L78), [../legendlayout/layout.go#L24](../legendlayout/layout.go#L24), [../../legend/render.go#L51](../../legend/render.go#L51)).

**Proper-use patterns.**

- Produce swatches from the ink that actually coloured the visualization, so the legend and the drawing cannot drift ([../../inks/legend_data.go#L27](../../inks/legend_data.go#L27)).
- Use `LegendLabelSampleShape` to preserve the visualization’s native label context — square, circle, or arc — without exposing visualization types to layout or rendering ([legend.go#L51](legend.go#L51), [../../legend/config.go#L109](../../legend/config.go#L109)).

**Anti-patterns.**

- Do not re-derive legend geometry with local constants; measurement, reservation, and drawing all read the constants defined here ([legend.go#L89](legend.go#L89)).

**Source locations.**

- [legend.go#L44](legend.go#L44) — `LegendData`, `LegendEntryData`, `LegendSwatch`, and the layout constants.
- [../legendlayout/helpers_test.go#L43](../legendlayout/helpers_test.go#L43) — shared sample-measurement behavior.
