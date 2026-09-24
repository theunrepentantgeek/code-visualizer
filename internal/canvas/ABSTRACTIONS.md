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

## LineSpec

**Purpose.**

- `LineSpec` is the reusable visual template for straight lines and multi-point paths: an ink supplies the stroke colour and `StrokeWidth` supplies its thickness ([spec.go#L29](spec.go#L29), [line.go#L9](line.go#L9), [path.go#L9](path.go#L9)).

**Boundary and invariants.**

- A line or path instance carries geometry only; its shared spec resolves a fixed `MetricValue` when drawing, so line strokes represent visualization structure rather than per-instance metric data ([line.go#L15](line.go#L15), [path.go#L14](path.go#L14)).
- `Line` represents one segment, while `Path` preserves an ordered point sequence for a connected stroke ([line.go#L9](line.go#L9), [path.go#L9](path.go#L9)).

**Related operations.**

- `Canvas.AddLine` and `Canvas.AddPath` retain the geometry and dispatch it through the same backend stroke vocabulary ([canvas.go#L167](canvas.go#L167), [canvas.go#L177](canvas.go#L177), [model/backend.go#L19](model/backend.go#L19)).

**Proper-use patterns.**

- Construct one `LineSpec` for a visual role such as grid lines, edges, or tracks and reuse it for every segment or path in that role ([../scatter/render.go#L74](../scatter/render.go#L74), [../radialtree/render.go#L66](../radialtree/render.go#L66), [../spiral/render.go#L182](../spiral/render.go#L182)).

**Anti-patterns.**

- Do not create a new spec for every segment or pre-resolve its stroke colour; the shared spec is the package’s styling boundary ([spec.go#L30](spec.go#L30), [line.go#L16](line.go#L16)).

**Source locations.**

- [spec.go#L29](spec.go#L29) — `LineSpec`.
- [canvas_test.go#L144](canvas_test.go#L144) — line dispatch and resolved-stroke coverage.

## TextSpec

**Purpose.**

- `TextSpec` is the reusable visual template for standalone text, combining ink, point size, horizontal anchor, and rotation independently of a text instance’s content and position ([text_spec.go#L21](text_spec.go#L21), [text.go#L9](text.go#L9)).

**Boundary and invariants.**

- Font family is deliberately not configurable: raster text uses Go Regular and SVG text uses sans-serif, while the spec controls only cross-backend properties ([text_spec.go#L21](text_spec.go#L21), [raster/backend.go#L300](raster/backend.go#L300), [svg/backend.go#L209](svg/backend.go#L209)).
- Rotation is in radians and anchoring uses the shared `TextAnchor` vocabulary ([text_spec.go#L24](text_spec.go#L24), [model/backend.go#L26](model/backend.go#L26)).
- Like `LineSpec`, a `TextSpec` resolves a fixed `MetricValue`; metric-dependent text colour should be selected before constructing or choosing the spec ([text.go#L16](text.go#L16)).

**Related operations.**

- `Canvas.AddText` retains text instances, and `TextColourFor` selects readable black or white label ink from a resolved fill colour ([canvas.go#L157](canvas.go#L157), [text_colour.go#L14](text_colour.go#L14)).

**Proper-use patterns.**

- Reuse one spec for repeated labels with the same visual role ([../bubbletree/render.go#L182](../bubbletree/render.go#L182), [../alluvial/render.go#L191](../alluvial/render.go#L191)).
- Use `model.DefaultFontSize` rather than a magic zero when asking a backend for its default size ([model/backend.go#L37](model/backend.go#L37)).

**Anti-patterns.**

- Do not encode backend-specific font choices in visualization packages; that belongs behind `model.Backend` ([text_spec.go#L21](text_spec.go#L21)).

**Source locations.**

- [text_spec.go#L24](text_spec.go#L24) — `TextSpec` and the shared anchor aliases.
- [canvas_test.go#L119](canvas_test.go#L119) — text dispatch coverage.

## ArcTextSpec

**Purpose.**

- `ArcTextSpec` and `ArcText` describe text curved around a circle, separating reusable ink and font size from each arc’s centre, radius, and content ([text_spec.go#L31](text_spec.go#L31), [text_spec.go#L38](text_spec.go#L38)).

**Boundary and invariants.**

- `Radius` is the reference arc radius; both backends apply the shared `model.ArcTextInset`, and layout code that reserves space must use that same inset ([text_spec.go#L41](text_spec.go#L41), [model/backend.go#L41](model/backend.go#L41)).
- The backend owns glyph placement along the arc; callers provide semantic circle geometry rather than pre-positioned glyphs ([model/backend.go#L21](model/backend.go#L21), [raster/backend.go#L333](raster/backend.go#L333)).

**Related operations.**

- `Canvas.AddArcText` retains the instance; bubble-tree labels and arc-shaped legend samples consume it ([canvas.go#L187](canvas.go#L187), [../bubbletree/render.go#L235](../bubbletree/render.go#L235), [../legend/render.go#L462](../legend/render.go#L462)).

**Proper-use patterns.**

- Share one `ArcTextSpec` across labels that use the same visual treatment, while varying `ArcText.Position`, `Radius`, and `Text` ([../bubbletree/render.go#L235](../bubbletree/render.go#L235)).

**Anti-patterns.**

- Do not reproduce the backend’s inset or per-rune angular placement at a visualization call site ([model/backend.go#L41](model/backend.go#L41), [raster/backend.go#L371](raster/backend.go#L371)).

**Source locations.**

- [text_spec.go#L32](text_spec.go#L32) — `ArcTextSpec` and `ArcText`.
- [canvas_test.go#L235](canvas_test.go#L235) — arc-text dispatch coverage.

## FilledPath

**Purpose.**

- `FilledPath` is a borderless compound shape made from one or more closed point loops, used for continuous surfaces, annular regions, and alluvial bands that do not fit the rectangle/disc/polygon templates ([filled_path.go#L9](filled_path.go#L9), [../spiral/render.go#L144](../spiral/render.go#L144), [../alluvial/render.go#L36](../alluvial/render.go#L36)).

**Boundary and invariants.**

- All loops are filled together using the even-odd rule, allowing inner loops to cut holes; fill is already a resolved colour rather than an `inks.Ink` ([model/backend.go#L18](model/backend.go#L18), [raster/backend_test.go#L179](raster/backend_test.go#L179), [svg/backend_test.go#L276](svg/backend_test.go#L276)).
- `AddFilledPath` clones every loop before retaining it, so later caller mutation cannot change the recorded shape ([canvas.go#L139](canvas.go#L139), [canvas.go#L148](canvas.go#L148)).

**Related operations.**

- `Canvas.AddFilledPath` records the compound path and `Backend.DrawFilledPath` is the shared raster/SVG rendering boundary ([canvas.go#L140](canvas.go#L140), [model/backend.go#L18](model/backend.go#L18)).

**Proper-use patterns.**

- Supply explicit closed loops and resolve any metric-driven ink once for the whole filled region before recording it ([../spiral/render.go#L144](../spiral/render.go#L144), [../alluvial/render.go#L163](../alluvial/render.go#L163)).

**Anti-patterns.**

- Do not model holes by painting a second shape in the background colour; place both loops in one `FilledPath` so transparency and both backends preserve the intended region ([raster/backend_test.go#L179](raster/backend_test.go#L179), [svg/backend_test.go#L276](svg/backend_test.go#L276)).

**Source locations.**

- [filled_path.go#L10](filled_path.go#L10) — `FilledPath`.
- [canvas.go#L140](canvas.go#L140) — retained-loop cloning.

## ImageFormat

**Purpose.**

- `ImageFormat` is the supported output-format vocabulary shared by path validation, label degradation, and backend selection ([format.go#L10](format.go#L10), [../stages/paths.go#L38](../stages/paths.go#L38), [canvas.go#L292](canvas.go#L292)).

**Boundary and invariants.**

- PNG and JPG select the raster backend; SVG selects the vector backend ([canvas.go#L293](canvas.go#L293)).
- `FormatFromPath` is case-insensitive, treats `.jpg` and `.jpeg` identically, and rejects missing or unsupported extensions instead of choosing a default ([format.go#L22](format.go#L22), [format_test.go#L9](format_test.go#L9), [format_test.go#L44](format_test.go#L44)).

**Related operations.**

- `Canvas.Render` resolves the format from the output path and owns backend creation ([canvas.go#L238](canvas.go#L238), [canvas.go#L292](canvas.go#L292)).

**Proper-use patterns.**

- Resolve an output path with `FormatFromPath` when behavior genuinely differs by format, as block-label stages do for tiny raster labels ([../treemap/labels_stage.go#L17](../treemap/labels_stage.go#L17), [block_label.go#L42](block_label.go#L42)).

**Anti-patterns.**

- Do not parse extensions or instantiate raster/SVG backends in visualization packages; use `FormatFromPath` for format-aware behavior and `Canvas.Render` for output ([format.go#L24](format.go#L24), [canvas.go#L238](canvas.go#L238)).

**Source locations.**

- [format.go#L11](format.go#L11) — `ImageFormat`, supported values, and path resolution.

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

- Build labels during a labelling stage and hand them to the canvas with the output format, so SVG keeps real text while raster output greeks tiny labels ([../treemap/labels_stage.go#L10](../treemap/labels_stage.go#L10), [block_label.go#L42](block_label.go#L42)).

**Anti-patterns.**

- Do not pre-truncate or pre-scale label text to make it fit; the fitting and degradation rules live here ([block_label.go#L37](block_label.go#L37)).

**Source locations.**

- [block_label.go#L18](block_label.go#L18) — `BlockLabel`, `AddBlockLabel`, and the fitting rules.
