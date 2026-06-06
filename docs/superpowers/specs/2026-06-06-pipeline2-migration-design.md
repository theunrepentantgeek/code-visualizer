# Pipeline2 Migration Design

Date: 2026-06-06
Branch: `improve/migrate-pipeline`

## Goal

Replace the legacy `pipeline.Run` + `pipeline.Stage[S]` model with the
type-keyed `*pipeline.State` model already prototyped in
`internal/pipeline/pipeline2.go`. After this change, nothing in the workspace
uses `pipeline.Run` or `pipeline.Stage`, and the five viz commands compose
their pipelines from plain functions invoked through `pipeline.ApplyFunc*`
helpers.

## Non-goals

- No behavioural change. Every viz produces the same canvas, the same logs,
  the same exported artefacts, and the same error wrapping (`eris.Wrap(err,
  "<viz> pipeline failed")`).
- No reshuffling of stages between packages. Stages keep their current homes
  (`internal/stages` for shared, `internal/<viz>` for viz-specific).
- No expansion of the test suite beyond what the new API requires (new tests
  for the new helpers; updates to existing tests to match the new shape).

## State shape

For every viz pipeline `*pipeline.State` carries exactly three values, each
identified by its Go type:

1. `*stages.CommonState` — unchanged in field layout. The shared bag of
   `TargetPath`, `Output`, `Flags`, `RootConfig`, `VizName`, `CLIFilters`,
   `FilterRules`, `Requested`, `Root`, `Width`, `Height`, `Canvas`, git
   history, etc.
2. The per-viz config — one of `*config.Treemap`, `*config.Radial`,
   `*config.Bubbletree`, `*config.Spiral`, `*config.Scatter`.
3. The per-viz state — `*treemap.State`, `*radialtree.State`,
   `*bubbletree.State`, `*spiral.State`, `*scatter.State`. These are
   **slimmed** as described below.

`*config.Config` (the root config) is not added as a separate value; stages
that need it continue to reach it via `CommonState.RootConfig`.

### Viz state slimming

Each `internal/<viz>/state.go` is reduced to viz-specific input flags and
viz-specific resolved fields. The following are removed from every viz state
type:

- The embedded `stages.CommonState` field.
- The `Config *config.<Viz>` field.
- The `Common() *stages.CommonState` method.
- The `IncludeBinary() bool` method.

For treemap, `CanvasLabels() []canvas.BlockLabel` is also removed (see the
shared-stage section below).

What remains in each viz state:

- `treemap.State`: `IncludeBinaryFiles`, `Flat`, `Size`, `FillMetric`,
  `FillPalette`, `BorderMetric`, `BorderPalette`, `Inks`, `Root`,
  `LegendConfig`, `BlockLabels`.
- `radialtree.State`: `IncludeBinaryFiles`, `DiscSize`, `FillMetric`,
  `FillPalette`, `BorderMetric`, `BorderPalette`, `Labels`, `Inks`, `Nodes`,
  `LegendConfig`.
- `bubbletree.State`: `IncludeBinaryFiles`, `Flat`, `Size`, `FillMetric`,
  `FillPalette`, `BorderMetric`, `BorderPalette`, `Labels`, `Inks`, `Nodes`,
  `LegendConfig`.
- `spiral.State`: `IncludeBinaryFiles`, `Size`, `FillMetric`, `FillPalette`,
  `BorderMetric`, `BorderPalette`, `Resolution`, `Labels`, `Buckets`,
  `Inks`, `Layout`, `LegendConfig`.
- `scatter.State`: `IncludeBinaryFiles`, `XAxis`, `YAxis`, `Size`,
  `FillMetric`, `FillPalette`, `BorderMetric`, `BorderPalette`, `Dataset`,
  `Inks`, `Layout`, `LegendConfig`.

## Pipeline package changes

### Additions

Add to `internal/pipeline`:

- `Store[T any](s *State, v T)` — exported version of the current unexported
  `store`. Required because seeding three values up front, and any test
  helper that wants to pre-populate state, both need to write multiple
  values.
- `NewState(values ...any) *State` — replaces the current
  `NewState[S any](initial S) *State`. Iterates `values`, computes the key
  for each via `reflect.TypeOf(v)`, and stores it. Panics if a value is
  `nil` (no usable type information) or if the same type is supplied twice.
