---
name: "add-visualization"
description: "How to add a new visualization type to code-visualizer"
domain: "architecture"
confidence: "high"
source: "observed from treemap + radialtree implementations"
---

## Context

When adding a new visualization type (e.g., bubbletree, sunburst, flamegraph), the codebase follows a strict structural pattern. Every visualization has five integration points that must be wired up consistently.

## Patterns

### 1. Layout Package — `internal/{vizname}/`

Three files:
- `node.go` — Visual node struct with: position fields, `FillColour color.RGBA`, `BorderColour *color.RGBA`, `Label string`, `ShowLabel bool`, `IsDirectory bool`, `Children []{NodeType}`.
- `layout.go` — `Layout(root *model.Directory, ..., sizeMetric metric.Name, ...) {NodeType}` function. Takes the model tree and returns a positioned visual tree. Colours are NOT set here — the renderer/CLI does that.
- `layout_test.go` — Gomega assertions, `t.Parallel()`, covers: positioning, metric scaling, label modes, empty/edge cases.

### 2. Config — `internal/config/{vizname}.go`

Struct with pointer fields: `Fill`, `FillPalette`, `Border`, `BorderPalette`, `Labels` (all `*string`). Nil = unset. Add to `Config` struct in `config.go` and initialize in `New()`.

### 3. Renderer — `internal/render/`

Two files:
- `{vizname}.go` — Entry point `Render{Viz}(root *{pkg}.{NodeType}, ..., outputPath string) error`. Dispatches by `FormatFromPath()` to PNG/JPG (via `fogleman/gg`) or SVG.
- `svg_{vizname}.go` — SVG-specific rendering.

Three-pass rendering for z-order: background → shapes → labels.

### 4. CLI Command — `cmd/codeviz/{vizname}_cmd.go`

Kong struct with: `TargetPath`, `Output`, size metric flag, `Fill`/`FillPalette`/`Border`/`BorderPalette`, `Labels`, `Width`/`Height`, `Filter`. Methods: `Validate()`, `Run(flags *Flags)`, `applyOverrides(cfg)`, `validatePaths()`, `buildFilterRules()`, `checkGitRequirement()`, `filterBinaryFiles()`.

Register in `render_cmd.go` as a new field on `RenderCmd`.

### 5. Colour Application

Parallel walk of layout node tree + `model.Directory` tree. Two paths: numeric (bucket mapping via `metric.ComputeBuckets` + `palette.MapNumericToColour`) and categorical (`palette.NewCategoricalMapper`). Applied separately for fill and border.

## Examples

- Treemap: `internal/treemap/`, `internal/render/renderer.go`, `internal/render/svg_treemap.go`, `cmd/codeviz/treemap_cmd.go`, `internal/config/treemap.go`
- Radial: `internal/radialtree/`, `internal/render/radialtree.go`, `internal/render/svg_radial.go`, `cmd/codeviz/radialtree_cmd.go`, `internal/config/radialtree.go`

## Variant: Flat-Sequence Visualizations (e.g., spiral)

Some visualizations don't map a file tree to visual nodes — they map a *sequence* (e.g., time buckets) to a flat list. Key differences from tree-based visualizations:

- **Layout returns a slice**, not a root node: `Layout(...) []NodeType` instead of `Layout(...) NodeType`.
- **No Children field** on the node struct. Nodes are ordered, not hierarchical.
- **Colour application iterates a slice**, not parallel-walking a layout tree + model tree.
- **Data transformation step** in the CLI `Run()`: between provider execution and layout, there's an extra step to transform the model tree into the sequence the layout expects (e.g., building time buckets from git history).
- **Node struct may carry domain data** (e.g., `TimeStart`/`TimeEnd` for time-based sequences) that tree-based nodes don't need.

The five integration points still apply, but point 5 (colour application) simplifies to a linear loop.

## Anti-Patterns

- **Setting colours in the layout function.** Layout only computes positions and sizes. Colour application happens in the CLI command layer.
- **Using `default:` Kong tags for fields with config pointer equivalents.** This bypasses the config layer. Use empty defaults and let `config.New()` set defaults.
- **Single-pass rendering.** Causes z-order bugs. Always use separate passes for background, shapes, and labels.
- **Extending the core model for visualization-specific data.** If a visualization needs data structures not shared by others (e.g., time buckets), keep them in the visualization's own package.
