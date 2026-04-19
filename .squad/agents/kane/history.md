# Kane — History

## Core Context

- **Project:** A Go CLI tool that scans file trees and renders treemap visualizations as PNG images with configurable metrics and colour palettes.
- **Role:** CLI Dev
- **Joined:** 2026-04-14T09:49:33.772Z

## Learnings

<!-- Append learnings below -->

### RadialCmd structure and flags

- `RadialCmd` mirrors `TreemapCmd` but uses `DiscSize metric.Name` (short flag `-d`) instead of `Size` for the sizing metric.
- Additional `Labels string` flag with `default:"all"` and `enum:"all,folders,none"` controls label rendering via `radialtree.LabelMode`.
- `Width` and `Height` both default to 1920 (square canvas); canvas size is `min(width, height)`.
- All colour flags (Fill, FillPalette, Border, BorderPalette) are identical to `TreemapCmd`.

### How applyOverrides works for Radial config

- `applyOverrides` writes non-zero CLI flag values into `*config.Config`.
- Width/Height go to `cfg.Width`/`cfg.Height` (top-level); colour and labels fields go to `cfg.Radial.*`.
- A nil-guard `if cfg.Radial == nil { cfg.Radial = &config.Radial{} }` is needed because config may be loaded from file without a `radial:` section.
- Zero-valued CLI strings are transparent (config file values pass through unchanged).

### Bubble Tree — Architecture Proposal Ready (2026-04-19)

- **Issue #33:** Ripley completed architecture research for circle-packing bubble tree visualization.
- **Your role in Phase 4 (CLI + Config wiring):** Implement `BubbletreeCmd` struct in `cmd/codeviz/bubbletree_cmd.go` and update `RenderCmd` to register `Bubbletree` subcommand. Follow the radial/treemap pattern.
- **BubbletreeCmd fields:**
  - `TargetPath`, `Output` (required, format-aware like radial)
  - `Size metric.Name` (short `-s`, required, numeric metrics) — primary sizing metric
  - `Fill, FillPalette, Border, BorderPalette` (optional, identical to TreemapCmd/RadialCmd)
  - `Labels string` (enum: `all,folders,none`, default empty for config.New() to set)
  - `Width int` (default 1920), `Height int` (default 1080) — **non-square canvas, unlike radial** 
  - `Filter []string` (repeatable glob rules)
- **Default dimensions:** 1920×1080 (like treemap, not square like radial) — bubble layout adapts to non-square
- **Config defaults:** Handled in `config.New()` → `Labels = "folders"` (file dots unlabelled by design)
- **Colour application:** Parallel walk of BubbleNode tree + model.Directory tree (like radial). Separate implementation v1, consider extraction later.
- **Key files to create/update:** `cmd/codeviz/bubbletree_cmd.go` (new), `cmd/codeviz/render_cmd.go` (register subcommand), `internal/config/bubbletree.go` (new, with Bubbletree struct), `internal/config/config.go` (add Bubbletree field, update New()).
- **Dependency:** Awaits Phase 1 (Dallas) — `internal/bubbletree/layout.go` must exist before you can call Layout() in Run().

### Bubble Tree — Phase 4 CLI + Config Complete (2026-04-19)

- **Created `cmd/codeviz/bubbletree_cmd.go`:** Full BubbletreeCmd with Validate, Run, applyOverrides, validatePaths, buildFilterRules, checkGitRequirement, filterBinaryFiles, resolveLabels, and all colour application functions (numeric + categorical for both fill and border).
- **Created `internal/config/bubbletree.go`:** Bubbletree config struct with pointer fields (Fill, FillPalette, Border, BorderPalette, Labels).
- **Updated `internal/config/config.go`:** Added Bubbletree field to Config struct and initialized in New() with Labels default "folders".
- **Updated `cmd/codeviz/render_cmd.go`:** Registered BubbletreeCmd as `bubbletree` subcommand.
- **Key pattern differences from radial:** Size flag is `--size/-s` (not `--disc-size/-d`); default 1920×1080 (not square); Layout takes width+height (not canvasSize); LabelMode defaults to "folders" (not "all").
- **Won't compile yet:** Imports `internal/bubbletree` (Layout, BubbleNode, LabelMode) and `render.RenderBubble` — Dallas is building these in parallel.

