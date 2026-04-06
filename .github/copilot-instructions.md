# code-visualizer Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-04-06

## Active Technologies
- Docker, Bash, JSON (devcontainer config); Go 1.26 (project language) + `mcr.microsoft.com/devcontainers/go:2-1.26` base image (002-add-devcontainer)
- Go 1.26+ + Goldie v2 (`github.com/sebdah/goldie/v2`), Gomega (assertions), fogleman/gg (PNG rendering) (003-use-goldie)
- Go 1.26.1 + Kong (CLI), go-git (git metadata), fogleman/gg (PNG rendering), eris (error wrapping), Gomega (test assertions), Goldie v2 (golden-file snapshots) (004-exclude-binary-lines)
- N/A (stateless CLI) (004-exclude-binary-lines)

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
- 004-exclude-binary-lines: Added Go 1.26.1 + Kong (CLI), go-git (git metadata), fogleman/gg (PNG rendering), eris (error wrapping), Gomega (test assertions), Goldie v2 (golden-file snapshots)
- 003-use-goldie: Added Go 1.26+ + Goldie v2 (`github.com/sebdah/goldie/v2`), Gomega (assertions), fogleman/gg (PNG rendering)
- 002-add-devcontainer: Added Docker, Bash, JSON (devcontainer config); Go 1.26 (project language) + `mcr.microsoft.com/devcontainers/go:2-1.26` base image


<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
