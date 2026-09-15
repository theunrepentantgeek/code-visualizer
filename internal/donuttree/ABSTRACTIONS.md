# Abstractions — `internal/donuttree`

## DonutNode

**Purpose.**

- `DonutNode` is one directory's annular sector: the directory it stands for, its ring depth, and the angular and radial extent it occupies ([node.go#L8](node.go#L8), [node.go#L9](node.go#L9)).
- It is the single geometric vocabulary shared by sector rendering, border insetting, and label placement ([render.go#L126](render.go#L126), [render.go#L145](render.go#L145), [labels.go#L91](labels.go#L91)).

**Boundary and invariants.**

- Extent is stored as a start angle plus a sweep, never as a start/end pair; the end is derived through `EndAngle` ([node.go#L12](node.go#L12), [node.go#L20](node.go#L20)).
- `Depth` indexes the ring: children start one ring further out, with the root's own ring occupied by the central anchor instead ([layout.go#L14](layout.go#L14), [layout.go#L53](layout.go#L53), [layout.go#L65](layout.go#L65)).
- Sibling sectors tile their parent's sweep exactly — each child begins where the previous one ended, and every child receives a minimum sweep even at zero metric value ([layout.go#L52](layout.go#L52), [layout.go#L83](layout.go#L83)).

**Related operations.**

- `Layout` builds the sector tree inside a `LayoutResult`, which also carries the centre point and the root anchor disc that no `DonutNode` represents ([layout.go#L16](layout.go#L16), [node.go#L24](node.go#L24), [render.go#L55](render.go#L55)).

**Proper-use patterns.**

- Convert a sector to pixels by polar arithmetic against the layout's centre rather than caching absolute points on the node ([render.go#L131](render.go#L131), [labels.go#L98](labels.go#L98)).

**Anti-patterns.**

- Do not shrink a sector for borders by editing its radii or angles in place; insetting is computed as a separate point set so the node stays the authoritative extent ([render.go#L145](render.go#L145), [render.go#L147](render.go#L147)).

**Source locations.**

- [node.go#L9](node.go#L9) — `DonutNode`, `EndAngle`, `LayoutResult`.
- [layout.go#L16](layout.go#L16) — sector allocation.
