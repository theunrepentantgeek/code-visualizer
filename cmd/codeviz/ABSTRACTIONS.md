# Abstractions — `cmd/codeviz`

## Viz Command

**Purpose.**

- A *viz command* is the recurring shape of every visualization subcommand: a Kong-annotated flag struct plus a fixed set of methods that turn flags and config into a rendered image ([main.go#L28](main.go#L28), [treemap_cmd.go#L14](treemap_cmd.go#L14), [spiral_cmd.go#L128](spiral_cmd.go#L128)).
- The repetition is deliberate and acknowledged, not an accident awaiting refactoring ([treemap_cmd.go#L76](treemap_cmd.go#L76), [scatter_cmd.go#L167](scatter_cmd.go#L167)).

**Boundary and invariants.**

- Configuration is layered in one fixed order — auto-load the config file, apply CLI overrides, then validate the effective result — and `Run` starts by doing exactly that ([treemap_cmd.go#L66](treemap_cmd.go#L66), [treemap_cmd.go#L67](treemap_cmd.go#L67), [treemap_cmd.go#L71](treemap_cmd.go#L71)).
- Overrides are zero-transparent: an unset CLI field leaves the config value untouched ([treemap_cmd.go#L113](treemap_cmd.go#L113), [treemap_cmd.go#L115](treemap_cmd.go#L115)).
- `Run` owns only wiring: build `stages.CommonState`, build the viz's own `State`, register both plus the config in a `pipeline.State`, then hand off to the viz package's stage sequences ([treemap_cmd.go#L84](treemap_cmd.go#L84), [treemap_cmd.go#L98](treemap_cmd.go#L98), [treemap_cmd.go#L106](treemap_cmd.go#L106)).

**Related operations.**

- `Filters` merges include/exclude rules, `stagesFlagsForCommand` converts global flags plus the command's history range, and `validateConfig` checks metric specs before any scanning happens ([treemap_cmd.go#L42](treemap_cmd.go#L42), [main.go#L67](main.go#L67), [treemap_cmd.go#L48](treemap_cmd.go#L48)).

**Proper-use patterns.**

- Add a new visualization by following the same six-part shape and registering the struct on the CLI; presets then reuse it by constructing the command directly and calling `Run` ([main.go#L33](main.go#L33), [render_cmd.go#L161](render_cmd.go#L161), [render_cmd.go#L194](render_cmd.go#L194)).
- Let errors accumulate in the pipeline state and wrap once at the end, instead of checking each stage ([treemap_cmd.go#L109](treemap_cmd.go#L109)).

**Anti-patterns.**

- Do not put layout, metric, or rendering logic in a command; stages live in the viz package and are invoked through the pipeline ([treemap_cmd.go#L104](treemap_cmd.go#L104), [treemap_cmd.go#L107](treemap_cmd.go#L107)).
- Do not validate flag values in isolation before merging; validation runs on the merged configuration, since config files can supply the same values ([treemap_cmd.go#L46](treemap_cmd.go#L46), [treemap_cmd.go#L73](treemap_cmd.go#L73)).

**Source locations.**

- [treemap_cmd.go#L14](treemap_cmd.go#L14) — the canonical command: flags, `Filters`, `validateConfig`, `mergeConfigAndValidate`, `Run`, `applyOverrides`.
- [render_cmd.go#L166](render_cmd.go#L166) — presets reusing commands through a shared `Run` interface.

## Alluvial Delta Fill

**Purpose.**

- An *alluvial delta fill* is the alluvial-only `.delta` suffix on a fill metric. It changes band colour from the metric's final-snapshot value to the change between the first and last ordered references ([alluvial_cmd.go#L23](alluvial_cmd.go#L23), [alluvial documentation](../../docs/content/docs/visualizations/alluvial.md#L76)).

**Boundary and invariants.**

- `.delta` modifies only the fill encoding: flow widths remain the configured width metric at each snapshot ([alluvial data model](../../internal/alluvial/data.go#L52), [alluvial documentation](../../docs/content/docs/visualizations/alluvial.md#L68)).
- The suffix is stripped before normal numeric-metric validation and metric collection, while the suffixed name is retained as the user-facing fill label ([alluvial_cmd.go#L80](alluvial_cmd.go#L80), [alluvial metric resolution](../../internal/alluvial/pipeline.go#L20)).
- Delta values are keyed by selected directory path and computed as last-reference fill value minus first-reference fill value; a path absent at either endpoint contributes zero at that endpoint ([alluvial fill calculation](../../internal/alluvial/data.go#L164)).

**Related operations.**

- `validateAlluvialFill` accepts the suffix through `ParseFillMetric`, `ResolveMetrics` records the base metric and delta mode, and `BuildDataStage` carries that mode into data construction ([alluvial_cmd.go#L80](alluvial_cmd.go#L80), [pipeline.go#L20](../../internal/alluvial/pipeline.go#L20), [pipeline.go#L263](../../internal/alluvial/pipeline.go#L263)).
- Legend construction preserves the suffixed label and uses the computed delta values to resolve the colour ink ([pipeline.go#L289](../../internal/alluvial/pipeline.go#L289)).

**Proper-use patterns.**

- Select the behavior as `--fill <numeric-metric>.delta[,palette]`, or use the same metric spec in alluvial configuration; command-line specs replace configured fill specs through normal command override handling ([alluvial_cmd_test.go#L36](alluvial_cmd_test.go#L36), [alluvial_cmd_test.go#L68](alluvial_cmd_test.go#L68)).
- Omit the suffix when bands should be coloured by the selected metric at the final ordered reference ([alluvial documentation](../../docs/content/docs/visualizations/alluvial.md#L76), [alluvial data model](../../internal/alluvial/data.go#L77)).

**Anti-patterns.**

- Do not apply `.delta` to the width metric or interpret it as adjacent-snapshot change; parsing is intentionally confined to the fill metric and compares only the first and last references ([alluvial_cmd.go#L57](alluvial_cmd.go#L57), [pipeline.go#L322](../../internal/alluvial/pipeline.go#L322), [alluvial fill calculation](../../internal/alluvial/data.go#L164)).
- Do not bypass base-metric validation by adding the suffix: unknown or non-numeric base metrics remain invalid ([alluvial_cmd.go#L80](alluvial_cmd.go#L80), [alluvial_cmd_test.go#L126](alluvial_cmd_test.go#L126)).

**Source locations.**

- [alluvial_cmd.go#L20](alluvial_cmd.go#L20) — CLI syntax, validation, and command/config boundary.
- [internal/alluvial/pipeline.go#L20](../../internal/alluvial/pipeline.go#L20) — fill resolution, retained label, delta mode, and legend construction.
- [internal/alluvial/data.go#L52](../../internal/alluvial/data.go#L52) — endpoint-value collection and delta calculation.
- [alluvial_cmd_test.go#L36](alluvial_cmd_test.go#L36) — CLI parsing, override, and validation coverage.
