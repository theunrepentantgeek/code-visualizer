# Ripley — History

## Core Context

- **Project:** A Go CLI tool that scans file trees and renders treemap visualizations as PNG images with configurable metrics and colour palettes.
- **Role:** Lead
- **Joined:** 2026-04-14T09:49:33.769Z

## Learnings

<!-- Append learnings below -->

### PR #51 Review — Palette Documentation (2026-04-18) — COMPLETED

- **Review comments addressed:** Two items in `tools/swatches/main.go` — replaced hard-coded palette list with `palette.Names()` (new function), added directory existence guard in `writeSwatch`.
- **CI fix:** `revive` cognitive-complexity (11 > 10) and `wsl_v5` whitespace lint. Resolved by extracting `createSwatchImage` helper.
- **Key files:** `internal/palette/palette.go` (added `Names()` func), `tools/swatches/main.go`.
- **Pattern:** CI runs `task ci` inside devcontainer; local lint requires `golangci-lint-custom` with nilaway plugin. Use `go vet` locally as smoke check.
- **Palette package:** `palettes` map is source of truth. `Names()` returns sorted slice from it — single source of truth for tooling.
- **Result:** Build and tests pass. 2 commits pushed to `docs/palette-documentation`.
