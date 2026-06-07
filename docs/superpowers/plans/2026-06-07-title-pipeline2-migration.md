# Title Flag — Pipeline2 Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Re-deliver the `--title` CLI flag for all five visualisations on top of `origin/main`, conformed to the typed-state pipeline that landed in PR #368.

**Architecture:** The original `feat(title)` work (commit `4817c8d`) was authored against the old `pipeline.Run` + generic `Stage[S VizState]` API. Main now uses `pipeline.NewState` with `ApplyFuncX/XY/XYZ` and shared stages with concrete `func(*CommonState) error` signatures. Config + canvas + CLI layers carry over verbatim from the original commit; only the shared-stages and per-viz layout/render hooks change shape.

**Tech Stack:** Go 1.26.1, `pipeline` typed-state package, `stages` shared-stages package, Kong CLI, fogleman/gg renderer, Gomega + Goldie tests.

**Spec:** [docs/superpowers/specs/2026-06-07-title-pipeline2-migration-design.md](../specs/2026-06-07-title-pipeline2-migration-design.md)

---

## Task 0: Reset branch to main

Discard the obsolete `4817c8d` commit and start from a clean `origin/main`.

**Files:**
- Modify: working tree (no file edits in this task)

- [ ] **Step 1: Stash uncommitted noise so it isn't lost**

```bash
git stash push -u -m "title-port: pre-reset working tree" -- \
  .github/workflows/repo-assist.md .go-version
```

Expected: stash created. The `.vscode/launch.json`, `cmd/codeviz/debug-sample.png`, and `.agents/skills/thermo-nuclear-code-quality-review/` are not staged via include paths above; they remain in the working tree as untracked files (out of scope).

- [ ] **Step 2: Reset branch hard to origin/main**

```bash
git fetch origin main
git reset --hard origin/main
```

Expected: HEAD now matches `origin/main` (`5e6e709` or newer). The previous title commit (`4817c8d`) is gone from the branch but remains in the reflog.

- [ ] **Step 3: Restore the docs/spec from the previous tip**

The design doc lives in commit `742a796` (parent reflog entry); cherry-pick just that file from the reflog so it survives the reset.

```bash
git checkout 'HEAD@{2}' -- docs/superpowers/specs/2026-06-07-title-pipeline2-migration-design.md
git add docs/superpowers/specs/2026-06-07-title-pipeline2-migration-design.md
git commit -m "docs: add title pipeline2 migration design"
```

Expected: spec restored, single new commit on top of `origin/main`. (If `HEAD@{2}` no longer points at the doc commit, use `git reflog` to find the right ref.)

- [ ] **Step 4: Confirm branch state**

```bash
git log --oneline origin/main..HEAD
git status --short
```

Expected: exactly one commit ahead of `origin/main` (the design doc). Working tree clean apart from the pre-existing untracked files.

---

## Task 1: Add `Title` config struct and `Config.OverrideTitleText`

Mirrors the existing `Footer` config plumbing.

**Files:**
- Create: `internal/config/title.go`
- Create: `internal/config/title_test.go`
- Modify: `internal/config/config.go`

- [ ] **Step 1: Write `title_test.go`**

Create `internal/config/title_test.go`:

```go
package config

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestTitle_ShowTitle_FalseByDefault(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ti := &Title{}
	g.Expect(ti.ShowTitle()).To(BeFalse())
}

func TestTitle_ShowTitle_FalseWhenHidden(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	hidden := true
	text := "My Repo"
	ti := &Title{
		Hidden: &hidden,
		Text:   &text,
	}

	g.Expect(ti.ShowTitle()).To(BeFalse())
}

func TestTitle_ShowTitle_FalseWhenTextEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	empty := ""
	ti := &Title{Text: &empty}
	g.Expect(ti.ShowTitle()).To(BeFalse())
}

func TestTitle_ShowTitle_TrueWhenTextSet(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	text := "My Repository"
	ti := &Title{Text: &text}
	g.Expect(ti.ShowTitle()).To(BeTrue())
}

func TestTitle_ShowTitle_NilTitle_ReturnsFalse(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var ti *Title
	g.Expect(ti.ShowTitle()).To(BeFalse())
}

func TestConfig_OverrideTitleText_SetsText(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := New()
	cfg.OverrideTitleText("My Project")

	g.Expect(cfg.Title).NotTo(BeNil())
	g.Expect(*cfg.Title.Text).To(Equal("My Project"))
}

func TestConfig_OverrideTitleText_Empty_LeavesNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := New()
	cfg.OverrideTitleText("")

	g.Expect(cfg.Title).To(BeNil())
}
```

- [ ] **Step 2: Run the tests; expect compile failures**

