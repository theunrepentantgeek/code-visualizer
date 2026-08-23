# Centered Treemap Directory Captions

## Problem

Aspect-aware directory rails currently position captions from a start anchor.
Top captions therefore appear left-aligned, while rotated left-rail captions
appear bottom-aligned. This loses the centered presentation used for directory
headings.

## Design

Every rendered directory caption will be centered within its rail in both
dimensions.

- Top-rail captions use the rail midpoint with no rotation.
- Left-rail captions use the same rail midpoint with a -90 degree rotation.
- Both caption specifications use the canvas middle text anchor.

The caption position is:

```text
x = rail.X + rail.W / 2
y = rail.Y + rail.H / 2
```

Using the canvas anchor keeps centering behavior consistent between raster and
SVG backends without duplicating text measurement or backend alignment logic.

## Unchanged Behavior

This change does not alter:

- rail orientation or geometry;
- label measurement, truncation, or omission thresholds;
- directory depth colors;
- directory borders;
- font size or text rotation direction;
- root-label suppression.

## Testing and Artifacts

Capture-backend tests will assert that top and left captions both use
`AnchorMiddle` at the exact rail midpoint, with their existing rotations.
Treemap PNG and SVG golden snapshots and the tree-map sample artifacts will be
regenerated to reflect only the caption-position change.

Pre-existing sample modifications outside the tree-map visualization remain
out of scope and must not be staged or restored.
