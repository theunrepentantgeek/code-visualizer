# Promote Visualization Commands to Top Level

**Date:** 2026-06-28
**Status:** Proposed

## Summary

Drop the `render` subcommand grouping and promote all five visualization
commands to top-level `codeviz` commands. Rename `radial` to `radial-tree` so
the tree-family commands share a consistent `<adj>-tree` form. Align the
config YAML section keys with the CLI command names, and fix the existing
`samples` task so its viz list matches real sample file names.

This is a hard break — no `render` alias is preserved — because the project is
pre-alpha and the `render` prefix carries no information.

## Motivation

`codeviz render tree-map` reads as redundant: every visualization command
renders, so the `render` group adds keystrokes without adding meaning. Help
output buries the actual choices one level deeper than necessary. Promoting
the five viz commands to the top level puts them next to `run` and `help`
where they belong.

While doing the rename, three pre-existing inconsistencies become visible and
are fixed together:

1. `radial` is the only viz that does not name what it draws. `radial-tree`
   matches `bubble-tree`, `tree-map`, and the internal `internal/radialtree/`
   package layout.
2. The YAML config section key for radial is `radial:`; once the CLI command
   is `radial-tree`, the section key should match.
3. The `samples` Taskfile target is broken: its viz matrix is
   `[treemap, bubbletree, radial, spiral, scatter]` but the sample files are
   `codeviz-tree-map.yml`, `codeviz-bubble-tree.yml`, etc., so the matrix
   never resolves to a real config path. Fixing the names is the natural
   place to fix the matrix.

## Command Surface

| Before                          | After                       |
| ------------------------------- | --------------------------- |
| `codeviz render tree-map …`     | `codeviz tree-map …`        |
| `codeviz render radial …`       | `codeviz radial-tree …`     |
| `codeviz render bubble-tree …`  | `codeviz bubble-tree …`     |
| `codeviz render spiral …`       | `codeviz spiral …`          |
| `codeviz render scatter …`      | `codeviz scatter …`         |
| `codeviz render` (group help)   | *(removed)*                 |
| `codeviz run …`                 | unchanged                   |
| `codeviz help …`                | unchanged                   |

`codeviz render <anything>` returns kong's standard unknown-command error
after this change. There is no alias and no deprecation warning.

## Changes

### CLI wiring (`cmd/codeviz/`)

- Delete `render_cmd.go`. `RenderCmd` is no longer needed.
- In `main.go`, replace the `Render RenderCmd` field of `CLI` with five
  sibling fields that mount the existing `*Cmd` types directly at the top
  level:

  ```go
  TreeMap    TreemapCmd    `cmd:"" name:"tree-map"    help:"Generate a tree-map visualization."`
  RadialTree RadialCmd     `cmd:"" name:"radial-tree" help:"Generate a radial tree visualization."`
  BubbleTree BubbletreeCmd `cmd:"" name:"bubble-tree" help:"Generate a bubble tree visualization."`
  Spiral     SpiralCmd     `cmd:""                    help:"Generate a spiral timeline visualization."`
  Scatter    ScatterCmd    `cmd:""                    help:"Generate a scatter plot visualization."`
  Run        RunCmd        `cmd:""                    help:"Run a preset visualization."`
  Help       HelpCmd       `cmd:""                    help:"Show this help message."`
  ```

  The existing `TreemapCmd`, `RadialCmd`, `BubbletreeCmd`, `SpiralCmd`, and
  `ScatterCmd` types — and their `Run(flags *Flags) error` methods — are not
  changed. Only the mount point changes.

### Tests

- `cmd/codeviz/main_test.go` — every test that builds an args slice
  containing `"render"` drops that element; entries with `"radial"` become
  `"radial-tree"`.
- `cmd/codeviz/render_matrix_test.go` — same edit; the matrix driver invokes
  the CLI via `os/exec`, so its arg list must use the promoted command name.
- No new test files are added. Existing coverage already exercises every viz
  through the CLI; updating the args is sufficient.

### Config / YAML

- `internal/config/config.go`:
  - Change the `Radial` field's struct tag from
    `yaml:"radial,omitempty" json:"radial,omitempty"` to
    `yaml:"radial-tree,omitempty" json:"radial-tree,omitempty"`.
  - In `Config.ForExport`, change `case "radial":` to
    `case "radial-tree":`.
- `internal/config/config_test.go` — the one
  `cfg.ForExport("radial")` call becomes `cfg.ForExport("radial-tree")`.
- `samples/`:
  - Rename `codeviz-radial.yml` → `codeviz-radial-tree.yml`.
  - In all five `samples/codeviz-*.yml` files, rename the `radial:` block
    key to `radial-tree:`.

### Docs

- `docs/usage.md`:
  - Update the synopsis (`codeviz [global flags] <viz> [flags] <target>`)
    and the subcommand list (`tree-map`, `radial-tree`, `bubble-tree`,
    `spiral`, `scatter`).
  - Rename every `## render <viz>` section heading and every
    `codeviz render <viz> …` example to the promoted form.

### Build

- `Taskfile.yml` — fix the `samples` task in one shot:

  ```yaml
  cmds:
    - for:
        matrix:
          IMAGE: [png, svg]
          VIZ: [tree-map, bubble-tree, radial-tree, spiral, scatter]
      cmd: "{{.CODEVIZ}} {{.ITEM.VIZ}} . --config samples/codeviz-{{.ITEM.VIZ}}.yml --output samples/codeviz-{{.ITEM.VIZ}}.{{.ITEM.IMAGE}} ; echo ''"
  ```

  This drops `render`, uses the new `radial-tree` name, and aligns matrix
  values with real `samples/codeviz-<name>.yml` filenames.

## Out of Scope

Historical records are left untouched:

- `docs/superpowers/plans/**`
- `docs/superpowers/specs/**` (everything other than this spec)
- `.squad/decisions.md`

These describe what the CLI was at the time each plan or decision was
written. Rewriting their command lines would distort the historical record.

`RunCmd` preset identifiers (`structure-tree-map`, `history-tree-map`, etc.)
are unchanged — they are preset names, not viz command names.

## Verification

- `task build` succeeds.
- `task test` passes after test arg updates.
- `task lint` is clean.
- `task samples` runs without error and produces all ten expected output
  files (5 viz × {png, svg}) under `samples/`.
- Manual smoke:
  - `./bin/codeviz tree-map . -o /tmp/t.png -s file-size` renders a PNG.
  - `./bin/codeviz radial-tree . -o /tmp/r.png -d file-size` renders a PNG.
  - `./bin/codeviz render tree-map .` exits non-zero with an
    unknown-command error.
