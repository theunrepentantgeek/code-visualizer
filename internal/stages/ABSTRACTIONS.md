# Abstractions — `internal/stages`

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