```bash
go test ./internal/config/... -run 'Title|OverrideTitleText' -count=1
```

Expected: build fails — `Title` undefined, `cfg.OverrideTitleText` undefined.

- [ ] **Step 3: Create `title.go`**

Create `internal/config/title.go`:

```go
package config

// Title holds configuration for the title rendered at the top of each
// generated image.
type Title struct {
	Text   *string `yaml:"text,omitempty"   json:"text,omitempty"`
	Hidden *bool   `yaml:"hidden,omitempty" json:"hidden,omitempty"`
}

// ShowTitle reports whether the title should be rendered.
func (t *Title) ShowTitle() bool {
	if t == nil {
		return false
	}

	if t.Hidden != nil && *t.Hidden {
		return false
	}

	if t.Text == nil || *t.Text == "" {
		return false
	}

	return true
}

// OverrideText sets Text to v if v is non-empty.
func (t *Title) OverrideText(v string) { overrideString(&t.Text, v) }
```

- [ ] **Step 4: Add `Title` field and `OverrideTitleText` to `Config`**

In `internal/config/config.go`, add the `Title` field to the `Config` struct (place it on the line directly above `Footer`):

```go
	Scatter    *Scatter      `yaml:"scatter,omitempty"    json:"scatter,omitempty"`
	Title      *Title        `yaml:"title,omitempty"      json:"title,omitempty"`
	Footer     *Footer       `yaml:"footer,omitempty"     json:"footer,omitempty"`
```

Then add (immediately above the existing `OverrideFooterText` method):

```go
// OverrideTitleText sets Title.Text to v if v is non-empty.
func (c *Config) OverrideTitleText(v string) {
	if c.Title == nil && v == "" {
		return
	}

	c.ensureTitle()
	c.Title.OverrideText(v)
}

// ensureTitle initialises Title if it is nil.
func (c *Config) ensureTitle() {
	if c.Title == nil {
		c.Title = &Title{}
	}
}
```

- [ ] **Step 5: Run tests; expect pass**

```bash
go test ./internal/config/... -count=1
```

Expected: all `Title*` and `OverrideTitleText*` tests pass; existing tests stay green.

- [ ] **Step 6: Commit**

```bash
git add internal/config/title.go internal/config/title_test.go internal/config/config.go
git commit -m "feat(config): add Title struct and OverrideTitleText"
```

---

## Task 2: Render title on the canvas

Adds the canvas-side machinery: constants, field, setter/getter, and the actual draw call.

**Files:**
- Modify: `internal/canvas/canvas.go`

- [ ] **Step 1: Add title constants alongside the footer constants**

Find the existing constant block in `internal/canvas/canvas.go` containing `FooterReservedHeight` and append the title constants. After:

```go
	// FooterReservedHeight is the vertical space (in pixels) the footer claims.
	// Layout stages should subtract this from the canvas height when the footer
	// is enabled, preventing content from being drawn underneath it.
	FooterReservedHeight = footerFontSize + footerMarginY
)
```

…the block becomes:

```go
	// FooterReservedHeight is the vertical space (in pixels) the footer claims.
	// Layout stages should subtract this from the canvas height when the footer
	// is enabled, preventing content from being drawn underneath it.
	FooterReservedHeight = footerFontSize + footerMarginY

	titleFontSize = 18.0
	titleMarginY  = 20.0

	// TitleReservedHeight is the vertical space (in pixels) that the title
	// occupies when rendered. Layout stages subtract this from the available
	// height (offset from the top) when the title is enabled.
	TitleReservedHeight = titleFontSize + titleMarginY
)
```

- [ ] **Step 2: Add `titleColor` next to `footerColor`**

Find:

```go
var footerColor = color.RGBA{R: 128, G: 128, B: 128, A: 200}
```

Append on the next line:

```go
var titleColor = color.RGBA{R: 40, G: 40, B: 40, A: 255}
```

- [ ] **Step 3: Add `title` field to `Canvas`**

Locate the `Canvas` struct and add `title *string` immediately before `footer *string`:

```go
type Canvas struct {
	width  int
	height int
	shapes []layeredShape
	legend *LegendConfig
	title  *string
	footer *string
}
```

- [ ] **Step 4: Add `SetTitle` and `TitleText` methods**

Immediately after the existing `SetFooter` method, add:

```go
// TitleText returns the current title text, or an empty string if no title
// has been set. Primarily useful for testing.
func (c *Canvas) TitleText() string {
	if c.title == nil {
		return ""
	}

	return *c.title
}

// SetTitle configures the title text for this canvas.
// An empty string clears a previously set title.
func (c *Canvas) SetTitle(text string) {
	if text == "" {
		c.title = nil

		return
	}

	c.title = &text
}
```