- `ApplyFuncXY[X, Y any](s *State, f func(X, Y) error)` — two typed inputs,
  in-place mutation, no return value other than `error`.
- `ApplyFuncXYZ[X, Y, Z any](s *State, f func(X, Y, Z) error)` — three
  typed inputs, in-place mutation, no return value other than `error`.

All four follow the existing convention: short-circuit when `s.Err() != nil`,
panic if a required input type is absent from the state, and call
`s.setErr(err)` if the stage returns a non-nil error.

The existing `ApplyFuncX`, `ApplyFuncXR`, `ApplyFuncXYR` are kept as-is.

### Removals

- `internal/pipeline/pipeline.go` — `Stage[S]` and `Run[S]` are deleted.
- `internal/pipeline/pipeline_test.go` — deleted.
- `internal/pipeline/state.go` — `NewState`'s old signature is replaced;
  the unexported `store` is removed (callers use the exported `Store`).

### Pipeline-package test updates

- `state_test.go` — adjusted to the new `NewState(values ...any)` shape;
  add coverage for nil-value panic and duplicate-type panic.
- `pipeline2_test.go` — internal `store(state, k)` call sites become
  `Store(state, k)`; add coverage for `ApplyFuncXY` and `ApplyFuncXYZ`
  (input present, input missing → panic, function returns error → sticky
  error, sticky error → short-circuit).

## Stage signatures

Stages become plain functions whose parameters spell out exactly which
typed values they consume from `*pipeline.State`. The `Stage[S]` type alias
is gone, so are the compile-time `var _ pipeline.Stage[...]` assertions in
every viz package.

### Shared stages (`internal/stages`)

Every generic-over-`VizState` (or `BinaryFilterToggler`, or
`CanvasLabelledState`) stage becomes a concrete function over the
specific typed values it needs.

| Current (generic) | New |
| --- | --- |
| `ValidatePaths[S VizState](s S) error` | `ValidatePaths(c *CommonState) error` |
| `BuildFilterRules[S VizState](s S) error` | `BuildFilterRules(c *CommonState) error` |
| `ScanFilesystem[S VizState](s S) error` | `ScanFilesystem(c *CommonState) error` |
| `CheckGitRequirement[S VizState](s S) error` | `CheckGitRequirement(c *CommonState) error` |
| `RunProviders[S VizState](s S) error` | `RunProviders(c *CommonState) error` |
| `ResolveDimensions[S VizState](s S) error` | `ResolveDimensions(c *CommonState) error` |
| `ExportConfig[S VizState](s S) error` | `ExportConfig(c *CommonState) error` |
| `ExportData[S VizState](s S) error` | `ExportData(c *CommonState) error` |
| `ApplyFooter[S VizState](s S) error` | `ApplyFooter(c *CommonState) error` |
| `WriteCanvas[S VizState](s S) error` | `WriteCanvas(c *CommonState) error` |
| `FilterBinaryFiles[S BinaryFilterToggler](s S) error` | moved to per-viz packages (see below) |
| `ApplyCanvasBlockLabels[S CanvasLabelledState](s S) error` | moved to `treemap` package (see below) |

Two functions cannot stay in `internal/stages` once the `VizState`
interface is gone, because they need a concretely-typed viz state value
as their second argument:

- **`FilterBinaryFiles`** moves to each viz package as a concrete function
  with signature `func FilterBinaryFiles(c *stages.CommonState, t *State) error`.
  Each viz's function is a one-liner that reads `t.IncludeBinaryFiles` and
  delegates to a shared unexported helper
  `stages.filterBinaryFiles(c *CommonState, include bool) error` containing
  the actual filtering logic. Orchestrators call
  `pipeline.ApplyFuncXY(s, <viz>.FilterBinaryFiles)`.
- **`ApplyCanvasBlockLabels`** moves to the `treemap` package as
  `func ApplyCanvasBlockLabels(c *stages.CommonState, t *State) error` and
  reads `t.BlockLabels` directly. Only the treemap pipeline calls it; no
  other viz produces block labels today.

Interface removals:

- `stages.VizState` — deleted.
- `stages.BinaryFilterToggler` — deleted.
- `stages.CanvasLabelledState` — deleted.

### Viz-specific stages (`internal/<viz>/stages.go`)

