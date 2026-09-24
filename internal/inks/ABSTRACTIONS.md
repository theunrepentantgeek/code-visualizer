# Abstractions — `internal/inks`

## Ink

**Purpose.**

- `Ink` is the abstraction that turns a metric value into colour: a solid colour for a shape (`Dip`), a full fill specification (`Fill`), introspection (`Info`), and the legend swatches that explain it (`LegendData`) ([ink.go#L1](ink.go#L1), [ink.go#L27](ink.go#L27)).
- Because one object answers both "what colour is this value" and "what does the legend look like", the drawing and its legend cannot disagree ([ink.go#L30](ink.go#L30), [legend_data.go#L27](legend_data.go#L27)).

**Boundary and invariants.**

- Three concrete behaviours exist — fixed, numeric (bucketed through `metric.BucketBoundaries`), and categorical — and `Info().Kind` is how callers tell them apart without type assertions ([ink.go#L14](ink.go#L14), [introspection.go#L10](introspection.go#L10)).
- Inks are built from the whole dataset, not per shape: numeric inks bucket all values against the palette size, categorical inks take the distinct categories ([ink.go#L73](ink.go#L73), [ink.go#L88](ink.go#L88)).
- An unknown metric or an empty value set degrades to a fixed fallback colour rather than failing, and opacity is fixed at construction rather than per shape ([inks.go#L31](inks.go#L31), [options.go#L16](options.go#L16)).

**Related operations.**

- `FixedInk`, `NumericInk`, `CategoricalInk` construct; `BuildMetricInk` picks the right constructor from a metric descriptor ([ink.go#L57](ink.go#L57), [inks.go#L17](inks.go#L17)).
- `BuildDirectoryMetricInk` performs the same construction from directory values; `NumericBreakpoints` exposes a defensive copy for rendering that must align with an ink’s numeric buckets ([inks.go#L108](inks.go#L108), [introspection.go#L30](introspection.go#L30)).
- `ShapeInks` pairs a fill ink with a border ink for visualization packages to embed ([ink.go#L163](ink.go#L163), [../treemap/inks.go#L41](../treemap/inks.go#L41)).

**Proper-use patterns.**

- Build inks once per run from the model tree and the resolved descriptor, then hand them to canvas shape specs ([inks.go#L17](inks.go#L17), [../canvas/spec.go#L9](../canvas/spec.go#L9)).
- Derive each shape's `MetricValue` through `MetricValueForFile`/`MetricValueForDirectory`, which consult the ink's own metric ([inks.go#L46](inks.go#L46)).
- Use `BuildDirectoryMetricInk` for directory shapes so the buckets and categories come from the same population that will be rendered ([inks.go#L108](inks.go#L108), [../donuttree/inks.go#L35](../donuttree/inks.go#L35)).

**Anti-patterns.**

- Do not build a separate legend colour scheme; ask the ink for its `LegendData` ([legend_data.go#L27](legend_data.go#L27), [../legend/config.go#L78](../legend/config.go#L78)).
- Do not type-assert to the concrete ink types to find out what an ink does; use `Info()` ([introspection.go#L15](introspection.go#L15)).

**Source locations.**

- [ink.go#L27](ink.go#L27) — `Ink`, `Kind`, and the three constructors.
- [inks.go#L17](inks.go#L17) — file/directory ink builders and metric-value helpers.
- [introspection.go#L10](introspection.go#L10) — stable kind/metric introspection and numeric breakpoints.

## MetricValue

**Purpose.**

- `MetricValue` is the self-describing value handed to an ink: a `metric.Kind` plus the one field that kind uses ([metric_value.go#L7](metric_value.go#L7), [metric_value.go#L9](metric_value.go#L9)).

**Boundary and invariants.**

- `Kind` selects which field is meaningful; the others are ignored ([metric_value.go#L8](metric_value.go#L8), [ink.go#L114](ink.go#L114)).
- The zero value means "no value for this metric", which is what `MetricValueForFile` returns for a nil file, a fixed ink, or a missing metric ([inks.go#L43](inks.go#L43), [inks.go#L64](inks.go#L64)).
- Shapes carry the metric value, not a resolved colour, so resolution happens at draw time ([../canvas/rectangle.go#L13](../canvas/rectangle.go#L13), [../canvas/rectangle.go#L19](../canvas/rectangle.go#L19)).

**Related operations.**

- `MeasureValue`, `QuantityValue`, `CategoryValue` construct one directly; `MetricValueForFile`/`MetricValueForDirectory` derive one from a model node and an ink ([metric_value.go#L17](metric_value.go#L17), [inks.go#L46](inks.go#L46)).

**Proper-use patterns.**

- Use the kind-specific constructors instead of populating the struct literally, so the kind and the field always agree ([metric_value.go#L17](metric_value.go#L17)).

**Anti-patterns.**

- Do not set several value fields "just in case"; only the one matching `Kind` is read ([ink.go#L114](ink.go#L114)).

**Source locations.**

- [metric_value.go#L9](metric_value.go#L9) — `MetricValue` and its constructors.

## RadialGradientInk

**Purpose.**

- `RadialGradientInk` is a decorator that gives any ink a pincushion-shaded appearance: the inner ink supplies the centre colour and the edge is a darkened version of it ([radial_gradient.go#L11](radial_gradient.go#L11), [radial_gradient.go#L28](radial_gradient.go#L28)).

**Boundary and invariants.**

- It only changes `Fill`: `Dip`, `Info`, and `LegendData` delegate to the inner ink, so shading never alters the legend or introspection ([radial_gradient.go#L24](radial_gradient.go#L24), [radial_gradient.go#L38](radial_gradient.go#L38)).
- The darkening fraction is fixed at 40% by the constructor ([radial_gradient.go#L9](radial_gradient.go#L9), [radial_gradient.go#L19](radial_gradient.go#L19)).
- The focus point comes from the shape being drawn, not from the ink, so the same ink shades every shape relative to its own geometry ([radial_gradient.go#L28](radial_gradient.go#L28), [../canvas/rectangle.go#L19](../canvas/rectangle.go#L19)).

**Related operations.**

- `NewRadialGradientInk` wraps an existing ink; the resulting `model.RadialGradientFill` is what backends render ([radial_gradient.go#L20](radial_gradient.go#L20), [../canvas/model/fill.go#L21](../canvas/model/fill.go#L21)).

**Proper-use patterns.**

- Wrap the fill ink at the stage that knows whether flat rendering was requested, leaving the rest of the pipeline unchanged ([../treemap/stages.go#L42](../treemap/stages.go#L42), [../bubbletree/stages.go#L50](../bubbletree/stages.go#L50)).

**Anti-patterns.**

- Do not implement per-visualization shading; wrapping the ink keeps shading orthogonal to how colours are chosen ([radial_gradient.go#L11](radial_gradient.go#L11)).

**Source locations.**

- [radial_gradient.go#L14](radial_gradient.go#L14) — `RadialGradientInk`, `NewRadialGradientInk`.

## ShapeInks

**Purpose.**

- `ShapeInks` is the shared fill-and-border pair embedded or aliased by visualization-specific ink state, avoiding incompatible copies of the same styling vocabulary ([ink.go#L163](ink.go#L163), [../treemap/inks.go#L39](../treemap/inks.go#L39), [../radialtree/inks.go#L22](../radialtree/inks.go#L22)).

**Boundary and invariants.**

- It groups roles only; each member remains a complete `Ink` with its own metric, palette, opacity, fill behavior, and legend data ([ink.go#L165](ink.go#L165)).
- Visualization packages may embed it and add role-specific state, but `ShapeInks` itself does not say whether either role is metric-driven ([../bubbletree/inks.go#L21](../bubbletree/inks.go#L21), [../scatter/inks.go#L25](../scatter/inks.go#L25)).

**Related operations.**

- Visualization ink builders populate the pair, canvas `ShapeStyle` consumes it, and legend builders reuse the same two inks ([../donuttree/inks.go#L31](../donuttree/inks.go#L31), [../canvas/spec.go#L7](../canvas/spec.go#L7), [../legend/legend.go#L28](../legend/legend.go#L28)).

**Proper-use patterns.**

- Alias `ShapeInks` when fill and border are the complete visualization state; embed it when the visualization needs additional inks or flags ([../spiral/inks.go#L19](../spiral/inks.go#L19), [../radialtree/inks.go#L22](../radialtree/inks.go#L22)).

**Anti-patterns.**

- Do not duplicate fill/border records per visualization or infer metric semantics from whether a field is non-nil; inspect each ink through `Info()` ([ink.go#L163](ink.go#L163), [introspection.go#L15](introspection.go#L15)).

**Source locations.**

- [ink.go#L163](ink.go#L163) — `ShapeInks`.
- [../donuttree/render_test.go#L160](../donuttree/render_test.go#L160) — embedded pair used in rendering tests.
