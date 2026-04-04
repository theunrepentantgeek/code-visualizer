# code-visualizer Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-04-04

## Active Technologies

- Go 1.26+ + Kong (CLI parsing), fogleman/gg (PNG rendering), go-git (git metadata) (001-cli-treemap-viz)

## Project Structure

```text
cmd/
  codeviz/
    main.go
internal/
  metric/
  palette/
  render/
  scan/
  treemap/
```

## Commands

- `task build` — Build the codeviz binary
- `task test` — Run all tests
- `task lint` — Run golangci-lint
- `task fmt` — Format with gofumpt
- `task tidy` — Format, mod tidy, lint fix
- `task ci` — Build, test, lint

## Code Style

Go 1.26+: Follow standard conventions, gofumpt formatting, eris error wrapping

## Recent Changes

- 001-cli-treemap-viz: Added Go 1.26+ + Kong (CLI parsing), fogleman/gg (PNG rendering), go-git (git metadata)

<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
