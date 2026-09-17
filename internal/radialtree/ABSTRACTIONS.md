# Abstractions — `internal/radialtree`

## RadialNode

**Purpose.**

- `RadialNode` is a placed disc in the radial tree: its offset from the canvas centre, its disc radius, its sector angle, and its children ([node.go#L25](node.go#L25), [node.go#L27](node.go#L27)).

**Boundary and invariants.**

- `Position` is a vector *from the canvas centre*, not an absolute pixel point; renderers translate a centre point by it ([node.go#L26](node.go#L26), [render.go#L73](render.go#L73), [render.go#L108](render.go#L108)).
- `Angle` is the midpoint of the node's angular sector, kept alongside `Position` because labels are oriented by angle rather than by position ([layout.go#L164](layout.go#L164), [render.go#L274](render.go#L274)).
- Children are appended files-first, then subdirectories, and consumers rely on that order to re-pair nodes with model files and directories ([layout.go#L201](layout.go#L201), [render.go#L121](render.go#L121)).

**Related operations.**

- `Layout` places the whole tree from a `model.Directory`; the render pass walks it for edges, discs, and labels ([layout.go#L30](layout.go#L30), [render.go#L37](render.go#L37)).

**Proper-use patterns.**

- Derive pixel positions with `center.Translate(node.Position)` at render time, so the same layout works for any canvas centre ([render.go#L34](render.go#L34), [render.go#L145](render.go#L145)).
- Detect the root by its zero-length position rather than tracking depth separately ([render.go#L285](render.go#L285)).

**Anti-patterns.**

- Do not walk node children against model children without honouring the directory/file flag; the pairing is by kind and order, not by index ([render.go#L122](render.go#L122), [render.go#L125](render.go#L125)).

**Source locations.**

- [node.go#L27](node.go#L27) — `RadialNode`.
- [layout.go#L157](layout.go#L157) — placement, including child ordering.
