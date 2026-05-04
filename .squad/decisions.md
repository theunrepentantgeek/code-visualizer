# Squad Decisions

## Active Decisions

### Spiral Visualization — Architecture Proposal

**Author:** Ripley  
**Date:** 2026-04-29  
**Status:** Proposed  
**Issue:** #127 — Add spiral visualization  

**Summary:**  
Add `codeviz render spiral` — a time-based visualization where data points are placed along a clockwise outward Archimedean spiral. Time starts at the centre and moves outward. Each point represents a time bucket (hour or day) and carries three metric destinations: disc size (numeric), fill colour (any kind), and border colour (any kind).

This is fundamentally different from treemap/radial/bubbletree — those visualize a file tree structure. The spiral visualizes a **time series**, which means new data infrastructure is required.

**Integration Points (5):**
1. **Layout Package — `internal/spiral/`**
   - `node.go`: `SpiralNode` struct with X, Y, DiscRadius, Angle, SpiralRadius, TimeStart, TimeEnd, Label, ShowLabel, FillColour, BorderColour
   - `layout.go`: `Layout(buckets []TimeBucket, width, height, resolution, labels) []SpiralNode`
   - Algorithm: Archimedean spiral with inner diameter = 1/3 outer, uniform angular spacing
   - `layout_test.go`: 8+ tests covering spiral geometry, angular distribution, edge cases

2. **Config — `internal/config/spiral.go`**
   - `Spiral` struct with Resolution, Size, Fill, FillPalette, Border, BorderPalette, Labels, Legend fields
   - Add `Spiral *Spiral` to root Config struct
   - Defaults: `Resolution: "daily"`, `Labels: "laps"`

3. **Renderer — `internal/render/`**
   - `spiral.go`: `RenderSpiral(nodes, width, height, outputPath, legend) error`
   - Three-pass PNG rendering: background → discs → labels
   - `svg_spiral.go`: SVG with `<circle>` elements and spiral path

4. **CLI Command — `cmd/codeviz/spiral_cmd.go`**
   - `SpiralCmd` struct with TargetPath, Output, Resolution, Size, Fill, FillPalette, Border, BorderPalette, Labels, Width, Height, Filter
   - Register in render_cmd.go
   - Run() flow: scan → build time buckets → aggregate metrics → Layout → render

5. **Time Bucketing — `internal/spiral/`**
   - `timebucket.go`: `TimeBucket` struct (Start, End, Files, SizeValue, FillValue, FillLabel, BorderValue, BorderLabel)
   - `BuildTimeBuckets(root, resolution, startTime, endTime) []TimeBucket`
   - `githistory.go`: `LoadCommitHistory(root) ([]CommitRecord, error)` — fetches commit timestamps from git service

**Design Decisions (8):**
- D1: Flat node list, not a tree (`[]SpiralNode` vs tree with children)
- D2: Clockwise from north angle convention (matches clock reading)
- D3: Inner diameter = 1/3 outer (from spec)
- D4: Resolution determines spots-per-lap (24 hourly, 28 daily)
- D5: Three metric destinations (size/fill/border) map to existing metric pipeline
- D6: Default size metric is commit count (natural "activity" measure)
- D7: Empty time buckets rendered as grey dots (preserve temporal fidelity)
- D8: LabelLaps is v1 default (full labels too crowded; users can override)

**Risks (5):**
1. **Git history performance (MEDIUM):** Large repos with deep history could be slow. Mitigated by git service caching and optional `--since`/`--until` flags.
2. **Non-git targets (LOW):** Spiral requires git. Fail gracefully if no repo found.
3. **Dense spirals (MEDIUM):** 10+ years of daily history = 130+ laps at 1920px. Mitigated by auto-resolution or range suggestions.
4. **Aggregation semantics (LOW):** "Sum of file-lines" is pragmatic; historic metric computation is v2+ scope.
5. **No time-series in model (ADDRESSED):** Spiral package owns its bucketing; doesn't extend core model.

**Implementation Phases:**
| Phase | Owner | Description |
|-------|-------|-------------|
| 1 | Dallas | `internal/spiral/` — node, time buckets, layout, tests |
| 2 | Dallas | `internal/render/spiral.go` + SVG rendering |
| 3 | Kane | CLI command + config + colour application |
| 4 | Lambert | Comprehensive test suite |
| 5 | Bishop | Visual polish |

---

### Spiral Tests Expect Time-Series Input

**Author:** Lambert  
**Date:** 2026-04-29  
**Context:** Issue #127 — Spiral Visualization  

**Observation:**  
The spiral visualization is the first viz type that operates on **time-series data** rather than a directory tree (`model.Directory`). The Layout function will need a fundamentally different input type — something like a slice of timestamped metric entries rather than a `*model.Directory`.

