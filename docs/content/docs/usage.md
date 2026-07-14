---
title: Usage
weight: 1
---

## Synopsis

```
codeviz [global flags] <visualization> [flags] <target-path>
```

Visualizations: `tree-map`, `radial-tree`, `bubble-tree`, `spiral`, `scatter`

## Global Flags

These flags apply to all subcommands.

| Flag              | Short | Description                                              |
| ----------------- | ----- | -------------------------------------------------------- |
| `--quiet`         | `-q`  | Suppress all non-essential output (warnings/errors only) |
| `--verbose`       | `-v`  | Show detailed progress during scanning and metrics       |
| `--debug`         |       | Show per-directory scan progress (implies `--verbose`)   |
| `--config`        |       | Path to configuration file (`.yaml`, `.yml`, or `.json`) |
| `--export-config` |       | Write effective configuration to file (`.yaml`, `.yml`, or `.json`) |
| `--export-data`   |       | Write computed metrics to file (`.json` or `.yaml`/`.yml`) |

## `tree-map`

Generate a tree-map visualization where each file is a rectangle sized by a metric.

### Synopsis

```
codeviz tree-map [flags] <target-path>
```

### Required Flags

| Flag       | Short | Values                                                                  | Description               |
| ---------- | ----- | ----------------------------------------------------------------------- | ------------------------- |
| `--output` | `-o`  | `.png`, `.jpg`, `.jpeg`, `.svg`                                         | Output image file path    |
| `--size`   | `-s`  | see `codeviz help-metrics`                                              | Metric for rectangle area |

### Optional Flags

| Flag               | Short | Default          | Description                                                   |
| ------------------ | ----- | ---------------- | ------------------------------------------------------------- |
| `--fill`           | `-f`  | same as `--size` | Metric for fill colour                                        |
| `--fill-palette`   |       | metric default   | Palette for fill colour                                       |
| `--border`         | `-b`  | none             | Metric for border colour                                      |
| `--border-palette` |       | metric default   | Palette for border colour                                     |
| `--width`          |       | `1920`           | Image width in pixels                                         |
| `--height`         |       | `1080`           | Image height in pixels                                        |
| `--filter`         |       | none             | Filter rule: glob to include, `!glob` to exclude (repeatable) |

## `radial-tree`

Generate a radial tree visualization with the repository root at the centre.

### Synopsis

```
codeviz radial-tree [flags] <target-path>
```

### Required Flags

| Flag          | Short | Values                                                                  | Description            |
| ------------- | ----- | ----------------------------------------------------------------------- | ---------------------- |
| `--output`    | `-o`  | `.png`, `.jpg`, `.jpeg`, `.svg`                                         | Output image file path |
| `--disc-size` | `-d`  | see `codeviz help-metrics`                                              | Metric for disc size   |

### Optional Flags

| Flag               | Short | Default        | Description                                                   |
| ------------------ | ----- | -------------- | ------------------------------------------------------------- |
| `--fill`           | `-f`  | none           | Metric for fill colour                                        |
| `--fill-palette`   |       | metric default | Palette for fill colour                                       |
| `--border`         | `-b`  | none           | Metric for border colour                                      |
| `--border-palette` |       | metric default | Palette for border colour                                     |
| `--labels`         |       | none           | Labels to display: `all`, `folders`, or `none`                |
| `--width`          |       | `1920`         | Image width in pixels                                         |
| `--height`         |       | `1920`         | Image height in pixels                                        |
| `--filter`         |       | none           | Filter rule: glob to include, `!glob` to exclude (repeatable) |

## `bubble-tree`

Generate a bubble-tree visualization where each file is a circle sized by a metric.

### Synopsis

```
codeviz bubble-tree [flags] <target-path>
```

### Required Flags

| Flag       | Short | Values                                                                  | Description            |
| ---------- | ----- | ----------------------------------------------------------------------- | ---------------------- |
| `--output` | `-o`  | `.png`, `.jpg`, `.jpeg`, `.svg`                                         | Output image file path |
| `--size`   | `-s`  | see `codeviz help-metrics`                                              | Metric for circle size |

### Optional Flags

| Flag               | Short | Default        | Description                                                   |
| ------------------ | ----- | -------------- | ------------------------------------------------------------- |
| `--fill`           | `-f`  | none           | Metric for fill colour                                        |
| `--fill-palette`   |       | metric default | Palette for fill colour                                       |
| `--border`         | `-b`  | none           | Metric for border colour                                      |
| `--border-palette` |       | metric default | Palette for border colour                                     |
| `--labels`         |       | none           | Labels to display: `all`, `folders`, or `none`                |
| `--width`          |       | `1920`         | Image width in pixels                                         |
| `--height`         |       | `1080`         | Image height in pixels                                        |
| `--filter`         |       | none           | Filter rule: glob to include, `!glob` to exclude (repeatable) |

## `spiral`

Generate a spiral visualization showing git commit history over time.
Each lap of the spiral represents one time period (day or hour); each file is a disc sized by an optional metric.
Requires the target directory to be inside a git repository.

### Synopsis

```
codeviz spiral [flags] <target-path>
```

### Required Flags

| Flag       | Short | Values                          | Description            |
| ---------- | ----- | ------------------------------- | ---------------------- |
| `--output` | `-o`  | `.png`, `.jpg`, `.jpeg`, `.svg` | Output image file path |

### Optional Flags

