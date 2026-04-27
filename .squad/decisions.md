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
**Status:** Complete (16 tests, all passing)

**Coverage:**
- Root positioning (at origin)
- Ring placement (by depth)
- Angular spread (no duplicates, full circle)
- Disc scaling (metric-based)
- Label modes (all three variants)
- Edge cases (empty tree, single child, nested depth)

**Test files:** 
- `internal/radialtree/layout_test.go` (12 tests)
- `internal/render/radialtree_test.go` (4 smoke tests, new)

---

### Kong struct fields use pointers; defaults in config.New()

**Author:** Dallas  
**Date:** 2026-04-15  
**Status:** Implemented

**Context:** PR review identified that `Labels string \`default:"all"\`` in `RadialCmd` caused Kong to always write `"all"` into `c.Labels`, silently ignoring user-configured defaults.

**Decision:** All Kong CLI struct fields that mirror a `config.*` pointer field must use pointer-compatible semantics:
- Remove `default:` tags from Kong string fields with corresponding `*string` in config
- Add `""` as first value in `enum:` so Kong accepts unset/empty state
- Handle defaults in `config.New()` only — single authoritative source

**Consequence:** `config.New()` sets `Radial.Labels = "all"`. CLI `--labels` only overrides on explicit user flag. `resolveLabels` simplified to single code path.

**Pattern:** Apply to any future CLI flags mapping to config pointer fields.

---

### Render + Layout Fixes — Code Quality

**Author:** Parker  
**Date:** 2026-04-15  
**Status:** Implemented  
**Files:** `internal/render/radialtree.go`, `internal/radialtree/layout.go`

**RenderRadialPNG signature:** Takes `*radialtree.RadialNode` pointer (Dallas updates call site; Lambert uses `&node` in tests).

**External label colour:** All non-root labels use fixed dark constant `#222222` (canvas background is white; disc-fill-based colour always wrong).

**Disc z-order:** Collect-sort-draw in `drawDiscs` (no recursion) guarantees smaller nodes render on top regardless of traversal order.

**Stroke batching:** One `dc.Stroke()` per node level (after all child edges) reduces GPU/CPU round-trips.

**Crowding prevention:** 
- Ring spacing floor: minimum ≥ `n * (2*minFileDisc + 4)` pixels
- Disc shrink: `adjustedDiscFactor()` reduces max when `n > π/maxDiscFactor`
- `dirDiscFactor` halved (`0.12 → 0.06`) for proportionate directory dots

**Docs:** `layout.go` computeLeafCount doc fixed (returns 0; callers guard).

---

### Foliage Palette — Issue #46

**Author:** Dallas  
**Date:** 2026-04-18  
**Status:** Implemented  
**Files:** `internal/palette/palette.go`, `internal/palette/palette_test.go`

**Decision:**
Added `Foliage` palette (`"foliage"`) — an 11-step ordered colour ramp from near-black (dead plant) through brown, orange, yellow, to intense green (healthy plant). Intended for "health" visualisations where low values = dead/brown and high values = healthy green.

**Colour Progression:**
Black → dark brown → brown → dark orange → orange → yellow-orange → yellow → yellow-green → medium green → intense green.

**Rationale:**
- Plant-health metaphor is intuitive for code-health metrics (age, churn, coverage).
- 11 steps matches the temperature palette granularity.
- All adjacent steps pass WCAG 2.0 contrast ratio >= 1.0.
- Palette is `Ordered: true` so it works with the existing numeric metric mapper.

---

### Bubble Tree Visualization — Architecture Proposal

**Author:** Ripley  
**Date:** 2026-04-19  
**Status:** Proposed  
**Issue:** #33 — Add Bubble visualization

**Summary:**
Add `codeviz render bubbletree` — a circle-packing visualization where directories are labelled circles containing nested child circles, and files are unlabelled dots. This follows patterns from GitHub Next and repo-visualizer.

