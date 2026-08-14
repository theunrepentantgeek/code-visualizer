# Radial-Tree Sample

Demonstrates the **radial-tree** visualization, which arranges files as discs
radiating outward from the repository root, one ring per folder depth.

![Radial-tree sample](code-visualizer.png)

## What it shows

| Visual property | Metric | Palette |
| --------------- | ------ | ------- |
| Disc size       | `file-lines` | — |
| Fill colour     | `file-type` | `categorization` |
| Border colour   | `file-freshness` | `good-bad` |
| Labels          | folders | — |

Disc area scales with file length, fill colour groups files by type, and the
border reflects how recently each file changed.

## Try it yourself

```sh
codeviz radial-tree . --config samples/radial-tree/code-visualizer.yml --output out.png
```

Key knobs in [`code-visualizer.yml`](code-visualizer.yml) to experiment with:

- `radial-tree.fileDiscSize` — the metric that drives file disc area.
- `radial-tree.fileFill` / `radial-tree.fileBorder` — swap in other file metrics and palettes.
- `radial-tree.directoryDiscSize`, `radial-tree.directoryFill`, and
  `radial-tree.directoryBorder` — override the metrics used for directory discs.
- `legend.position` / `legend.orientation` — where the legend is drawn.
