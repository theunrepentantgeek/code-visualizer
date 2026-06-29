# Rename `run` to `render` for Preset Invocation

**Date:** 2026-06-29
**Status:** Proposed

## Summary

Rename the preset-runner subcommand from `codeviz run <preset>` to
`codeviz render <preset>`. Pure rename — no behaviour, feature, or
preset-identifier changes. The Go type `RunCmd` becomes `RenderCmd` and its
file pair is renamed accordingly.

This is a hard break — no `run` alias is preserved — consistent with the
precedent set by the prior viz-promotion rename.

## Motivation

`run` is an underspecified verb. Every command in this CLI "runs"; the word
adds no information about what the user is asking for. `render <preset>` is
specific: it names the action (produce an image) and dovetails with the
mental model already established by the explicit viz commands (`codeviz
tree-map`, `codeviz radial-tree`, etc.).

The distinction this CLI now draws is:

- **Explicit viz commands** name the visualization type and take full control
  of metrics and palettes (`codeviz tree-map . -o out.png -s file-size`).
- **`render <preset>`** is the easy mode: pick a curated combination by name
  without knowing which metrics and palettes pair well
  (`codeviz render structure-tree-map . -o out.png`).

Both produce a rendered image. Naming the entry point `render` makes the
shared purpose explicit and removes the "what does run do?" friction.

## Command Surface

| Before                                  | After                                      |
| --------------------------------------- | ------------------------------------------ |
| `codeviz run`                           | `codeviz render`                           |
| `codeviz run <preset> <target> -o …`    | `codeviz render <preset> <target> -o …`    |

`codeviz run <anything>` returns kong's standard unknown-command error after
this change. There is no alias and no deprecation warning.

## Changes

### CLI mounting (`cmd/codeviz/main.go`)

Replace the `Run RunCmd` field of `CLI` with `Render RenderCmd`. Kong derives
the command name `render` from the field name, so no explicit `name:` tag is
needed. The `help:` text is updated to match.

Before:

```go
Run        RunCmd        `cmd:""                    help:"Run a preset visualization."`
```

After:

```go
Render     RenderCmd     `cmd:""                    help:"Render a preset visualization."`
```

### File rename

- `cmd/codeviz/run_cmd.go` → `cmd/codeviz/render_cmd.go`
- `cmd/codeviz/run_cmd_test.go` → `cmd/codeviz/render_cmd_test.go`

Done via `git mv` so history follows.

A `cmd/codeviz/render_cmd.go` file existed in the previous (`render`-as-viz-
group) iteration of the CLI and was deleted as part of the viz-promotion
work. The name is now free.

### Inside `render_cmd.go`

- Rename Go type `RunCmd` → `RenderCmd` (struct definition + every method
  receiver: `Validate`, `Run`, `listPresets`, `effectiveTitle`, `runPreset`,
  `structureTreemap`, `structureBubbletree`, `historyTreemap`, `ageTreemap`,
  `contributorsTreemap`).
- The `Run` method (invoked by kong) keeps its name. `runPreset` and other
  internal methods keep their lowercase names — they are implementation
  detail, not user-facing.
- Update the file-level doc comment block so the usage examples and the
  type-level description use `render`:

  ```go
  // RenderCmd renders a named preset — a predefined combination of
  // visualization, metrics, and palette that generates a useful image
  // without requiring the caller to know which metrics and palettes to
  // combine.
  //
  // Usage:
  //
  //	codeviz render                                  # list available presets
  //	codeviz render <preset> <target> -o <output>    # render a preset
  ```

- Update the `Preset` field tag's help text from "Name of the preset to run;
  omit to list available presets." to "Name of the preset to render; omit to
  list available presets."

### Inside `render_cmd_test.go`

- Replace every `RunCmd{…}` literal with `RenderCmd{…}`.
- Rename test functions `TestRunCmd_*` → `TestRenderCmd_*` for consistency
  (`TestRunCmd_Validate_NoArgs_ListMode`,
  `TestRunCmd_Validate_KnownPreset_Valid`,
  `TestRunCmd_Validate_UnknownPreset_ReturnsError`,
  `TestRunCmd_Validate_MissingTarget_ReturnsError`,
  `TestRunCmd_Validate_MissingOutput_ReturnsError`,
  `TestRunCmd_EffectiveTitle_UsesTitleWhenSet`,
  `TestRunCmd_EffectiveTitle_UsesPresetDefaultWhenTitleEmpty`,
  `TestRunCmd_AllPresets_RegisteredAndUnique`,
  `TestRunCmd_ParsedFromCLI_NoArgs`).
- Change the one `parser.Parse([]string{"run"})` site to
  `parser.Parse([]string{"render"})`.

## Out of Scope

Per Q1 scope check during brainstorming — these surfaces do not reference
`run` or presets today, so there is nothing to rename:

- `docs/usage.md` — presets are not documented; writing preset docs is a
  separate piece of work.
- `samples/` — no preset sample configs exist.
- `Taskfile.yml` — no preset-related targets exist.

Also unchanged:

- Preset identifiers (`structure-tree-map`, `structure-bubble-tree`,
  `history-tree-map`, `age-tree-map`, `contributors-tree-map`).
- `docs/superpowers/plans/**`, `docs/superpowers/specs/**` (other than this
  spec), `.squad/decisions.md` — historical record left intact.

## Verification

- `task build` succeeds.
- `task test` passes (renamed test functions still run).
- `task lint` is clean.
- Manual smoke:
  - `./bin/codeviz render` prints the preset table (no preset name = list
    mode).
  - `./bin/codeviz render structure-tree-map . -o /tmp/cv-preset.png` exits 0
    and writes a non-empty PNG.
  - `./bin/codeviz run` exits non-zero with kong's unknown-command error.
