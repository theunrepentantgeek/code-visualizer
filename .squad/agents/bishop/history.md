# Bishop — History

## Core Context

- **Project:** A Go CLI tool (`codeviz`) that scans file trees and renders treemap visualizations as PNG images with configurable metrics and colour palettes.
- **Role:** Artificer — abstractions, interfaces, types, design patterns, code smells, structural integrity
- **Joined:** 2026-04-15
- **Requested by:** Bevan Arps

## Project Knowledge

- Language: Go 1.26+
- Key packages: `cmd/codeviz/` (entry), `internal/metric/`, `internal/palette/`, `internal/render/`, `internal/scan/`, `internal/treemap/`
- Build: `task build` → `bin/codeviz`
- Test: `task test` (`go test ./... -count=1`)
- Lint: `task lint` (golangci-lint v2 with nilaway, wsl_v5, revive, wrapcheck, gci)
- Format: `task fmt` (gofumpt); imports ordered by gci
- Error handling: eris wrapping throughout
- Test assertions: Gomega (not testify); golden files via Goldie v2
- Go is already a pattern-friendly language: interfaces are implicit, types are first-class, package boundaries enforce encapsulation
- Key structural seams to watch: metric/palette/render boundaries, how scan feeds treemap, how types flow from scan → treemap → render

## Learnings

<!-- Append learnings below -->
