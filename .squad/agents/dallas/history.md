# Dallas — History

## Core Context

- **Project:** A Go CLI tool that scans file trees and renders treemap visualizations as PNG images with configurable metrics and colour palettes.
- **Role:** Go Dev
- **Joined:** 2026-04-14T09:49:33.771Z

## Recent Work

- **PR #39 resolution (2026-04-15)**: Fixed FolderAuthorCountProvider dependencies, mean-file-lines description, git error logging. Extracted folder load helpers. Fixed float-to-days conversion and test error handling. Branch squad/38-extend-provider-capabilities task ci passed and pushed.

## Learnings

<!-- Append learnings below -->

### PR review fixes — radialtree_cmd.go + config (2026-04-15)

- **Kong defaults silently override config**: `default:"all"` on a Kong string field causes Kong to always set the value, so `applyOverrides` always writes over config file values. Fix: remove `default:` tags from Kong fields, add `""` to enum, set defaults in `config.New()` only.
- **`defaultStr` helper**: Added `func defaultStr(s string) *string { return &s }` to `internal/config/config.go` for use in `New()`.
- **`resolveLabels` simplification**: After config.New() guarantees `Labels` is always set, the `c.Labels != ""` fallback branch in `resolveLabels` is dead code and was removed.
- **RenderRadialPNG pointer call site**: Changed `render.RenderRadialPNG(nodes, ...)` → `render.RenderRadialPNG(&nodes, ...)` to align with Parker's upcoming signature change. Build fails until Parker's change lands (expected).
- **Colour-apply function invariant**: Documented in all four `applyRadial*` functions that `node.Children` must be files-first, then subdirectories — matching `layoutDir` output order.

### radialtree package (2026-04-15)

- **Package path:** `github.com/bevan/code-visualizer/internal/radialtree` — files `node.go` and `layout.go`
- **RadialNode fields:** `X, Y float64` (pixel offset from canvas centre), `DiscRadius float64` (disc pixel radius), `Angle float64` (radians, 0=east, π/2=down), `Label string`, `ShowLabel bool`, `IsDirectory bool`, `FillColour color.RGBA`, `BorderColour *color.RGBA`, `Children []RadialNode`
- **Layout() signature:** `Layout(root *model.Directory, canvasSize int, discMetric metric.Name, labels LabelMode) RadialNode`
- **LabelMode constants:** `LabelAll = "all"`, `LabelFoldersOnly = "folders"`, `LabelNone = "none"`
- **Algorithm:** ring spacing = (canvasSize/2 - 40) / (maxDepth+1); angular sectors proportional to leaf counts; files placed before subdirs; disc sizes scaled by discMetric quantity/measure value
- Removed pre-existing `radialtree.go` stub (replaced by `node.go`)

### PR #39 review fixes (2026-04-15)

- **FolderAuthorCountProvider.Dependencies()**: Added `gitprovider.AuthorCount` dependency for correct scheduler ordering. Note: this provider queries git directly (not via file metrics) to compute author union sets — simple count summation wouldn't give correct cross-file deduplication.
- **mean-file-lines description**: Updated to explicitly say "skips binary files" per the issue spec.
- **Git debug logging**: Removed the `!errors.Is(err, errUntracked)` guard so untracked-file events are now logged at `slog.Debug` rather than silently swallowed.
- **Folder load helpers**: Added five higher-level helpers to `metrics.go` (`loadMaxQuantity`, `loadMinQuantity`, `loadSumQuantity`, `loadMeanMeasure`, `loadPositiveMeanMeasure`) that encapsulate the full WalkDirectories loop — all 8 folder Load() methods now delegate to a single helper call.
- **Float conversion fix**: Changed `int64(age.Hours()/24)` to `int64(age/(24*time.Hour))` to use integer duration arithmetic, avoiding float precision loss on long durations.
- **Test error handling**: Fixed ignored `os.WriteFile` errors in folder test setup.
