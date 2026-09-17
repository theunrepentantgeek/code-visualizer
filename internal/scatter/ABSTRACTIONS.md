# Abstractions — `internal/scatter`

## AxisSpec

**Purpose.**

- `AxisSpec` is the *request* for one axis: which metric it plots, that metric's kind, and the scale to map it with ([axis.go#L7](axis.go#L7), [axis.go#L8](axis.go#L8)).
- It is the single parameter threaded from CLI resolution through data collection into layout ([stages.go#L115](stages.go#L115), [data.go#L96](data.go#L96), [layout.go#L38](layout.go#L38)).

**Boundary and invariants.**

- `Kind` decides the axis's whole shape: `metric.Classification` means categorical bands, anything else means a numeric range ([axis_resolve.go#L29](axis_resolve.go#L29), [data.go#L169](data.go#L169)).
- `Scale` only applies to numeric axes, and `Log` additionally requires strictly positive data, validated before layout ([resolved_axis.go#L3](resolved_axis.go#L3), [stages.go#L209](stages.go#L209)).

**Related operations.**

- `resolveAxisSpec` builds one from CLI flags; `resolveAxis` turns it into a `ResolvedAxis` ([stages.go#L115](stages.go#L115), [axis_resolve.go#L27](axis_resolve.go#L27)).

**Proper-use patterns.**

- Read a node's axis value through `axisValueForContainer`, which switches on the spec's kind rather than probing metric storage directly ([data.go#L164](data.go#L164), [data.go#L169](data.go#L169)).

**Anti-patterns.**

- Do not infer an axis's kind from the values encountered; the spec is authoritative and a mismatch silently drops points ([data.go#L169](data.go#L169), [data.go#L142](data.go#L142)).

**Source locations.**

- [axis.go#L8](axis.go#L8) — `AxisSpec`, with `AxisValue`, `AxisTick`, `AxisBand`.
- [stages.go#L115](stages.go#L115) — CLI resolution into a spec.

## ResolvedAxis

**Purpose.**

- `ResolvedAxis` is the layout-ready form of an axis: the originating spec, a display title, and exactly one of a numeric range with ticks or a set of categorical bands ([resolved_axis.go#L30](resolved_axis.go#L30), [resolved_axis.go#L31](resolved_axis.go#L31)).

**Boundary and invariants.**

- `Numeric` and `Categorical` are alternatives, chosen by the spec's kind; both are pointers so "not this kind" is representable ([axis_resolve.go#L29](axis_resolve.go#L29), [axis_resolve.go#L38](axis_resolve.go#L38)).
- Tick and band positions are absolute canvas coordinates, which is why moving a layout must offset the axes too ([axis.go#L20](axis.go#L20), [axis.go#L27](axis.go#L27), [resolved_axis.go#L58](resolved_axis.go#L58)).
- `Centers` is an optional O(1) index over bands; code must still work when it is nil, as it is for hand-built axes ([resolved_axis.go#L22](resolved_axis.go#L22), [axis_resolve.go#L306](axis_resolve.go#L306)).

**Related operations.**

- `NumericTicks`, `CategoricalBands`, and `Offset` are nil-safe accessors used by rendering and by layout translation ([resolved_axis.go#L40](resolved_axis.go#L40), [resolved_axis.go#L50](resolved_axis.go#L50), [resolved_axis.go#L61](resolved_axis.go#L61)).

**Proper-use patterns.**

- Iterate through `NumericTicks`/`CategoricalBands` rather than dereferencing `Numeric`/`Categorical`, so the other axis kind is simply an empty loop ([render.go#L72](render.go#L72), [render.go#L156](render.go#L156)).

**Anti-patterns.**

- Do not shift point positions without calling `Offset` on both axes; ticks and bands would be left behind ([layout.go#L86](layout.go#L86), [layout.go#L88](layout.go#L88)).

**Source locations.**

- [resolved_axis.go#L31](resolved_axis.go#L31) — `ResolvedAxis`, `NumericAxis`, `CategoricalAxis`, `Offset`.
- [axis_resolve.go#L27](axis_resolve.go#L27) — resolution from a spec plus data.

## Dataset

**Purpose.**

- `Dataset` is the plottable subset of the model: one `PointDatum` per node that has all of X, Y, and size, together with counts of what was dropped ([data.go#L64](data.go#L64), [data.go#L65](data.go#L65), [data.go#L15](data.go#L15)).

**Boundary and invariants.**

- A node is kept only if all three values resolve; any missing value increments the matching skip counter and excludes the node entirely ([data.go#L143](data.go#L143), [data.go#L154](data.go#L154)).
- A `PointDatum` refers to either a file or a directory, never both, per the selected grain ([data.go#L16](data.go#L16), [data.go#L116](data.go#L116), [data.go#L121](data.go#L121)).
- `SkipCounts.Total` double-counts nodes missing more than one value, so it is an upper bound rather than a distinct-node count ([data.go#L57](data.go#L57), [data.go#L60](data.go#L60)).

**Related operations.**

- `CollectDataset` builds it by walking at the chosen grain; `Files` and `metricSources` project it for ink building and legends ([data.go#L96](data.go#L96), [data.go#L71](data.go#L71), [data.go#L82](data.go#L82)).

**Proper-use patterns.**

- Build inks and layout from the dataset, not from the model root, so colour scales cover exactly the plotted nodes ([stages.go#L194](stages.go#L194), [stages.go#L200](stages.go#L200)).
- Report `Skipped` to the user rather than silently losing nodes ([stages.go#L290](stages.go#L290), [stages.go#L296](stages.go#L296)).

**Anti-patterns.**

- Do not read `Directory` or `File` without a nil check; the accessors exist because only one is set ([data.go#L25](data.go#L25), [data.go#L37](data.go#L37)).

**Source locations.**

- [data.go#L65](data.go#L65) — `Dataset`, `PointDatum`, `SkipCounts`.
- [data.go#L136](data.go#L136) — the all-or-nothing collection rule.

## ScatterLayout

**Purpose.**

- `ScatterLayout` is the finished plot geometry: the plot rectangle, both resolved axes, and one positioned disc per plotted node ([layout.go#L29](layout.go#L29), [layout.go#L30](layout.go#L30), [layout.go#L21](layout.go#L21)).

**Boundary and invariants.**

- The plot rect is inset from the canvas by fixed margins that leave room for axis labels; points are positioned inside it ([layout.go#L39](layout.go#L39), [layout.go#L42](layout.go#L42)).
- All coordinates are absolute, and translation must go through `OffsetLayout` so plot, axes, and points move together ([layout.go#L86](layout.go#L86), [stages.go#L272](stages.go#L272)).

**Related operations.**

- `Layout` builds it from a `Dataset` and two specs; `RenderToCanvas` draws it ([layout.go#L38](layout.go#L38), [render.go#L23](render.go#L23)).

**Proper-use patterns.**

- Map a datum to a position with `positionForValue` against the resolved axis, so numeric scaling and categorical banding share one path ([layout.go#L63](layout.go#L63), [axis_resolve.go#L284](axis_resolve.go#L284)).

**Anti-patterns.**

- Do not translate points directly for layout offsets; that would desynchronise the axes and the plot rect ([layout.go#L87](layout.go#L87), [layout.go#L91](layout.go#L91)).

**Source locations.**

- [layout.go#L30](layout.go#L30) — `ScatterLayout`, `ScatterPoint`.
- [layout.go#L86](layout.go#L86) — `OffsetLayout`.
