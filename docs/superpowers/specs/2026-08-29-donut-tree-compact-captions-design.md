# Donut Tree Compact Captions

## Goal

Render donut-tree directory captions as compact multiline text blocks instead
of distributing their lines across the full ring thickness. Align each line's
baseline using the same left/right orientation rules as radial-tree labels.

## Current Problem

Donut-tree captions place each line on a separate concentric arc. Their radii
divide the sector's entire radial span evenly, which makes ordinary line
spacing appear doubled or larger. Each line is also decomposed into individual
glyphs and rotated tangentially to its arc. This differs from radial-tree text,
whose baseline points radially and flips on the left half to remain readable.

## Caption Geometry

Each visible sector retains its existing ordered caption lines: directory name,
size value, optional fill value, and optional border value.

The caption is rendered as one text element per line. The sector midpoint angle
and midpoint radius define the center of the complete multiline block. Lines
use the measured font line height as their center-to-center spacing and are
offset perpendicular to their baselines, leaving the complete block centered
on the sector midpoint.

The baseline orientation follows the radial-tree rule:

- On the right half, rotation equals the sector midpoint angle.
- On the left half, rotation equals the sector midpoint angle plus pi.

Every line uses the middle text anchor. This keeps each line centered on the
caption block while allowing the shared rotation rule to preserve readability.

## Font Fitting

The font starts at the existing default size and is reduced only as needed.
Fitting accounts for both dimensions of the rotated multiline block:

- The widest line must fit within the sector's available radial thickness.
- The complete stack of lines must fit within the tangential arc length at the
  sector midpoint radius.

If the resulting size is below the existing minimum, the caption is omitted.
No new configuration or fallback rendering mode is introduced.

## Implementation Scope

The change is confined to donut-tree caption layout and its tests. It replaces
per-glyph curved-line placement with one canvas text element per caption line.
The canvas API and radial-tree implementation remain unchanged; donut tree
copies the radial tree's established orientation rule.

## Testing

Focused donut-tree tests will verify:

- one draw call per caption line;
- measured normal line spacing and a block centered on the sector midpoint;
- radial baseline rotation on the right half;
- pi-flipped radial baseline rotation on the left half;
- font fitting in both the radial and tangential dimensions;
- omission when the fitted size falls below the minimum.

The focused tests must fail against the current concentric-arc implementation
before production code changes. After implementation, the donut-tree unit and
golden tests will be run, followed by the repository's full CI task. Only
donut-tree golden output should change as a result of this fix.

## Non-Goals

- Adding a general multiline canvas primitive.
- Changing radial-tree rendering.
- Changing caption content or metric ordering.
- Updating unrelated visualization samples or golden files.