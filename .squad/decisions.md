# Squad Decisions

## Active Decisions

### Radial Tree — Type Reference and Layout

**Author:** Dallas  
**Status:** Implemented

**RadialNode struct:**
```go
type RadialNode struct {
    X, Y         float64     // pixel position relative to canvas centre
    DiscRadius   float64     // radius in pixels
    Angle        float64     // angle in radians (0 = right/east)
    Label        string      // directory or file name
    ShowLabel    bool        // render this label
    IsDirectory  bool        // directory node flag
    FillColour   color.RGBA  // zero = use default
    BorderColour *color.RGBA // nil = use default
    Children     []RadialNode
}
```

**LabelMode constants:**
- `LabelAll` — show labels on all nodes
- `LabelFoldersOnly` — directories only
- `LabelNone` — hide all labels

**Layout() function:** `func Layout(root *model.Directory, canvasSize int, discMetric metric.Name, labels LabelMode) RadialNode`
- Root at (0, 0); coordinates relative to canvas centre
- Angle stored on every node for label rotation
- FillColour/BorderColour set by renderer, not layout

---

### Radial Tree — CLI Design

**Author:** Kane  
**Status:** Implemented

**Key flags:**
- `-d/--disc-size` (required, metric.Name) — numeric metrics only
- `-f/--fill` (optional, metric) — fill colour mapping
- `-b/--border` (optional, metric) — border colour mapping
- `--labels all|folders|none` (default: all)
- `--width`, `--height` (default: 1920)

**Canvas size:** `min(width, height)` — square layout for radial geometry

**Config struct:** `config.Radial` with Fill, FillPalette, Border, BorderPalette, Labels fields

---

### Radial Tree — Three-Pass Rendering

**Author:** Parker  
**Status:** Implemented

**Rendering order:**
1. Edges pass — all parent→child lines
2. Discs pass — all filled circles and borders
3. Labels pass — all text labels

**Why:** Single-pass recursion creates z-order problems. Separating passes ensures edges < discs < labels visually.

**Radial label rotation:**
- Right half (angle ≤ π/2 or > 3π/2): rotate by angle, anchor left
- Left half (angle > π/2 and ≤ 3π/2): rotate by angle + π, anchor right
- Root: centred, unrotated

This keeps text upright on both canvas halves.

---

### Radial Tree — Test Coverage

**Author:** Lambert  
**Status:** Complete (12 tests, all passing)

**Coverage:**
- Root positioning (at origin)
- Ring placement (by depth)
- Angular spread (no duplicates, full circle)
- Disc scaling (metric-based)
- Label modes (all three variants)
- Edge cases (empty tree, single child, nested depth)

**Test file:** `internal/radialtree/layout_test.go`

## Governance

- All meaningful changes require team consensus
- Document architectural decisions here
- Keep history focused on work, decisions focused on direction
