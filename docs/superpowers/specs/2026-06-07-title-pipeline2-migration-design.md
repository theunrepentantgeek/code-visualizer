# Title Flag — Pipeline2 Migration Design

**Date:** 2026-06-07
**Branch:** `repo-assist/feat-viz-title-354-a0b6fbf2660dd9f4`
**Supersedes:** branch commit `4817c8d` (`feat(title): add --title flag to all visualization commands (#354)`)

## Context

The branch above adds a `--title` CLI flag to every visualization command. It was authored against the previous pipeline architecture (`pipeline.Run` + generic `Stage[S VizState]`), where shared stages were parametrised over a `VizState` interface.

Since then, PR #368 (`improve/migrate-pipeline`) has landed on `main`, replacing that architecture with a typed-state pipeline:

- `pipeline.NewState(values...)` — type-keyed value store.
- `pipeline.ApplyFuncX(s, fn) / ApplyFuncXY(...) / ApplyFuncXYZ(...)` — apply stages whose signatures are `func(X) error`, `func(X, Y) error`, `func(X, Y, Z) error`.
- Shared stages (e.g. `stages.ApplyFooter`, `stages.WriteCanvas`) are now plain functions over `*CommonState`, not generics.

The branch's title work needs to be reshaped onto this new architecture. The config layer, canvas layer, and CLI flag wiring carry over verbatim; only the *pipeline plumbing* changes.

## Goal

Re-deliver the `--title` feature on top of `origin/main`, using main's typed-state pipeline idioms, with zero behavioural change relative to the original PR #354.

## Strategy

1. `git reset --hard origin/main` on the current branch.
2. Re-apply the title work as a single new commit, conformed to the new pipeline.

This produces a clean diff against `origin/main` and avoids carrying conflict markers or stale migration scaffolding.

## Component Plan

### 1. Config layer (lift from original commit; no shape changes)

- **New** `internal/config/title.go`:
  - `type Title struct { Text *string; Hidden *bool }` with `yaml`/`json` tags.
  - `(*Title).ShowTitle() bool` — true iff non-nil, not hidden, and `Text` is non-empty.
  - `(*Title).OverrideText(v string)` — same `overrideString` helper as `Footer.OverrideText`.

- **New** `internal/config/title_test.go` — port from branch (mirrors `footer_test.go`).

- **Modify** `internal/config/config.go`:
  - Add field `Title *Title` to `Config` (next to `Footer`).
  - Add method `(*Config).OverrideTitleText(v string)` mirroring `OverrideFooterText`.
  - `New()` does not initialise `Title` (matches main's treatment of `Title` — opt-in via flag/config).

### 2. Canvas layer (lift from original commit; no shape changes)

- **Modify** `internal/canvas/canvas.go`:
  - Add constants `titleFontSize`, `titleMarginY`, exported `TitleReservedHeight = titleFontSize + titleMarginY`.
  - Add field `title *string` to `Canvas`.
  - Add `SetTitle(text string)` and `TitleText() string` mirroring `SetFooter` / `FooterText`.
  - Render the title in the appropriate backend draw method, mirroring how the footer is drawn at the bottom but anchored at the top.

### 3. Shared stages (this is the migration delta)

- **Modify** `internal/stages/canvas.go` — add (signatures match main's `ApplyFooter` / `EffectiveFooterHeight`):

  ```go
  // ApplyTitle sets the title on c.Canvas from RootConfig.Title.
  // No-op if the canvas or root config is nil, or if the title is hidden / empty.
  func ApplyTitle(c *CommonState) error {
      if c.Canvas == nil || c.RootConfig == nil {
          return nil
      }
      title := c.RootConfig.Title
      if !title.ShowTitle() {
          return nil
      }
      c.Canvas.SetTitle(*title.Text)
      return nil
  }

  // EffectiveTitleHeight returns the pixel height the title occupies when
  // rendered. Layout stages subtract this from the top of the available area.
  // Returns 0 when cfg is nil or the title is not shown.
  func EffectiveTitleHeight(cfg *config.Config) int {
      if cfg == nil || !cfg.Title.ShowTitle() {
          return 0
      }
      return int(canvas.TitleReservedHeight)
  }
  ```

- **Modify** `internal/stages/canvas_test.go` — add tests for `ApplyTitle` and `EffectiveTitleHeight` next to the existing footer tests, exercising `*CommonState` directly (no fake `VizState`).

### 4. Per-viz layout stages

For each of `bubbletree`, `radialtree`, `scatter`, `spiral`, `treemap` in `internal/<viz>/stages.go`:

- Inside `LayoutStage`, change:

  ```go
  availH := c.Height - stages.EffectiveFooterHeight(c.RootConfig)
  ```

  to:

  ```go
  titleH := stages.EffectiveTitleHeight(c.RootConfig)
  availH := c.Height - stages.EffectiveFooterHeight(c.RootConfig) - titleH
  ```

- Where the viz positions content vertically, offset it down by `titleH`. The exact mechanism is per-viz (e.g. `OffsetNodes` for bubbletree, gradient/centering offsets for radial). Mirror what the original commit did, transplanted to main's layout-stage shape.

- For `radialtree`, additionally re-apply the small `internal/radialtree/render.go` and `internal/radialtree/render_internal_test.go` deltas from the original commit (centering offset accounting for the title).

### 5. CLI commands

For each of `bubbletree_cmd.go`, `radialtree_cmd.go`, `scatter_cmd.go`, `spiral_cmd.go`, `treemap_cmd.go` in `cmd/codeviz/`:

- Add a struct field next to the existing `Footer` field:

  ```go
  Title string `default:"" help:"Override title text on the generated image." optional:""`
  ```

- In `applyOverrides`, add immediately after `cfg.OverrideFooterText(c.Footer)`:

  ```go
  cfg.OverrideTitleText(c.Title)
  ```

- In `Run`, insert immediately before `pipeline.ApplyFuncX(s, stages.ApplyFooter)`:

  ```go
  pipeline.ApplyFuncX(s, stages.ApplyTitle)
  ```

### 6. Out of scope

- Working-tree noise unrelated to the title port stays untouched: `.github/workflows/repo-assist.md`, `.go-version`, untracked `cmd/codeviz/debug-sample.png`, `.vscode/launch.json`, `.agents/skills/thermo-nuclear-code-quality-review/`.
- No refactors beyond what is strictly required to land the title feature on the new architecture.

## Verification

- `task test` — all tests green; new title tests in `internal/config/` and `internal/stages/` pass.
- `task ci` — fmt:check, mod:check, build, test, lint all green. Run via an Explore subagent (verbose linter output).
- Smoke-render: `bin/codeviz bubbletree . -o /tmp/x.png --title "Hello"` produces a PNG with the title visible at the top and the visualization content offset below it.

## Risk

Low. The migration target (the new pipeline) is well-established on `main` (already used by all five viz commands for `ApplyFooter`/`WriteCanvas`). Title is the smaller sibling of footer and follows the same pattern point for point.
