# Bubbletree Pipeline Migration

**Date:** 2026-05-17
**Status:** Draft
**Related spec:** [2026-05-16 Pipeline abstraction design](2026-05-16-pipeline-abstraction-design.md)

## Background

The pipeline abstraction landed with `treemap` as its reference
implementation. `internal/pipeline`, `internal/stages`, and `internal/legend`
are in place; `internal/treemap` owns the treemap-specific state, stages,
inks, and render code; and `cmd/codeviz/treemap_cmd.go` is a thin
orchestrator that composes `pipeline.Run`.

`cmd/codeviz/bubbletree_cmd.go` still uses the pre-pipeline open-coded
shape: a single ~80-line `Run` method that interleaves config merge, path
validation, scan, metric resolution, provider run, binary filter, export,
render, and logging. The shared lifecycle steps already live in
`internal/stages`; this spec migrates bubbletree to the same pattern the
treemap refactor established.

In addition, four helpers (`buildMetricInk`, `metricValueForFile`,
`collectNumericValues`, `collectDistinctTypes`) currently live in
`cmd/codeviz/ink_builder.go` and are duplicated inside
`internal/treemap/inks.go`. The bubble migration would create a third copy.
We extract them into a new `internal/inks` package as part of this work and
update treemap, radial, and spiral to consume it. Radial and spiral keep
their existing open-coded `Run()` methods — only their imports change.

## Goals

1. Move bubble-specific render and ink code out of `cmd/codeviz` into
   `internal/bubbletree`, exported under that package.
2. Rewrite `BubbletreeCmd.Run` as a `pipeline.Run` composition that reuses
   the shared stages from `internal/stages` plus six bubble-specific stages.
3. Extract the duplicated ink helpers into a new `internal/inks` package
   and convert all four visualizations to consume it.
4. Add end-to-end render coverage for bubbletree (PNG / SVG / JPG plus label
   modes and ink kinds) modelled on `internal/treemap/render_test.go`.

## Non-goals

- Refactoring `radial` or `spiral` commands to use the pipeline. Those get
  their own follow-up specs.
- Changing any observable CLI behavior. Bubbletree PNG/SVG/JPG output, log
  lines (keys, values, levels, ordering), error messages, and exit codes
  are unchanged.
- Reserving canvas space for the legend in bubbletree. Today the legend
  overlays the bubbles; that behavior is preserved. Aligning bubbletree
  with treemap's `legend.ReserveAndLayout` flow is out of scope and can be
  a follow-up spec if desired.
- Consolidating `bubbletree.Inks` and `treemap.Inks` into one struct. They
  are identical today but each viz keeps its own type for future divergence.
- Introducing byte-identical fixture-based golden tests (e.g. Goldie). The
  new render tests verify decodability and structural shape, matching the
  existing pattern in `internal/treemap/render_test.go`.

## Success criteria

- `task ci` passes.
- All existing bubbletree CLI validation tests in
  `cmd/codeviz/main_test.go` pass unchanged.
- `cmd/codeviz/bubbletree_cmd.go` `Run()` is a short pipeline composition
  with no inline scan/provider/export/render logic.
- `cmd/codeviz/ink_builder.go`, `cmd/codeviz/shape_inks.go`,
  `cmd/codeviz/bubble_canvas.go`, and `cmd/codeviz/bubble_canvas_test.go`
  no longer exist.
- `internal/inks` package exists and is consumed by `internal/treemap`,
  `internal/bubbletree`, and `cmd/codeviz/radial_canvas.go`.
  `cmd/codeviz/spiral_canvas.go` does not call the shared ink helpers
  today (it uses its own `buildBucketInk` / `spiralMetricValue`); it
  only loses its dependency on the deleted `shape_inks.go` struct.
- `internal/bubbletree/render_test.go` provides end-to-end render coverage
  matching the depth of `internal/treemap/render_test.go`.

## Architectural overview

