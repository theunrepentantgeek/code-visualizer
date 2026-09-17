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
