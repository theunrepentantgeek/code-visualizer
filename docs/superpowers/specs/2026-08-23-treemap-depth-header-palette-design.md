# Treemap Depth Header Palette

**Date:** 2026-08-23  
**Status:** Approved

## Summary

Color tree-map directory rails by visible nesting depth so adjacent parent and
child headers remain visually distinct. Use a private five-color cool-slate
ramp, ordered darkest to lightest, and wrap to the darkest color after five
visible levels.

The root remains unlabeled and does not consume a palette slot. Its immediate
child directories use the darkest color.

## Goals

- Distinguish adjacent directory rails that currently merge into one dark
  block.
- Make rail color a deterministic visual encoding of visible nesting depth.
- Preserve white-label readability at every depth.
- Keep directory chrome subordinate to metric-driven file colors.
- Preserve the allocation-conscious treemap render walk.

## Non-Goals

- A user-selectable metric palette.
- A CLI or configuration option for directory rail colors.
- Sibling-specific or adjacency-based color variation.
- Changes to rail geometry, label fitting, orientation, borders, or file
  rendering.
- Coloring border-only directories that have no visible rail.

## Palette

The private treemap rail palette contains five opaque colors:

| Visible depth | Color | White contrast |
|---:|---|---:|
| 0 | `#202631` | 15.18:1 |
| 1 | `#2F3B4D` | 11.33:1 |
| 2 | `#3D5268` | 8.06:1 |
| 3 | `#516A7D` | 5.66:1 |
| 4 | `#5F7888` | 4.64:1 |

All colors exceed the WCAG AA 4.5:1 contrast threshold for normal white text.
The ramp uses one hue family so depth is visible without competing with file
metrics.

## Depth Semantics

Depth is counted only for directories that could have visible chrome:

- The hidden root has a sentinel depth and consumes no palette slot.
- Every immediate child directory of the root has visible depth 0.
- Each nested directory increments its parent's visible depth by one.
- Files do not participate in directory depth.
- A rail color is selected with:

```text
paletteIndex = visibleDepth % 5
```

Visible depth 5 therefore wraps to the darkest color. Every directory at the
same visible depth uses exactly the same color, regardless of sibling position
or whether its rail is top-oriented or left-oriented.

## Architecture

### Layout metadata

`TreemapRectangle` gains directory-depth metadata named `VisibleDepth`.

The root receives `-1` as a sentinel. Recursive layout assigns immediate child
directories depth 0, then passes `parentDepth + 1` to nested directories. File
rectangles leave the field at its zero value and never consume it because
rendering consults depth only for directory rails.

Depth is structural metadata and belongs in layout rather than being inferred
from render recursion. This keeps the layout tree self-describing and makes
depth behavior independently testable.

### Private palette

The five colors live in `internal/treemap`, near the other structural treemap
colors. They are not added to `internal/palette`, because that registry is for
user-selectable metric palettes.

The existing single `headerFill` value is removed.

### Render specs

At the start of a render pass, build one immutable `canvas.RectangleSpec` per
rail palette color. Each spec uses its color for both fill and zero-width
border, matching the current rail style.

When a directory has visible chrome, rendering selects:

```text
railSpec = railSpecs[VisibleDepth % len(railSpecs)]
```

The selected spec is used for both top and left rails. Border-only directories
and the root emit no rail, so their depth value is never indexed. Directory
border and white text specs remain unchanged.

This avoids allocating a new rectangle spec for every directory and keeps PNG
and SVG behavior identical because both backends receive the same canvas
shapes.

## Data Flow

```text
root layout
    |
    | VisibleDepth = -1
    v
immediate child directory layout
    |
    | VisibleDepth = 0
    v
nested directory layout
    |
    | VisibleDepth = parent + 1
    v
render visible rail
    |
    `-- railSpecs[VisibleDepth % 5]
```

## Error and Fallback Behavior

No user-facing error path is added:

- The palette is a non-empty compile-time structure.
- Depth is generated internally and cannot be negative for a visible nested
  rail.
- The root and border-only directories do not index the palette.
- Existing tiny-rectangle behavior still omits the rail and its color.

## Compatibility

This replaces the single directory-header color in the current feature branch.
There is no compatibility flag.

The following behavior remains unchanged:

- root label suppression;
- top-versus-left rail selection;
- rail dimensions and padding;
- label truncation and rotation;
- directory borders;
- file sizing, fill, border, focus, and labels;
- legend, title, and footer behavior.

## Testing

### Layout tests

Verify:

- root depth is `-1`;
- immediate child directories receive depth 0;
- nested directories increment depth by one;
- same-depth siblings receive the same value;
- file rectangles do not influence directory depth assignment;
- offsetting rectangles does not change depth.

### Render tests

Verify:

- visible depths 0 through 4 select the five exact rail colors;
- visible depth 5 wraps to the depth-0 color;
- top and left rails use the same depth-selection rule;
- same-depth siblings render with the same fill;
- border-only directories emit no colored rail;
- directory borders and white labels remain unchanged.

### Golden and sample output

Regenerate and review:

- `internal/goldentest/testdata/treemap-png.golden`;
- `internal/goldentest/testdata/treemap-svg.golden`;
- the generated tree-map PNG and SVG sample images.

The review should confirm that adjacent nested rails remain distinguishable,
white labels remain readable, and metric-driven file colors retain visual
priority. Existing unrelated sample-image modifications must not be reverted
or included in commits for this change.
