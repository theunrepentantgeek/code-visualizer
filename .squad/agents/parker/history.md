# Parker — History

## Core Context

- **Project:** A Go CLI tool (`codeviz`) that scans file trees and renders treemap visualizations as PNG images with configurable metrics and colour palettes.
- **Role:** Staff Developer — technical quality, debt, maintainability, and long-term viability
- **Joined:** 2026-04-15
- **Requested by:** Bevan Arps

## Project Knowledge

- Language: Go 1.26+
- Key packages: `cmd/codeviz/` (entry), `internal/metric/`, `internal/palette/`, `internal/render/`, `internal/scan/`, `internal/treemap/`
- Build: `task build` → `bin/codeviz`
- Test: `task test` (`go test ./... -count=1`)
- Lint: `task lint` (golangci-lint v2 with nilaway, wsl_v5, revive, wrapcheck, gci)
- Format: `task fmt` (gofumpt)
- Full CI: `task ci` (build + test + lint)
- Error handling: eris wrapping throughout
- Test assertions: Gomega (not testify); golden files via Goldie v2
- Formatting enforced by gofumpt; import ordering by gci

## Learnings

<!-- Append learnings below -->

### RenderRadialPNG (2026-04-15)

- **Signature:** `func RenderRadialPNG(root radialtree.RadialNode, canvasSize int, outputPath string) error`  
  Located in `internal/render/radialtree.go`. Square canvas only; all node positions are offsets from canvas centre.

- **Three-pass rendering:** edges → discs → labels. Each pass is a full recursive traversal of the tree. Required to avoid z-order issues (e.g., parent discs drawn over child edges).

- **Label rotation:** Right half uses `RotateAbout(node.Angle)` + left anchor (ax=0). Left half uses `RotateAbout(node.Angle + π)` + right anchor (ax=1). This flips the baseline direction so characters stay upright. Root node (dist=0) gets an unrotated centred label.

- **Colour defaults:** file fill `#CCCCCC`, directory fill `#444444`, border `#333333`, edge `#999999`. Custom colours applied if `FillColour.A > 0` (fill) or `BorderColour != nil` (border).

- **Dallas's radialtree package** (`internal/radialtree/`) was already in progress when this renderer was written: `node.go` defines `RadialNode`, `layout.go` defines `Layout`. The `render_cmd.go` already references `RadialCmd` (pre-existing lint failure, not mine to fix).