**Architecture highlights:**
- **Package:** `internal/bubbletree/` (mirrors radialtree/treemap pattern)
- **Node type:** `BubbleNode` with `X, Y, Radius` (single radius primitive vs RadialNode's Angle+DiscRadius)
- **Algorithm:** Front-chain circle packing (Wang et al. 2006) + Welzl's enclosing circle — implement in pure Go, no D3 dependency
- **Layout function:** `Layout(root, width, height, sizeMetric, labels) BubbleNode` — takes width/height like treemap (non-square canvas support)
- **CLI:** `BubbletreeCmd` with `--size` flag (not `--disc-size` as in radial); default 1920×1080 (like treemap)
- **Config:** `Bubbletree` struct with Fill/Border/Labels; defaults to `Labels="folders"` (file dots unlabelled)
- **Rendering:** Three-pass PNG (directories→files→labels for z-order); SVG with nested `<g>` groups
- **Reuse:** All existing packages (model, metric, palette, scan, render infrastructure); no new dependencies

**Implementation phases:**
1. Layout engine (front-chain + enclosing circle)
2. PNG rendering (three-pass with fogleman/gg)
3. SVG rendering (nested groups)
4. CLI + Config wiring
5. Optional: curved labels, force-sim polish

**No force-simulation required for v1** — front-chain alone produces good results; polish is a follow-up.

**Open questions:** Minimum file radius, directory transparency alpha, max-depth support, root-level file grouping.

**Risk:** Low (algorithm well-documented, O(n²) is acceptable for typical codebases). Visual quality without force-sim is medium risk — mitigated by front-chain being proven good.

**Decision:** Proceed with this architecture.

---

### Bubble Tree Layout Engine — Algorithm & Constants

**Author:** Dallas  
**Date:** 2026-04-19  
**Status:** Implemented (Phase 1)  
**Issue:** #33

**BubbleNode struct:**
```go
type BubbleNode struct {
    X, Y         float64
    Radius       float64
    Label        string
    ShowLabel    bool
    IsDirectory  bool
    FillColour   color.RGBA
    BorderColour *color.RGBA
    Children     []BubbleNode
}
```

**Layout() signature:** `func Layout(root *model.Directory, width, height int, sizeMetric metric.Name, labels LabelMode) BubbleNode`
- Takes width+height (like treemap, non-square canvas). Returns value type.

**Algorithm choices:**
- **Front-chain packing** without chain pruning. O(n³) per level, acceptable for typical directory sizes (<100 direct children).
- **Welzl's enclosing circle** adapted for circles (not points). Falls back to pairwise enclosing when degenerate.
- **Leaf sizing:** `radius = sqrt(metricValue)` with `minFileRadius = 2px` floor.
- **Padding constants:** `siblingPadding = 3px`, `parentPadding = 6px`.
- **Fallback placement:** Golden angle distribution on outer edge when no valid tangent position found.

**Consequence:** Renderers receive a fully-positioned tree with absolute pixel coordinates. Colours set by renderer/CLI, not layout.

---

### Validation Ordering — Validate() vs validateEffective()

**Author:** Kane  
**Date:** 2026-04-26  
**Status:** Implemented  
**Issue:** #99

**Problem:**
Kong calls `Validate()` on command structs during `ctx.Run()`, BEFORE the command's `Run()` method executes. Config file loading (`TryAutoLoad`) and CLI→config merging (`applyOverrides`) happen inside `Run()`. This means `Validate()` sees empty size fields when `--size`/`--disc-size`/`--bubble-size` wasn't passed on CLI, even though the config file supplies them.

**Decision:**
Split validation into two phases:

1. **`Validate()`** — Only checks CLI-only concerns that don't depend on config file values. Currently: filter glob syntax validation.
2. **`validateEffective()`** — Called from `Run()` after config load + merge + size backfill. Checks size metric existence and kind, fill/border metric-palette validity, and border-palette-requires-border constraint.

**Kong struct tag changes:**
- Removed `required:"true"` from size fields
- Added `default:""` (Kong requires either required or default for enum fields)
- Added leading comma to enum list: `enum:",file-size,file-lines,..."` to accept empty values

**Size backfill pattern in Run():**
After `applyOverrides()`, if the size field is still empty, read it back from the merged config:
```go
if c.Size == "" {
    if s := ptrString(flags.Config.Treemap.Size); s != "" {
        c.Size = metric.Name(s)
    }
}
```

**Applies to:** `TreemapCmd`, `RadialCmd`, `BubbletreeCmd`

**Consequence:** Any future command with config-dependent fields must follow this pattern — keep `Validate()` for CLI-only checks, defer config-dependent validation to `Run()`.

---

### User Directive — PR Review Thread Replies

**Author:** Bevan (via Copilot)  
**Date:** 2026-04-18  
**Status:** Active

**Directive:**
Anytime changes are pushed to address PR review comments, always reply to the review thread confirming the action taken (or explaining why no change was needed).

**Rationale:** User request — reinforces existing team practice of maintaining clear communication on review threads.

### Export Data — Issue #107

**Author:** Ripley  
**Date:** 2026-04-26  
**Status:** Implemented

#### Overview

Feature adds `--export-data` CLI flag to save computed metrics to JSON or YAML files after metrics computation but before rendering.

#### 1. Export Data Structure

```go
type ExportData struct {
	Directory *DirectoryExport `json:"directory"`
}

type DirectoryExport struct {
	Name         string                   `json:"name"`
	Path         string                   `json:"path"`
	Files        []*FileExport            `json:"files"`
	Directories  []*DirectoryExport       `json:"directories"`
	Quantities   map[string]int64         `json:"quantities"`
	Measures     map[string]float64       `json:"measures"`
	Classifications map[string]string     `json:"classifications"`
}

type FileExport struct {
	Name            string                 `json:"name"`
	Path            string                 `json:"path"`
	Extension       string                 `json:"extension"`
	IsBinary        bool                   `json:"isBinary"`
	Quantities      map[string]int64       `json:"quantities"`
	Measures        map[string]float64     `json:"measures"`
	Classifications map[string]string      `json:"classifications"`
}
```

**Rationale:**
- Flat metric maps simplify JSON/YAML serialization and make output self-describing
- String keys preserve human readability instead of using `metric.Name` constants
- Full paths enable post-export filtering or analysis without tree reconstruction
- IsBinary flag preserved for debugging verification
- Structure mirrors model tree (recursive directories + flat files)

#### 2. Package Location

**Decision:** `internal/export/` with single `Export()` function

**Rationale:**
- Mirrors existing package patterns (render, scan, config)
- Clear separation of concerns independent of CLI, rendering, and metric computation
- Allows independent implementation and testing
- Easy to extend with new formats

#### 3. API Signature

```go
func Export(root *model.Directory, requested []metric.Name, outputPath string) error
```

**Design points:**
- Format inferred from file extension (.json or .yaml/.yml)
- Takes requested metric names to ensure only computed metrics exported
- Returns error wrapped with eris for consistency
- No config dependency; export agnostic to visualization type

#### 4. Flag Placement

**Decision:** `--export-data` added to `Flags` struct (not per-command)

**Rationale:**
- Consistency with existing `--export-config` pattern
- Export works on any visualization command (treemap, radial, bubble)
- Avoids duplication; each command checks `flags.ExportData` after `provider.Run()`

#### 5. Metric Visibility

**Decision:** Use requested metric names passed to `Export()`

**Implementation:** 
- Each command collects requested metrics (e.g., `collectRequestedMetrics(size, fill, border)`)
- Passed to `export.Export()` along with root tree
- Export logic iterates through requested names and extracts values only for those metrics
- No new model methods required; leverages existing getters

**Rationale:**
- No model changes needed
- Explicit control; only metrics actually requested are exported
- Clean separation; export doesn't need metric registry knowledge

#### 6. Integration Flow

Each command's `Run()` method follows this pattern:

```
1. Merge config and validate
2. Scan filesystem
3. Compute metrics (collect requested list)
4. [NEW] Export metrics if --export-data flag provided
5. Render visualization
```

#### Summary Table

| Aspect | Decision |
|--------|----------|
| **Data structure** | Recursive `DirectoryExport` + flat `FileExport` with metric maps |
| **Package** | `internal/export/` with single `Export()` function |
| **API** | `Export(root *model.Directory, requested []metric.Name, outputPath string) error` |
| **Flag** | `--export-data` on `Flags` struct (like `--export-config`) |
| **Metrics** | Use requested list passed to Export; no new model methods |
| **Integration** | Call after `provider.Run()`, before render |

## Governance

- All meaningful changes require team consensus
- Document architectural decisions here
- Keep history focused on work, decisions focused on direction

---

### Git providers: use worktree root for relative paths

**Author:** Dallas
**Date:** 2026-07-21
**Status:** Implemented

**Context:** All git metric providers (file-age, file-freshness, author-count) were computing file paths relative to the scan root (`root.Path`), but go-git expects paths relative to the git worktree root. This caused zero-value metrics when scanning a subdirectory.

**Decision:** `repoService` now stores the worktree root path (via `Worktree().Filesystem.Root()`) and exposes it as `RepoRoot()`. All path computations use `RepoRoot()` as the base.

**Also:** Factored the triplicated Load() walk-and-compute pattern into a single `loadGitMetric()` helper to prevent this class of bug from recurring if new git providers are added.

---

### Decision: Filter false-positive commits in git metric providers

**Author:** Dallas
**Date:** 2026-04-27
**Issue:** #114
**PR:** #119

**Context:** go-git's `LogOptions{FileName}` filter includes merge commits that didn't modify the target file. This caused `file-freshness` to always return 0 because `data.newest` reflected the repo's most recent commit, not the file's last actual change.

**Decision:** Added `commitModifiedFile()` to `fetchCommitData` in `internal/provider/git/service.go`. Each commit returned by go-git is verified by comparing the file's blob hash against the first parent's tree. Commits where the file hash is unchanged are skipped.

**Impact:**
- Fixes `file-freshness` (was always 0, now returns correct days-since-last-change).
- Also improves accuracy of `file-age` and `author-count` — they shared the same inflated commit set, but the impact was less visible.
- Small performance cost: each commit now does a tree lookup + parent tree lookup. Acceptable for correctness.

---

### Decision: Replace times slice with explicit oldest/newest in commitData

**Author:** Dallas  
**Date:** 2026-07-21  
**Status:** Implemented  
**Scope:** `internal/provider/git/service.go`

**Context:** The `commitData` struct stored commit timestamps in a `times []time.Time` slice, then used positional indexing (`times[0]` for newest, `times[len-1]` for oldest) to compute file-age and file-freshness. This assumed go-git's iteration order (by committer time descending) matched author-time order. It doesn't — author time ≠ committer time for merge commits, rebases, cherry-picks, and PRs.

**Decision:** Replaced `times []time.Time` with two explicit fields: `oldest time.Time` and `newest time.Time`. During `fetchCommitData()` iteration, track min/max with `Before()`/`After()` comparisons. This eliminates any dependency on iteration order.

**Rationale:**
- Correct by construction — no ordering assumption, just min/max.
- Simpler — two fields instead of a growing slice. Less memory, no `len()` checks.
- All existing tests pass unchanged (synthetic repos have correctly ordered timestamps, but the fix is still correct for them).

---

### golangci-lint upgraded to v2.11.4

**Author:** Lambert
**Date:** 2026-04-26
**Issue:** #113
**PR:** #115

**What changed:**
- golangci-lint and golangci-lint-custom (nilaway) upgraded from v2.8.0 to v2.11.4
- All `sort.Slice`/`sort.Strings`/`sort.Float64s` replaced with `slices.SortFunc`/`slices.Sort` (new revive `use-slices-sort` rule)
- `slog.Error(err.Error())` replaced with structured `slog.Error("msg", "err", err)` (gosec G706)
- `filepath.Clean()` added for user-supplied paths (gosec G703)
- Nil guard added to `provider/registry.go:get()` for nilaway safety

**Impact on team:**
- **All code must now use `slices.Sort`/`slices.SortFunc` instead of `sort.Slice`/`sort.Strings`/`sort.Float64s`**. The linter will enforce this.
- **Use structured slog logging** (`slog.Error("message", "err", err)`) instead of `slog.Error(err.Error())`.
- **Sanitise user-supplied file paths** with `filepath.Clean()` before use.
- New revive rules suggested for adoption in issue #116.

**Version references:**
- `.devcontainer/install-dependencies.sh` line 149
- `.devcontainer/.custom-gcl.template.yml` line 1

---

### Ripley Review Verdict — PR #108 Export Metrics

**Author:** Ripley
**Date:** 2026-04-26
**PR:** #108
**Issue:** #107
**Verdict:** REJECT

**Decision:** Standardize YAML usage for export code on the repository's existing package:
- Use `go.yaml.in/yaml/v3`
- Do not introduce `gopkg.in/yaml.v3` alongside it

**Why:** The codebase already uses `go.yaml.in/yaml/v3` in config loading/saving and in the new export tests. Adding `gopkg.in/yaml.v3` in `internal/export/export.go` introduces a second YAML library with the same API surface for no functional gain, increases dependency surface, and breaks consistency.

**Required follow-up:**
1. Change `internal/export/export.go` to import `go.yaml.in/yaml/v3`
2. Run module tidy so `gopkg.in/yaml.v3` is removed from `go.mod`/`go.sum`
3. Re-run the targeted export and CLI tests, then let CI confirm the full suite

