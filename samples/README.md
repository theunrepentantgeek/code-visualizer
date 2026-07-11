# Samples

Each subfolder contains a self-contained example of one code-visualizer
visualization: a `code-visualizer.yml` config, the rendered `code-visualizer.png`
and `code-visualizer.svg` outputs, and a `README.md` explaining what the sample
shows and which knobs to try.

| Sample | Visualization | Description |
| ------ | ------------- | ----------- |
| [tree-map](tree-map/) | Tree-map | Space-filling nested rectangles sized by file length. |
| [bubble-tree](bubble-tree/) | Bubble-tree | Circles packed within circles, one level per folder. |
| [radial-tree](radial-tree/) | Radial-tree | Discs radiating outward from the repository root. |
| [spiral](spiral/) | Spiral | Commit history laid out along a time spiral. |
| [scatter](scatter/) | Scatter | Files plotted as points on a pair of metric axes. |

## Regenerating the samples

All samples are rendered from the code-visualizer repository itself:

```sh
task samples
```

This rebuilds the binary and regenerates every `code-visualizer.png` /
`code-visualizer.svg` from the matching `code-visualizer.yml`.
