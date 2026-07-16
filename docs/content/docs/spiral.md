---
title: spiral
weight: 5
---

The `spiral` visualisation plots project activity along a spiral of time. Each
lap represents one period — a day or an hour — and every spot is a time bucket
whose discs are sized by an optional metric. It reveals when a codebase was busy
and when it lay dormant, so it requires the target directory to be inside a git
repository.

## Synopsis

```text
codeviz spiral [flags] <target-path>
```

## Required flags

| Flag       | Short | Values                          | Description            |
| ---------- | ----- | ------------------------------- | ---------------------- |
| `--output` | `-o`  | `.png`, `.jpg`, `.jpeg`, `.svg` | Output image file path |

## Optional flags

| Flag                   | Short | Default        | Description                                                        |
| ---------------------- | ----- | -------------- | ----------------------------------------------------------------- |
| `--size`               | `-s`  | none           | Numeric metric for disc size; see `codeviz help metrics`          |
| `--resolution`         | `-r`  | `daily`        | Time resolution: `daily` or `hourly`                              |
| `--fill`               | `-f`  | none           | Fill colour: `metric[,palette]` (e.g. `file-type,categorization`) |
| `--border`             | `-b`  | none           | Border colour: `metric[,palette]` (e.g. `file-lines,foliage`)     |
| `--labels`             |       | `laps`         | Labels to display: `all`, `laps`, or `none`                       |
| `--legend`             |       | `bottom-right` | Legend position, or `none` to hide it                             |
| `--legend-orientation` |       | auto           | Legend orientation: `vertical` or `horizontal`                    |
| `--width`              |       | `1920`         | Canvas width in pixels                                            |
| `--height`             |       | `1920`         | Canvas height in pixels                                           |
| `--title`              |       | none           | Override the title text on the generated image                    |
| `--footer`             |       | none           | Override the footer text on the generated image                   |
| `--hide-footer`        |       | `false`        | Suppress the attribution footer                                   |
| `--include`            |       | none           | Include matching files; simple glob (repeatable)                  |
| `--exclude`            |       | none           | Exclude matching files; simple glob (repeatable)                  |
| `--include-binary-files` |     | `false`        | Include binary files, which are excluded by default               |

See [Shared concepts]({{< relref "/docs/shared-concepts" >}}) for the list of metric names,
palettes, and the include and exclude filter rules.

## Examples

Plot the daily commit history of a repository:

```sh
codeviz spiral ./src -o spiral.png
```

Switch to an hourly resolution and size discs by line count:

```sh
codeviz spiral ./src -o spiral.png -s file-lines -r hourly
```