- [ ] **Step 5: Draw the title in `RenderTo`**

Locate `(c *Canvas) RenderTo(backend Backend) error`. Find the block that draws the footer (begins `if c.footer != nil {`). Insert the title block immediately above it:

```go
	if c.title != nil {
		pos := model.Position{
			X: float64(c.width) / 2,
			Y: titleMarginY,
		}
		backend.DrawText(pos, *c.title, titleColor, titleFontSize, model.AnchorMiddle, 0)
	}

	if c.footer != nil {
		pos := model.Position{
```

- [ ] **Step 6: Build to verify the canvas package compiles**

```bash
go build ./internal/canvas/...
```

Expected: clean build.

- [ ] **Step 7: Run canvas tests**

```bash
go test ./internal/canvas/... -count=1
```

Expected: all existing tests pass (no new tests added in this task — coverage comes via the stages tests in Task 3).

- [ ] **Step 8: Commit**

```bash
git add internal/canvas/canvas.go
git commit -m "feat(canvas): add SetTitle/TitleText and draw title at top"
```

---

## Task 3: Add `ApplyTitle` stage and `EffectiveTitleHeight` helper

Conforms to main's typed-state pipeline shape: a plain `func(*CommonState) error`, NOT a generic over `VizState`.

**Files:**
- Modify: `internal/stages/canvas.go`
- Modify: `internal/stages/canvas_test.go`

- [ ] **Step 1: Write failing tests for `ApplyTitle` and `EffectiveTitleHeight`**

In `internal/stages/canvas_test.go`, append (after the existing `TestEffectiveFooterHeight_*` tests):

```go
func TestApplyTitle_NilCanvas_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := &stages.CommonState{
		Canvas:     nil,
		RootConfig: config.New(),
	}

	g.Expect(stages.ApplyTitle(c)).To(Succeed())
}

func TestApplyTitle_NilRootConfig_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := &stages.CommonState{
		Canvas:     canvas.NewCanvas(100, 100),
		RootConfig: nil,
	}

	g.Expect(stages.ApplyTitle(c)).To(Succeed())
}

func TestApplyTitle_NoTitle_NoTitleOnCanvas(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	// Default config has no title text; ShowTitle() is false.

	cv := canvas.NewCanvas(800, 600)
	c := &stages.CommonState{
		Canvas:     cv,
		RootConfig: cfg,
	}

	g.Expect(stages.ApplyTitle(c)).To(Succeed())
	g.Expect(cv.TitleText()).To(BeEmpty())
}

func TestApplyTitle_WithTitle_SetsTitleOnCanvas(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	cfg.OverrideTitleText("My Project")

	cv := canvas.NewCanvas(800, 600)
	c := &stages.CommonState{
		Canvas:     cv,
		RootConfig: cfg,
	}

	g.Expect(stages.ApplyTitle(c)).To(Succeed())
	g.Expect(cv.TitleText()).To(Equal("My Project"))
}

func TestApplyTitle_TitleHidden_NoTitleOnCanvas(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	cfg.OverrideTitleText("My Project")
	hidden := true
	cfg.Title.Hidden = &hidden

	cv := canvas.NewCanvas(800, 600)
	c := &stages.CommonState{
		Canvas:     cv,
		RootConfig: cfg,
	}

	g.Expect(stages.ApplyTitle(c)).To(Succeed())
	g.Expect(cv.TitleText()).To(BeEmpty())
}

func TestEffectiveTitleHeight_NilConfig_ReturnsZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(stages.EffectiveTitleHeight(nil)).To(Equal(0))
}

func TestEffectiveTitleHeight_NoTitle_ReturnsZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	g.Expect(stages.EffectiveTitleHeight(cfg)).To(Equal(0))
}

func TestEffectiveTitleHeight_TitleShown_ReturnsPositive(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	cfg.OverrideTitleText("My Project")

	height := stages.EffectiveTitleHeight(cfg)
	g.Expect(height).To(BeNumerically(">", 0))
	g.Expect(height).To(Equal(int(canvas.TitleReservedHeight)))
}
```

If `canvas` is not yet imported in `canvas_test.go`, it already is (used by the footer tests). If `config` is not yet imported, add it. The existing footer tests show the import block.

- [ ] **Step 2: Run tests; expect compile failure**

```bash
go test ./internal/stages/... -run 'ApplyTitle|EffectiveTitleHeight' -count=1
```

Expected: build fails — `stages.ApplyTitle` and `stages.EffectiveTitleHeight` undefined.

- [ ] **Step 3: Add `ApplyTitle` and `EffectiveTitleHeight` to `internal/stages/canvas.go`**

