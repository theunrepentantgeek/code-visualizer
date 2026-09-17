# Abstractions — `internal/bubbletree`

## BubbleNode

**Purpose.**

- `BubbleNode` is the packed circle tree the visualization renders from: one disc per directory or file, carrying its circle, model path, label decision, and children ([node.go#L17](node.go#L17), [node.go#L19](node.go#L19)).

**Boundary and invariants.**

- Layout builds each node in its *parent's* local frame; absolute pixel coordinates only exist after `scaleToFit` has run over the whole tree ([layout.go#L45](layout.go#L45), [transforms.go#L60](transforms.go#L60)).
- A labelled directory's radius includes `LabelReservation` on top of its disc, so the drawn disc radius is the stored radius minus that reservation ([layout.go#L19](layout.go#L19), [layout.go#L69](layout.go#L69), [render.go#L204](render.go#L204)).
- `Path` is the stable identifier used to join nodes back to the model for colour mapping — not `Label`, which is only a display name ([node.go#L21](node.go#L21), [node.go#L30](node.go#L30)).

**Related operations.**

- `Layout` produces the tree; `Index` flattens it into path-keyed directory and file maps for rendering ([layout.go#L25](layout.go#L25), [node.go#L30](node.go#L30)).

**Proper-use patterns.**

- Look nodes up by path through `Index` rather than re-walking the tree in each render step ([render.go#L38](render.go#L38)).
- Ask `bubbleDirDiscRadius` for a directory's drawn radius so the label reservation is removed consistently ([render.go#L204](render.go#L204)).

**Anti-patterns.**

- Do not treat `Geometry.Center` as pixel coordinates before scaling, nor a labelled directory's `Geometry.Radius` as its visible disc ([transforms.go#L60](transforms.go#L60), [render.go#L205](render.go#L205)).

**Source locations.**

- [node.go#L19](node.go#L19) — `BubbleNode`, `Index`.
- [transforms.go#L65](transforms.go#L65) — local-to-absolute scaling.
