# Abstractions — `internal/stages`

## CommonState

**Purpose.**

- `CommonState` is the shared run state on which acquisition, metric, canvas, and output stages operate; visualization-specific layout and rendering state lives alongside it, not inside it ([common.go](common.go)).

**Boundary and invariants.**

- The orchestrator supplies inputs (`TargetPath`, output, merged root config, flags, filters, and visualization identity). Later stages populate fields in dependency order: source before scan, model before providers/aggregation, bounds before layout, canvas before output ([common.go](common.go), [../treemap/pipeline.go](../treemap/pipeline.go)).
- `Source` is the content/repository context; `Root` is the acquired model; `Canvas` is the retained drawing. They are different stages of the run, not interchangeable trees or surfaces ([common.go#L48](common.go#L48)).
- Loaded `GitHistory` is shared read-only after acquisition; consumers may retain pointers into it for the state's lifetime ([common.go#L61](common.go#L61)).

**Related operations.**

- `ResolveSource`, `ScanFilesystem`, provider/aggregation stages, drawing-bound reservations, and `WriteCanvas` populate or consume shared state. The type-keyed `pipeline.State` wires it together with the selected config section and visualization state ([../treemap/pipeline.go](../treemap/pipeline.go), [../../cmd/codeviz/treemap_cmd.go](../../cmd/codeviz/treemap_cmd.go)).

**Proper-use patterns.**

- Seed `*CommonState` and the separate visualization state once for ordinary runs. Alluvial creates reference-specific acquisition state and flags before scanning each snapshot, rather than treating one shared root as all revisions ([../../cmd/codeviz/treemap_cmd.go#L97](../../cmd/codeviz/treemap_cmd.go#L97), [../alluvial/pipeline.go#L296](../alluvial/pipeline.go#L296)).

**Anti-patterns.**

- Do not put visualization-specific layout records in `CommonState`, or share mutable acquisition fields across concurrent reference loads; the separate state objects and snapshot preparation establish the boundary ([common.go](common.go), [../alluvial/state.go](../alluvial/state.go), [../alluvial/pipeline.go](../alluvial/pipeline.go)).

**Source locations.**

- [common.go](common.go) — inputs, stage outputs, history lifetime.
- [../treemap/pipeline.go](../treemap/pipeline.go), [../scatter/stages.go](../scatter/stages.go), and [../alluvial/pipeline.go](../alluvial/pipeline.go) — independent workflow consumers.

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
- Resolve at the level the workflow needs to compute: shared metric collection classifies at directory level for tree roll-ups; scatter selects its target from grain. A tree map's inclusion of files is not itself a directory-grain setting ([metrics.go](metrics.go), [../scatter/stages.go#L170](../scatter/stages.go#L170)).

**Anti-patterns.**

- Do not ask `provider.GetBase` directly for a metric that might be an expression result; `DescriptorFor` is the lookup that covers both ([requested.go#L38](requested.go#L38), [../spiral/inks.go#L68](../spiral/inks.go#L68)).

**Source locations.**

- [requested.go#L12](requested.go#L12) — `RequestedMetrics`, `DescriptorFor`, `ClassifyRequestedMetrics`.
- [metrics.go#L12](metrics.go#L12) — `CollectRequestedMetrics`.
- [../treemap/stages.go](../treemap/stages.go), [../scatter/stages.go](../scatter/stages.go), and [../spiral/inks.go](../spiral/inks.go) — collection at the appropriate level and descriptor lookup.
