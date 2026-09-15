# Abstractions — `internal/canvas`

## Canvas

**Purpose.**

- `Canvas` is a retained-then-render drawing surface: shapes are recorded with a layer assignment and only rasterised (or serialised) later, in one batch, against a chosen backend ([canvas.go#L1](canvas.go#L1), [canvas.go#L53](canvas.go#L53)).
- Retaining shapes is what lets later pipeline stages add labels, legends, title, and footer to a drawing that earlier stages produced ([../treemap/pipeline.go#L37](../treemap/pipeline.go#L37)).

**Boundary and invariants.**

- Draw order is layer first, then insertion order within a layer — a stable sort, so equal-layer shapes keep the order they were added ([canvas.go#L264](canvas.go#L264), [canvas.go#L47](canvas.go#L47)).
- Title and footer are drawn last, after every recorded shape, and are always horizontally centred ([canvas.go#L277](canvas.go#L277), [canvas.go#L282](canvas.go#L282)).
- The content area is narrower than the canvas: `SetDrawingBounds` records the vertical band left after the title and footer reservations, and `DrawingSize` reports it ([canvas.go#L74](canvas.go#L74), [canvas.go#L104](canvas.go#L104), [../stages/canvas.go#L119](../stages/canvas.go#L119)).

**Related operations.**

- `NewCanvas`, the `Add*` recorders, `Render` (picks a backend from the output extension) and `RenderTo` (explicit backend) ([canvas.go#L66](canvas.go#L66), [canvas.go#L112](canvas.go#L112), [canvas.go#L240](canvas.go#L240), [canvas.go#L264](canvas.go#L264)).
- `FooterReservedHeight` and `TitleReservedHeight` are the layout contract that layout stages subtract from the available height ([canvas.go#L21](canvas.go#L21), [canvas.go#L30](canvas.go#L30)).

**Proper-use patterns.**

- Reserve title/footer space and set the drawing bounds as pipeline stages before laying out content, so a viz never draws underneath the chrome ([../stages/canvas.go#L119](../stages/canvas.go#L119), [../stages/canvas.go#L131](../stages/canvas.go#L131)).
- Lay out against `DrawingSize()`/`DrawingMinY()` rather than the raw canvas height ([canvas.go#L86](canvas.go#L86), [canvas.go#L104](canvas.go#L104)).

**Anti-patterns.**

- Do not rely on call order alone for z-ordering across concerns; assign the appropriate `Layer` so overlays cannot be hidden by later content ([canvas.go#L266](canvas.go#L266), [layer.go#L3](layer.go#L3)).
- Do not render to a format-specific backend directly from a visualization; `Render(outputPath)` chooses the backend from the output format ([canvas.go#L240](canvas.go#L240), [../stages/canvas.go#L23](../stages/canvas.go#L23)).

**Source locations.**

- [canvas.go#L53](canvas.go#L53) — `Canvas`, drawing bounds, `RenderTo`.
- [canvas.go#L21](canvas.go#L21) — the reserved-height constants.

## Layer

**Purpose.**

- `Layer` is the z-ordering vocabulary for canvas content: background, surface, structure, content, overlay ([layer.go#L3](layer.go#L3), [layer.go#L8](layer.go#L8)).

**Boundary and invariants.**

- Lower values draw first; the 10-unit gaps between constants exist so intermediate layers can be added without renumbering ([layer.go#L4](layer.go#L4), [layer.go#L5](layer.go#L5)).
- Each constant names a role rather than a specific visualization's shape, so different visualizations compose consistently ([layer.go#L9](layer.go#L9), [layer.go#L17](layer.go#L17)).

**Related operations.**

- Every `Canvas.Add*` method takes a layer; `RenderTo` sorts by it ([canvas.go#L112](canvas.go#L112), [canvas.go#L266](canvas.go#L266)).

**Proper-use patterns.**

- Put backgrounds on `LayerBackground`, plot furniture on `LayerStructure`, data shapes on `LayerContent`, and labels or legends on `LayerOverlay` ([../scatter/render.go#L41](../scatter/render.go#L41), [../scatter/render.go#L63](../scatter/render.go#L63)).

**Anti-patterns.**

- Do not invent numeric layer values at call sites; use the named constants so the gaps stay meaningful ([layer.go#L5](layer.go#L5)).

**Source locations.**

- [layer.go#L6](layer.go#L6) — `Layer` and its constants.

## ShapeStyle

**Purpose.**

- `ShapeStyle`, embedded by `RectangleSpec`, `DiscSpec`, and `PolygonSpec`, is the reusable visual template for a class of filled shapes: which inks paint fill and border, and how wide the border is ([spec.go#L7](spec.go#L7), [spec.go#L14](spec.go#L14)).
- Splitting the template from the instance means many shapes share one style while each carries only its own geometry and metric values ([rectangle.go#L9](rectangle.go#L9), [rectangle.go#L10](rectangle.go#L10)).

**Boundary and invariants.**

- Specs hold `inks.Ink`, not colours: the colour for a given shape is resolved at draw time from the instance's `inks.MetricValue` ([spec.go#L9](spec.go#L9), [rectangle.go#L18](rectangle.go#L18)).
- Instances reference their spec by pointer and are expected to share it across a whole family of shapes ([rectangle.go#L11](rectangle.go#L11), [../bubbletree/render.go#L48](../bubbletree/render.go#L48)).
- Rectangles may supply a per-instance gradient focus, while discs and polygons consistently resolve gradients from their centre ([rectangle.go#L15](rectangle.go#L15), [disc.go#L18](disc.go#L18), [polygon.go#L18](polygon.go#L18)).

**Related operations.**

- `Canvas.AddRectangle`, `AddDisc`, and `AddPolygon` record instances; each instance's `drawTo` resolves the shared style's inks and hands the resulting fills to the backend ([canvas.go#L112](canvas.go#L112), [rectangle.go#L18](rectangle.go#L18)).

**Proper-use patterns.**

- Build one spec per visual role and reuse it for every shape in that role ([../bubbletree/render.go#L48](../bubbletree/render.go#L48), [../bubbletree/render.go#L56](../bubbletree/render.go#L56)).
- Use a fixed ink for chrome such as backgrounds, and a metric-driven ink for data shapes ([../bubbletree/render.go#L50](../bubbletree/render.go#L50)).

**Anti-patterns.**

- Do not resolve a metric to a colour before recording a shape; pass the `MetricValue` and let the ink decide at draw time ([rectangle.go#L13](rectangle.go#L13), [rectangle.go#L18](rectangle.go#L18)).

**Source locations.**

- [spec.go#L7](spec.go#L7) — `ShapeStyle` and the filled-shape specs that embed it.
- [rectangle.go#L10](rectangle.go#L10) — an instance carrying geometry, metric values, and focus.

## BlockLabel

**Purpose.**

- `BlockLabel` is a centred multi-line label constrained to a rectangular area, sized automatically to the largest font that fits ([block_label.go#L17](block_label.go#L17), [block_label.go#L87](block_label.go#L87)).

**Boundary and invariants.**

- Fitting is analytic rather than a search: glyph metrics scale with point size, so one measurement at a reference size yields the tight-fitting size ([block_label.go#L84](block_label.go#L84), [block_label.go#L87](block_label.go#L87)).
- Below a line-height threshold the label degrades rather than rendering unreadable text — greeked bars first, then omitted entirely — but only for raster formats and only when the caller has not set `PreserveText` ([block_label.go#L42](block_label.go#L42), [block_label.go#L13](block_label.go#L13)).
- `LayoutSize` exists because bounds derived from position-plus-size arithmetic cannot always be reversed exactly; when set and valid it wins over `Bounds.Size()` ([block_label.go#L20](block_label.go#L20), [block_label.go#L56](block_label.go#L56)).

**Related operations.**

- `Canvas.AddBlockLabel` performs the fit and records either text or greeked bars; empty lines are dropped first ([block_label.go#L29](block_label.go#L29), [block_label.go#L64](block_label.go#L64)).

**Proper-use patterns.**

- Build labels during a labelling stage and hand them to the canvas with the output format, so SVG keeps real text while raster output greeks tiny labels ([../treemap/state.go#L24](../treemap/state.go#L24), [block_label.go#L42](block_label.go#L42)).

**Anti-patterns.**

- Do not pre-truncate or pre-scale label text to make it fit; the fitting and degradation rules live here ([block_label.go#L37](block_label.go#L37)).

**Source locations.**

- [block_label.go#L18](block_label.go#L18) — `BlockLabel`, `AddBlockLabel`, and the fitting rules.
