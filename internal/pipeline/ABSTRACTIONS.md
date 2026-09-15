# Abstractions — `internal/pipeline`

## State

**Purpose.**

- `State` is a type-keyed bag of pipeline values plus a single latched error: each value is stored under its own Go type, so stages declare what they need by parameter type rather than by string key ([state.go#L6](state.go#L6), [state.go#L14](state.go#L14), [state.go#L58](state.go#L58)).
- It turns a visualization run into a flat list of `Apply*` calls with no per-step error handling at the call site ([../../cmd/codeviz/treemap_cmd.go#L98](../../cmd/codeviz/treemap_cmd.go#L98)).

**Boundary and invariants.**

- One value per type, and wiring mistakes are programming errors: `NewState` panics on a nil value or a duplicate type, and applying a function whose input type is absent panics ([state.go#L18](state.go#L18), [state.go#L23](state.go#L23), [apply.go#L20](apply.go#L20)).
- The error is sticky: once a stage returns an error it is stored and every subsequent `Apply*` returns immediately without running, so the first failure wins ([apply.go#L16](apply.go#L16), [apply.go#L27](apply.go#L27), [state.go#L68](state.go#L68)).
- Results are stored by *static* type parameter, so a stage returning `R` overwrites the previous `R` ([apply.go#L59](apply.go#L59), [state.go#L52](state.go#L52)).

**Related operations.**

- `NewState` seeds the state; `ApplyFuncX`, `ApplyFuncXR`, `ApplyFuncXY`, `ApplyFuncXYR` and `ApplyFuncXYZ` run a stage over one to three state-resident inputs, optionally storing a result ([state.go#L14](state.go#L14), [apply.go#L12](apply.go#L12), [apply.go#L38](apply.go#L38), [apply.go#L69](apply.go#L69), [apply.go#L103](apply.go#L103), [apply.go#L127](apply.go#L127)).
- `Err` reports the latched error at the end of the run ([state.go#L68](state.go#L68)).

**Proper-use patterns.**

- Seed the state once with the distinct state objects a visualization needs — shared state, config section, and viz state — then apply stages in order and wrap `s.Err()` at the end ([../../cmd/codeviz/treemap_cmd.go#L84](../../cmd/codeviz/treemap_cmd.go#L84), [../../cmd/codeviz/treemap_cmd.go#L109](../../cmd/codeviz/treemap_cmd.go#L109)).
- Write stages as plain functions of their state types returning `error`, so they are directly testable without a `State` ([../stages/filter.go#L21](../stages/filter.go#L21), [../stages/canvas.go#L15](../stages/canvas.go#L15)).
- Group a visualization's stage sequence into package-level helpers that take `*pipeline.State` ([../treemap/pipeline.go#L11](../treemap/pipeline.go#L11)).

**Anti-patterns.**

- Do not put two values of the same type (for example two `*config.Config` values) into one state — construction panics, and by design there is no per-key namespace ([state.go#L23](state.go#L23)).
- Do not check for errors between `Apply*` calls; the error latch already short-circuits the remaining stages ([apply.go#L16](apply.go#L16)).
- Do not use `State` as a general mutable scratchpad shared across goroutines: it is an unsynchronised map ([state.go#L6](state.go#L6), [state.go#L52](state.go#L52)).

**Source locations.**

- [state.go#L6](state.go#L6) — `State`, `NewState`, `Err`.
- [apply.go#L12](apply.go#L12) — the typed `Apply*` family and its panic/latch rules.
