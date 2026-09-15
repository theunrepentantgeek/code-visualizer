# Abstractions — `internal/geometry`

All types here are immutable value types: every operation returns a new value rather than mutating the receiver, and operations that cannot produce a meaningful value report failure with a second `bool` result or `NaN` instead of panicking.

## Point

**Purpose.**

- A `Point` is an absolute position in 2D space — a *location*, not a displacement ([point.go#L5](point.go#L5)).
- `OriginPoint` names `(0, 0)` as a value, because Go cannot declare struct constants ([point.go#L10](point.go#L10)).

**Boundary and invariants.**

- A `Point` is `Valid()` only when both coordinates are finite (no NaN, no ±Inf); distance operations propagate invalidity as `NaN` rather than a bogus number ([point.go#L19](point.go#L19), [point.go#L32](point.go#L32), [point.go#L40](point.go#L40)).
- Point arithmetic is typed: a point plus a `Vector` is a point, and the difference of two points is a `Vector` — points are never added to points ([point.go#L24](point.go#L24), [point.go#L28](point.go#L28)).

**Related operations.**

- `Translate`, `VectorTo`, `DistanceTo` / `DistanceSquaredTo`, and the free functions `Midpoint` and `Lerp` ([point.go#L24](point.go#L24), [point.go#L48](point.go#L48), [point.go#L52](point.go#L52)).

**Proper-use patterns.**

- Derive displacements with `VectorTo` rather than subtracting fields by hand ([../bubbletree/packing.go#L167](../bubbletree/packing.go#L167), [../surface/mesh.go#L236](../surface/mesh.go#L236)).
- Use `Midpoint`/`Lerp` for interpolation along a segment ([../surface/mesh.go#L169](../surface/mesh.go#L169), [../surface/subdivide.go#L226](../surface/subdivide.go#L226)).

**Anti-patterns.**

- Do not use `Point` to express an offset or direction; the codebase uses `Vector` for that (for example `RadialNode.Position` is a `Vector` from the canvas centre) ([../radialtree/node.go#L27](../radialtree/node.go#L27)).
- Do not compare distances without guarding validity: an invalid endpoint yields `NaN`, which silently fails every comparison ([point.go#L32](point.go#L32)).

**Source locations.**

- [point.go#L5](point.go#L5) — `Point` and its operations.

## Vector

**Purpose.**

- A `Vector` is a displacement (direction plus magnitude), the only value that may be added to a `Point` ([vector.go#L5](vector.go#L5), [point.go#L24](point.go#L24)).
- `ZeroVector` is the named additive identity ([vector.go#L10](vector.go#L10)).

**Boundary and invariants.**

- Validity means both components are finite ([vector.go#L25](vector.go#L25)).
- `Unit` returns `(ZeroVector, false)` for invalid or zero-length vectors, and pre-scales by the largest component so direction survives at the extremes of `float64` range instead of collapsing to zero or overflowing ([vector.go#L61](vector.go#L61), [vector.go#L49](vector.go#L49)).
- Polar construction is explicit: `NewRadialVector(angle, length)` converts angle/length to components ([vector.go#L21](vector.go#L21)).

**Related operations.**

- `Add`, `Subtract`, `Scale`, `Dot`, `Length` / `LengthSquared`, `Unit` ([vector.go#L30](vector.go#L30), [vector.go#L46](vector.go#L46)).

**Proper-use patterns.**

- Build radial layouts from an angle and radius with `NewRadialVector` instead of hand-rolled `cos`/`sin` ([../radialtree/layout.go#L170](../radialtree/layout.go#L170)).
- Negate a displacement with `Scale(-1)` when mirroring a position ([../bubbletree/packing.go#L186](../bubbletree/packing.go#L186)).
- Prefer `LengthSquared`/`DistanceSquaredTo` when only comparing magnitudes ([vector.go#L46](vector.go#L46), [point.go#L32](point.go#L32)).

**Anti-patterns.**

- Do not normalise by dividing by `Length()` yourself — that is exactly the overflow/underflow failure `Unit` exists to avoid ([vector.go#L49](vector.go#L49)).
- Do not ignore the `ok` result of `Unit`: a zero-length vector has no direction ([vector.go#L61](vector.go#L61)).

**Source locations.**

- [vector.go#L5](vector.go#L5) — `Vector` and its operations.

## Size

**Purpose.**

- A `Size` is a width/height extent with no position, used for canvas dimensions, reserved layout space, and measured content ([size.go#L5](size.go#L5), [../canvas/canvas.go#L99](../canvas/canvas.go#L99), [../legend/config.go#L70](../legend/config.go#L70)).

**Boundary and invariants.**

- Validity requires finite *and* non-negative dimensions — unlike `Point`/`Vector`, a negative extent is meaningless ([size.go#L15](size.go#L15)).
- `Empty()` is only meaningful for a valid size: it reports a zero width or height ([size.go#L21](size.go#L21)).
- `AspectRatio` returns `false` rather than dividing by zero for invalid or zero-height sizes ([size.go#L31](size.go#L31)).

**Related operations.**

- `Area`, `Scale`, `AspectRatio`; `RectFromPositionSize` places a size at a point ([size.go#L25](size.go#L25), [rect.go#L10](rect.go#L10)).

**Proper-use patterns.**

- Report required layout space as a `Size` and let the caller decide where to place it ([../legend/config.go#L70](../legend/config.go#L70), [../canvas/legendlayout/layout.go#L24](../canvas/legendlayout/layout.go#L24)).
- Scale a whole measured block uniformly with `Scale` ([../legend/render.go#L68](../legend/render.go#L68)).

**Anti-patterns.**

- Do not treat a `Size` as a position or rectangle; combine it with a `Point` via `RectFromPositionSize` ([rect.go#L10](rect.go#L10), [../legend/render.go#L498](../legend/render.go#L498)).

**Source locations.**

- [size.go#L5](size.go#L5) — `Size` and its operations.

## Rect

**Purpose.**

- A `Rect` is an axis-aligned rectangle expressed as `Min`/`Max` corners, and is the canonical bounding-box currency for layout and drawing ([rect.go#L5](rect.go#L5), [../canvas/model/backend.go#L15](../canvas/model/backend.go#L15)).

**Boundary and invariants.**

- Validity requires both corners valid *and* `Min <= Max` on both axes; a "backwards" rectangle is never silently normalised ([rect.go#L23](rect.go#L23)).
- Operations that can leave the valid domain (`Inset`, `ExpandToInclude`, `Union`) return `(Rect, bool)` and yield the zero `Rect` on failure ([rect.go#L47](rect.go#L47), [rect.go#L60](rect.go#L60), [rect.go#L71](rect.go#L71)).
- `Bounds()` returns the receiver, letting a `Rect` stand in wherever a boundable region is expected — the same interface `Circle` and `surface.Region` satisfy ([rect.go#L17](rect.go#L17), [circle.go#L48](circle.go#L48), [../surface/types.go#L23](../surface/types.go#L23)).

**Related operations.**

- `RectFromPositionSize`, `Width`/`Height`/`Size`/`Center`, `Contains`, `Translate`, `Inset`, `ExpandToInclude`, `Union` ([rect.go#L10](rect.go#L10), [rect.go#L32](rect.go#L32), [rect.go#L37](rect.go#L37)).

**Proper-use patterns.**

- Check the `ok` result before using an inset or unioned rectangle, and skip the work when it collapses ([../treemap/labels.go#L120](../treemap/labels.go#L120), [../bubbletree/transforms.go#L47](../bubbletree/transforms.go#L47)).
- Convert a position-plus-size layout result with `RectFromPositionSize` rather than computing `Max` inline ([rect.go#L10](rect.go#L10), [../legend/render.go#L498](../legend/render.go#L498)).

**Anti-patterns.**

- Do not assume `Width()`/`Height()` are non-negative without checking `Valid()`; they are plain subtractions ([rect.go#L32](rect.go#L32), [rect.go#L23](rect.go#L23)).
- Do not discard the `bool` from `Inset`/`Union`/`ExpandToInclude` — the zero `Rect` it pairs with is not a usable rectangle ([rect.go#L47](rect.go#L47)).

**Source locations.**

- [rect.go#L5](rect.go#L5) — `Rect` and its operations.

## Circle

**Purpose.**

- A `Circle` is a centre plus radius, and is the shape currency for disc-based visualizations and the canvas disc primitive ([circle.go#L5](circle.go#L5), [../canvas/model/backend.go#L16](../canvas/model/backend.go#L16), [../bubbletree/node.go#L19](../bubbletree/node.go#L19), [../scatter/layout.go#L22](../scatter/layout.go#L22)).

**Boundary and invariants.**

- Validity requires a valid centre and a finite, non-negative radius; every predicate returns `false` for invalid inputs rather than guessing ([circle.go#L14](circle.go#L14), [circle.go#L20](circle.go#L20), [circle.go#L28](circle.go#L28)).
- `Contains`, `Encloses` and `Intersects` are *exact*, inclusive comparisons with no tolerance ([circle.go#L20](circle.go#L20), [circle.go#L28](circle.go#L28), [circle.go#L38](circle.go#L38)).
- `Bounds()` returns the zero `Rect` for an invalid circle, so it also satisfies the boundable-region shape ([circle.go#L48](circle.go#L48)).

**Related operations.**

- `NewCircle`, `Contains`, `Encloses`, `Intersects`, `Bounds`, `Translate` ([circle.go#L10](circle.go#L10), [circle.go#L61](circle.go#L61)).

**Proper-use patterns.**

- Store laid-out disc geometry as a single `Circle` field rather than separate centre/radius fields ([../bubbletree/node.go#L19](../bubbletree/node.go#L19), [../spiral/node.go#L12](../spiral/node.go#L12)).
- When floating-point slack is needed, wrap `Encloses` in a package-local tolerant check instead of loosening the shared predicate ([../bubbletree/geometry.go#L57](../bubbletree/geometry.go#L57)).

**Anti-patterns.**

- Do not rely on `Encloses`/`Intersects` to absorb accumulated floating-point error — they compare exactly, which is why circle packing keeps its own tolerance helper ([circle.go#L28](circle.go#L28), [../bubbletree/geometry.go#L61](../bubbletree/geometry.go#L61)).

**Source locations.**

- [circle.go#L5](circle.go#L5) — `Circle` and its operations.