**Implications for Architecture:**
1. **Layout signature differs from existing visualizations.** It won't take `*model.Directory`. Tests assume a time-series input type and time-resolution parameter.
2. **Empty time buckets matter.** Gaps (e.g., no events between hours 2–4) must render as spots with zero/default metrics. The spiral is a continuous path.
3. **Half-open aggregation intervals** (`[start, end)`) need careful boundary handling. Events at exact boundaries (midnight, on-the-hour) must land in the correct bucket.
4. **Two geometric constraints unique to spiral:** inner diameter ≈ 1/3 outer, and consistent angular spacing. These don't exist in treemap/radialtree/bubbletree.

**Test Specification:**  
`.squad/agents/lambert/spiral-test-spec.md` contains 50 test cases across 10 categories:
- Time-series input validation (4 tests)
- Bucket aggregation (6 tests)
- Angular spacing (5 tests)
- Spiral geometry (6 tests)
- Disc sizing constraints (5 tests)
- Label placement modes (5 tests)
- Empty/edge-case buckets (4 tests)
- Colour mapping (4 tests)
- Rendering output (3 tests)
- CLI integration (3 tests)

**Assumed Node Structure:**  
`SpiralNode` with at minimum: `X`, `Y`, `DiscRadius`, `FillColour`, `BorderColour`, `Label`, `ShowLabel` fields.  
`TimeResolution` type (hourly/daily) as a parameter.  
Time-series input with timestamps and per-timestamp metric values.

**Next Steps:**  
Once Ripley's layout signature is finalized, Lambert will adapt the 50 specs into compilable Go tests with Gomega assertions and Goldie golden-file snapshots.

---

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

---

### MetricSpec — Combined metric+palette type

