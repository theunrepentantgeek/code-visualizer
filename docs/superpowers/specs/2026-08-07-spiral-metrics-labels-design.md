# Spiral Metrics Labels Design

## Purpose

Make spiral visualizations self-explanatory by placing a compact date and the
configured metric values inside every activity dot. Replace the existing
timeline-axis labels and add a matching circular key to the legend.

## User Experience

Every active spiral dot displays upright, centered text in this order:

```text
day D
month MMM
<size metric value>
<fill metric value>
<border metric value>
<other configured metric values>
```

Metric values appear without metric names. The existing legend continues to
name the configured metrics and explain their encodings.

Each metric contributes at most one value, even when it is assigned to more
than one role. Values retain the configured-role order: size, fill, border,
then other active roles such as surface. Values that cannot be obtained from a
bucket are omitted, while both date lines remain.

The in-dot label always appears for every active dot. The spiral `--labels`
option and the existing labels positioned along the 12-o'clock axis are
removed.

Dots have a larger minimum radius so the fixed date-and-metrics text block is
legible. A bounded range above that minimum remains available for the size
metric, so dot area continues to communicate relative value.

The legend includes an annotated circular key that uses the same stacked date
and value positions as a dot. It demonstrates date, size, fill, border, and
other active metric values while the normal legend entries retain metric
names.

## Architecture

Introduce a spiral-specific label model derived from a positioned `TimeBucket`
and its configured metric roles. It owns:

- formatting the two date lines;
- collecting distinct metric values in role order;
- formatting values with the existing treemap-compatible metric formatter; and
- providing both per-dot text and the legend-key sample.

The renderer adds a shared upright, centered text specification on the overlay
layer for the dot labels. It no longer renders the rotated external labels.
The spiral legend renderer consumes the same label model for its circular key,
ensuring it stays aligned with the active metric encodings.

## Data Flow

1. Resolve the spiral's size, fill, border, and other active metric roles.
2. For each active time bucket, construct its label model, skipping repeated
   metric names and values unavailable from the bucket.
3. Determine a radius floor from the fixed label block; map the optional size
   metric into a bounded range from that floor to the non-overlapping maximum.
4. Render each disc and its centered label.
5. Render normal legend entries plus the circular key generated from the same
   label model.

## Error Handling

No new user-facing configuration is introduced. Missing or unavailable metric
values follow existing label behavior: omit that value rather than failing the
render. A dot always retains its two date lines. The normal no-size-metric path
uses the larger fixed minimum radius.

## Testing

- Unit-test date formatting, role ordering, metric-name deduplication, and
  omission of unavailable values.
- Unit-test the increased minimum radius and preserved metric-driven range.
- Verify rendering contains centered in-dot labels and no external timeline
  labels.
- Verify the circular legend key mirrors the active dot label structure.
- Update focused raster and SVG golden snapshots for representative
  configurations, including repeated metric roles and a surface metric.

## Non-goals

- Adding a new label configuration option.
- Preserving the legacy spiral `--labels` behavior.
- Changing metric names or encoding entries in the existing legend.
