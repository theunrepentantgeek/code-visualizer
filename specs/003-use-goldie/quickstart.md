# Quickstart: Use Goldie for Golden File Testing

**Date**: 2026-04-05  
**Feature**: 003-use-goldie

## What Changed

The handwritten golden file comparison infrastructure in `internal/render/renderer_test.go` has been replaced with the [Goldie v2](https://github.com/sebdah/goldie) library.

## Running Tests

```bash
task test
```

All golden file tests run automatically as part of the standard test suite.

## Updating Golden Files

When rendered output intentionally changes, regenerate golden files using either:

```bash
# Via Taskfile
task update-golden-files

# Via go test directly
go test ./... -update

# Via environment variable
GOLDIE_UPDATE=1 go test ./...
```

## Files Affected

- `internal/render/renderer_test.go` — migrated golden file helper to use Goldie
- `internal/render/testdata/*.png` — golden files (unchanged in location)
- `go.mod` / `go.sum` — Goldie v2 dependency added
- `Taskfile.yml` — `update-golden-files` task updated to use `GOLDIE_UPDATE`
