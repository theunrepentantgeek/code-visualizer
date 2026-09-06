---
title: Usage
weight: 1
---

## Synopsis

```text
codeviz [global flags] <visualization> [flags] <target-path>
```

Visualisations: `tree-map`, `radial-tree`, `bubble-tree`, `spiral`, and
`scatter`. The `render` command produces the same images from named presets, so
you do not have to know which metrics and palettes to combine.

## Global flags

These flags apply to every subcommand.

| Flag              | Short | Description                                                          |
| ----------------- | ----- | ------------------------------------------------------------------- |
| `--quiet`         | `-q`  | Suppress all non-essential output (warnings and errors only)        |
| `--verbose`       | `-v`  | Show detailed progress during scanning and metrics                  |
| `--debug`         |       | Show per-directory scan progress (implies `--verbose`)              |
| `--config`        |       | Path to configuration file (`.yaml`, `.yml`, or `.json`)            |
| `--export-config` |       | Write effective configuration to file (`.yaml`, `.yml`, or `.json`) |
| `--export-data`   |       | Write computed metrics to file (`.json` or `.yaml`/`.yml`)          |

## Commands

Each visualisation has its own reference page describing the flags it accepts:

- [tree-map]({{< relref "tree-map" >}}) — files as nested rectangles sized by a metric.
- [radial-tree]({{< relref "radial-tree" >}}) — the folder hierarchy fanned out from a central root.
- [bubble-tree]({{< relref "bubble-tree" >}}) — files as circles packed into enclosing bubbles.
- [spiral]({{< relref "spiral" >}}) — commit activity plotted along a spiral of time.
- [scatter]({{< relref "scatter" >}}) — files positioned by two metrics, one on each axis.
- [render]({{< relref "render" >}}) — named presets that combine a visualisation, metrics, and a palette.

See [Shared concepts]({{< relref "/docs/shared-concepts" >}}) for the metric names, palettes, and
the include and exclude filter rules that every command shares.

## Git history constraints

Every visualization command and `render` accepts the following Git history
constraints:

| Flag            | Includes                                                     |
| --------------- | ------------------------------------------------------------ |
| `--from <ref>`  | Commits after a revision, or on/after a date or timestamp    |
| `--until <ref>` | A revision and its history, or commits on/before a timestamp |

`<ref>` is resolved as an exact tag, then a unique short or full commit ID,
then a date. Use `tag:`, `sha:`, or `date:` to force an interpretation. Dates
accept ISO 8601 or `YYYYMMDD[-HHMM][Z]`. Values without a timezone use local
time; a date-only `--until` includes the complete day.

Reference types may be mixed:

```sh
codeviz tree-map . -o release.png -f commit-count \
  --from v1.0 --until tag:v2.0

codeviz tree-map . -o recent.png -f commit-count \
  --from sha:a1b2c3d --until 20260905
```

Revision ranges follow the full Git commit graph. The lower revision is
exclusive and must be an ancestor of the upper revision, or of `HEAD` when the
upper bound is a date or omitted. The upper revision is inclusive and may be
outside the current `HEAD` history.

By default, these options constrain Git-derived metrics and timeline history
without removing unchanged files from the current checkout. Add
`--changed-only` (or set `changedOnly: true` in configuration) to retain only
current-tree files modified by commits in the effective range:

```sh
codeviz tree-map . -o release.png --from v1.0 --until v2.0 --changed-only
```

`--changed-only` requires at least one `--from` or `--until` bound. Existing
include/exclude rules and binary-file exclusion run first. Deleted files remain
absent because the visualization uses the current tree; a renamed file is shown
under its current destination path. Empty directories are removed.

## Examples

The examples below exercise the global flags. Each visualisation page carries
examples specific to that command.

### Export the effective configuration

```sh
codeviz --export-config config.yaml tree-map ./src -o treemap.png -s file-size
```

### Export computed metrics to JSON

Writes a JSON file containing the full file tree and all computed metric values,
which is useful for downstream analysis or for building custom visualisations.

```sh
codeviz --export-data metrics.json tree-map ./src -o treemap.png -s file-size -f file-type
```

### Export computed metrics to YAML

```sh
codeviz --export-data metrics.yaml tree-map ./src -o treemap.png -s file-lines
```

## Exit codes

| Code | Meaning                                              |
| ---- | ---------------------------------------------------- |
| 0    | Success — image written to output path               |
| 1    | Invalid arguments or validation failure              |
| 2    | Target path does not exist or is not a directory     |
| 3    | Git-required metric used on non-git directory        |
| 4    | Output path error (parent missing, permission)       |
| 5    | Internal error during scan or render                 |
| 6    | No files available after filtering (e.g. all binary) |
