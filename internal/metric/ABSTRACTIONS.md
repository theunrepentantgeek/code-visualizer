# Abstractions — `internal/metric`

## Name

**Purpose.**

- `Name` is the identity of a metric — the key under which a value is stored and the token users type on the command line or in config ([metric.go#L6](metric.go#L6)).

**Boundary and invariants.**

- Provider packages own the constants; this package deliberately defines none, so the vocabulary stays open ([metric.go#L5](metric.go#L5), [../provider/git/base_metrics.go#L26](../provider/git/base_metrics.go#L26)).
- A `Name` may be a bare base metric or the canonical spelling of a metric expression. For snapshot computation, use `ResolvedMetric.ResultName`: it omits the temporal suffix, while `MetricExpression.ResultName()` preserves the entire expression ([expression.go](expression.go), [../provider/resolution.go](../provider/resolution.go)).
- Names are the key type for every metric map on a model node ([../model/metric_container.go#L12](../model/metric_container.go#L12)).

**Related operations.**

- `ParseExpression` parses a name into its parts; `provider.ResolveName` validates it and reports the kind it produces ([expression.go#L24](expression.go#L24), [../provider/resolution.go#L34](../provider/resolution.go#L34)).

**Proper-use patterns.**

- Take `metric.Name` (not `string`) in APIs that read or write metric values, including CLI flag fields ([../model/metric_container.go#L20](../model/metric_container.go#L20), [../../cmd/codeviz/treemap_cmd.go#L21](../../cmd/codeviz/treemap_cmd.go#L21)).
- Resolve a user-supplied name through `provider.ResolveName` before using it, so config validation, CLI validation, and the pipeline agree on its meaning ([../provider/resolution.go#L28](../provider/resolution.go#L28)).

**Anti-patterns.**

- Do not invent a private naming scheme for aggregated results; use the resolved result name. A temporal comparison reads underlying snapshot values rather than requesting a provider value named with `.delta` or `.stepdelta` ([../provider/resolution.go](../provider/resolution.go), [../alluvial/pipeline.go](../alluvial/pipeline.go)).

**Source locations.**

- [metric.go#L6](metric.go#L6) — `Name`.
- [../model/metric_container.go](../model/metric_container.go) and [../provider/resolution.go](../provider/resolution.go) — storage keys and request validation.

## Kind

**Purpose.**

- `Kind` says what *type* of value a metric carries: `Quantity` (integers), `Measure` (floats), or `Classification` (strings) ([metric.go#L9](metric.go#L9), [metric.go#L11](metric.go#L11)).
- It is the single switch that decides numeric versus categorical handling all the way through storage, colouring, and legends ([../inks/inks.go#L29](../inks/inks.go#L29), [../canvas/model/legend.go#L34](../canvas/model/legend.go#L34)).

**Boundary and invariants.**

- The three kinds map one-to-one onto the three metric maps on a model node ([../model/metric_container.go#L12](../model/metric_container.go#L12)).
- Kind is not fixed by the base metric alone: aggregation can change it (for example `.mean` over quantities yields a `Measure`) ([../provider/resolution.go#L158](../provider/resolution.go#L158)).
- Each known kind has a conventional aggregation — quantity uses sum, measure uses mean, and classification uses mode — while an unknown kind has no default ([metric.go#L18](metric.go#L18)).

**Related operations.**

- `DefaultAggregation` returns the kind's conventional aggregation and an `ok` result; `provider.BaseMetricDescriptor.Kind` declares a base metric's kind, while `ResolvedMetric.ResultKind` reports the kind after aggregation ([metric.go#L18](metric.go#L18), [../provider/base_descriptor.go#L25](../provider/base_descriptor.go#L25), [../provider/resolution.go#L12](../provider/resolution.go#L12)).

**Proper-use patterns.**

- Branch on kind when building an ink: numeric kinds get bucketed colours, classifications get categorical mapping ([../inks/inks.go#L29](../inks/inks.go#L29)).
- Carry the kind alongside the value when passing a metric value to the colour layer ([../inks/metric_value.go#L9](../inks/metric_value.go#L9)).
- Use `DefaultAggregation` when a metric must cross levels without an explicit aggregation, then let provider resolution validate that the metric supports it ([../donuttree/stages.go#L68](../donuttree/stages.go#L68), [../provider/resolution.go#L126](../provider/resolution.go#L126)).

**Anti-patterns.**

- Do not infer kind from a value's Go type at the call site; ask the descriptor or the resolved metric, because aggregation may have changed it ([../provider/resolution.go#L158](../provider/resolution.go#L158)).

**Source locations.**

- [metric.go#L9](metric.go#L9) — `Kind` and its constants.
- [../model/metric_container.go](../model/metric_container.go) and [../inks/inks.go](../inks/inks.go) — storage and numeric/categorical handling.

## MetricLevel

**Purpose.**

- `MetricLevel` says where raw data for a metric lives in the model hierarchy: file, declaration, commit, or directory ([level.go#L4](level.go#L4), [level.go#L6](level.go#L6)).

**Boundary and invariants.**

- A base metric declares its *native* level; a request declares a *target* level, and the difference is exactly what makes aggregation necessary ([../provider/base_descriptor.go#L28](../provider/base_descriptor.go#L28), [../provider/resolution.go#L119](../provider/resolution.go#L119)).
- Aggregation is only meaningful across levels: an explicit aggregation on a metric already at the target level is rejected ([../provider/resolution.go#L136](../provider/resolution.go#L136)).
- `String()` produces the user-facing level word used in those error messages ([level.go#L14](level.go#L14), [../provider/resolution.go#L140](../provider/resolution.go#L140)).

**Related operations.**

- `provider.AllBaseForLevel` selects registered metrics by level; `ResolvedMetric` records both source and target level ([../provider/base_registry.go#L188](../provider/base_registry.go#L188), [../provider/resolution.go#L15](../provider/resolution.go#L15)).

**Proper-use patterns.**

- Choose the target level from the visualization's grain before resolving metric names ([../scatter/stages.go#L184](../scatter/stages.go#L184)).
- Use the level recorded on a resolved expression to decide whether declaration or commit data must be loaded at all ([../stages/requested.go#L20](../stages/requested.go#L20)).

**Anti-patterns.**

- Do not resolve every name at file level "to be safe" — that falsely rejects valid aggregation expressions, which is why validation picks the level from the expression ([../provider/resolution.go#L43](../provider/resolution.go#L43)).

**Source locations.**

- [level.go#L4](level.go#L4) — `MetricLevel` and its constants.
- [../provider/resolution.go](../provider/resolution.go) and [../scatter/stages.go](../scatter/stages.go) — source/target legality and grain-driven selection.

## AggregationName

**Purpose.**

- `AggregationName` names a roll-up function (`sum`, `min`, `max`, `mean`, `count`, `mode`, `distinct`, `range`) applied when lifting values from a finer level to a coarser one ([aggregation_name.go#L4](aggregation_name.go#L4), [aggregation_name.go#L7](aggregation_name.go#L7)).

**Boundary and invariants.**

- The set is closed: `IsKnown` checks membership of a fixed table, and expression parsing uses it to decide whether a trailing segment is an aggregation or part of the name ([aggregation_name.go#L36](aggregation_name.go#L36), [expression.go#L72](expression.go#L72)).
- `IsZero` means "no aggregation requested", distinct from an unknown one; and each base metric declares which aggregations it permits, so the name alone is never sufficient authority ([aggregation_name.go#L31](aggregation_name.go#L31), [../provider/base_descriptor.go#L62](../provider/base_descriptor.go#L62)).
- The `Aggregate*` implementations define the empty-input behaviour (`0` for numeric, `""` for mode) and the tie-break for `mode` (lexicographically first) ([aggregation.go#L18](aggregation.go#L18), [aggregation.go#L88](aggregation.go#L88)).

**Related operations.**

- `AggregateSum`, `AggregateMin`, `AggregateMax`, `AggregateMean`, `AggregateCount`, `AggregateRange` over floats; `AggregateMode` and `AggregateDistinct` over strings ([aggregation.go#L8](aggregation.go#L8), [aggregation.go#L88](aggregation.go#L88), [aggregation.go#L119](aggregation.go#L119)).

**Proper-use patterns.**

- Dispatch to the `Aggregate*` functions from a single place that also rejects unknown names, rather than inlining sums and means ([../stages/aggregation.go#L184](../stages/aggregation.go#L184)).
- List the metric's supported aggregations in its descriptor so validation can produce actionable errors ([../provider/base_descriptor.go#L31](../provider/base_descriptor.go#L31), [../provider/resolution.go#L126](../provider/resolution.go#L126)).

**Anti-patterns.**

- Do not treat an unknown trailing segment as an aggregation: three-segment expressions fail rather than guessing ([expression.go#L88](expression.go#L88)).

**Source locations.**

- [aggregation_name.go#L4](aggregation_name.go#L4) — `AggregationName`, constants, `IsKnown`.
- [aggregation.go#L8](aggregation.go#L8) — the aggregation implementations.
- [../provider/resolution.go](../provider/resolution.go) and [../stages/aggregation.go](../stages/aggregation.go) — permitted operations and computed-value storage.

## FilterName

**Purpose.**

- `FilterName` names a provider-defined subset of finer-grained nodes (for example `public` or `private` declarations) that a metric expression can restrict to ([filter.go#L4](filter.go#L4), [../model/declaration.go#L25](../model/declaration.go#L25)).

**Boundary and invariants.**

- The vocabulary belongs to the provider: `ProviderDescriptor.Filters` declares the names and their descriptions, and each base metric lists the ones it supports ([../provider/base_descriptor.go#L12](../provider/base_descriptor.go#L12), [../provider/base_descriptor.go#L68](../provider/base_descriptor.go#L68)).
- `IsZero` means "no filter", which is always valid; a non-empty name that the metric does not support is an error ([filter.go#L7](filter.go#L7), [../provider/resolution.go#L97](../provider/resolution.go#L97)).
- Evaluation is delegated to the descriptor's `FilterFunc`, which defaults to "passes" when absent ([../provider/base_descriptor.go#L52](../provider/base_descriptor.go#L52)).

**Related operations.**

- `MetricExpression.Filter` carries the requested filter; `Declaration.MatchesFilter` implements the declaration-level vocabulary ([expression.go#L12](expression.go#L12), [../model/declaration.go#L25](../model/declaration.go#L25)).

**Proper-use patterns.**

- Express a filtered metric as the leading segment of the expression (`public.declarations.count`) rather than as a separate flag ([expression.go#L24](expression.go#L24)).

**Anti-patterns.**

- Do not hard-code filter semantics in aggregation code; route the decision through the descriptor so providers stay in control of their vocabulary ([../provider/base_descriptor.go#L34](../provider/base_descriptor.go#L34)).

**Source locations.**

- [filter.go#L4](filter.go#L4) — `FilterName`.
- [../provider/base_descriptor.go](../provider/base_descriptor.go) and [../model/declaration.go](../model/declaration.go) — provider-owned vocabulary and declaration predicates.

## MetricExpression

**Purpose.**

- `MetricExpression` is the parsed form of a user-supplied metric string in the grammar `[filter.]base-metric[.aggregation][.temporal]` ([expression.go#L12](expression.go#L12), [expression.go#L25](expression.go#L25)).

**Boundary and invariants.**

- At most four dot-separated segments, each matching the lowercase identifier grammar. A recognized temporal modifier is allowed only at the end and is removed before parsing the remaining one-to-three core segments ([expression.go#L25](expression.go#L25), [expression_test.go](expression_test.go)).
- Ambiguity in a two-segment core is resolved by the known-aggregation table: a known trailing word is an aggregation, otherwise the leading word is a filter ([expression.go#L83](expression.go#L83)).
- Parsing is syntactic; existence, filter applicability, and aggregation legality belong to provider resolution. `String()` and `ResultName()` retain the temporal suffix; `WithoutTemporal()` produces the expression evaluated within each snapshot ([expression.go](expression.go), [../provider/resolution.go](../provider/resolution.go)).

**Related operations.**

- `ParseExpression` to build; `WithoutTemporal` to select snapshot computation; `provider.ResolveExpression` to validate the underlying metric and produce a `ResolvedMetric`. Ordinary `ResolveForValidation` explicitly rejects temporal requests ([expression.go](expression.go), [../provider/resolution.go](../provider/resolution.go)).

**Proper-use patterns.**

- Parse first, then resolve; treat a parse failure as "this is just a base metric name" only where that fallback is intended ([../stages/requested.go#L62](../stages/requested.go#L62)).
- Store aggregated values under `ResolvedMetric.ResultName`; retain the full expression separately when a visualization must compare those values across snapshots ([../stages/aggregation.go](../stages/aggregation.go), [../alluvial/pipeline.go](../alluvial/pipeline.go)).

**Anti-patterns.**

- Do not hand-split metric strings on `.`; filter/aggregation disambiguation and terminal temporal recognition live in `ParseExpression` ([expression.go](expression.go)).
- Do not mistake successful parsing or underlying provider resolution for authorization to use a temporal modifier in every visualization ([../provider/resolution.go#L57](../provider/resolution.go#L57), [../config/alluvial.go](../config/alluvial.go)).

**Source locations.**

- [expression.go#L12](expression.go#L12) — `MetricExpression`, parsing, canonical names, and `WithoutTemporal`.
- [../provider/resolution.go](../provider/resolution.go) and [../config/alluvial.go](../config/alluvial.go) — ordinary and ordered-snapshot validation.

## TemporalName

**Purpose.**

- `TemporalName` identifies a comparison across ordered snapshots, separate from aggregation within a snapshot ([temporal_name.go](temporal_name.go)).

**Boundary and invariants.**

- The recognized values are `delta` and `stepdelta`; the zero value means no comparison. Recognition does not establish that the selected visualization or metric role supports it ([temporal_name.go](temporal_name.go), [../provider/resolution.go#L57](../provider/resolution.go#L57)).
- In alluvial fill evaluation, `delta` is last minus first; `stepdelta` is each snapshot minus its predecessor, with no fill value at the first snapshot. Caller order, not date sorting, defines those relationships ([../alluvial/data.go#L211](../alluvial/data.go#L211), [../../docs/content/docs/visualizations/alluvial.md](../../docs/content/docs/visualizations/alluvial.md)).

**Related operations.**

- `IsZero`/`IsKnown` query the vocabulary; expression parsing recognizes a terminal modifier; `WithoutTemporal` and provider resolution select the underlying per-snapshot result ([temporal_name.go](temporal_name.go), [expression.go](expression.go), [../provider/resolution.go](../provider/resolution.go)).

**Proper-use patterns.**

- Validate role support in configuration, resolve the underlying numeric metric, then compare acquired snapshot values. Alluvial keeps the full fill label separate from the metric used for acquisition ([../config/alluvial.go](../config/alluvial.go), [../alluvial/pipeline.go#L77](../alluvial/pipeline.go#L77)).

**Anti-patterns.**

- Do not implement temporal comparison in a base provider or treat it as a directory roll-up: providers and aggregation compute each snapshot's values before the visualization compares them ([../provider/resolution.go](../provider/resolution.go), [../alluvial/data.go#L211](../alluvial/data.go#L211)).

**Source locations.**

- [temporal_name.go](temporal_name.go) and [expression_test.go](expression_test.go) — vocabulary and grammar.
- [../provider/resolution_test.go](../provider/resolution_test.go), [../config/alluvial.go](../config/alluvial.go), and [../alluvial/data.go](../alluvial/data.go) — resolution, role validation, and evaluation.

## BucketBoundaries

**Purpose.**

- `BucketBoundaries` holds the quantile breakpoints that map a set of numeric metric values onto a fixed number of palette steps, plus the observed min/max ([bucket.go#L9](bucket.go#L9), [bucket.go#L18](bucket.go#L18)).

**Boundary and invariants.**

- `NumBuckets()` is `len(Boundaries)+1`, and `BucketIndex` assigns a value to the first bucket whose boundary it falls below — boundaries are exclusive upper bounds ([bucket.go#L64](bucket.go#L64), [bucket.go#L69](bucket.go#L69)).
- Breakpoints are rounded to two significant figures and de-duplicated, so the actual bucket count may be smaller than the requested step count ([bucket.go#L18](bucket.go#L18), [bucket.go#L43](bucket.go#L43)).
- Breakpoints at or below the minimum are dropped (they would leave bucket 0 permanently empty), and degenerate input — empty, one step, or all values equal — yields no boundaries at all ([bucket.go#L46](bucket.go#L46), [bucket.go#L30](bucket.go#L30)).

**Related operations.**

- `ComputeBuckets` builds them from the full value set; numeric inks own one and use `BucketIndex`/`NumBuckets` to pick a palette colour ([bucket.go#L18](bucket.go#L18), [../inks/ink.go#L81](../inks/ink.go#L81), [../inks/ink.go#L121](../inks/ink.go#L121)).

**Proper-use patterns.**

- Compute boundaries once from the whole dataset with `steps` equal to the palette size, then reuse them for every value ([../inks/ink.go#L81](../inks/ink.go#L81)).
- Derive legend breakpoint labels from `Boundaries` so the legend and the colouring cannot drift apart ([../inks/legend_data.go#L32](../inks/legend_data.go#L32), [../inks/introspection.go#L31](../inks/introspection.go#L31)).

**Anti-patterns.**

- Do not assume `NumBuckets()` equals `StepCount`: deduplication and the minimum-boundary rule can collapse buckets ([bucket.go#L50](bucket.go#L50)).
- Do not derive colours by linear interpolation between `Min` and `Max`; the mapping is quantile-based by design ([bucket.go#L38](bucket.go#L38), [../palette/mapper.go#L10](../palette/mapper.go#L10)).

**Source locations.**

- [bucket.go#L9](bucket.go#L9) — `BucketBoundaries`, `ComputeBuckets`, `BucketIndex`.
- [../inks/ink.go](../inks/ink.go) and [../inks/legend_data.go](../inks/legend_data.go) — the same boundaries for drawing and legends.
