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
### PR #51 Review — Palette Documentation (2026-04-18) — COMPLETED

- **Review comments addressed:** Two items in `tools/swatches/main.go` — replaced hard-coded palette list with `palette.Names()` (new function), added directory existence guard in `writeSwatch`.
- **CI fix:** `revive` cognitive-complexity (11 > 10) and `wsl_v5` whitespace lint. Resolved by extracting `createSwatchImage` helper.
- **Key files:** `internal/palette/palette.go` (added `Names()` func), `tools/swatches/main.go`.
- **Pattern:** CI runs `task ci` inside devcontainer; local lint requires `golangci-lint-custom` with nilaway plugin. Use `go vet` locally as smoke check.
- **Palette package:** `palettes` map is source of truth. `Names()` returns sorted slice from it — single source of truth for tooling.
- **Result:** Build and tests pass. 2 commits pushed to `docs/palette-documentation`.

### Merge Conflict Resolution — PR #45 vs PR #51 (2026-04-18)

- **Cause:** PR #51 (palette documentation) merged into `main` while PR #45 (image format support) was still open. Both PRs touched `cmd/codeviz/treemap_cmd.go`. PR #51's `main` kept the old `render.RenderPNG` call; PR #45 had replaced it with the format-aware `render.Render`. Git couldn't auto-merge the overlapping hunk.
- **Conflict scope:** Single file (`cmd/codeviz/treemap_cmd.go`), single hunk — the render call inside `renderAndLog()`.
- **Resolution:** Kept PR #45's `render.Render` call (multi-format entrypoint) with its debug log line, discarding the stale `render.RenderPNG` reference from main. Both branches' intent preserved.
- **Verification:** `task build` and `task test` pass. All 14 packages green. Pushed merge commit to origin.
- **Process learning:** Always rebase PR branches onto latest main before pushing fixes, especially when multiple PRs are in flight touching related code. A quick `git fetch origin && git merge origin/main` (or rebase) before pushing would have caught this conflict locally instead of leaving the PR in a dirty state on GitHub.
