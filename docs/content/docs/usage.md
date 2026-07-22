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
