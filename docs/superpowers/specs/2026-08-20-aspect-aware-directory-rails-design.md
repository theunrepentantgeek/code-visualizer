# Aspect-Aware Treemap Directory Rails

**Date:** 2026-08-20  
**Status:** Approved

## Summary

Replace the tree-map's unconditional directory header bars with conditional,
aspect-aware label rails. Wide and square nested directories use a top rail;
tall nested directories use a left rail with rotated text. A rail is omitted
when it cannot show a complete name or four-rune truncated name, or when the
remaining content area would be less than 20 pixels in either dimension.

The root directory has no label rail. Every directory retains its thin
structural border and gutter, including directories whose label is omitted.

## Goals

- Reduce the low-signal blocks created by adjacent headers in deeply nested
  directories.
- Preserve directory names when the directory can display either the complete
  name or at least four runes plus an ellipsis.
- Keep labels in dedicated space rather than overlaying file tiles.
- Preserve legible directory boundaries when labels are omitted.
- Produce identical layout decisions for raster and SVG output.

## Non-Goals

- Interactive labels, hover states, or tooltips.
- A CLI or configuration option for restoring the old header layout.
- Changes to file sizing, squarification, colors, pincushion shading, file
  labels, legends, or title and footer layout.
- Collapsing single-child directory levels.

## Chosen Approach

### Directory chrome

Directory chrome is resolved during layout from the directory's final
rectangle:

1. The root receives no label rail.
2. A nested rectangle with `width >= height` is a top-rail candidate.
3. A nested rectangle with `height > width` is a left-rail candidate.
4. The chosen orientation does not fall back to the other orientation when it
   cannot fit.
5. A candidate rail is accepted only when:
   - the post-rail child-content rectangle is at least 20 pixels wide and
     20 pixels high; and
   - the rail can display either the complete name or a truncated name with at
     least four visible runes followed by an ellipsis.
6. A rejected candidate becomes a border-only directory and gives its full
   padded interior to child layout.

The rail remains 20 pixels thick and uses the existing dark directory-header
fill and white 12-point text. Top-rail text reads left to right. Left-rail text
is placed on the left edge and rotated -90 degrees, reading bottom to top.

All directories keep the existing structural border. Their child content also
keeps the existing 4-pixel inset on sides not occupied by a rail:

- border-only: 4 pixels on all sides;
- top rail: rail at the top, then 4-pixel left, right, and bottom insets;
- left rail: rail at the left, then 4-pixel top, right, and bottom insets.

Sibling gaps remain unchanged.

### Label fitting

The layout uses the existing canvas text-measurement package with the same
12-point font used by both rendering backends. It reserves 4 pixels of text
padding at each end of a rail.

If the complete name fits, it is retained. Otherwise, the fitter removes
Unicode runes from the end and appends `…` until the text fits. Truncation is
accepted only when at least four original runes remain. Short complete names
remain valid when they fit; the four-rune minimum applies only to truncated
names.

Names are never truncated by byte, so UTF-8 directory names remain valid.

## Architecture

### Layout helper

A focused treemap layout helper accepts a directory rectangle, its name, and a
root flag. It returns resolved directory-chrome metadata:

- orientation: none, top, or left;
- rail bounds;
- child-content bounds;
- fitted display text.

The helper owns orientation selection, text measurement, truncation, and
minimum-content checks. It is pure and can be tested independently.

### Layout rectangle metadata

`TreemapRectangle` carries the resolved directory-chrome metadata alongside
its existing geometry and children. Recursive directory layout follows this
sequence:

1. Establish the directory's rectangle from its parent's squarification.
2. Resolve directory chrome.
3. Squarify children inside the returned child-content bounds.
4. Recurse into child directories.

Because the parent rectangle is fixed before its chrome is resolved, showing
or hiding a label changes only that directory's internal child layout. It does
not move or resize sibling directories.

### Rendering

The render walk consumes the resolved metadata without measuring text or
recomputing geometry:

- draw the directory's structural border;
- draw a rail rectangle only for top or left orientation;
- draw the fitted label at 0 degrees for top rails or -90 degrees for left
  rails;
- emit no rail shape or label text for orientation none.

This keeps raster and SVG behavior aligned because both backends receive the
same canvas shapes and text rotation.

## Data Flow

```text
directory model
    |
    v
parent squarification -> directory rectangle
    |
    v
resolve directory chrome
    |-- orientation
    |-- fitted text
    |-- rail bounds
    `-- child-content bounds
    |
    v
child squarification -> recursive layout
    |
    v
render border + optional rail + optional label
```

## Edge Cases and Failure Behavior

- Empty directories still render their border and may render a rail when both
  the name and minimum post-rail interior fit.
- Tiny or degenerate rectangles render border-only and do not attempt text
  drawing.
- A directory name that cannot meet the truncation minimum is omitted with its
  rail, rather than rendering an ellipsis-only label.
- Square directories deterministically choose a top rail.
- The root retains its structural border and padding but never reserves label
  space.
- Existing non-positive child-content guards continue to stop recursive
  layout safely.

No new user-facing errors are introduced. Insufficient space degrades
deterministically to a border-only directory.

## Testing

### Unit tests

Test the pure directory-chrome helper for:

- root label suppression;
- top rail for wide and square rectangles;
- left rail for tall rectangles;
- full-name fitting;
- four-rune-plus-ellipsis truncation;
- omission when fewer than four runes can be retained;
- rune-safe truncation for non-ASCII names;
- omission when the post-rail content area is below either 20-pixel minimum;
- exact rail and child-content bounds for all three orientations.

### Layout tests

Verify that:

- children remain within the resolved child-content bounds;
- sibling rectangles do not overlap;
- a hidden rail returns its space to child layout;
- resolving one directory's rail does not alter its siblings' rectangles;
- root children begin within the root's padded, header-free interior.

### Render tests

Using the capture backend, verify that:

- top labels are emitted at 0-degree rotation;
- left labels are emitted at -90-degree rotation;
- omitted labels emit neither a header rectangle nor text;
- directory borders remain for unlabeled directories.

### Golden tests

Regenerate and review existing tree-map golden images for PNG and SVG. The
review should confirm that nested header blocks are reduced, tall directories
use left rails, file areas remain readable, and directory boundaries remain
clear.
