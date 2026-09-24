# Abstractions — `internal/alluvial`

## Data

**Purpose.**

- `Data` is the renderer-independent release comparison: caller-ordered snapshot
  columns, path-aligned transitions between adjacent columns, and one colour
  value per represented directory ([data.go#L14](data.go#L14),
  [data.go#L52](data.go#L52)).
- Directory identity is the repository-relative path carried by both `Value`
  and `Transition` ([data.go#L28](data.go#L28),
  [data.go#L35](data.go#L35)).

**Boundary and invariants.**

- Column order is snapshot order, while values and transition paths are sorted
  for deterministic output ([data.go#L62](data.go#L62),
  [data.go#L102](data.go#L102), [data.go#L218](data.go#L218)).
- Transitions exist only between adjacent columns and contain the union of
  their paths; an absent endpoint has width zero, which represents an
  introduction or removal ([data.go#L34](data.go#L34),
  [data.go#L189](data.go#L189)).
- Width is always the selected metric's value in that snapshot, never a
  release-to-release delta ([data.go#L50](data.go#L50),
  [data_test.go#L28](data_test.go#L28)).
- Fill values are independent of widths: they come from the last snapshot, or
  from last-minus-first when delta fill is selected
  ([data.go#L73](data.go#L73), [data.go#L164](data.go#L164),
  [data_test.go#L119](data_test.go#L119)).
- Expanding a directory replaces it with its direct children; expansion does
  not recurse unless a child is also explicitly named
  ([data.go#L108](data.go#L108), [data_test.go#L81](data_test.go#L81)).

**Related operations.**

- `BuildData` constructs the model from `Snapshot` values and `Options`;
  `LayoutData` converts it into drawable geometry
  ([data.go#L52](data.go#L52), [state.go#L59](state.go#L59),
  [state.go#L65](state.go#L65), [layout.go#L51](layout.go#L51)).

**Proper-use patterns.**

- Resolve directory-level metrics and finish every snapshot before building
  data, then use the resulting `Data` as the sole input to layout
  ([pipeline.go#L181](pipeline.go#L181), [pipeline.go#L263](pipeline.go#L263),
  [pipeline.go#L81](pipeline.go#L81)).
- Preserve repository paths as the join key across releases; display labels
  and slice positions are not stable identities ([data.go#L28](data.go#L28),
  [data.go#L204](data.go#L204)).

**Anti-patterns.**

- Do not derive widths from adjacent-reference changes; delta is a colour
  option only and does not alter column geometry
  ([data.go#L50](data.go#L50), [data.go#L79](data.go#L79)).
- Do not synthesize an “other” directory or automatically recurse into large
  directories; visible hierarchy is selected only by explicit expansion
  ([data.go#L108](data.go#L108),
  [../../docs/plans/2026-09-19-2051-feat-alluvial-diagram-plan.md#scope-boundaries](../../docs/plans/2026-09-19-2051-feat-alluvial-diagram-plan.md#scope-boundaries)).

**Source locations.**

- [data.go#L14](data.go#L14) — `Data`, `Column`, `Value`, and `Transition`.
- [data.go#L52](data.go#L52) — deterministic construction and fill semantics.
- [data_test.go#L28](data_test.go#L28) — ordering and transition contract.

## Layout

**Purpose.**

- `Layout` is the drawable alluvial geometry: positioned release columns,
  metric-proportional directory bands, and the flows connecting matching
  paths ([layout.go#L15](layout.go#L15), [layout.go#L24](layout.go#L24),
  [layout.go#L41](layout.go#L41)).

**Boundary and invariants.**

- Every column uses one shared scale based on the largest column total, making
  band heights comparable across releases ([layout.go#L68](layout.go#L68),
  [layout.go#L177](layout.go#L177),
  [layout_test.go#L38](layout_test.go#L38)).
- Active paths receive zero-width placeholders in every column before bands
  are laid out, preserving path order and vertical gaps across releases
  ([layout.go#L60](layout.go#L60), [layout.go#L97](layout.go#L97),
  [layout_test.go#L94](layout_test.go#L94)).
- Only positive finite widths establish active paths. A one-sided transition
  becomes a flow whose missing endpoint has zero height; a transition absent
  at both ends is omitted ([layout.go#L104](layout.go#L104),
  [layout.go#L136](layout.go#L136), [layout.go#L258](layout.go#L258)).
- Coordinates are absolute canvas coordinates after `LayoutStage`; columns,
  bands, flow endpoints, and vertical bounds must all be offset together
  ([pipeline.go#L81](pipeline.go#L81), [pipeline.go#L106](pipeline.go#L106)).

**Related operations.**

- `LayoutData` constructs geometry from `Data`; `RenderToCanvas` turns its
  bands and valid flows into shared-canvas shapes
  ([layout.go#L51](layout.go#L51), [render.go#L23](render.go#L23)).

**Proper-use patterns.**

- Reserve title, footer, and legend space before laying out, then offset the
  complete layout into the remaining drawing bounds
  ([pipeline.go#L69](pipeline.go#L69), [pipeline.go#L81](pipeline.go#L81)).
- Render introduction and removal flows from their zero-height endpoints
  rather than inventing off-canvas origins
  ([layout.go#L148](layout.go#L148), [render.go#L47](render.go#L47)).

**Anti-patterns.**

- Do not scale each column independently; doing so would make equal band
  heights represent different metric values
  ([layout.go#L68](layout.go#L68), [layout_test.go#L38](layout_test.go#L38)).
- Do not render a flow with reversed columns, inverted bands, two zero-height
  endpoints, or non-finite coordinates
  ([render.go#L261](render.go#L261)).

**Source locations.**

- [layout.go#L15](layout.go#L15) — `Layout`, `ColumnLayout`, `Band`, and `Flow`.
- [layout.go#L51](layout.go#L51) — shared-scale geometry construction.
- [render.go#L23](render.go#L23) — layout consumption.
