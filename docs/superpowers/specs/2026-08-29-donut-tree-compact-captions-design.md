# Donut Tree Compact Captions

## Goal

Render donut-tree directory captions as compact multiline text blocks instead
of distributing their lines across the full ring thickness. Center each block
on a radius from the donut center, with tangential text baselines.

## Current Problem

Donut-tree captions place each line on a separate concentric arc. Their radii
divide the sector's entire radial span evenly, which makes ordinary line
spacing appear doubled or larger. Each line is also decomposed into individual
glyphs and rotated tangentially to its arc. This differs from radial-tree text,
and its line spacing is much larger than normal multiline text. The first
compact-caption revision corrected the spacing but oriented the baselines
radially, making the resulting labels look unnatural within donut sectors.

## Caption Geometry

Each visible sector retains its existing ordered caption lines: directory name,
size value, optional fill value, and optional border value.

The caption is rendered as one text element per line. The sector midpoint angle
and midpoint radius define the center of the complete multiline block. Lines
use the measured font line height as their center-to-center spacing and are
offset along the sector midpoint radius, leaving the complete block centered on
the sector midpoint. The radius is therefore the centerline through the stack.

Each text baseline is perpendicular to that centerline and tangent to the ring:

- On the upper half, rotation equals the sector midpoint angle plus pi over two.
- On the lower half, rotation adds another pi to keep the text upright.

Every line uses the middle text anchor. This keeps each line centered on the
caption block while allowing the shared rotation rule to preserve readability.

## Font Fitting

The font starts at the existing default size and is reduced only as needed.
Fitting accounts for both dimensions of the rotated multiline block:

- The widest line must fit within the tangential arc length at the sector
  midpoint radius.
- The complete stack of lines must fit within the sector's radial thickness.

If the resulting size is below the existing minimum, the caption is omitted.
No new configuration or fallback rendering mode is introduced.

## Implementation Scope

The change is confined to donut-tree caption layout and its tests. It replaces
per-glyph curved-line placement with one canvas text element per caption line.
The canvas API and radial-tree implementation remain unchanged.

## Testing

Focused donut-tree tests will verify:

- one draw call per caption line;
- measured normal line spacing and a block centered on the sector midpoint;
- radial alignment of the multiline block centerline;
- tangential baseline rotation on the upper half;
- pi-flipped tangential baseline rotation on the lower half;
- font fitting in both the radial and tangential dimensions;
- omission when the fitted size falls below the minimum.

The revised focused tests must fail against the current radial-baseline compact
implementation before production code changes. After implementation, the
donut-tree unit and golden tests will be run, followed by the repository's full
CI task. Only donut-tree golden output should change as a result of this fix.

## Non-Goals

- Adding a general multiline canvas primitive.
- Changing radial-tree rendering.
- Changing caption content or metric ordering.
- Updating unrelated visualization samples or golden files.