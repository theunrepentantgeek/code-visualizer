# Abstractions — `internal/treemap`

## TreemapRectangle

**Purpose.**

- `TreemapRectangle` is the laid-out tree the whole visualization works from: one positioned node per directory or file, with its bounds, label, chrome, and children ([node.go#L23](node.go#L23)).
- Producing it is the entire job of layout; rendering and labelling are pure walks over it ([layout.go#L26](layout.go#L26), [render.go#L206](render.go#L206), [labels.go#L108](labels.go#L108)).

**Boundary and invariants.**

- `layoutSize` preserves the third-party layout box's dimensions so rendering stays bit-for-bit stable after the position-plus-size to `Rect` conversion; use `size()` rather than re-deriving from `Bounds` ([node.go#L25](node.go#L25), [node.go#L39](node.go#L39)).
- `VisibleDepth` counts *visible* directory nesting, with the synthetic root at -1, and is meaningless for files, which always keep zero ([node.go#L28](node.go#L28), [render.go#L99](render.go#L99)).
- Consumers must tolerate the negative root depth: depth-indexed lookups clamp rather than wrap ([render.go#L103](render.go#L103)).

**Related operations.**

- `Layout` builds the tree from a `model.Directory`; the render and label stages consume it ([layout.go#L26](layout.go#L26), [../treemap/stages.go#L76](../treemap/stages.go#L76)).

**Proper-use patterns.**

- Take sizes through `rect.size()` wherever layout arithmetic matters, so golden output does not shift ([render.go#L232](render.go#L232), [labels.go#L108](labels.go#L108)).

**Anti-patterns.**

- Do not compute a node's width as `Bounds.Max - Bounds.Min` for layout purposes; that is not guaranteed to reproduce the original box width exactly ([directory_chrome.go#L19](directory_chrome.go#L19)).

**Source locations.**

- [node.go#L23](node.go#L23) — `TreemapRectangle` and `size()`.
- [layout.go#L26](layout.go#L26) — `Layout`.

## DirectoryChrome

**Purpose.**

- `DirectoryChrome` is a directory's decoration budget: which label orientation was chosen, the label text actually used, the rail the label sits in, and the content area left for children ([node.go#L14](node.go#L14), [node.go#L15](node.go#L15)).

**Boundary and invariants.**

- Orientation is decided by box shape — wide boxes get a top rail, tall boxes a left rail — and falls back to border-only when the label or the remaining content would be too small ([directory_chrome.go#L28](directory_chrome.go#L28), [directory_chrome.go#L52](directory_chrome.go#L52)); `DirectoryLabelNone` means "no rail and no label" ([node.go#L9](node.go#L9), [render.go#L206](render.go#L206)).
- `Text` is the *fitted* label, possibly truncated with an ellipsis, so renderers never re-truncate ([directory_chrome.go#L15](directory_chrome.go#L15), [directory_chrome.go#L92](directory_chrome.go#L92)).
- Chrome geometry is computed in third-party box terms and converted to `Rect` only once, at the point it is stored ([directory_chrome.go#L19](directory_chrome.go#L19)).

**Related operations.**

- `resolveDirectoryChrome` chooses it during layout; the render stage draws the rail and label from it ([directory_chrome.go#L28](directory_chrome.go#L28), [render.go#L207](render.go#L207)).

**Proper-use patterns.**

- Children occupy the content area, never the full node bounds; layout recomputes that area from the original box rather than reading `Chrome.Content` back, to keep Squarify's arithmetic identical ([node.go#L19](node.go#L19), [layout.go#L62](layout.go#L62), [layout.go#L65](layout.go#L65)).

**Anti-patterns.**

- Do not decide at render time whether a directory label fits; that decision is already baked into the chrome ([directory_chrome.go#L56](directory_chrome.go#L56)).

**Source locations.**

- [node.go#L15](node.go#L15) — `DirectoryChrome`, `DirectoryLabelOrientation`.
- [directory_chrome.go#L28](directory_chrome.go#L28) — chrome resolution.
