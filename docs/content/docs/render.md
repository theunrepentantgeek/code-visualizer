---
title: render
weight: 7
---

The `render` command renders a named preset — a predefined combination of
visualisation, metrics, and palette that produces a useful image without
requiring you to know which metrics and palettes to pair. It is the fastest way
to get a meaningful picture of a repository.

## Synopsis

```text
codeviz render                               # list the available presets
codeviz render <preset> <target> -o <output> # render a preset
```

Run `codeviz render` with no arguments to print the presets and their
descriptions.

## Arguments and flags

| Flag or argument | Short | Default | Description                                                     |
| ---------------- | ----- | ------- | -------------------------------------------------------------- |
| `<preset>`       |       | none    | Name of the preset to render; omit to list the presets         |
| `<target>`       |       | none    | Path to the directory to scan; required when a preset is given |
| `--output`       | `-o`  | none    | Output image file path; required when a preset is given        |
| `--title`        |       | preset  | Override the preset's default title                            |
| `--width`        |       | `1920`  | Image width in pixels                                          |
| `--height`       |       | `1080`  | Image height in pixels                                         |
| `--hide-footer`  |       | `false` | Suppress the attribution footer                               |

## Available presets

| Preset                   | Description                                                                              |
| ------------------------ | --------------------------------------------------------------------------------------- |
| `structure-tree-map`     | Tree-map sized by file lines, coloured by file type. A quick overview of code structure. |
| `structure-bubble-tree`  | Bubble tree sized by file lines, coloured by file type. An alternative structure view.   |
| `history-tree-map`       | Tree-map sized by file lines, coloured by commit count. Highlights frequently-changed hotspots. |
| `age-tree-map`           | Tree-map sized by file lines, coloured by file age. Reveals stale and actively-maintained areas. |
| `contributors-tree-map`  | Tree-map sized by file lines, coloured by distinct author count. Useful for bus-factor analysis. |

The `history-tree-map`, `age-tree-map`, and `contributors-tree-map` presets read
git metadata, so their target directory must be inside a git repository.

## Examples

List the available presets:

```sh
codeviz render
```

Render the structure overview of a repository:

```sh
codeviz render structure-tree-map ./src -o structure.png
```

Render the commit hotspots with a custom title:

```sh
codeviz render history-tree-map ./src -o hotspots.png --title "Commit Hotspots"
```