Each viz stage function takes the typed values it needs by parameter, in
the order `*stages.CommonState`, `*<viz>.State`, `*config.<Viz>`. Examples
(treemap):

| Current | New |
| --- | --- |
| `func ResolveMetrics(s *State) error` | `func ResolveMetrics(c *stages.CommonState, t *State, cfg *config.Treemap) error` |
| `func BuildInksStage(s *State) error` | `func BuildInksStage(c *stages.CommonState, t *State) error` |
| `func BuildLegendStage(s *State) error` | `func BuildLegendStage(c *stages.CommonState, t *State, cfg *config.Treemap) error` |
| `func LayoutStage(s *State) error` | `func LayoutStage(c *stages.CommonState, t *State) error` |
| `func RenderStage(s *State) error` | `func RenderStage(c *stages.CommonState, t *State) error` |
| `func LabelStage(s *State) error` | `func LabelStage(c *stages.CommonState, t *State) error` |
| `func LogResult(s *State) error` | `func LogResult(c *stages.CommonState, t *State) error` |

Bodies mechanically swap `s.Common()` for the passed-in `c`, `s.Config` for
the passed-in `cfg`, and leave viz-local field accesses (`s.Inks`, `s.Nodes`,
…) renamed to whatever the new parameter is called. The choice of parameter
which receives the third argument (config) is dictated by whether the stage
actually reads any config fields; stages that touch only viz-local data
take only `(*stages.CommonState, *State)`.

The same translation applies to `internal/bubbletree/stages.go`,
`internal/radialtree/stages.go`, `internal/spiral/stages.go`,
`internal/scatter/stages.go`, and any sibling files containing additional
stages (e.g. `internal/spiral/*.go`).

The compile-time `var _ pipeline.Stage[*State] = ...` blocks at the bottom
of each viz `stages.go` are deleted; the new functions are not assignable to
the (now-removed) `Stage[*State]` type.

## Orchestrator (`cmd/codeviz/*_cmd.go`)

Each `*Cmd.Run` is rewritten as a flat sequence of `pipeline.ApplyFunc*`
calls against one `*pipeline.State`. Treemap is the worked example; the
other four follow the same pattern with their own typed values.

```go
func (c *TreemapCmd) Run(flags *Flags) error {
    if err := c.mergeConfigAndValidate(flags); err != nil {
        return err
    }

    common := &stages.CommonState{
        TargetPath: c.TargetPath,
        Output:     c.Output,
        Flags:      toStagesFlags(flags),
        RootConfig: flags.Config,
        VizName:    "treemap",
        CLIFilters: c.Filters(),
    }
    cfg := flags.Config.Treemap
    viz := &treemap.State{
        IncludeBinaryFiles: c.IncludeBinaryFiles,
        Flat:               c.Flat,
    }

    s := pipeline.NewState(common, cfg, viz)

    pipeline.ApplyFuncX(s, stages.ValidatePaths)
    pipeline.ApplyFuncX(s, stages.ExportConfig)
    pipeline.ApplyFuncX(s, stages.BuildFilterRules)
    pipeline.ApplyFuncXYZ(s, treemap.ResolveMetrics)
    pipeline.ApplyFuncX(s, stages.ScanFilesystem)
    pipeline.ApplyFuncX(s, stages.CheckGitRequirement)
    pipeline.ApplyFuncX(s, stages.RunProviders)
    pipeline.ApplyFuncXY(s, treemap.FilterBinaryFiles)
    pipeline.ApplyFuncX(s, stages.ExportData)
    pipeline.ApplyFuncX(s, stages.ResolveDimensions)
    pipeline.ApplyFuncXY(s, treemap.BuildInksStage)
    pipeline.ApplyFuncXYZ(s, treemap.BuildLegendStage)
    pipeline.ApplyFuncXY(s, treemap.LayoutStage)
    pipeline.ApplyFuncXY(s, treemap.RenderStage)
    pipeline.ApplyFuncXY(s, treemap.LabelStage)
    pipeline.ApplyFuncXY(s, treemap.ApplyCanvasBlockLabels)
    pipeline.ApplyFuncX(s, stages.ApplyFooter)
    pipeline.ApplyFuncX(s, stages.WriteCanvas)
    pipeline.ApplyFuncXY(s, treemap.LogResult)

    return eris.Wrap(s.Err(), "treemap pipeline failed")
}
```

Properties:

- No per-call error check; `ApplyFunc*` short-circuits on a sticky error.
- One `eris.Wrap(s.Err(), …)` at the end, matching the existing wrap text.
- The `[*treemap.State]` type parameter at the `pipeline.Run` call site
  disappears; types are inferred from each stage's signature.

The same shape applies to `bubbletree_cmd.go`, `radialtree_cmd.go`,
`spiral_cmd.go`, and `scatter_cmd.go`, each seeding `pipeline.NewState`
with its own `*stages.CommonState`, `*config.<Viz>`, and `*<viz>.State`.

## Test fallout

### Pipeline package

Covered above (`state_test.go`, `pipeline2_test.go`).

### Shared stages (`internal/stages/*_test.go`)

- Tests that constructed a fake state implementing `VizState` /
  `BinaryFilterToggler` / `CanvasLabelledState` are rewritten to call the
  now-concrete functions with a directly-constructed `*CommonState`.
- `binary_test.go` now exercises the unexported helper
  `stages.filterBinaryFiles(c, include)`; the per-viz `FilterBinaryFiles`
  wrappers are exercised in their respective viz packages' `stages_test.go`.
- `labels_test.go` moves to `internal/treemap` (since
  `ApplyCanvasBlockLabels` now lives there) and switches to
  `*treemap.State` instead of `fakeLabelState`.
- `canvas_test.go` drops the `[*fakeState]` type parameters and calls
  `stages.ApplyFooter(c)` / `stages.WriteCanvas(c)` directly.
- Tests that asserted satisfaction of removed interfaces are deleted.

### Viz packages (`internal/<viz>/stages_test.go`)

- `&<viz>.State{CommonState: stages.CommonState{...}, Config: ..., ...}`
  literals are split into three values (`*stages.CommonState`,
  `*config.<Viz>`, `*<viz>.State`).
- Stage invocations either call the now-plain function with its concrete
  typed args, or set up a `*pipeline.State` and call via
  `pipeline.ApplyFunc*`. Either is acceptable; the existing single-stage
  unit-test style is best served by the former.
- Assertions like `var _ stages.BinaryFilterToggler = on` (in bubbletree,
  radialtree) are deleted.
- The `TestState_CommonReturnsEmbeddedPointer` test in
  `internal/treemap/stages_test.go` is deleted along with the embedded
  field it asserted.

## Migration order

All five viz commands are migrated together, in the same change-set, so
the codebase never sits in a hybrid state. Recommended sequence within the
single change:

1. Add `Store`, the new `NewState`, `ApplyFuncXY`, and `ApplyFuncXYZ` to
   `internal/pipeline`. Update `pipeline2_test.go` and `state_test.go`.
2. Slim each viz `state.go` (remove embedded `CommonState`, `Config`
   field, `Common()`, `IncludeBinary()`, `CanvasLabels()`).
3. Rewrite each viz `stages.go` (and siblings) to the new signatures.
4. Rewrite each shared stage in `internal/stages` to the new concrete
   signatures; delete `VizState`, `BinaryFilterToggler`,
   `CanvasLabelledState`.
5. Rewrite each `cmd/codeviz/<viz>_cmd.go` to use `pipeline.NewState` +
   `pipeline.ApplyFunc*`.
6. Update all affected tests.
7. Delete `internal/pipeline/pipeline.go` and
   `internal/pipeline/pipeline_test.go`.

The branch lands once `task ci` is green.

## Verification

- `task ci` (build + unit tests + lint) is the gate.
- A manual run of each viz against a sample (e.g. `samples/codeviz-treemap.yml`)
  is **not** strictly required because there are no behavioural changes, but
  a quick spot-check of one viz is cheap insurance and is recommended once
  CI is green.

## Risks

- **Stage parameter ordering.** A consistent order — `*CommonState`,
  `*<viz>.State`, `*config.<Viz>` — for every multi-argument stage keeps
  the orchestrator readable and prevents accidental swaps that the type
  system would still accept when two parameters happen to share a type.
  The convention is enforced by review, not by the compiler.
- **`FilterBinaryFiles` duplication.** Five near-identical one-liners now
  live in the viz packages instead of a single generic function. They
  share the actual filtering logic via the unexported
  `stages.filterBinaryFiles` helper, so duplication is limited to the
  signature plumbing, which is the price of removing the `VizState`
  interface.