Append at the bottom of `internal/stages/canvas.go`:

```go
// ApplyTitle sets the title on c.Canvas from RootConfig.Title.
// If the Title is nil, hidden, or has no text, the canvas title is left unset.
func ApplyTitle(c *CommonState) error {
	if c.Canvas == nil || c.RootConfig == nil {
		return nil
	}

	title := c.RootConfig.Title
	if !title.ShowTitle() {
		return nil
	}

	c.Canvas.SetTitle(*title.Text)

	return nil
}

// EffectiveTitleHeight returns the number of pixels that the title occupies
// when rendered. Layout stages subtract this from the top of the available
// area so that visualisation content does not overlap the title.
// Returns 0 when cfg is nil or the title is not shown.
func EffectiveTitleHeight(cfg *config.Config) int {
	if cfg == nil {
		return 0
	}

	if !cfg.Title.ShowTitle() {
		return 0
	}

	return int(canvas.TitleReservedHeight)
}
```

- [ ] **Step 4: Run tests; expect pass**

```bash
go test ./internal/stages/... -count=1
```

Expected: all new title tests pass; footer tests still green.

- [ ] **Step 5: Commit**

```bash
git add internal/stages/canvas.go internal/stages/canvas_test.go
git commit -m "feat(stages): add ApplyTitle stage and EffectiveTitleHeight"
```

---

## Task 4: Reserve title space in `bubbletree.LayoutStage`

**Files:**
- Modify: `internal/bubbletree/stages.go`

- [ ] **Step 1: Update `LayoutStage`**

In `internal/bubbletree/stages.go`, replace the existing `LayoutStage` body so it accounts for `titleH` both when subtracting from `availH` and when offsetting nodes. Replace:

```go
func LayoutStage(c *stages.CommonState, b *State) error {
	availH := c.Height - stages.EffectiveFooterHeight(c.RootConfig)
	layoutW, layoutH := legend.ReserveAndLayout(b.LegendConfig, c.Width, availH)

	b.Nodes = Layout(c.Root, layoutW, layoutH, b.Size, b.Labels)

	if layoutW < c.Width || layoutH < availH {
		if b.LegendConfig != nil {
			wReduce, hReduce := b.LegendConfig.ReserveSpace()
			dx, dy := legend.LayoutOffset(b.LegendConfig, wReduce, hReduce)
			OffsetNodes(&b.Nodes, dx, dy)
		}
	}

	return nil
}
```

with:

```go
func LayoutStage(c *stages.CommonState, b *State) error {
	titleH := stages.EffectiveTitleHeight(c.RootConfig)
	availH := c.Height - stages.EffectiveFooterHeight(c.RootConfig) - titleH
	layoutW, layoutH := legend.ReserveAndLayout(b.LegendConfig, c.Width, availH)

	b.Nodes = Layout(c.Root, layoutW, layoutH, b.Size, b.Labels)

	dx, dy := float64(0), float64(titleH)
	if layoutW < c.Width || layoutH < availH {
		if b.LegendConfig != nil {
			wReduce, hReduce := b.LegendConfig.ReserveSpace()
			ldx, ldy := legend.LayoutOffset(b.LegendConfig, wReduce, hReduce)
			dx += ldx
			dy += ldy
		}
	}

	OffsetNodes(&b.Nodes, dx, dy)

	return nil
}
```

- [ ] **Step 2: Build and run bubbletree tests**

```bash
go test ./internal/bubbletree/... -count=1
```

