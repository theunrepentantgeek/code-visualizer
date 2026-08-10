# Adaptive Spiral Cadence Design

## Goal

Make the daily spiral cadence adapt to timeline length so dots have similar
spacing in the radial and arc directions near the middle of the spiral. A
daily lap must contain exactly 14, 28, 42, or 56 dots.

## Scope

The change affects only daily spiral layout. Hourly spirals keep their existing
24 dots per lap and one-hour bucket duration. No command-line or configuration
setting is introduced.

## Cadence Selection

After daily time buckets are created, select one candidate dots-per-lap value
from `14`, `28`, `42`, and `56`.

For each candidate:

1. Derive the candidate's Archimedean spiral parameters for the current bucket
   count and drawing bounds.
2. Measure the radial gap between two points at the same angle on adjacent
   turns.
3. Measure the arc gap between consecutive points at the spiral's midpoint
   radius.
4. Score the candidate by the absolute difference between those two gaps.

Select the candidate with the lowest score. A tie selects 28 dots per lap,
preserving the established four-week cadence when the geometry gives no
preference.

The selector returns an explicit dots-per-lap value. `Resolution` remains
responsible for time-bucket duration only; it must not be repurposed to encode
the selected daily cadence.

## Integration

Store the selected cadence in spiral pipeline state after bucket construction.
Pass it to layout and maximum-disc-radius calculations instead of obtaining
daily dots per lap from `Resolution`. The layout keeps its current canvas
centering, inner-to-outer radius ratio, clockwise orientation, and rendering
interfaces.

Daily bucket boundaries remain calendar-day boundaries. The adaptive cadence
only changes the visual number of daily buckets per revolution, not the buckets
or aggregated data themselves.

## Validation

Unit tests will confirm:

- Daily selection always returns one of 14, 28, 42, or 56.
- Each selected candidate minimizes the defined midpoint spacing score for
  representative short, medium, and long daily timelines.
- A score tie selects 28.
- Hourly cadence remains 24.
- Layout and maximum-disc-radius calculations use the supplied cadence, while
  existing geometry invariants remain true.