| Package                        | Change                                                                                                                                                  |
| ------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `internal/inks` (new)          | Owns `BuildMetricInk`, `MetricValueForFile`, `CollectNumericValues`, `CollectDistinctTypes`. No knowledge of any specific visualization.                |
| `internal/bubbletree` (extend) | Adds `State`, viz stages, `Inks` struct + `BuildInks`, and `RenderToCanvas`. Existing `layout.go` / `node.go` are unchanged.                            |
| `internal/treemap`             | `inks.go` keeps `Inks` and `BuildInks` but delegates to `internal/inks` for the four helpers; local copies removed.                                     |
| `cmd/codeviz`                  | `bubbletree_cmd.go` shrinks to Kong wiring + pipeline composition. Bubble render/ink files deleted. Radial/spiral keep their code; only imports change. |

## Design

### `internal/inks`

```go
package inks

// BuildMetricInk creates an Ink for a given metric, using the appropriate
// constructor based on the metric kind (numeric vs categorical). Returns a
// fixed-colour ink when the metric is unknown or when no values are present.
func BuildMetricInk(
    root *model.Directory,
    m metric.Name,
    palName palette.PaletteName,
    fallback color.RGBA,
) canvas.Ink

// MetricValueForFile builds a MetricValue from a file's data for the given
// ink. Returns the zero MetricValue when file is nil, when the ink is fixed,
// or when the file has no value for the ink's metric.
func MetricValueForFile(file *model.File, ink canvas.Ink) canvas.MetricValue

// CollectNumericValues walks the directory tree and returns every file's
// numeric value for metric m (quantity or measure, in that order).
func CollectNumericValues(root *model.Directory, m metric.Name) []float64

// CollectDistinctTypes returns the sorted distinct classification values
// observed for metric m across all files under root.
func CollectDistinctTypes(root *model.Directory, m metric.Name) []string
```

Bodies move verbatim from `cmd/codeviz/ink_builder.go`. Tests move from
`cmd/codeviz/main_test.go` (the `TestCollectDistinctTypes_*` test) plus
new table-driven tests added for `BuildMetricInk` (numeric kind,
categorical kind, unknown-metric fallback) and `MetricValueForFile` (nil
file, file missing the metric, numeric/categorical happy paths).

### `internal/bubbletree`

New files:

- `state.go` — `State` and method set.
- `stages.go` — six viz-specific stage functions plus the unexported
  `resolveFillMetric` / `resolveLabels` helpers called from
  `ResolveMetrics`.
- `inks.go` — `Inks` struct and `BuildInks` constructor that wraps
  `inks.BuildMetricInk`.
- `render.go` — content of today's `cmd/codeviz/bubble_canvas.go`, with
  exported names where the package boundary requires it (`RenderToCanvas`
  and the helpers it needs across files). Internal-only helpers
  (`addBubbleBackground`, `addBubbleDirDiscs`, `addBubbleFileDiscs`,
  `addBubbleLabels`, `bubbleArcFontSize`, `indexBubbleNodes`,
  `collectBubbleDirEntries`) stay unexported within the package.
- `render_test.go` — new, see Testing.
- `inks_test.go` — moved bubble canvas / ink tests, retargeted at the new
  package and exported API.

`State`:

```go
type State struct {
    stages.CommonState

    Config             *config.Bubbletree
    IncludeBinaryFiles bool

    // Resolved during the pipeline:
    Size          metric.Name
    FillMetric    metric.Name
    FillPalette   palette.PaletteName
    BorderMetric  metric.Name
    BorderPalette palette.PaletteName
    Labels        LabelMode
    Inks          Inks
    Nodes         BubbleNode
    LegendConfig  *canvas.LegendConfig
}

func (s *State) Common() *stages.CommonState { return &s.CommonState }
func (s *State) IncludeBinary() bool         { return s.IncludeBinaryFiles }
```

Viz-specific stages (all `pipeline.Stage[*State]`):

