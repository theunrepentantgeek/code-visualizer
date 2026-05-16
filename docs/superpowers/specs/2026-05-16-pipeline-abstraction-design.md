# Pipeline Abstraction for Visualization Rendering

**Date:** 2026-05-16
**Status:** Draft
**Related issue:** [#194 — Introduce shared `vizCmd` pipeline](https://github.com/theunrepentantgeek/code-visualizer/issues/194)

## Background

All four current visualization commands (`treemap`, `bubbletree`, `radialtree`,
`spiral`) follow the same ~15-step lifecycle: validate config and paths, export
config, build filter rules, scan filesystem, collect metrics, run providers,
filter binary files, export data, resolve layout dimensions, build inks,
compute layout, render to canvas, set legend, write output, log result. The
shared scaffold today is copy-pasted across four `*_cmd.go` files (~200 lines
each). Steps 1–9 and 13–15 are structurally identical; only the layout / ink /
render block in the middle is genuinely viz-specific.

A minimal `internal/pipeline` package already exists with a generic
`Stage[S any]` type and a `Run` function. It needs unit tests and a usable
shape around it before any visualization can be decomposed into stages.

## Goals

1. Document and test the existing `internal/pipeline` primitive.
2. Provide a reusable scaffold (`internal/stages`) so each visualization
   declares its lifecycle as a list of stage functions rather than open-coded
   procedural code.
3. Extract legend-construction code (currently entangled with treemap) into a
   dedicated `internal/legend` package so it is reusable.
4. Refactor the treemap command end-to-end as the reference implementation,
   proving the design before applying it to the other three visualizations in
   follow-up specs.

## Non-goals

- Refactoring `bubbletree`, `radialtree`, or `spiral`. Each gets its own short
  follow-up spec once the pattern is validated on treemap.
- Changing any observable CLI behavior. Treemap PNG/SVG output, log lines,
  error messages, and exit codes are unchanged. Existing golden-file snapshots
  must still pass byte-for-byte.
- Changing how `config.Config`, `*Flags`, or Kong command wiring work.
- Introducing branching, parallel, conditional, or middleware-style pipeline
  features.
- Refactoring the interactive UX referenced as future motivation; this spec
  only positions the code so that work becomes easier later.

## Success criteria

- `task ci` passes.
- All existing treemap golden tests pass without regenerated fixtures.
- `cmd/codeviz/treemap_cmd.go` `Run()` is a short pipeline composition; viz
  scaffolding no longer lives in `cmd/codeviz`.
- `internal/pipeline` has tests covering the documented semantics below.
- `cmd/codeviz` is reduced to orchestration: Kong structs, config merging,
  pipeline composition, and viz-agnostic glue. Reusable building blocks live
  under `internal/`.

## Architectural overview

Five packages participate, organized by responsibility:

| Package                       | Responsibility                                                                                                                                                                                     |
| ----------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `internal/pipeline`           | Domain-agnostic `Stage[S]` and `Run`. No knowledge of visualizations, metrics, or config.                                                                                                          |
| `internal/stages`             | `CommonState`, `VizState` interface, and shared lifecycle stages used by every visualization. Owns shared helpers (path validation, filter rules, git check, palette resolution, progress wiring). |
| `internal/legend`             | Legend construction, layout-space reservation, and offset math. Reusable across all visualizations.                                                                                                |
| `internal/treemap` (extended) | Treemap-specific state type and viz stages (resolve metrics, build inks, build legend, layout, render, log result). Existing layout code unchanged.                                                |
| `cmd/codeviz`                 | Orchestration only: Kong command structs, config merging, pipeline composition.                                                                                                                    |

The other viz packages (`internal/bubbletree`, `internal/radialtree`,
`internal/spiral`) are not touched in this spec.

## Design

### `internal/pipeline`

The existing API loses its unused `C` type parameter, and `Stage` is
relaxed so that it can be parameterized on pointer types directly (see
"Why `Stage[S]` takes `S`, not `*S`" below).

```go
type Stage[S any] func(S) error

// Run executes stages in order against initialState. Each stage receives
// the state and may mutate it (when S is a pointer type, mutations are
// visible to subsequent stages and to the caller). If any stage returns
// an error, execution halts immediately and the (partial) state plus
// error are returned. Run does not wrap stage errors; callers and stages
// own wrapping conventions.
func Run[S any](initialState S, stages ...Stage[S]) (S, error)
```

**Why `Stage[S]` takes `S`, not `*S`.** Visualization state types expose
their shared common state through a pointer-receiver method
(`func (s *State) Common() *stages.CommonState`). With
`Stage[S any] = func(*S) error`, satisfying a `VizState` constraint on
`S` requires the two-type-parameter idiom
(`[S any, PS interface { *S; Common() *stages.CommonState }]`), which is
awkward at every shared-stage definition and at every call site. Taking
`S` directly lets callers instantiate stages with a pointer type
(`Stage[*treemap.State]`), at which point `*treemap.State` satisfies
`VizState` naturally via its pointer-receiver method. Non-pointer uses
remain available by passing a value type as `S`.

Tested semantics:

1. Empty pipeline returns the input state and a nil error.
2. A single stage runs once; its mutations are visible in the returned state.
3. Multiple stages run in declaration order; each observes earlier mutations.
4. When a stage returns an error, subsequent stages are not invoked.
5. State mutations from stages that ran before the error are preserved in the
   returned value.
6. `Run` returns the stage's error unwrapped.
7. A nil stage panics. This matches Go's default behavior for calling a nil
   func value and is documented rather than guarded.

Tests live in `internal/pipeline/pipeline_test.go`, use Gomega for assertions
(matching repo convention), and define a tiny in-test state struct so the
pipeline package keeps zero domain imports.

### `internal/stages`

`CommonState` holds the fields used by shared stages. Its final field list
will track what the treemap refactor actually needs; the working set is:

```go
type CommonState struct {
    // Inputs set by the orchestrator before Run:
    TargetPath   string
    Output       string
    Flags        *cmd.Flags        // progress, --export-config, --export-data
    RootConfig   *config.Config    // shared config: width, height, file filters
    CLIFilters   []string          // raw --filter values from the CLI

    // Populated by shared stages during the pipeline:
    FilterRules  []filter.Rule     // BuildFilterRules
    Requested    []metric.Name     // populated by a viz-specific ResolveMetrics stage
    Root         *model.Directory  // ScanFilesystem
    Width        int               // ResolveDimensions
    Height       int               // ResolveDimensions
    Canvas       *canvas.Canvas    // viz-specific render stage
}
```

The `VizState` interface is the constraint shared stages use:

```go
type VizState interface {
    Common() *CommonState
}
```

Each viz state struct embeds `CommonState` and exposes it via a `Common()`
method on a pointer receiver, returning a pointer to the embedded value.
Shared stages are generic functions parameterized over a `VizState`
implementation; in practice the type argument is always a pointer type
(e.g. `stages.ScanFilesystem[*treemap.State]`), which is what satisfies
the `Common()` method set.

Shared stages provided by the package:

| Stage                             | Purpose                                                                            |
| --------------------------------- | ---------------------------------------------------------------------------------- |
| `ValidatePaths[S VizState]`       | Stat target and output paths; check output format.                                 |
| `ExportConfig[S VizState]`        | Save merged config when `--export-config` is set.                                  |
| `BuildFilterRules[S VizState]`    | Merge `RootConfig.FileFilter` with CLI `--filter` flags.                           |
| `ScanFilesystem[S VizState]`      | Walk the target directory and populate `Root`. Includes scan progress wiring.      |
| `CheckGitRequirement[S VizState]` | Verify the target is inside a git repository when any `Requested` metric needs it. |
| `RunProviders[S VizState]`        | Run metric providers against `Root`. Includes metric progress wiring.              |
| `FilterBinaryFiles[S VizState]`   | Remove binary files unless the viz state's `IncludeBinaryFiles` flag is set.       |
| `ExportData[S VizState]`          | Save raw data when `--export-data` is set.                                         |
| `ResolveDimensions[S VizState]`   | Resolve effective width and height from config plus defaults.                      |
| `WriteCanvas[S VizState]`         | Render `Canvas` to `Output`.                                                       |

Each shared stage emits the same `slog` lines as today, in the same places.
No log-level or message changes.

The final summary log is **viz-specific**: each viz defines its own
`LogResult` stage because the fields logged (size metric, fill metric,
palettes, file/dir counts) differ per visualization.

Helpers moved here (non-stage utilities used by stages):

- Path validation and the `outputPathError` / `targetPathError` sentinel types.
- `buildFilterRules`.
- Git-requirement helpers (`checkGitRequirement`, `findGitMetric`,
  `verifyGitRepo`, `gitRequiredError`).
- Binary file filtering (`filterBinaryFiles`, `countAll`,
  `noFilesAfterFilterError`).
- Palette and metric resolution shared across viz types
  (`resolveFillPalette`, `resolveBorderMetricAndPalette`, `specMetric`,
  `specPalette`, `collectRequestedMetrics`).
- Progress wiring (`buildScanProgress`, `buildMetricProgress`).

`ptrInt` and `ptrString` may stay in `cmd/codeviz` or move with the helpers;
the choice is left to the implementation.

Errors from shared stages are wrapped with `eris.Wrap` using today's
messages (`"scan failed"`, `"failed to load metrics"`, `"failed to export
data"`, etc.). Sentinel error types move alongside the helpers that produce
them.

### `internal/legend`

Pulls legend code out of `cmd/codeviz` and treemap-local helpers into one
package that any visualization can call.

Moves to `internal/legend`:

- `buildLegendConfig` (composes `canvas.LegendConfig` from inks, metric
  names, and a position/orientation pair).
- `resolveLegendOptions` (parses position and orientation strings, applies
  auto-detection).
- `reserveAndLayout` (returns layout dimensions after reserving legend
  space, with the `minReservableSize` fallback when reservation would shrink
  the canvas too far).
- `legendLayoutOffset` and `cornerLegendOffset` (compute offset to apply to
  layout output once space is reserved).
- `minReservableSize` constant.

Stays where it is:

- `canvas.LegendConfig`, `canvas.LegendPosition*`, `canvas.LegendOrientation*`
  — these are canvas-level types owned by the canvas package.
- `internal/canvas/legend*.go` — render-side legend code, unchanged.

`cmd/codeviz/legend_builder.go` is deleted; its functions and tests migrate
to `internal/legend`.

### Treemap refactor

A new file under `internal/treemap` (likely `state.go` or `pipeline.go`)
adds the viz state type and viz-specific stages:

```go
package treemap

type State struct {
    stages.CommonState                       // embedded
    Config              *config.Treemap
    IncludeBinaryFiles  bool

    // Resolved during the pipeline:
    Size           metric.Name
    FillMetric     metric.Name
    FillPalette    palette.PaletteName
    BorderMetric   metric.Name
    BorderPalette  palette.PaletteName
    Inks           Inks                       // existing treemapInks, moved/renamed
    Rects          []Rect                     // treemap.Layout output
    LegendConfig   *canvas.LegendConfig
}

func (s *State) Common() *stages.CommonState { return &s.CommonState }
```

Viz-specific stages (all `pipeline.Stage[State]`):

| Stage            | Responsibility                                                                                             |
| ---------------- | ---------------------------------------------------------------------------------------------------------- |
| `ResolveMetrics` | Computes `Size`, `FillMetric`, `FillPalette`, `BorderMetric`, `BorderPalette`; fills `Common().Requested`. |
| `BuildInks`      | Calls the existing `buildTreemapInks` (moved into the package).                                            |
| `BuildLegend`    | Calls `legend.Build` to populate `LegendConfig`.                                                           |
| `Layout`         | Reserves space via `legend.ReserveAndLayout`, runs `treemap.Layout`, applies any offset, stores `Rects`.   |
| `Render`         | Calls existing `renderTreemapToCanvas`, assigns to `Common().Canvas`, attaches legend.                     |
| `LogResult`      | Emits the viz-specific final `slog.Info` line.                                                             |

`WriteCanvas` (shared) performs the actual `cv.Render(Output)` call.

`cmd/codeviz/treemap_cmd.go` after refactor keeps the Kong struct, config
merging (`mergeConfigAndValidate`, `applyOverrides`, `validateConfig`), and
becomes a thin orchestrator:

```go
func (c *TreemapCmd) Run(flags *Flags) error {
    if err := c.mergeConfigAndValidate(flags); err != nil {
        return err
    }

    state := treemap.State{
        CommonState: stages.CommonState{
            TargetPath: c.TargetPath,
            Output:     c.Output,
            Flags:      flags,
            RootConfig: flags.Config,
            CLIFilters: c.Filter,
        },
        Config:             flags.Config.Treemap,
        IncludeBinaryFiles: c.IncludeBinaryFiles,
    }

    _, err := pipeline.Run(&state,
        stages.ValidatePaths[*treemap.State],
        stages.ExportConfig[*treemap.State],
        stages.BuildFilterRules[*treemap.State],
        treemap.ResolveMetrics,
        stages.ScanFilesystem[*treemap.State],
        stages.CheckGitRequirement[*treemap.State],
        stages.RunProviders[*treemap.State],
        stages.FilterBinaryFiles[*treemap.State],
        stages.ExportData[*treemap.State],
        stages.ResolveDimensions[*treemap.State],
        treemap.BuildInks,
        treemap.BuildLegend,
        treemap.Layout,
        treemap.Render,
        stages.WriteCanvas[*treemap.State],
        treemap.LogResult,
    )
    return err
}
```

`ResolveMetrics` must run before `ScanFilesystem` and `RunProviders`
because it populates `Common().Requested`; the ordering above reflects
that. Viz-specific stages (`treemap.ResolveMetrics`, `treemap.BuildInks`,
…) are declared directly as `pipeline.Stage[*treemap.State]` and so need
no explicit type argument at the call site.

## Testing strategy

1. `internal/pipeline` — new unit tests covering the seven semantic rules
   listed above.
2. `internal/stages` — unit tests per stage using a minimal fake state that
   satisfies `VizState`. Focus on branching behavior: `FilterBinaryFiles`
   no-op when the include flag is set; `CheckGitRequirement` skips when no
   git metrics are requested; `ExportConfig` and `ExportData` no-op when
   their flags are empty. Pure helpers (`buildFilterRules`,
   `resolveFillPalette`, `collectRequestedMetrics`) get table tests.
3. `internal/legend` — port the existing legend tests from
   `cmd/codeviz/legend_builder_test.go` essentially unchanged.
4. `internal/treemap` — new unit tests for the `ResolveMetrics` and `Layout`
   stage wrappers; the underlying algorithms already have coverage.
5. `cmd/codeviz` — existing `main_test.go` and treemap golden tests are the
   integration safety net. They must pass unchanged.

## Implementation sequencing

Each step keeps `task ci` green. Commit boundaries are decided during
implementation; this is the ordering, not a commit list.

1. Add tests to `internal/pipeline` and drop the unused `C` type parameter
   from `Run`.
2. Create `internal/legend`, move legend code and tests. Update existing
   callers in `cmd/codeviz/treemap_cmd.go` and any other current consumers.
3. Create `internal/stages` skeleton: `CommonState`, `VizState`, and the
   shared stages plus helpers listed above. The other three commands still
   use their existing inline code paths and call into helpers either via
   thin forwarders in `cmd/codeviz` or by importing the new package
   directly — whichever is least invasive while keeping CI green.
4. Create `internal/treemap` state and viz stages; rewrite `TreemapCmd.Run`
   to compose the pipeline. Verify golden tests pass.
5. Delete dead code from `cmd/codeviz` (helpers now living in
   `internal/stages` or `internal/legend`); `viz_cmd_helpers.go` shrinks
   to only what the three not-yet-refactored commands still need.

## Risks

- **Shared helpers used by the not-yet-refactored commands break when
  moved.** Mitigation: during step 3, leave thin forwarders in `cmd/codeviz`
  so the bubbletree, radialtree, and spiral commands compile unchanged.
  Forwarders are removed when each of those commands is refactored in its
  own follow-up spec.
- **`CommonState`'s shape ends up wrong for the other three vizes.**
  Mitigation: spec scope is treemap-only; field additions and renames are
  expected when the next viz lands. The `Common()` accessor isolates
  changes to one method.
- **Generic type inference noise.** Mitigation: shared stages are
  invoked with explicit `[*treemap.State]` type arguments at composition
  sites. This is verbose but local to one function per viz and not a
  correctness issue.
- **Subtle render differences sneak through during the move.** Mitigation:
  existing golden-file tests must pass without regenerated fixtures.

## YAGNI exclusions

- No conditional, branching, or parallel pipeline primitives. A stage that
  needs to be conditional inspects state and returns early.
- No pipeline-level logging or tracing wrappers; stages log via `slog`
  directly as they do today.
- No middleware or decorator pattern around stages.
- No premature abstraction over the other three visualizations. Their
  shape informs follow-up specs, not this one.
