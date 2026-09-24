# Abstractions — `internal/palette`

## PaletteName

**Purpose.**

- `PaletteName` is the user-facing identity of a colour palette, used in config, on the command line, and in metric descriptors ([palette.go#L15](palette.go#L15), [palette.go#L16](palette.go#L16)).

**Boundary and invariants.**

- The set is closed: the named constants are the whole vocabulary and `IsValid` checks membership ([palette.go#L18](palette.go#L18), [palette.go#L42](palette.go#L42)).
- An unknown name is not an error at lookup time — `GetPalette` returns the zero palette — so validation must happen earlier ([palette.go#L271](palette.go#L271), [../config/metric_spec.go#L108](../config/metric_spec.go#L108)).
- Each base metric names a `DefaultPalette`, so a metric always has a sensible colouring when the user does not choose one ([../provider/base_descriptor.go#L33](../provider/base_descriptor.go#L33), [../stages/metrics.go#L36](../stages/metrics.go#L36)).

**Related operations.**

- `IsValid` validates input, `GetPalette` resolves it, and `Names`/`Infos` enumerate the registry in deterministic name order for tools and help output ([palette.go#L42](palette.go#L42), [palette.go#L239](palette.go#L239), [palette.go#L257](palette.go#L257), [palette.go#L271](palette.go#L271)).

**Proper-use patterns.**

- Validate the name where the user supplies it — as part of `config.MetricSpec.Validate` — rather than at the point of drawing ([../config/metric_spec.go#L107](../config/metric_spec.go#L107)).

**Anti-patterns.**

- Do not pass a raw string palette name into rendering code and hope the lookup succeeds; an unknown name silently yields a palette with no colours ([palette.go#L271](palette.go#L271)).

**Source locations.**

- [palette.go#L16](palette.go#L16) — `PaletteName`, its constants, and `IsValid`.
- [palette.go#L239](palette.go#L239) — deterministic registry discovery via `Names` and `Infos`.
- [../../cmd/codeviz/help_palettes_cmd.go#L16](../../cmd/codeviz/help_palettes_cmd.go#L16) — palette metadata consumed by user-facing help.

## ColourPalette

**Purpose.**

- `ColourPalette` is the runtime representation of a palette: its name, a human-readable description, its ordered list of colours, and whether that order is meaningful ([palette.go#L42](palette.go#L42), [palette.go#L43](palette.go#L43)).

**Boundary and invariants.**

- `Ordered` distinguishes sequential palettes suitable for numeric gradients from unordered ones intended for classifications ([palette.go#L54](palette.go#L54), [palette.go#L80](palette.go#L80)).
- The palette's length is the available step count: numeric inks bucket their values into exactly `len(Colours)` steps ([../inks/ink.go#L81](../inks/ink.go#L81)).
- `Description` is user-facing help text, so palettes are self-documenting in CLI output ([palette.go#L51](palette.go#L51), [palette.go#L257](palette.go#L257)).

**Related operations.**

- `GetPalette` resolves a name; `MapNumericToColour` scales a bucket index onto the palette range, returning the middle colour for a single bucket and opaque black for an empty palette ([palette.go#L271](palette.go#L271), [mapper.go#L10](mapper.go#L10), [mapper.go#L15](mapper.go#L15)).
- `RelativeLuminance` and `ContrastRatio` provide the shared WCAG 2.0 interpretation used when choosing readable foregrounds and testing palette contrast ([palette.go#L275](palette.go#L275), [palette.go#L284](palette.go#L284), [../canvas/text_colour.go#L14](../canvas/text_colour.go#L14)).

**Proper-use patterns.**

- Size numeric bucketing from the palette rather than from a fixed constant, so every palette uses its full range ([../inks/ink.go#L81](../inks/ink.go#L81)).
- Map bucket index to colour through `MapNumericToColour` in both drawing and legend construction ([../inks/ink.go#L122](../inks/ink.go#L122), [../inks/legend_data.go#L38](../inks/legend_data.go#L38)).

**Anti-patterns.**

- Do not index `Colours` directly with a bucket index; the bucket range and the palette range are different sizes ([mapper.go#L21](mapper.go#L21)).

**Source locations.**

- [palette.go#L43](palette.go#L43) — `ColourPalette` and the built-in palettes.
- [mapper.go#L10](mapper.go#L10) — `MapNumericToColour`.
- [palette.go#L275](palette.go#L275) — luminance and contrast operations.
- [../../docs/content/docs/palettes/index.md](../../docs/content/docs/palettes/index.md) — user-facing ordered/unordered palette guidance.

## CategoricalMapper

**Purpose.**

- `CategoricalMapper` assigns each distinct category value a stable palette colour, which is what makes categorical colouring consistent between shapes and the legend ([mapper.go#L30](mapper.go#L30), [mapper.go#L35](mapper.go#L35)).

**Boundary and invariants.**

- Assignment follows the order of the values passed in, so callers control colour assignment by controlling that order ([mapper.go#L54](mapper.go#L54), [../inks/ink.go#L94](../inks/ink.go#L94)).
- More values than colours is allowed but degrades explicitly: colours wrap around and a warning is logged ([mapper.go#L36](mapper.go#L36), [mapper.go#L48](mapper.go#L48)).
- Unknown values and empty palettes both fall back to mid grey rather than failing ([mapper.go#L38](mapper.go#L38), [mapper.go#L68](mapper.go#L68)).

**Related operations.**

- `NewCategoricalMapper` builds one from the distinct values and a palette; categorical inks own one ([mapper.go#L37](mapper.go#L37), [../inks/ink.go#L88](../inks/ink.go#L88)).

**Proper-use patterns.**

- Build the mapper once from the full set of distinct values collected across the tree, then reuse it for every shape ([../inks/ink.go#L88](../inks/ink.go#L88), [../inks/inks.go#L38](../inks/inks.go#L38)).

**Anti-patterns.**

- Do not hash category strings to colours; the mapper exists so the same value keeps the same colour for the whole run ([mapper.go#L63](mapper.go#L63)).

**Source locations.**

- [mapper.go#L31](mapper.go#L31) — `CategoricalMapper`, `NewCategoricalMapper`, `Map`.