| Stage              | Responsibility                                                                                                                                                                                                                            |
| ------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ResolveMetrics`   | Sets `Size`, `FillMetric`, `FillPalette`, `BorderMetric`, `BorderPalette`, `Labels`; populates `Common().Requested` via `stages.CollectRequestedMetrics`.                                                                                 |
| `BuildInksStage`   | Emits the `slog.Info("Rendering image", "output", …, "width", …, "height", …)` line; calls `BuildInks(Common().Root, …)`; stores result in `s.Inks`.                                                                                      |
| `BuildLegendStage` | Calls `legend.ResolveOptions(ptrString(cfg.Legend), ptrString(cfg.LegendOrientation))` then `legend.Build(...)` with `s.Inks.Fill / FillMetric / Inks.Border / BorderMetric / Size`; stores in `s.LegendConfig`.                          |
| `LayoutStage`      | Runs `bubbletree.Layout(Common().Root, Common().Width, Common().Height, s.Size, s.Labels)` and stores into `s.Nodes`. No legend reservation, no offset.                                                                                   |
| `RenderStage`      | Calls `RenderToCanvas(&s.Nodes, Common().Root, Common().Width, Common().Height, s.Inks)`. If `s.LegendConfig != nil`, calls `cv.SetLegend(*s.LegendConfig)`. Assigns to `Common().Canvas`.                                                |
| `LogResult`        | Emits the final `slog.Info("Rendered bubble tree", …)` line with the same keys and values as today: `files`, `directories`, `output`, `width`, `height`, `size_metric`, `fill_metric`, `fill_palette`, `border_metric`, `border_palette`. |

`Inks`:

```go
type Inks struct {
    Fill   canvas.Ink
    Border canvas.Ink
}

func BuildInks(
    root *model.Directory,
    fillMetric metric.Name,
    fillPaletteName palette.PaletteName,
    borderMetric metric.Name,
    borderPaletteName palette.PaletteName,
) Inks
```

Default colours (`bubbleDefaultFileFill`, `bubbleDefaultDirFill`,
`bubbleDefaultBorder`, `bubbleLabelColour`, `bubbleBgColour`) and the
const block at the top of `bubble_canvas.go` move into `render.go`
unchanged.

### `internal/treemap` change

`internal/treemap/inks.go` is simplified to:

```go
package treemap

type Inks struct {
    Fill   canvas.Ink
    Border canvas.Ink
}