Expected: all tests still pass. (Existing tests don't set a title, so `titleH == 0` and the offset is `(0, 0)` plus whatever the legend already produced — behaviour unchanged.)

- [ ] **Step 3: Commit**

```bash
git add internal/bubbletree/stages.go
git commit -m "feat(bubbletree): reserve title space in LayoutStage"
```

---

## Task 5: Reserve title space in `treemap.LayoutStage`

**Files:**
- Modify: `internal/treemap/stages.go`

- [ ] **Step 1: Update `LayoutStage`**

Replace the existing function body with:

```go
func LayoutStage(c *stages.CommonState, t *State) error {
	titleH := stages.EffectiveTitleHeight(c.RootConfig)
	availH := c.Height - stages.EffectiveFooterHeight(c.RootConfig) - titleH
	layoutW, layoutH := legend.ReserveAndLayout(t.LegendConfig, c.Width, availH)

	rect := Layout(c.Root, layoutW, layoutH, t.Size)

	dx, dy := float64(0), float64(titleH)
	if layoutW < c.Width || layoutH < availH {
		if t.LegendConfig != nil {
			wReduce, hReduce := t.LegendConfig.ReserveSpace()
			ldx, ldy := legend.LayoutOffset(t.LegendConfig, wReduce, hReduce)
			dx += ldx
			dy += ldy
		}
	}

	OffsetRects(&rect, dx, dy)
	t.Root = rect

	return nil
}
```

- [ ] **Step 2: Run treemap tests**

```bash
go test ./internal/treemap/... -count=1
```

Expected: all tests pass.

- [ ] **Step 3: Commit**

```bash
git add internal/treemap/stages.go
git commit -m "feat(treemap): reserve title space in LayoutStage"
```

---

## Task 6: Reserve title space in `scatter.LayoutStage`

**Files:**
- Modify: `internal/scatter/stages.go`

- [ ] **Step 1: Update `LayoutStage`**

Replace the existing function body with:

```go
func LayoutStage(c *stages.CommonState, x *State) error {
	titleH := stages.EffectiveTitleHeight(c.RootConfig)
	availH := c.Height - stages.EffectiveFooterHeight(c.RootConfig) - titleH
	layoutW, layoutH := legend.ReserveAndLayout(x.LegendConfig, c.Width, availH)

	layout := Layout(x.Dataset, layoutW, layoutH, x.XAxis, x.YAxis)

	dx, dy := float64(0), float64(titleH)
	if layoutW < c.Width || layoutH < availH {
		if x.LegendConfig != nil {
			wReduce, hReduce := x.LegendConfig.ReserveSpace()
			ldx, ldy := legend.LayoutOffset(x.LegendConfig, wReduce, hReduce)
			dx += ldx
			dy += ldy
		}
	}

	OffsetLayout(&layout, dx, dy)
	x.Layout = layout

	return nil
}
```

- [ ] **Step 2: Run scatter tests**

```bash
go test ./internal/scatter/... -count=1
```

Expected: all tests pass.

- [ ] **Step 3: Commit**

```bash
git add internal/scatter/stages.go
git commit -m "feat(scatter): reserve title space in LayoutStage"
```

---

## Task 7: Reserve title space in `spiral.LayoutStage`

Spiral has no `Offset*` helper; it shifts the layout centre and node positions inline.

**Files:**
- Modify: `internal/spiral/stages.go`

- [ ] **Step 1: Update `LayoutStage`**

Replace the existing function body with:

```go
func LayoutStage(c *stages.CommonState, p *State) error {
	titleH := stages.EffectiveTitleHeight(c.RootConfig)
	availH := c.Height - stages.EffectiveFooterHeight(c.RootConfig) - titleH

	layout := Layout(p.Buckets, c.Width, availH, p.Resolution, p.Labels)
	maxDisc := MaxDiscRadius(len(p.Buckets), c.Width, availH, p.Resolution)

	ApplyDiscSizes(layout.Nodes, p.Buckets, maxDisc)

	if titleH > 0 {
		dy := float64(titleH)
		layout.CY += dy

		for i := range layout.Nodes {
			layout.Nodes[i].Y += dy
		}
	}

	p.Layout = layout

	return nil
}
```

- [ ] **Step 2: Run spiral tests**

```bash
go test ./internal/spiral/... -count=1
```

Expected: all tests pass.

- [ ] **Step 3: Commit**

```bash
git add internal/spiral/stages.go
git commit -m "feat(spiral): reserve title space in LayoutStage"
```

---

## Task 8: Reserve title space in radialtree

Radialtree differs: it uses a square content area, not a generic offset helper. The canvas height grows by `titleH` and the content centre shifts down. This requires changing `RenderToCanvas`'s signature, so we update its tests in lockstep.

**Files:**
- Modify: `internal/radialtree/render.go`
- Modify: `internal/radialtree/render_internal_test.go`
- Modify: `internal/radialtree/stages.go`

- [ ] **Step 1: Update `RenderToCanvas` to accept a `topOffset`**

In `internal/radialtree/render.go`, change the function signature and body. Replace the existing `RenderToCanvas` and the helper `addBackground`:

```go
// RenderToCanvas walks the layout and model trees, adding shapes to the canvas.
// canvasSize is the side length (pixels) of the square content area.
// topOffset is the number of pixels to reserve at the top (e.g. for a title);
// the canvas height becomes canvasSize + topOffset and the content centre is
// shifted down by topOffset so it fits below the reserved area.
func RenderToCanvas(
	nodes *RadialNode,
	root *model.Directory,
	canvasSize int,
	topOffset int,
	inks Inks,
) *canvas.Canvas {
	canvasHeight := canvasSize + topOffset
	cv := canvas.NewCanvas(canvasSize, canvasHeight)

	cx := float64(canvasSize) / 2.0
	cy := float64(canvasSize)/2.0 + float64(topOffset)

	addBackground(cv, canvasSize, canvasHeight)
	addEdges(cv, *nodes, cx, cy)
	addDiscs(cv, nodes, root, cx, cy, inks)
	addLabels(cv, *nodes, cx, cy, inks)

	return cv
}

// addBackground adds a white background rectangle.
func addBackground(cv *canvas.Canvas, canvasWidth, canvasHeight int) {
	bgSpec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        canvas.FixedInk(bgColour),
			Border:      canvas.FixedInk(borderColour),
			BorderWidth: 0,
		},
	}

	cv.AddRectangle(canvas.LayerBackground, canvas.Rectangle{
		Spec:  bgSpec,
		W:     float64(canvasWidth),
		H:     float64(canvasHeight),
		Focus: canvasmodel.Point{X: 0.5, Y: 0.5},
	})
}
```

- [ ] **Step 2: Update existing `RenderToCanvas` callers in tests**

In `internal/radialtree/render_internal_test.go`, every call site currently reads:

```go
cv := RenderToCanvas(&nodes, root, <size>, inks)
```

Replace each with:

```go
cv := RenderToCanvas(&nodes, root, <size>, 0, inks)
```

There are four call sites: `TestRenderRadialToCanvas_PNG` (size 800), `TestRenderRadialToCanvas_SVG` (size 400), `TestRenderRadialToCanvas_NestedDirs` (size 800), and `TestRenderRadialToCanvas_EmptyDir` (size 400). Insert `0,` before `inks` in each.

- [ ] **Step 3: Update `LayoutStage` and `RenderStage` to thread `titleH`**

In `internal/radialtree/stages.go`, replace `LayoutStage` and `RenderStage`:

```go
func LayoutStage(c *stages.CommonState, r *State) error {
	titleH := stages.EffectiveTitleHeight(c.RootConfig)
	availH := c.Height - stages.EffectiveFooterHeight(c.RootConfig) - titleH
	canvasSize := min(c.Width, availH)

	r.Nodes = Layout(c.Root, canvasSize, r.DiscSize, r.Labels)

	return nil
}

func RenderStage(c *stages.CommonState, r *State) error {
	titleH := stages.EffectiveTitleHeight(c.RootConfig)
	availH := c.Height - stages.EffectiveFooterHeight(c.RootConfig) - titleH
	canvasSize := min(c.Width, availH)

	cv := RenderToCanvas(&r.Nodes, c.Root, canvasSize, titleH, r.Inks)
	if r.LegendConfig != nil {
		cv.SetLegend(*r.LegendConfig)
	}

	c.Canvas = cv

	return nil
}
```

- [ ] **Step 4: Run radialtree tests**

```bash
go test ./internal/radialtree/... -count=1
```

Expected: all tests pass.

- [ ] **Step 5: Update golden files if needed**

If the radial tests use Goldie snapshots that differ because the canvas size changed when no title is set: when `titleH == 0`, `canvasSize + topOffset == canvasSize`, so behaviour is unchanged. No golden updates expected. If a snapshot fails anyway, regenerate:

```bash
GOLDIE_UPDATE=1 go test ./internal/radialtree/... -count=1
```

…and inspect the diff before accepting.

- [ ] **Step 6: Commit**

```bash
git add internal/radialtree/render.go internal/radialtree/render_internal_test.go internal/radialtree/stages.go
git commit -m "feat(radialtree): reserve title space via RenderToCanvas topOffset"
```

---

## Task 9: Wire `--title` flag into `treemap` CLI

**Files:**
- Modify: `cmd/codeviz/treemap_cmd.go`

- [ ] **Step 1: Add `Title` struct field**

Find the `Footer` field declaration:

```go
	Footer     string `default:"" help:"Override footer text on the generated image." optional:""`
	HideFooter bool   `default:"false" help:"Suppress the attribution footer." name:"hide-footer" optional:""`
```

Insert the `Title` field on the line directly above `Footer`:

```go
	Title      string `default:"" help:"Override title text on the generated image." optional:""`
	Footer     string `default:"" help:"Override footer text on the generated image." optional:""`
	HideFooter bool   `default:"false" help:"Suppress the attribution footer." name:"hide-footer" optional:""`
```

- [ ] **Step 2: Apply the override in `applyOverrides`**

Find:

```go
	cfg.OverrideFooterText(c.Footer)
	cfg.OverrideHideFooter(c.HideFooter)
```

Insert above it:

```go
	cfg.OverrideTitleText(c.Title)
	cfg.OverrideFooterText(c.Footer)
	cfg.OverrideHideFooter(c.HideFooter)
```

- [ ] **Step 3: Wire the `ApplyTitle` stage into the pipeline**

Find:

```go
	pipeline.ApplyFuncX(s, stages.ApplyFooter)
```

Insert above it:

```go
	pipeline.ApplyFuncX(s, stages.ApplyTitle)
	pipeline.ApplyFuncX(s, stages.ApplyFooter)
```

- [ ] **Step 4: Build**

```bash
go build ./cmd/...
```

Expected: clean build.

- [ ] **Step 5: Commit**

```bash
git add cmd/codeviz/treemap_cmd.go
git commit -m "feat(treemap): add --title CLI flag"
```

---

## Task 10: Wire `--title` flag into `bubbletree` CLI

**Files:**
- Modify: `cmd/codeviz/bubbletree_cmd.go`

- [ ] **Step 1: Add `Title` struct field**

Find the `Footer` field block in the `BubbletreeCmd` struct (immediately after `Width`/`Height`):

```go
	Footer     string `default:"" help:"Override footer text on the generated image." optional:""`
	HideFooter bool   `default:"false" help:"Suppress the attribution footer." name:"hide-footer" optional:""`
```

Insert the `Title` field directly above:

```go
	Title      string `default:"" help:"Override title text on the generated image." optional:""`
	Footer     string `default:"" help:"Override footer text on the generated image." optional:""`
	HideFooter bool   `default:"false" help:"Suppress the attribution footer." name:"hide-footer" optional:""`
```

- [ ] **Step 2: Apply the override in `applyOverrides`**

Find:

```go
	cfg.OverrideFooterText(c.Footer)
	cfg.OverrideHideFooter(c.HideFooter)
```

Insert above:

```go
	cfg.OverrideTitleText(c.Title)
	cfg.OverrideFooterText(c.Footer)
	cfg.OverrideHideFooter(c.HideFooter)
```

- [ ] **Step 3: Wire the stage into `Run`**

Find:

```go
	pipeline.ApplyFuncX(s, stages.ApplyFooter)
```

Insert above:

```go
	pipeline.ApplyFuncX(s, stages.ApplyTitle)
	pipeline.ApplyFuncX(s, stages.ApplyFooter)
```

- [ ] **Step 4: Build**

```bash
go build ./cmd/...
```

Expected: clean build.

- [ ] **Step 5: Commit**

```bash
git add cmd/codeviz/bubbletree_cmd.go
git commit -m "feat(bubbletree): add --title CLI flag"
```

---

## Task 11: Wire `--title` flag into `radialtree` CLI

**Files:**
- Modify: `cmd/codeviz/radialtree_cmd.go`

- [ ] **Step 1: Add `Title` struct field**

Insert directly above the existing `Footer` field:

```go
	Title      string `default:"" help:"Override title text on the generated image." optional:""`
	Footer     string `default:"" help:"Override footer text on the generated image." optional:""`
	HideFooter bool   `default:"false" help:"Suppress the attribution footer." name:"hide-footer" optional:""`
```

- [ ] **Step 2: Apply the override**

In `applyOverrides`, insert `cfg.OverrideTitleText(c.Title)` directly above `cfg.OverrideFooterText(c.Footer)`.

- [ ] **Step 3: Wire the stage**

In `Run`, insert `pipeline.ApplyFuncX(s, stages.ApplyTitle)` directly above `pipeline.ApplyFuncX(s, stages.ApplyFooter)`.

- [ ] **Step 4: Build**

```bash
go build ./cmd/...
```

Expected: clean build.

- [ ] **Step 5: Commit**

```bash
git add cmd/codeviz/radialtree_cmd.go
git commit -m "feat(radialtree): add --title CLI flag"
```

---

## Task 12: Wire `--title` flag into `scatter` CLI

**Files:**
- Modify: `cmd/codeviz/scatter_cmd.go`

- [ ] **Step 1: Add `Title` field above `Footer`**

```go
	Title      string `default:"" help:"Override title text on the generated image." optional:""`
	Footer     string `default:"" help:"Override footer text on the generated image." optional:""`
	HideFooter bool   `default:"false" help:"Suppress the attribution footer." name:"hide-footer" optional:""`
```

- [ ] **Step 2: Apply the override**

In `applyOverrides`, insert `cfg.OverrideTitleText(c.Title)` directly above `cfg.OverrideFooterText(c.Footer)`.

- [ ] **Step 3: Wire the stage**

In `Run`, insert `pipeline.ApplyFuncX(s, stages.ApplyTitle)` directly above `pipeline.ApplyFuncX(s, stages.ApplyFooter)`.

- [ ] **Step 4: Build**

```bash
go build ./cmd/...
```

Expected: clean build.

- [ ] **Step 5: Commit**

```bash
git add cmd/codeviz/scatter_cmd.go
git commit -m "feat(scatter): add --title CLI flag"
```

---

## Task 13: Wire `--title` flag into `spiral` CLI

**Files:**
- Modify: `cmd/codeviz/spiral_cmd.go`

- [ ] **Step 1: Add `Title` field above `Footer`**

```go
	Title      string `default:"" help:"Override title text on the generated image." optional:""`
	Footer     string `default:"" help:"Override footer text on the generated image." optional:""`
	HideFooter bool   `default:"false" help:"Suppress the attribution footer." name:"hide-footer" optional:""`
```

- [ ] **Step 2: Apply the override**

In `applyOverrides`, insert `cfg.OverrideTitleText(c.Title)` directly above `cfg.OverrideFooterText(c.Footer)`.

- [ ] **Step 3: Wire the stage**

In `Run`, insert `pipeline.ApplyFuncX(s, stages.ApplyTitle)` directly above `pipeline.ApplyFuncX(s, stages.ApplyFooter)`.

- [ ] **Step 4: Build**

```bash
go build ./cmd/...
```

Expected: clean build.

- [ ] **Step 5: Commit**

```bash
git add cmd/codeviz/spiral_cmd.go
git commit -m "feat(spiral): add --title CLI flag"
```

---

## Task 14: Full verification

Run the project's CI bundle to ensure nothing else regressed.

- [ ] **Step 1: Run the full test suite**

```bash
task test
```

Expected: all tests pass.

- [ ] **Step 2: Run lint and full CI via Explore subagent**

Per repo convention (`.github/copilot-instructions.md`), `task lint` and `task ci` must be dispatched through an Explore subagent because of verbose output. Ask the subagent for: exit status; the count and identity of failing linters/tests; offending `file:line` and message for each issue; or a one-line "no issues" note.

```bash
task ci
```

Expected (via subagent): zero failing linters, zero failing tests.

- [ ] **Step 3: Smoke-render**

```bash
task build
./bin/codeviz bubbletree . -o /tmp/codeviz-title-smoke.png --title "Hello"
```

Expected: PNG written; opening it shows the title text near the top, with bubbletree content rendered below — not overlapping.

- [ ] **Step 4: Run the no-title path to confirm zero regression**

```bash
./bin/codeviz bubbletree . -o /tmp/codeviz-no-title-smoke.png
```

Expected: PNG written; title area is absent and the bubbletree fills the same space it did on `origin/main` (no spurious top margin).

---

## Task 15: Push and open PR

- [ ] **Step 1: Push branch**

The branch was force-reset; the remote needs `--force-with-lease`.

```bash
git push --force-with-lease
```

Expected: push succeeds.

- [ ] **Step 2: Open / update PR**

Via `gh` or the GitHub UI, target `main`. Title: `feat: add --title flag to all visualization commands`. Body should reference the design doc and note that the original PR (#354) was rebased onto the new pipeline.

---

## Self-Review

**Spec coverage:**
- §1 Config layer → Task 1.
- §2 Canvas layer → Task 2.
- §3 Shared stages (`ApplyTitle`, `EffectiveTitleHeight`) → Task 3.
- §4 Per-viz layout stages → Tasks 4–8 (one per viz, plus the radialtree render-signature change).
- §5 CLI commands → Tasks 9–13 (one per cmd file).
- §6 Out of scope → respected via the stash list in Task 0 step 1.
- Verification (`task test`, `task ci`, smoke-render) → Task 14.

**Placeholder scan:** No TBDs, "implement later", or "similar to Task N". Every code step shows the actual code.

**Type consistency:**
- `Title` struct fields: `Text *string`, `Hidden *bool` — used the same in `title.go`, `title_test.go`, and the `cfg.Title.Hidden = &hidden` line in Task 3.
- `(*Title).ShowTitle()`, `(*Title).OverrideText(string)`, `(*Config).OverrideTitleText(string)` — same names everywhere.
- `Canvas.SetTitle(string)`, `Canvas.TitleText() string` — same names in canvas, stages tests, and stages function.
- `stages.ApplyTitle(*CommonState) error`, `stages.EffectiveTitleHeight(*config.Config) int` — same signatures in definition, tests, and viz call sites.
- `pipeline.ApplyFuncX(s, stages.ApplyTitle)` — matches `ApplyFooter`'s wiring on main.
- `radialtree.RenderToCanvas(nodes, root, canvasSize, topOffset, inks)` — new signature is consistent across `render.go`, four test call sites in `render_internal_test.go`, and `RenderStage` in `stages.go`.

No issues found.
