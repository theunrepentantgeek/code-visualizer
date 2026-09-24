# Abstractions — `internal/provider`

## ProviderDescriptor

**Purpose.**

- `ProviderDescriptor` is the shared identity of a metric provider package: its name and the filter vocabulary available to every metric it registers ([base_descriptor.go#L10](base_descriptor.go#L10), [base_descriptor.go#L12](base_descriptor.go#L12)).

**Boundary and invariants.**

- One descriptor per provider package, declared as a package-level value and reused for every registration from that package ([git/base_metrics.go#L10](git/base_metrics.go#L10), [git/base_metrics.go#L23](git/base_metrics.go#L23)).
- `Filters` names the filters the provider defines and documents them; a provider with no filters leaves it nil ([base_descriptor.go#L14](base_descriptor.go#L14), [git/base_metrics.go#L12](git/base_metrics.go#L12)).
- The association is recorded per metric at registration time and retrieved with `GetBaseProvider` ([base_registry.go#L45](base_registry.go#L45), [base_registry.go#L166](base_registry.go#L166)).

**Related operations.**

- `HasFilter` for vocabulary membership; `RegisterBaseWithProvider` to bind a metric to its provider ([base_descriptor.go#L18](base_descriptor.go#L18), [base_registry.go#L156](base_registry.go#L156)).

**Proper-use patterns.**

- Register a provider's metrics from a single exported `RegisterBase()` entry point rather than from scattered `init()` functions ([git/base_metrics.go#L16](git/base_metrics.go#L16)).

**Anti-patterns.**

- Do not declare a filter on an individual metric that its provider does not define; the provider owns the vocabulary and metrics select from it ([base_descriptor.go#L14](base_descriptor.go#L14), [base_descriptor.go#L30](base_descriptor.go#L30)).

**Source locations.**

- [base_descriptor.go#L12](base_descriptor.go#L12) — `ProviderDescriptor`, `HasFilter`.

## BaseMetricDescriptor

**Purpose.**

- `BaseMetricDescriptor` is the static metadata for one composable base metric: its name, kind, native level, description, the filters and aggregations it permits, its default palette, and how to evaluate a filter ([base_descriptor.go#L24](base_descriptor.go#L24), [base_descriptor.go#L25](base_descriptor.go#L25)).
- It is the authority every other layer consults — validation, resolution, ink building, and help output ([resolution.go#L70](resolution.go#L70), [../stages/requested.go#L38](../stages/requested.go#L38)).

**Boundary and invariants.**

- Describing is separate from computing: the descriptor carries no loading logic, which lives in a `BaseMetricLoader` ([base_descriptor.go#L24](base_descriptor.go#L24), [loader.go#L10](loader.go#L10)).
- Absent restrictions mean "no restriction": a nil `DeclKinds` matches all declaration kinds, and a nil `FilterFunc` passes everything ([base_descriptor.go#L42](base_descriptor.go#L42), [base_descriptor.go#L52](base_descriptor.go#L52)).
- Names are globally unique: registering the same metric name twice panics at startup ([base_registry.go#L30](base_registry.go#L30), [base_registry.go#L150](base_registry.go#L150)).

**Related operations.**

- `SupportsFilter`, `SupportsAggregation`, `MatchesDeclKind`, `PassesFilter` ([base_descriptor.go#L42](base_descriptor.go#L42), [base_descriptor.go#L62](base_descriptor.go#L62), [base_descriptor.go#L68](base_descriptor.go#L68)).
- Registry access: `RegisterBase`/`RegisterBaseWithProvider`, `GetBase`, `AllBase`, `AllBaseForLevel`, `BaseNames` — all sorted by name for stable output ([base_registry.go#L150](base_registry.go#L150), [base_registry.go#L104](base_registry.go#L104), [base_registry.go#L116](base_registry.go#L116)).

**Proper-use patterns.**

- Declare the metric's supported aggregations and a `DefaultPalette` at registration so validation errors can suggest alternatives and rendering has a sensible default ([git/base_metrics.go#L23](git/base_metrics.go#L23), [resolution.go#L126](resolution.go#L126)).
- Register configuration-derived metrics through the same API as built-in ones so they validate identically ([classification/provider.go#L33](classification/provider.go#L33)).

**Anti-patterns.**

- Do not synthesise descriptors ad hoc in rendering code; ask `RequestedMetrics.DescriptorFor`, which covers both registered and expression-derived metrics ([../stages/requested.go#L33](../stages/requested.go#L33)).

**Source locations.**

- [base_descriptor.go#L25](base_descriptor.go#L25) — `BaseMetricDescriptor` and its predicates.
- [base_registry.go#L146](base_registry.go#L146) — the process-wide registry and its exported accessors.

## BaseMetricLoader

**Purpose.**

- `BaseMetricLoader` is a unit of metric-loading work: the base metrics it populates, the metrics it must run after, the `LoadFunc` that does the work, and an optional progress reporter ([loader.go#L8](loader.go#L8), [loader.go#L10](loader.go#L10)).
- Separating loaders from descriptors lets one pass over the tree populate several metrics at once ([loader.go#L9](loader.go#L9)).

**Boundary and invariants.**

- A loader is only invoked for metrics that were actually requested: `LoadFunc` receives the subset of its own `Metrics` that the run needs ([loader.go#L21](loader.go#L21), [run.go#L113](run.go#L113)).
- `LoadersFor` selects by requested metric and preserves registration order; ordering by dependency is done by the runner, not the registry ([base_registry.go#L80](base_registry.go#L80), [base_registry.go#L175](base_registry.go#L175)).
- Dependencies are resolved by topological sort into levels that run in parallel; a dependency no selected loader provides is an error, not a silent skip ([run.go#L45](run.go#L45), [run.go#L192](run.go#L192)).

**Related operations.**

- `RegisterLoader`, `LoadersFor`, `RunLoaders`, and `FileProgressTotal` for progress estimation ([base_registry.go#L171](base_registry.go#L171), [run.go#L45](run.go#L45), [run.go#L31](run.go#L31)).

**Proper-use patterns.**

- Register the descriptor and the loader together for a metric family so a metric can never be resolvable but unloadable ([classification/provider.go#L33](classification/provider.go#L33)).
- Declare `Dependencies` instead of ordering loaders by registration order or sequencing them by hand ([loader.go#L13](loader.go#L13), [run.go#L184](run.go#L184)).
- Load file content through an attached `model.File` source when present; filesystem and Go loaders keep explicit OS-path fallbacks only for the current source-less `scan.Scan` compatibility boundary documented in [issue #755](https://github.com/theunrepentantgeek/code-visualizer/issues/755) ([../model/file.go#L27](../model/file.go#L27), [filesystem/metrics.go#L81](filesystem/metrics.go#L81), [golang/file_loader.go#L70](golang/file_loader.go#L70)).

**Anti-patterns.**

- Do not load every metric the loader knows about; honour the `requested` slice so unrequested work is not performed ([loader.go#L21](loader.go#L21)).
- Do not add a new unconditional `os.Open(File.Path)` path to a provider; it cannot read historical or virtual sources ([../source/gitfs.go#L39](../source/gitfs.go#L39), [../model/file.go#L27](../model/file.go#L27)).

**Source locations.**

- [loader.go#L10](loader.go#L10) — `BaseMetricLoader`, `LoadFunc`.
- [run.go#L45](run.go#L45) — dependency-ordered execution.

## ResolvedMetric

**Purpose.**

- `ResolvedMetric` is a fully validated metric request: the parsed expression, the descriptor it resolved to, its source and target levels, the kind and name of the result, and whether aggregation is needed ([resolution.go#L11](resolution.go#L11), [resolution.go#L12](resolution.go#L12)).

**Boundary and invariants.**

- Resolution is the only place where existence, filter applicability, and aggregation legality are checked; producing a `ResolvedMetric` means all three passed ([resolution.go#L75](resolution.go#L75), [resolution.go#L79](resolution.go#L79)).
- `ResultKind` may differ from the descriptor's kind, because some aggregations change it ([resolution.go#L84](resolution.go#L84), [resolution.go#L158](resolution.go#L158)).
- `ResultName` is the expression's canonical name, which is also the key the value is stored under ([resolution.go#L92](resolution.go#L92), [../metric/expression.go#L107](../metric/expression.go#L107)).

**Related operations.**

- `ResolveExpression` for a parsed expression, `ResolveName` as the canonical entry point for a user-supplied name, and `ResolveForValidation` for config and CLI validation ([resolution.go#L24](resolution.go#L24), [resolution.go#L34](resolution.go#L34), [resolution.go#L51](resolution.go#L51)).

**Proper-use patterns.**

- Validate user input with `ResolveForValidation`, which picks the target level from the expression so neither bare metrics nor aggregations are falsely rejected ([resolution.go#L43](resolution.go#L43), [../config/metric_spec.go#L117](../config/metric_spec.go#L117)).
- Resolve at the level the pipeline will aggregate to when classifying requested metrics for a run ([../stages/requested.go#L75](../stages/requested.go#L75)).

**Anti-patterns.**

- Do not re-derive a metric's kind or storage name downstream; carry the `ResolvedMetric` so config validation, the CLI, and the pipeline agree ([resolution.go#L28](resolution.go#L28)).

**Source locations.**

- [resolution.go#L12](resolution.go#L12) — `ResolvedMetric` and the three resolver entry points.
