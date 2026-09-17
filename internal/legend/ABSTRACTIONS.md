# Abstractions — `internal/legend`

## Entry

**Purpose.**

- `Entry` is one metric's presence in the legend: the visual `Role` it explains (fill, border, size, surface), the metric's name, and the ink that coloured it ([config.go#L13](config.go#L13), [config.go#L23](config.go#L23)).

**Boundary and invariants.**

- The entry references the *same* ink object the visualization drew with, so swatches are derived from the actual colouring rather than recomputed ([config.go#L27](config.go#L27), [config.go#L86](config.go#L86)).
- `Role` is a closed vocabulary and doubles as the displayed label; the border role additionally changes how swatches are drawn ([config.go#L16](config.go#L16), [config.go#L88](config.go#L88)).
- A size entry has no colour meaning, so it is given a fixed white ink ([legend.go#L64](legend.go#L64)).

**Related operations.**

- `Builder` assembles the standard fill/border/size entries plus any visualization-specific ones ([legend.go#L27](legend.go#L27), [legend.go#L41](legend.go#L41)).

**Proper-use patterns.**

- Append visualization-specific entries through `Builder.AdditionalEntries` rather than constructing a `Config` by hand ([legend.go#L36](legend.go#L36), [../spiral/stages.go#L145](../spiral/stages.go#L145)).

**Anti-patterns.**

- Do not build an entry with a freshly constructed ink; reuse the ink from the visualization's state so the legend matches the drawing ([../treemap/stages.go#L57](../treemap/stages.go#L57)).

**Source locations.**

- [config.go#L14](config.go#L14) — `Role` and its constants.
- [config.go#L24](config.go#L24) — `Entry`.

## Config

**Purpose.**

- `Config` is everything needed to render a legend overlay: where it goes, how it is oriented, an optional inline label sample, and its entries ([config.go#L48](config.go#L48), [config.go#L49](config.go#L49)).
- It is the boundary between visualization state and the backend-facing `model.LegendData` ([config.go#L76](config.go#L76)).

**Boundary and invariants.**

- "No legend" is represented by a nil `*Config` or by conversion returning nil — position `none` or zero entries both mean nothing is drawn or reserved ([legend.go#L42](legend.go#L42), [config.go#L79](config.go#L79), [render.go#L17](render.go#L17)).
- Orientation is optional: an empty value is resolved from the position, with top-center and bottom-center defaulting to horizontal ([config.go#L56](config.go#L56), [config.go#L96](config.go#L96)).
- Space reservation and rendering both go through `toLegendData`, so the space reserved matches what is drawn ([config.go#L70](config.go#L70), [render.go#L21](render.go#L21)).
- `ReserveLayout` returns content dimensions and their matching offset together; if reservation would make the content too small, both fall back to the full area and zero offset ([reserve.go#L14](reserve.go#L14), [reserve.go#L24](reserve.go#L24)).

**Related operations.**

- `Builder.Build` constructs it, `ResolveOptions` turns raw CLI/config strings into position and orientation, `ReserveLayout` sizes and positions content around it, and `RenderInto` decomposes it into canvas primitives ([legend.go#L41](legend.go#L41), [legend.go#L13](legend.go#L13), [reserve.go#L24](reserve.go#L24), [render.go#L16](render.go#L16)).

**Proper-use patterns.**

- Resolve options, build the config in a stage, then attach the label sample if the visualization draws inline labels ([../treemap/stages.go#L50](../treemap/stages.go#L50), [../treemap/stages.go#L61](../treemap/stages.go#L61)).
- Reserve legend space before laying out content, and apply the returned offset to that layout before rendering the same config ([../treemap/stages.go#L76](../treemap/stages.go#L76), [../treemap/stages.go#L80](../treemap/stages.go#L80)).

**Anti-patterns.**

- Do not guard call sites with ad-hoc "legend enabled" flags; passing a nil config is the supported way to disable it ([render.go#L17](render.go#L17)).

**Source locations.**

- [config.go#L49](config.go#L49) — `Config`, `DefaultOrientation`, `ReserveSpace`, `toLegendData`.
- [reserve.go#L14](reserve.go#L14) — `Reservation` and `ReserveLayout`.
- [legend.go#L28](legend.go#L28) — `Builder` and `Build`.