func BuildInks(root *model.Directory, fillMetric metric.Name, fillPaletteName palette.PaletteName, borderMetric metric.Name, borderPaletteName palette.PaletteName) Inks {
    inks := Inks{Border: canvas.FixedInk(structuralBorder)}
    inks.Fill = inks.BuildMetricInk(...)  // -> inks.BuildMetricInk via internal/inks
    ...
}
```

The local `buildMetricInk`, `metricValueForFile`, `collectNumericValues`,
`collectDistinctTypes` are deleted. `render.go` updates its two call
sites for `metricValueForFile` to use `inks.MetricValueForFile`.

### `cmd/codeviz` changes

**`bubbletree_cmd.go`** after the refactor keeps:

- The `BubbletreeCmd` Kong struct (unchanged).
- `Validate`, `validateConfig`, `mergeConfigAndValidate`, `applyOverrides`
  (unchanged).
- A `Run` method shaped exactly like `TreemapCmd.Run`:

```go
func (c *BubbletreeCmd) Run(flags *Flags) error {
    if err := c.mergeConfigAndValidate(flags); err != nil {
        return err
    }

    state := &bubbletree.State{
        CommonState: stages.CommonState{
            TargetPath: c.TargetPath,
            Output:     c.Output,
            Flags:      toStagesFlags(flags),
            RootConfig: flags.Config,
            CLIFilters: c.Filter,
        },
        Config:             flags.Config.Bubbletree,
        IncludeBinaryFiles: c.IncludeBinaryFiles,
    }

    _, err := pipeline.Run[*bubbletree.State](
        state,
        stages.ValidatePaths,
        stages.ExportConfig,
        stages.BuildFilterRules,
        bubbletree.ResolveMetrics,
        stages.ScanFilesystem,
        stages.CheckGitRequirement,
        stages.RunProviders,
        stages.FilterBinaryFiles,
        stages.ExportData,
        stages.ResolveDimensions,
        bubbletree.BuildInksStage,
        bubbletree.BuildLegendStage,
        bubbletree.LayoutStage,
        bubbletree.RenderStage,
        stages.WriteCanvas,
        bubbletree.LogResult,
    )

    return eris.Wrap(err, "bubbletree pipeline failed")
}
```

`resolveFillMetric` and `resolveLabels` move into `internal/bubbletree`.
`renderAndLog` is deleted; its logic now lives in `BuildInksStage`,
`BuildLegendStage`, `LayoutStage`, `RenderStage`, `WriteCanvas`, and
`LogResult`.

**`bubble_canvas.go`, `bubble_canvas_test.go`** — deleted. Contents
relocated to `internal/bubbletree`.

**`shape_inks.go`** — deleted. The struct existed only to hold `fill` /
`border` ink pairs for the bubble, radial, and spiral renderers.
`radial_canvas.go` gets a local unexported `radialInks` struct identical
in shape; `spiral_canvas.go` gets `spiralInks` if it uses one. Mechanical
change in both files.

**`ink_builder.go`** — deleted. All four helpers now live in
`internal/inks`. `radial_canvas.go` imports the new package and calls
`inks.BuildMetricInk` / `inks.MetricValueForFile` at its existing call
sites. `spiral_canvas.go` is not touched by this change beyond the
`shape_inks` rename, since it uses its own `buildBucketInk` /
`spiralMetricValue` helpers.

## Testing strategy

1. **`internal/inks/inks_test.go`** — `TestCollectDistinctTypes_*` ported
   from `cmd/codeviz/main_test.go` (plus a renaming pass to its exported
   form). New table tests: `TestBuildMetricInk_Numeric`,
   `TestBuildMetricInk_Categorical`,
   `TestBuildMetricInk_UnknownMetricFallback`,
   `TestBuildMetricInk_EmptyValuesFallback`,
   `TestMetricValueForFile_Numeric`,
   `TestMetricValueForFile_Categorical`,
   `TestMetricValueForFile_NilFile`,
   `TestMetricValueForFile_FixedInk`.

2. **`internal/bubbletree/inks_test.go`** — the existing
   `TestBuildBubbleInks_DefaultColours`, four `TestBubbleArcFontSize_*`,
   and index/walk tests from `bubble_canvas_test.go`, ported to the new
   package and exported names.

3. **`internal/bubbletree/render_test.go`** — new, modelled on
   `internal/treemap/render_test.go`:
   - `TestRenderBubbleToCanvas_PNG` — multi-file fixture renders to a
     decodable PNG of the requested dimensions.
   - `TestRenderBubbleToCanvas_SVG` — output XML root element is `<svg>`.
   - `TestRenderBubbleToCanvas_JPG` — decodable JPEG.
   - `TestRenderBubbleToCanvas_EmptyDirectory` — port of the existing
     test of the same name.
   - `TestRenderBubbleToCanvas_LabelsAll` — labels mode `all` adds file
     labels to the canvas; assert at least one text shape exists.
   - `TestRenderBubbleToCanvas_LabelsNone` — labels mode `none` adds no
     text shapes.
   - `TestRenderBubbleToCanvas_NumericFill` — numeric fill ink kind.
   - `TestRenderBubbleToCanvas_CategoricalFill` — categorical fill ink
     kind with a non-default border palette.

4. **`internal/treemap`** — existing tests must pass unchanged after the
   `internal/inks` extraction. The two `metricValueForFile` call sites in
   `render.go` switch to `inks.MetricValueForFile`; no test changes.

5. **`cmd/codeviz/main_test.go`** — `TestBubbletreeCmd_Validate_*` tests
   continue to pass unchanged. `TestCollectDistinctTypes_*` is removed
   from this file (relocated to `internal/inks`).

## Implementation sequencing

Each step keeps `task ci` green and can be committed independently.

1. **Create `internal/inks`.**
   - Move the four helpers verbatim, export them, and export their
     supporting types if any.
   - Update `internal/treemap/inks.go` to delegate.
   - Update `cmd/codeviz/bubble_canvas.go` and `radial_canvas.go` to
     import `internal/inks` at their `buildMetricInk` and
     `metricValueForFile` call sites. `spiral_canvas.go` does not use
     these helpers, so no import change there.
   - Move `TestCollectDistinctTypes_*` to the new package; add the new
     table tests.
   - Delete `cmd/codeviz/ink_builder.go`.

2. **Move bubble render code into `internal/bubbletree`.**
   - Create `render.go` (from `bubble_canvas.go`), `inks.go` (`Inks` +
     `BuildInks`).
   - Export the function called from `cmd/codeviz` (`RenderToCanvas`,
     `BuildInks`). Helpers stay package-private.
   - Move `bubble_canvas_test.go` content to
     `internal/bubbletree/inks_test.go` (the ink-related tests) and any
     render-related tests into `render_test.go` as a starting point.
   - Update `bubbletree_cmd.go` to call the new exported funcs while
     keeping its open-coded `Run` for the moment.
   - Delete `cmd/codeviz/bubble_canvas.go`, `bubble_canvas_test.go`, and
     `shape_inks.go` (introducing per-viz local structs in
     `radial_canvas.go` and `spiral_canvas.go` to replace `shapeInks`).

3. **Add comprehensive `render_test.go` coverage** — the eight tests
   listed in Testing point 3. This locks render behavior before the
   pipeline rewrite.

4. **Introduce the pipeline.**
   - Add `internal/bubbletree/state.go` and `stages.go`.
   - Rewrite `BubbletreeCmd.Run` as a `pipeline.Run` composition.
   - Remove `renderAndLog`, `resolveFillMetric`, and `resolveLabels` from
     `bubbletree_cmd.go`.

## Risks

- **`shape_inks` deletion fan-out.** Removing the shared struct touches
  `radial_canvas.go` and `spiral_canvas.go`. Mitigation: define
  per-file unexported `radialInks` / `spiralInks` structs with the same
  shape; the diff is mechanical and confined to one file each.
- **Behavior drift during the cross-package move.** Bubble render code
  moving from `cmd/codeviz` to `internal/bubbletree` could regress
  pixel-level output. Mitigation: step 3 adds render tests before the
  pipeline rewrite, locking decodability + structural shape (text shape
  count for labels modes, ink kinds via `inks.BuildInks` constructors).
  Canvas output isn't deterministic byte-wise across runs, which is why
  we don't use fixture-based comparisons.
- **`internal/inks` API churn.** Exporting four helpers locks their
  signatures. The signatures are simple and unchanged from today; risk
  is low. Anything that later wants to consume them from a different
  angle can add new functions without breaking the existing ones.
- **Forgotten `slog` fields.** `LogResult` must match today's log keys
  and values byte-for-byte (matters for log-driven tooling). Mitigation:
  side-by-side diff of the existing `renderAndLog` log call against
  `LogResult` during step 4.

## YAGNI exclusions

- No legend space reservation for bubbletree (preserves current overlay
  behavior).
- No consolidation of `bubbletree.Inks` and `treemap.Inks` types.
- No byte-identical golden fixtures (Goldie not adopted).
- No pipeline migration for radial or spiral in this spec; only their
  ink-helper imports change.
- No new conditional, parallel, or middleware-style pipeline features.
