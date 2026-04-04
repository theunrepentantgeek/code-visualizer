# Quickstart: CLI Treemap Visualization

**Branch**: `001-cli-treemap-viz` | **Date**: 2026-04-04

## Prerequisites

- Go 1.22+ installed
- `task` (Taskfile runner) installed
- Git (for running tests against git repos)

## Build

```bash
# Clone and enter the repo
cd code-visualizer

# Install dependencies
go mod download

# Build the CLI binary
task build
# Or directly:
go build -o codeviz ./cmd/codeviz/
```

## Run

```bash
# Simplest usage: treemap of a directory sized by file size
./codeviz ./myproject -o treemap.png --size file-size

# See all options
./codeviz --help
```

## Test

```bash
# Run all tests
task test
# Or directly:
go test ./...

# Run tests with verbose output
go test -v ./...

# Update golden files after intentional output changes
go test ./... -update
```

## Lint

```bash
task lint
# Or directly:
gofmt -l .
golangci-lint run
```

## Example Workflows

### Visualise a project by file size
```bash
./codeviz . -o size-treemap.png --size file-size
```

### Visualise a git repo coloured by file age
```bash
./codeviz . -o age-treemap.png --size file-lines --fill file-age --fill-palette temperature
```

### Three-metric visualisation
```bash
./codeviz . -o full-treemap.png \
  --size file-size \
  --fill file-freshness --fill-palette temperature \
  --border author-count --border-palette good-bad
```

## Project Layout

```
cmd/codeviz/main.go       # CLI entrypoint
internal/scan/             # Directory + git scanning
internal/metric/           # Metric computation + bucketing
internal/palette/          # Colour palette definitions + mapping
internal/treemap/          # Squarified treemap layout
internal/render/           # PNG rendering
tests/                     # Test files + testdata + golden files
```
