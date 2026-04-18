# Ripley — History

## Core Context

- **Project:** A Go CLI tool that scans file trees and renders treemap visualizations as PNG images with configurable metrics and colour palettes.
- **Role:** Lead
- **Joined:** 2026-04-14T09:49:33.769Z

## Learnings

<!-- Append learnings below -->

### PR #45 Review — Image Format Support (2026-04-18) — COMPLETED

- **Review comments addressed:** 6 issues across 6 files — all confirmed by repo owner.
- **Double-close pattern:** `defer f.Close()` + explicit `f.Close()` at return causes double-close. Fix: named return `(err error)` + deferred closure that conditionally assigns close error. Applied to `svg_radial.go`, `svg_treemap.go`, `save.go`.
- **XML unmarshal in tests:** `xml.Unmarshal(data, new(any))` fails because `encoding/xml` can't unmarshal into `*interface{}`. Use `var parsed struct{}` + `&parsed` instead.
- **CLI help text:** Both `treemap_cmd.go` and `radialtree_cmd.go` needed `jpeg` added to the supported formats list alongside `jpg`.
- **Key files:** `internal/render/svg_radial.go`, `internal/render/svg_treemap.go`, `internal/render/save.go`, `internal/render/renderer_test.go`, `cmd/codeviz/treemap_cmd.go`, `cmd/codeviz/radialtree_cmd.go`.
- **CI status:** All 8 check runs were green before push; build+tests pass locally after fixes.
- **Result:** 1 commit pushed to `feature/44-image-format-support`.
