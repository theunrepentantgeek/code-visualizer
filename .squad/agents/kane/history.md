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

