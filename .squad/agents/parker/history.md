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