**Author:** Kane
**Status:** Implemented (PR #120)
**Issue:** #118

**Decision:** Introduced `config.MetricSpec` type to bundle metric name and palette name into a single value. This replaces the four separate fields (`Fill`, `FillPalette`, `Border`, `BorderPalette`) on both CLI structs and config structs with two `MetricSpec` fields (`Fill`, `Border`).

**CLI format:**
```
--fill file-type,categorization --border file-lines,foliage
```
Palette is optional — `--fill file-type` uses the provider's default palette.

**Config format:**
```yaml
treemap:
  fill: file-type,categorization
  border: file-lines,foliage
```

**Breaking changes:**
- `--fill-palette` and `--border-palette` CLI flags removed.
- Config file fields `fillPalette` and `borderPalette` removed (combine into `fill`/`border` values).

**Impact:**
- **All command structs** (`TreemapCmd`, `RadialCmd`, `BubbletreeCmd`): Updated to use `MetricSpec`.
- **Config structs** (`Treemap`, `Radial`, `Bubbletree`): `*MetricSpec` replaces separate pointer strings.
- **Helper functions**: `specMetric()` and `specPalette()` replace `ptrString()` for MetricSpec access.
- **Existing config files** using old `fillPalette`/`borderPalette` format will need migration.

---

### go-git FileName log includes non-modifying merge commits

**Author:** Lambert
**Status:** Implemented (PR #119, fixes #114)

**Context:** go-git's `repo.Log(&LogOptions{FileName: &path})` returns merge commits that have the file in their tree but didn't actually change it. This pollutes `commitData.newest` with very recent timestamps, causing `file-freshness` to always be 0.

**Decision:** Added `commitModifiedFile()` TREESAME check to `fetchCommitData()` in `internal/provider/git/service.go`. Each commit's blob hash is compared against all parent commits; if the hash matches any parent, the commit is skipped. This correctly filters merge commits that merely carried the file through.

**Impact:** All three git metrics (file-age, file-freshness, author-count) now correctly exclude non-modifying merge commits. This is most visible for file-freshness but also improves accuracy for file-age and author-count.

**Known limitation:** go-git's history simplification may still miss some commits in complex merge topologies. This hasn't been observed to cause practical issues.


---

### Spiral Layout — Phase 1 Implementation Decisions

**Author:** Dallas
**Date:** 2026-04-29
**Status:** Implemented
**Issue:** #127 — Add spiral visualization

## Decisions

### LabelMode is string-typed, not int-typed

Ripley's architecture proposed `LabelMode int` with iota constants. String type was used (`"all"`, `"laps"`, `"none"`) — matching the radialtree and bubbletree packages. This makes Kong enum integration straightforward and avoids a string-to-int conversion layer.

### BuildTimeBuckets does not take *model.Directory

The architecture showed `BuildTimeBuckets(root *model.Directory, resolution, start, end)`. The `root` parameter was dropped because bucket construction only needs time range and resolution. File assignment to buckets happens in the CLI layer, consistent with the principle that layout computes positions and the CLI layer handles data binding.

### Resolution type is int-based (internal)

Resolution uses `iota` (Hourly=0, Daily=1) because it's internal to the spiral engine and never exposed as a CLI string directly. The CLI command will map `"hourly"/"daily"` strings to these constants.

### Total angle uses n-1 for 0-indexed spacing

For n buckets, the last bucket sits at angle `(n-1) * step` (not `n * step`), since bucket 0 is at θ=0. This means the inner/outer ratio holds precisely at the endpoints of the placed sequence.

## Files

- `internal/spiral/node.go` — SpiralNode struct, LabelMode type
- `internal/spiral/timebucket.go` — Resolution, TimeBucket, BuildTimeBuckets
- `internal/spiral/layout.go` — Layout function, spiral geometry
- `internal/spiral/layout_test.go` — 19 layout tests
- `internal/spiral/timebucket_test.go` — 9 bucket tests

---

### Spiral Config Uses MetricSpec (Not Separate Fields)

**Author:** Kane
**Date:** 2026-04-29
**Status:** Implemented

## Decision

The `config.Spiral` struct uses `*MetricSpec` for Fill and Border fields (matching Treemap, Radial, and Bubbletree post-issue #118), instead of separate `*string` fields for Fill/FillPalette/Border/BorderPalette.

## Rationale

The architecture proposal was written before MetricSpec consolidation (#118/#120). All three existing visualization config structs now use `*MetricSpec`. Keeping spiral consistent avoids a special case and ensures the CLI `--fill metric,palette` syntax works the same way everywhere.

## Impact

- `cmd/codeviz/spiral_cmd.go` uses `config.MetricSpec` for Fill/Border CLI flags
- `internal/config/spiral.go` uses `*MetricSpec` for Fill/Border config fields
- No `FillPalette`/`BorderPalette` separate fields exist on spiral config or CLI

---

### PR #144 & #145 Review Outcomes

**Author:** Ripley  
**Date:** 2026-05-02  
**Status:** Decision Made

## PR #144 — Fix Spiral Visualization Bugs (Fixes #139)

**Decision:** Changes requested (unable to submit review via gh; own PRs)

**What looks good**
- Empty bucket handling and size clamping are aligned with the research direction.
- Shared `spiralBorderWidth` is used in both PNG and SVG renderers, keeping parity.
- `MaxDiscRadius` is exposed from layout and used to clamp disc sizes.

**Blocker**
- `applySpiralDiscSizes` still returns early when `maxSize == 0`, which means empty buckets retain the default disc radius if all size values are zero. This violates the "no commits → no dot" requirement in that edge case (e.g., size metric set to a value that is zero for all files).

**Requested change**
- Ensure empty buckets get `DiscRadius = 0` even when `maxSize == 0` (e.g., move the empty-bucket handling ahead of the `maxSize == 0` return, and default active buckets to `minDiscRadius` if needed).

## PR #145 — Add New Git Metrics (Fixes #136)

**Decision:** Approved (unable to submit review via gh; own PRs)

**Summary**
- Requirements are met: new metrics added, `commitData` extended, Patch API used for churn stats, commit-density matches spec, registration and `IsGitMetric` updated.
- Tests cover metadata, churn calculations, density edge cases, and non-git error handling.

## Notes

- PR #144 edge case was fixed by Dallas (moved empty-bucket handling before maxSize check).
- `task ci` not available in environment (task not installed).
- `go test ./...` passes locally.

---

### Structural Audit — Codebase Review and Refactoring Strategy

**Author:** Bishop (Artificer)  
**Date:** 2026-05-03  
**Status:** Proposed  

## Summary

Completed a full structural audit of `cmd/codeviz/` and all `internal/` packages. Filed 11 issues (#152–#162) covering the most impactful structural improvements needed to reduce duplication and clarify abstractions.

## Structural Health

The codebase is **well-designed at the package level** — boundaries are mostly correct, types are meaningful, and the metric/provider/model/layout/render pipeline makes sense. Issues are almost entirely about **duplication and missing intermediate abstractions**, not fundamental design flaws. This is natural for a codebase that grew from one viz type to four without extracting common patterns.

## Top 3 High-Leverage Refactoring Opportunities

### 1. Extract Shared Command Workflow (Issue #152)
The four viz commands (treemap, radial, bubbletree, spiral) duplicate ~60% of their code. A shared pipeline/template method would eliminate ~1,500 lines and make new viz types trivial to add. **Single highest-leverage change.**

### 2. Unify Raster/SVG Rendering (Issue #158)
Each viz type has paired raster + SVG renderers that duplicate traversal and drawing logic. A rendering abstraction (draw-list or backend interface) would halve the render package and prevent duplication from scaling.

### 3. Declarative Git Providers (Issue #155)
Seven git provider files are structurally identical wrappers differing only in 4 parameters. Replace with table-driven registration to eliminate ~180 lines and make new metrics a one-liner.

## Sequencing Recommendation

- **Quick wins:** Issues #155 (git providers) and #153 (config base) are independent and low-risk
- **Major refactor:** Issue #152 (command workflow) is largest but most impactful; should be planned carefully
- **Dependent:** Issue #158 (render unification) best tackled after #152, since command layer cleanup will clarify render API surface

## Issues Filed

#152–#162 (11 total):
- #152: Extract shared command workflow
- #153: Config base abstraction
- #154: Metrics registration
- #155: Git provider consolidation
- #156–#162: Additional refactoring opportunities

All issues include detailed scope, acceptance criteria, and implementation notes.
