# Donut Ring Spacing Design

## Goal

Add visible spacing between adjacent depth rings in the donut-tree
visualization. Each ring uses 90% of its existing radial slot, leaving the
outer 10% transparent.

## Scope

Ring spacing is a fixed visual behavior. It does not add CLI flags,
configuration fields, validation, or runtime error paths.

The change is limited to donut-tree sector geometry. Angular allocation,
directory hierarchy, metrics, palettes, root-anchor geometry, maximum-layer
pruning, legends, and canvas dimensions remain unchanged.

## Layout Geometry

`Layout` continues dividing the available radius into equal slots based on the
maximum visible directory depth. For a sector at a given depth:

- its inner radius remains at the current slot's inner boundary;
- its outer radius is `inner radius + 90% of the slot width`;
- the remaining outer 10% of the slot is transparent.

The next depth starts at its existing slot boundary. This creates a consistent
gap between adjacent rings while keeping the first ring flush with the root
anchor. The outermost ring leaves the same proportional margin before the
canvas edge.

## Rendering and Labels

`DonutNode` remains the source of annular-sector geometry. Fill polygons,
metric-border inset polygons, and curved labels consume the narrowed inner and
outer radii without renderer-specific spacing logic.

As a result:

- fills and borders stop at the narrowed outer radius;
- the existing white canvas background shows through each gap;
- labels remain centered within the visible 90%-width band;
- PNG and SVG rendering follow the same geometry.

Existing narrow-sector border clamping remains responsible for keeping border
geometry finite and non-inverted.

## Edge Cases

Empty and root-only trees continue rendering without sectors. Because the gap
is proportional to each radial slot, every non-empty ring retains 90% of its
previous positive width, including deeply nested trees and trees limited by
`max-layers`.

## Testing and Documentation

Focused layout tests will verify:

- visible ring width equals 90% of the radial slot;
- ring inner radii remain on their existing slot boundaries;
- the gap between adjacent depths equals 10% of the slot width;
- the root anchor remains flush with the first ring.

Rendering tests will verify that sector polygons, border geometry, and label
placement use the narrowed radii. Donut-tree PNG and SVG golden snapshots will
capture the visual change. User-facing documentation needs no new option
description; generated donut-tree imagery may be refreshed where the existing
workflow requires it.
