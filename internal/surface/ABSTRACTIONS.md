# Abstractions — `internal/surface`

## Sample

**Purpose.**

- `Sample` is a scalar field observation: a position plus the value there, used both for the data points fed in and for every interpolated mesh vertex ([types.go#L16](types.go#L16), [interpolation.go#L19](interpolation.go#L19), [poisson.go#L25](poisson.go#L25)).

**Boundary and invariants.**

- `Original` distinguishes a supplied observation from a generated infill point ([types.go#L20](types.go#L20), [mesh.go#L47](mesh.go#L47)).
- Support is tracked privately: interpolation marks a sample unsupported when it falls outside the observations' influence, and mesh building drops any triangle touching one ([types.go#L19](types.go#L19), [interpolation.go#L163](interpolation.go#L163), [mesh.go#L207](mesh.go#L207)).
- Non-finite positions or values are rejected rather than propagated ([mesh.go#L50](mesh.go#L50), [poisson.go#L204](poisson.go#L204)).

**Related operations.**

- `Interpolate` assigns a value at an arbitrary position; `PoissonSamples` generates evenly spaced ones; `BoundaryLoops` returns them along a region's edge ([interpolation.go#L19](interpolation.go#L19), [poisson.go#L25](poisson.go#L25), [types.go#L90](types.go#L90)).

**Proper-use patterns.**

- Callers supply only position and value; everything else is filled in by this package ([../spiral/surface.go#L17](../spiral/surface.go#L17)).

**Anti-patterns.**

- Do not reconstruct support or originality outside the package — the flags travel with the sample through interpolation and meshing ([interpolation.go#L160](interpolation.go#L160), [mesh.go#L209](mesh.go#L209)).

**Source locations.**

- [types.go#L16](types.go#L16) — `Sample`.
- [interpolation.go#L160](interpolation.go#L160) — how value and support are assigned.

## Region

**Purpose.**

- `Region` is the area a surface is built over, expressed as just two questions: what is its bounding rectangle, and does it contain a given point ([types.go#L23](types.go#L23)).
- Everything that generates geometry — infill sampling, mesh building, triangle culling — is written against that interface ([poisson.go#L25](poisson.go#L25), [mesh.go#L19](mesh.go#L19), [mesh.go#L270](mesh.go#L270)).

**Boundary and invariants.**

- `Bounds` must be finite and non-empty, and a typed-nil implementation counts as invalid; both are checked before any work starts ([poisson.go#L111](poisson.go#L111), [poisson.go#L119](poisson.go#L119), [poisson.go#L137](poisson.go#L137)).
- `Annulus` is the shape this package ships, with `Contains` rejecting inverted or negative radii ([types.go#L28](types.go#L28), [types.go#L42](types.go#L42)).
- Boundary loops are not part of the interface: `BoundaryLoops` type-switches over the shapes it knows, so a new region type needs a case there ([types.go#L90](types.go#L90), [types.go#L96](types.go#L96)).

**Related operations.**

- `Build` restricts a Delaunay mesh to the region; annulus containment additionally drives triangle-level culling ([mesh.go#L19](mesh.go#L19), [mesh.go#L340](mesh.go#L340)).

**Proper-use patterns.**

- Describe the target area as a region and let the package sample it, rather than pre-computing point sets ([../spiral/surface.go#L25](../spiral/surface.go#L25), [../spiral/surface.go#L28](../spiral/surface.go#L28)).

**Anti-patterns.**

- Do not rely on `Contains` alone to keep a mesh inside a curved region; edges also have to avoid the inner radius, which is why annulus culling is chord-aware ([mesh.go#L378](mesh.go#L378), [mesh.go#L420](mesh.go#L420)).

**Source locations.**

- [types.go#L23](types.go#L23) — `Region` and `Annulus`.
- [mesh.go#L270](mesh.go#L270) — region-based triangle culling.

## Triangle

**Purpose.**

- `Triangle` is the output unit of the mesh: three samples and a single representative value for the facet ([types.go#L179](types.go#L179), [mesh.go#L19](mesh.go#L19)).

**Boundary and invariants.**

- Degenerate triangles and those touching an unsupported sample are never returned ([mesh.go#L207](mesh.go#L207), [mesh.go#L231](mesh.go#L231)).
- Edges are kept short — refinement targets the longest edge against `MaxTriangleEdge` — so flat-shaded facets stay small enough to read as a smooth field ([types.go#L11](types.go#L11), [mesh.go#L151](mesh.go#L151), [mesh.go#L197](mesh.go#L197)).

**Related operations.**

- `Build` produces triangles; `LongestEdge` measures one; `SubdivideTriangle` splits one into banded polygons ([mesh.go#L19](mesh.go#L19), [mesh.go#L244](mesh.go#L244), [subdivide.go#L9](subdivide.go#L9)).

**Proper-use patterns.**

- Render triangles directly for a flat surface, and only subdivide when discrete colour bands are wanted ([../spiral/render.go#L88](../spiral/render.go#L88), [../spiral/render.go#L112](../spiral/render.go#L112)).

**Anti-patterns.**

- Do not interpolate colour across a triangle's vertices; the facet carries one `Value`, and banding is done by subdivision instead ([types.go#L181](types.go#L181), [subdivide.go#L9](subdivide.go#L9)).

**Source locations.**

- [types.go#L179](types.go#L179) — `Triangle`.
- [mesh.go#L19](mesh.go#L19) — construction and culling rules.

## Polygon

**Purpose.**

- `Polygon` is a value-banded fragment of a triangle: an ordered ring of samples plus the single value that names its band ([types.go#L184](types.go#L184), [subdivide.go#L9](subdivide.go#L9)).

**Boundary and invariants.**

- Subdividing with no breakpoints yields exactly one polygon covering the whole triangle, so callers never need a separate "unbanded" path ([subdivide.go#L14](subdivide.go#L14), [subdivide.go#L21](subdivide.go#L21)).
- Invalid triangles or breakpoints yield no polygons at all, rather than partial output ([subdivide.go#L10](subdivide.go#L10), [subdivide.go#L77](subdivide.go#L77)).
- A band's `Value` is a representative value inside that band, not the parent triangle's value ([subdivide.go#L45](subdivide.go#L45), [subdivide.go#L258](subdivide.go#L258)).

**Related operations.**

- `SubdivideTriangle` is the only producer; consumers convert the ring to canvas points ([subdivide.go#L9](subdivide.go#L9), [../spiral/render.go#L152](../spiral/render.go#L152)).

**Proper-use patterns.**

- Group the resulting polygons by resolved colour before drawing, so each band becomes one fill path ([../spiral/render.go#L112](../spiral/render.go#L112), [../spiral/render.go#L130](../spiral/render.go#L130)).

**Anti-patterns.**

- Do not assume a fixed fragment count per triangle; a triangle may be clipped away entirely or split into several bands ([subdivide.go#L28](subdivide.go#L28), [subdivide.go#L45](subdivide.go#L45)).

**Source locations.**

- [types.go#L184](types.go#L184) — `Polygon`.
- [subdivide.go#L9](subdivide.go#L9) — band subdivision.
