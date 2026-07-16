---
title: bubble-tree
weight: 4
---

The `bubble-tree` visualisation draws every file as a circle sized by a metric,
with directories packed as enclosing bubbles. It offers a softer, more organic
alternative to the tree-map while conveying the same nested structure.

## Synopsis

```text
codeviz bubble-tree [flags] <target-path>
```

## Required flags

| Flag       | Short | Values                          | Description                    |
| ---------- | ----- | ------------------------------- | ------------------------------ |
| `--output` | `-o`  | `.png`, `.jpg`, `.jpeg`, `.svg` | Output image file path         |
| `--size`   | `-s`  | see `codeviz help metrics`      | Numeric metric for circle size |

## Optional flags

| Flag                   | Short | Default        | Description                                                        |
| ---------------------- | ----- | -------------- | ----------------------------------------------------------------- |
| `--fill`               | `-f`  | none           | Fill colour: `metric[,palette]` (e.g. `file-type,categorization`) |
| `--border`             | `-b`  | none           | Border colour: `metric[,palette]` (e.g. `file-lines,foliage`)     |
| `--labels`             |       | `none`         | Labels to display: `all`, `folders`, or `none`                    |
| `--legend`             |       | `bottom-right` | Legend position, or `none` to hide it                             |
| `--legend-orientation` |       | auto           | Legend orientation: `vertical` or `horizontal`                    |
| `--width`              |       | `1920`         | Image width in pixels                                             |
| `--height`             |       | `1080`         | Image height in pixels                                            |
| `--title`              |       | none           | Override the title text on the generated image                    |
| `--footer`             |       | none           | Override the footer text on the generated image                   |
| `--hide-footer`        |       | `false`        | Suppress the attribution footer                                   |
| `--flat`               |       | `false`        | Disable radial gradient shading and use flat solid fills          |
| `--include`            |       | none           | Include matching files; simple glob (repeatable)                  |
| `--exclude`            |       | none           | Exclude matching files; simple glob (repeatable)                  |
| `--include-binary-files` |     | `false`        | Include binary files, which are excluded by default               |

See [Shared concepts]({{< relref "/docs/shared-concepts" >}}) for the list of metric names,
palettes, and the include and exclude filter rules.

## Examples

Size circles by line count:

```sh
codeviz bubble-tree ./src -o bubbles.png -s file-lines
```

Colour by file type, show every label, and render to SVG:

```sh
codeviz bubble-tree ./src -o bubbles.svg -s file-size -f file-type --labels all
```
