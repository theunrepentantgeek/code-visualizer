# Abstractions — `internal/stages`

## CommonState

**Purpose.**

- `CommonState` is the shared run record threaded through visualization pipelines: orchestrator inputs, source and scan results, requested metrics, drawing state, and optional Git/authorship history ([common.go#L35](common.go#L35)).

**Boundary and invariants.**

- Fields are phase-owned rather than freely interchangeable: the orchestrator seeds inputs, shared/viz stages populate named outputs, and later stages consume them in pipeline order ([common.go#L36](common.go#L36), [../pipeline/ABSTRACTIONS.md](../pipeline/ABSTRACTIONS.md)).
- `Source` is resolved before scanning, `Root` is published only after a successful scan, and `ReferenceNow` follows the selected snapshot clock when historical content is used ([source.go#L14](source.go#L14), [scan.go#L12](scan.go#L12), [source.go#L72](source.go#L72)).
- Loaded `GitHistory` is written once and treated as immutable; per-file indexes and author history are derived run state, not alternate acquisition APIs ([common.go#L57](common.go#L57), [git_history.go#L79](git_history.go#L79)).

**Related operations.**

- `ResolveSource`, `ScanFilesystem`, `RunProviders`, history stages, aggregation stages, and canvas stages each fill or consume a defined portion of the state ([source.go#L14](source.go#L14), [scan.go#L12](scan.go#L12), [providers.go#L15](providers.go#L15), [compute_aggregations_stage.go#L8](compute_aggregations_stage.go#L8)).
- `Flags` is the package-local cross-cutting CLI bundle used without importing package `main` ([common.go#L20](common.go#L20)).

**Proper-use patterns.**

- Seed one `*CommonState` beside viz-specific state in `pipeline.State`, then express phase order as typed stage applications ([common.go#L31](common.go#L31), [../../cmd/codeviz/treemap_cmd.go#L84](../../cmd/codeviz/treemap_cmd.go#L84)).
- Reuse a resolved `source.Tree` across equivalent scans; `ScanFilesystem` resolves it only when `Source.FS` is absent ([scan.go#L16](scan.go#L16)).

**Anti-patterns.**

- Do not populate the same field from multiple unrelated stages or consume a field before its owner stage; `CommonState` has no runtime phase guard ([common.go#L35](common.go#L35)).
- Do not mutate shared Git commit records after loading; consumers retain pointers into the history slice ([common.go#L57](common.go#L57), [../provider/git/commit.go#L26](../provider/git/commit.go#L26)).

**Source locations.**

- [common.go#L20](common.go#L20) — cross-cutting `Flags`.
- [common.go#L35](common.go#L35) — `CommonState` and phase-owned fields.

## Aggregation engine

**Purpose.**

- The aggregation engine materializes resolved metric expressions from file-, declaration-, or commit-level source values onto the file/directory nodes that visualizations consume ([aggregation.go#L22](aggregation.go#L22), [compute_aggregations_stage.go#L8](compute_aggregations_stage.go#L8)).

**Boundary and invariants.**

- Every directory aggregate is source-grounded from all descendant source nodes, never folded from child aggregates; this preserves means, ranges, ratios, and filtered results ([aggregation.go#L47](aggregation.go#L47), [aggregation.go#L57](aggregation.go#L57)).
- Declaration and commit expressions first create per-file results, then independently compute each directory's result from descendant declarations or commits ([aggregation.go#L209](aggregation.go#L209), [aggregation.go#L375](aggregation.go#L375)).
- Missing source values produce no stored result, while result kind and storage name come from `ResolvedMetric` rather than being inferred locally ([aggregation.go#L75](aggregation.go#L75), [aggregation.go#L158](aggregation.go#L158)).

**Related operations.**

- `ComputeAggregations` dispatches by source level; metric package `Aggregate*` functions implement the actual reductions ([aggregation.go#L22](aggregation.go#L22), [aggregation.go#L176](aggregation.go#L176)).
- Filtered file-level metrics use the canonical `filter.base` lookup key before the terminal aggregation suffix is added to the result name ([aggregation.go#L12](aggregation.go#L12)).

**Proper-use patterns.**

- Pass only provider-resolved expressions after providers and optional declaration/commit loading have populated their source values ([compute_aggregations_stage.go#L8](compute_aggregations_stage.go#L8), [requested.go#L12](requested.go#L12)).
- Store results using `ResolvedMetric.ResultName` and `ResultKind` so downstream descriptor lookup and value access agree ([aggregation.go#L158](aggregation.go#L158), [requested.go#L33](requested.go#L33)).

**Anti-patterns.**

- Do not aggregate a directory from already aggregated child directories; aggregate-of-aggregates is incorrect for non-additive operations ([aggregation.go#L47](aggregation.go#L47)).
- Do not silently coerce unsupported source or result kinds; aggregation returns an error for invalid resolved input ([aggregation.go#L31](aggregation.go#L31), [aggregation.go#L158](aggregation.go#L158)).

**Source locations.**

- [aggregation.go#L12](aggregation.go#L12) — filtered lookup key.
- [aggregation.go#L22](aggregation.go#L22) — source-level dispatch and source-grounded aggregation.
- [compute_aggregations_stage.go#L8](compute_aggregations_stage.go#L8) — pipeline adapter.

## RequestedMetrics

**Purpose.**

- `RequestedMetrics` is the classified answer to "what did the user ask for": base metric names that providers load directly, and resolved expressions that need aggregation ([requested.go#L10](requested.go#L10), [requested.go#L12](requested.go#L12)).

**Boundary and invariants.**

- Classification is best-effort and never fails: a name that will not parse or will not resolve is kept as a base metric so later stages report the error in context ([requested.go#L68](requested.go#L68), [requested.go#L75](requested.go#L75)).
- An expression that does not actually need aggregation collapses to its base name, and only file-level sources are added to `BaseMetrics` because declaration- and commit-level data comes from separate stages ([requested.go#L82](requested.go#L82), [requested.go#L90](requested.go#L90)).
- `DescriptorFor` synthesises a descriptor for expression results, so the rendering layer can describe metrics such as `public.methods.count` that do not exist in the base registry ([requested.go#L33](requested.go#L33), [requested.go#L41](requested.go#L41)).

**Related operations.**

- `ClassifyRequestedMetrics` builds it from a flat name list at a target level; `CollectRequestedMetrics` is the convenience wrapper that gathers a viz's size and metric specs ([requested.go#L62](requested.go#L62), [metrics.go#L12](metrics.go#L12)).
- `HasDeclarationExpressions` and `HasCommitExpressions` let stages decide whether expensive declaration or commit loading is required at all ([requested.go#L20](requested.go#L20), [requested.go#L27](requested.go#L27)).

**Proper-use patterns.**

- Populate `CommonState.Requested` from the viz's own metric selection stage, then have ink-building ask it for descriptors ([../treemap/stages.go#L22](../treemap/stages.go#L22), [../treemap/inks.go#L56](../treemap/inks.go#L56)).
- Resolve at the level the visualization draws at — directory grain for tree maps, the axis level for scatter plots ([metrics.go#L23](metrics.go#L23), [../scatter/stages.go#L170](../scatter/stages.go#L170)).

**Anti-patterns.**

- Do not ask `provider.GetBase` directly for a metric that might be an expression result; `DescriptorFor` is the lookup that covers both ([requested.go#L38](requested.go#L38), [../spiral/inks.go#L68](../spiral/inks.go#L68)).

**Source locations.**

- [requested.go#L12](requested.go#L12) — `RequestedMetrics`, `DescriptorFor`, `ClassifyRequestedMetrics`.
- [metrics.go#L12](metrics.go#L12) — `CollectRequestedMetrics`.