| Flag                  | Short | Default        | Description                                                   |
| --------------------- | ----- | -------------- | ------------------------------------------------------------- |
| `--size`              | `-s`  | none           | Metric for disc size; see `codeviz help-metrics`              |
| `--fill`              | `-f`  | none           | Metric for fill colour                                        |
| `--fill-palette`      |       | metric default | Palette for fill colour                                       |
| `--border`            | `-b`  | none           | Metric for border colour                                      |
| `--border-palette`    |       | metric default | Palette for border colour                                     |
| `--resolution`        | `-r`  | `daily`        | Time resolution: `daily` or `hourly`                          |
| `--labels`            |       | `laps`         | Labels to display: `all`, `laps`, or `none`                   |
| `--width`             |       | `1920`         | Image width in pixels                                         |
| `--height`            |       | `1920`         | Image height in pixels                                        |
| `--filter`            |       | none           | Filter rule: glob to include, `!glob` to exclude (repeatable) |

## Shared Concepts

### Metric values

| Metric                 | Valid for `--size`/`--disc-size` | Valid for `--fill`/`--border` | Description                                         |
| ---------------------- | :------------------------------: | :---------------------------: | --------------------------------------------------- |
| `file-size`            |                ✓                 |               ✓               | File size in bytes                                  |
| `file-lines`           |                ✓                 |               ✓               | Number of non-binary lines                          |
| `file-age`             |            ✓ *(git)*             |           ✓ *(git)*           | Time since first commit (days)                      |
| `file-freshness`       |            ✓ *(git)*             |           ✓ *(git)*           | Time since last commit (days)                       |
| `author-count`         |            ✓ *(git)*             |           ✓ *(git)*           | Number of distinct commit authors                   |
| `commit-count`         |            ✓ *(git)*             |           ✓ *(git)*           | Total number of commits touching the file           |
| `total-lines-added`    |            ✓ *(git)*             |           ✓ *(git)*           | Accumulated lines added across all commits          |
| `total-lines-removed`  |            ✓ *(git)*             |           ✓ *(git)*           | Accumulated lines removed across all commits        |
| `commit-density`       |            ✓ *(git)*             |           ✓ *(git)*           | Commits per month of file lifetime                  |
| `file-type`            |                —                 |               ✓               | File extension category                             |

Metrics marked *(git)* require the target directory to be inside a git repository.

### Palette values

`categorization`, `temperature`, `good-bad`, `neutral`, `foliage`, `terrain`

See [Palettes](../palettes) for detailed descriptions and colour samples.

### Filter rules

The `--filter` flag accepts glob patterns. Prefix with `!` to exclude matching files.
Multiple `--filter` flags are evaluated in order, like a `.gitignore`.

```sh
# Include only Go files
codeviz tree-map ./src -o out.png -s file-size --filter '*.go' --filter '!*'

# Exclude generated Go files
codeviz tree-map ./src -o out.png -s file-size --filter '!*_gen.go' --filter '!*_gen_test.go'
```

### Selection metrics (user-defined classification)

You can define your own **classification metrics** in the config file.
Each metric assigns a category string to files by matching their relative path against an ordered list of glob rules — the first match wins.

```yaml
# codeviz.yaml
selectionMetrics:
  code-purpose:
    - category: test
      filename: "*_test.go"
    - category: source
      filename: "*"
  code-source:
    - category: generated
      filename: "*_gen.go"
    - category: authored
      filename: "*"
```

Use the metric name (e.g. `code-purpose`) anywhere a classification metric is accepted:

```sh
# Colour tree-map cells by whether each file is a test or source file
codeviz --config codeviz.yaml tree-map ./src -o out.png -s file-size -f code-purpose
```

Selection metrics use the `categorization` palette by default, showing each distinct category in a unique colour.
Files that match no rule receive no colour (rendered in the palette's neutral/fallback colour).

## Examples

### Tree-map by file size

```sh
codeviz tree-map ./src -o treemap.png -s file-size
```

### Tree-map coloured by file type

```sh
codeviz tree-map ./src -o treemap.png -s file-size -f file-type
```

### Tree-map with git freshness and temperature palette

```sh
codeviz tree-map ./src -o treemap.png -s file-lines -f file-freshness --fill-palette temperature
```

### Tree-map with border showing author count

```sh
codeviz tree-map ./src -o treemap.png -s file-lines -f file-freshness -b author-count
```

### Radial tree by file size

```sh
codeviz radial-tree ./src -o radial.png -d file-size
```

### Radial tree with folder labels

```sh
codeviz radial-tree ./src -o radial.png -d file-lines -f file-type --labels folders
```

### Bubble tree by file lines

```sh
codeviz bubble-tree ./src -o bubbles.png -s file-lines
```

### Bubble tree with all labels and SVG output

```sh
codeviz bubble-tree ./src -o bubbles.svg -s file-size -f file-type --labels all
```

### 4K tree-map with verbose logging

```sh
codeviz -v tree-map ./src -o treemap.png -s file-size --width 3840 --height 2160
```

### Export effective configuration

```sh
codeviz --export-config config.yaml tree-map ./src -o treemap.png -s file-size
```

### Export computed metrics to JSON

Writes a JSON file containing the full file tree and all computed metric values.
Useful for downstream analysis or building custom visualizations.

```sh
codeviz --export-data metrics.json tree-map ./src -o treemap.png -s file-size -f file-type
```

### Export computed metrics to YAML

```sh
codeviz --export-data metrics.yaml tree-map ./src -o treemap.png -s file-lines
```

## Exit Codes

| Code | Meaning                                              |
| ---- | ---------------------------------------------------- |
| 0    | Success — image written to output path               |
| 1    | Invalid arguments or validation failure              |
| 2    | Target path does not exist or is not a directory     |
| 3    | Git-required metric used on non-git directory        |
| 4    | Output path error (parent missing, permission)       |
| 5    | Internal error during scan/render                    |
| 6    | No files available after filtering (e.g. all binary) |
