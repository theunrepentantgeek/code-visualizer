# Donut Sector Adjacent Borders

## Goal

Render metric-colored donut-sector borders as adjacent strokes so neighboring
segments meet at their shared boundary without either stroke overlapping and
obscuring the other.

## Current Problem

Every donut sector currently uses its fill polygon as its border path. A canvas
stroke is centered on that path, so neighboring sectors place their strokes on
the same geometric boundary. Because sectors render sequentially, the later
sector paints over half of the earlier sector's border. This creates broken,
discontinuous-looking borders when adjacent sectors use different colors.

## Border Geometry

Sector fills retain their existing annular geometry and rendering order.

When a border metric is configured, each sector receives a separate border
polygon inset by half the configured stroke width:

- the outer arc radius decreases by half the stroke width;
- the inner arc radius increases by half the stroke width;
- the start radial edge moves inward from the start angle by the angular offset
  whose chord at the sector's midpoint radius equals half the stroke width;
- the end radial edge moves inward by the same amount.

The stroke remains centered on this inset path. Consequently, each stroke's
outer edge reaches the original sector boundary, while its inner edge remains
inside its own sector. Neighboring strokes are adjacent at their shared logical
boundary and no longer overlap.

The inset is clamped for narrow sectors so geometry remains valid. Fill-only
rendering remains unchanged.

## Implementation Scope

The donut renderer will draw the existing fill polygon without a border, then
draw an inset polygon with no fill and the configured metric border. This
requires canvas polygons to support a transparent fill; no backend-specific
stroke-alignment behavior will be introduced.

Root-anchor rendering, layout allocation, labels, non-donut visualizations, and
the public CLI configuration remain unchanged.

## Testing

Focused tests will verify that:

- fill polygons preserve the original annular sector geometry;
- adjacent border centerlines are separated by exactly one stroke width at their
  shared radial boundary;
- inset inner and outer border arcs reach the original ring boundaries at their
  stroke edges;
- border-free rendering retains its existing draw calls and geometry;
- narrow sectors produce finite, non-inverted border geometry.

The geometry regression test must fail against the current shared-centerline
implementation before production code changes. After implementation, donut
unit tests and golden tests will run, followed by the repository CI task. Only
donut golden output should change.

## Non-Goals

- Selecting one segment's metric color for a shared border.
- Changing canvas stroke alignment globally.
- Changing donut allocation, labels, or fill geometry.
- Updating unrelated visualization samples or golden files.
