# Surface Local Interpolation Design

## Purpose

Replace global inverse-distance weighting in `internal/surface` with a smooth,
compact spatial kernel. The current interpolation includes every observation,
so a large population of individually weak distant observations can pull much
of the surface toward the global mean.

The replacement must suppress distant influence without creating sharp local
peaks, cutoff rings, or visible steps. It remains a generic surface feature;
the interpolation model contains no spiral or timeline semantics.

## Scope

This change affects interpolation and unsupported mesh geometry in
`internal/surface`. Spiral remains the first consumer and requires no new
configuration or visualization-specific interpolation logic.

User-facing controls, timeline-aware distance, per-visualization tuning, and
adding surfaces to other visualizations are out of scope.

## Interpolation Model

### Compact smootherstep kernel

Replace inverse-square weighting with a reversed smootherstep kernel. For an
observation at spatial distance `d` and a surface-wide support radius `R`:

```text
t = d / R

w(d) = 1 - (6t^5 - 15t^4 + 10t^3)    when 0 <= t < 1
w(d) = 0                                when t >= 1
```

The interpolated value is the normalized weighted mean of observations with
positive weight:

```text
value = sum(observation.value * w(distance)) / sum(w(distance))
```

The kernel has zero slope at both endpoints. It decreases slowly near an
observation, falls most quickly around the middle of its support, and reaches
zero without a crease at `R`. It replaces IDW rather than multiplying IDW, so
there is only one radial falloff.

An interpolation request at an original observation's coordinates returns
that observation's exact value. This preserves measurements independently of
the kernel and floating-point normalization.

### Global adaptive support radius

Derive one fixed radius for each surface build from the spatial distribution
of its finite observations:

1. For each observation, find the shortest positive distance to another
   observation. Coincident observations are skipped when finding that
   positive distance.
2. Discard observations for which no positive nearest-neighbor distance
   exists.
3. Sort the resulting distances in ascending order.
4. Select the nearest-rank 90th percentile at one-based rank
   `ceil(0.90 * count)`.
5. Set `R` to twice that percentile distance.

The high percentile accommodates most spacing variation without allowing a
single isolated observation to determine the radius. The fixed multiplier of
two gives neighboring kernels substantial overlap while remaining an
internal, consistent surface characteristic.

If there are no positive nearest-neighbor distances, or if the derived radius
is zero or non-finite, the interpolation model is invalid and the build
returns no surface.

## Components And Data Flow

### Interpolation model

Add a private interpolation model in `internal/surface`. It owns the filtered
observations and derived support radius. `Build` creates it once and reuses it
for all boundary, Poisson infill, and mesh-refinement points, avoiding repeated
nearest-neighbor and percentile calculations.

The model's internal interpolation operation returns `(value, supported)`.
It is supported when the point exactly matches an observation or at least one
observation has positive kernel weight. It is unsupported when no observation
lies within `R`.

Keep `surface.Interpolate(point, originals) float64` as the simple public
entry point used by callers and tests. It constructs the same model and
returns zero when the model or point is unsupported, preserving the existing
single-value API. Mesh generation uses the internal operation directly so it
does not confuse an unsupported point with a legitimate zero value.

### Mesh support state

Surface points carry explicit support state during mesh construction.
Original observations are supported and retain their exact values. Generated
boundary, infill, and refinement points receive both value and support state
from the interpolation model.

Sampling, Delaunay triangulation, and refinement continue across the complete
supplied region. This preserves deterministic mesh geometry and existing edge
length constraints. Final triangle selection omits every triangle containing
an unsupported vertex. The result may therefore contain holes where the
region has no local data support; the surface does not invent fallback values
for those areas.

No changes are required to palette-band subdivision or rendering. They only
receive supported triangles.

## Edge Cases And Errors

- Fewer than three finite observations still produce no mesh.
- Observations with non-finite coordinates or values are excluded before
  radius calculation and interpolation.
- Coincident observations do not add zero distances to the percentile input.
- If multiple observations share an exact location, exact interpolation uses
  the first finite observation in input order, preserving deterministic
  existing behavior.
- An isolated observation may have no overlapping support and therefore only
  preserve its exact original vertex; surrounding unsupported triangles are
  omitted.
- Failure to derive a valid radius, triangulate, or satisfy refinement limits
  returns no surface through the existing failure path. Callers retain their
  current fallback behavior, such as rendering a spiral without its surface.

## Testing

### Kernel and radius tests

- Verify kernel weights at normalized distances `0`, `0.5`, and `1` are `1`,
  `0.5`, and `0` respectively.
- Verify the first derivative is flat at both endpoints through values close
  to `0` and `1`.
- Verify the support radius is twice the nearest-rank 90th percentile of
  positive nearest-neighbor distances.
- Verify coincident and non-finite observations do not distort radius
  selection.

### Interpolation tests

- Preserve exact values at original coordinates.
- Verify a known weighted average between observations.
- Verify observations at or beyond `R` contribute no weight.
- Verify unsupported public interpolation returns zero while the internal
  operation reports `supported = false`.
- Verify deterministic handling of duplicate coordinates.

### Mesh and integration tests

- Verify generated vertices within support receive the smootherstep value.
- Verify triangles touching an unsupported vertex are omitted.
- Preserve existing region, maximum-edge, refinement, boundary, and
  determinism guarantees.
- Update spiral PNG and SVG Goldie snapshots and inspect them for improved
  locality without sharp peaks, visible cutoff rings, or unsupported fill.
- Run the complete surface and spiral test suites, then the repository CI
  checks.

## Success Criteria

- Distant observations outside the adaptive radius have exactly zero effect.
- Nearby observations blend with a bounded S-shaped falloff rather than an
  inverse-square peak.
- Unsupported parts of a region are absent rather than assigned a fallback
  value.
- The interpolation policy remains entirely within the generic surface
  package and exposes no new user configuration.
- Existing mesh-quality and deterministic-rendering guarantees continue to
  pass.