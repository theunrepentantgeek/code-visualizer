# Centered Treemap Directory Captions Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Center every top and left treemap directory caption within its rail.

**Architecture:** Keep caption fitting and rail layout unchanged. Rendering will use the canvas middle text anchor and the rail midpoint for both orientations, with rotation remaining the only positional difference between top and left captions.

**Tech Stack:** Go 1.26.1, fogleman/gg canvas backends, Gomega, Goldie v2, Task

---

## File Map

| Path | Responsibility |
|---|---|
| `internal/treemap/render.go` | Center directory captions at each rail midpoint |
| `internal/treemap/render_directory_chrome_test.go` | Verify top and left caption anchors, positions, and rotations |
| `internal/goldentest/testdata/treemap-png.golden` | Updated raster snapshot |
| `internal/goldentest/testdata/treemap-svg.golden` | Updated SVG snapshot |
| `samples/tree-map/code-visualizer.png` | Updated tree-map raster sample |
| `samples/tree-map/code-visualizer.svg` | Updated tree-map SVG sample |

### Task 1: Center Directory Captions During Rendering

**Files:**
- Modify: `internal/treemap/render.go:21-32,220-236`
- Modify: `internal/treemap/render_directory_chrome_test.go:32-80`

- [ ] **Step 1: Change the top-rail capture assertion**

In `TestRenderToCanvas_DrawsTopDirectoryChrome`, change the expected text call
to the rail midpoint and middle anchor:

```go
g.Expect(hasText(backend.texts, textCall{
	pos:      canvas.Position{X: 50, Y: 20},
	text:     "source",
	fontSize: 12,
	anchor:   canvas.AnchorMiddle,
	rotation: 0,
})).To(BeTrue())
```

The fixture rail is `{X: 10, Y: 10, W: 80, H: 20}`, so its midpoint is
`{X: 50, Y: 20}`.

- [ ] **Step 2: Change the left-rail capture assertion**

In `TestRenderToCanvas_DrawsLeftDirectoryChrome`, change the expected text call
to the rail midpoint and middle anchor:

```go
g.Expect(hasText(backend.texts, textCall{
	pos:      canvas.Position{X: 20, Y: 50},
	text:     "source",
	fontSize: 12,
	anchor:   canvas.AnchorMiddle,
	rotation: -math.Pi / 2,
})).To(BeTrue())
```

The fixture rail is `{X: 10, Y: 10, W: 20, H: 80}`, so its midpoint is
`{X: 20, Y: 50}`.

- [ ] **Step 3: Run the focused tests and verify they fail**

Run:

```bash
go test ./internal/treemap \
  -run 'TestRenderToCanvas_Draws(Top|Left)DirectoryChrome' -count=1
```

Expected: FAIL because rendering still emits `AnchorStart` at the padded start
of each rail.

- [ ] **Step 4: Change both directory caption specs to middle anchoring**

In `internal/treemap/render.go`, update the two text specs without changing
font size or rotation:

```go
dirTopLabelSpec = &canvas.TextSpec{
	Ink:      inks.FixedInk(palette.White),
	FontSize: directoryLabelFontSize,
	Anchor:   canvas.AnchorMiddle,
	Rotation: 0,
}
dirLeftLabelSpec = &canvas.TextSpec{
	Ink:      inks.FixedInk(palette.White),
	FontSize: directoryLabelFontSize,
	Anchor:   canvas.AnchorMiddle,
	Rotation: -math.Pi / 2,
}
```

- [ ] **Step 5: Position every caption at the rail midpoint**

Replace the orientation-specific start coordinates in `addDirectoryShapes`
with one midpoint calculation:

```go
if rect.Chrome.Text != "" {
	spec := dirTopLabelSpec
	rail := rect.Chrome.Rail

	if rect.Chrome.Orientation == DirectoryLabelLeft {
		spec = dirLeftLabelSpec
	}

	cv.AddText(canvas.LayerOverlay, canvas.Text{
		Spec:    spec,
		X:       rail.X + rail.W/2,
		Y:       rail.Y + rail.H/2,
		Content: rect.Chrome.Text,
	})
}
```

Do not change rail geometry, fitting, truncation, omission, colors, borders, or
root handling.

- [ ] **Step 6: Format and run treemap tests**

Run:

```bash
task fmt
go test ./internal/treemap -count=1
```

Expected: PASS.

- [ ] **Step 7: Run targeted lint**

Run the repository's custom linter through a low-noise task agent:

```bash
./tools/golangci-lint-custom run ./internal/treemap/...
```

Expected: exit 0 with no issues.

- [ ] **Step 8: Commit centered rendering**

Stage only the source and capture-test files:

```bash
git add internal/treemap/render.go \
  internal/treemap/render_directory_chrome_test.go
git diff --cached --name-only
```

Expected staged paths:

```text
internal/treemap/render.go
internal/treemap/render_directory_chrome_test.go
```

Commit:

```bash
git commit -m "fix(treemap): center directory captions" \
  -m "Co-authored-by: Copilot App <223556219+Copilot@users.noreply.github.com>" \
  -m "Copilot-Session: 0113bb97-3cb7-4511-a2a7-f156a4d83cf6"
```

### Task 2: Update Centered Caption Artifacts

**Files:**
- Modify: `internal/goldentest/testdata/treemap-png.golden`
- Modify: `internal/goldentest/testdata/treemap-svg.golden`
- Modify: `samples/tree-map/code-visualizer.png`
- Modify: `samples/tree-map/code-visualizer.svg`

- [ ] **Step 1: Record all pre-existing sample modifications**

Run:

```bash
git status --short samples
```

The eight PNG/SVG files under `samples/bubble-tree`, `samples/radial-tree`,
`samples/scatter`, and `samples/spiral` are unrelated and must remain unstaged
and unrestored. The two tree-map artifacts are generated outputs and will be
deliberately replaced by this task.

- [ ] **Step 2: Confirm the old treemap goldens fail**

Run:

```bash
go test ./internal/goldentest -run TestGolden_Treemap -count=1
```

Expected: FAIL because the committed snapshots still contain start-anchored
directory captions.

- [ ] **Step 3: Regenerate and verify only treemap goldens**

Run:

```bash
GOLDIE_UPDATE=1 go test ./internal/goldentest -run TestGolden_Treemap -count=1
go test ./internal/goldentest -run TestGolden_Treemap -count=1
```

Expected: both commands PASS after regeneration; only the two treemap golden
files change.

- [ ] **Step 4: Regenerate only tree-map samples**

Run:

```bash
task samples-tree-map
```

Expected: only `samples/tree-map/code-visualizer.png` and
`samples/tree-map/code-visualizer.svg` are rewritten by this command. The
eight unrelated sample modifications remain untouched.

- [ ] **Step 5: Inspect centered SVG captions**

Run:

```bash
rg '<text .*fill="rgb\(255,255,255\)".*text-anchor="middle".*>(src|docs)</text>' \
  internal/goldentest/testdata/treemap-svg.golden
rg '<text .*fill="rgb\(255,255,255\)".*text-anchor="middle".*rotate\(-90\.00.*>docs</text>' \
  internal/goldentest/testdata/treemap-svg.golden
rg '<text .*fill="rgb\(255,255,255\)".*text-anchor="middle"' \
  samples/tree-map/code-visualizer.svg
rg '<text .*fill="rgb\(255,255,255\)".*text-anchor="middle".*rotate\(-90\.00' \
  samples/tree-map/code-visualizer.svg
```

Expected: top and rotated left directory captions use
`text-anchor="middle"` in both SVG artifacts. Their slate rail fills and white
text remain present.

- [ ] **Step 6: Run focused verification**

Run:

```bash
go test ./internal/treemap ./internal/goldentest \
  -run 'Test(RenderToCanvas|Golden_Treemap)' -count=1
```

Expected: PASS.

- [ ] **Step 7: Stage exactly the four generated artifacts**

Run:

```bash
git add internal/goldentest/testdata/treemap-png.golden \
  internal/goldentest/testdata/treemap-svg.golden \
  samples/tree-map/code-visualizer.png \
  samples/tree-map/code-visualizer.svg
git diff --cached --name-only
```

Expected staged paths:

```text
internal/goldentest/testdata/treemap-png.golden
internal/goldentest/testdata/treemap-svg.golden
samples/tree-map/code-visualizer.png
samples/tree-map/code-visualizer.svg
```

Commit:

```bash
git commit -m "test(treemap): update centered caption snapshots" \
  -m "Co-authored-by: Copilot App <223556219+Copilot@users.noreply.github.com>" \
  -m "Copilot-Session: 0113bb97-3cb7-4511-a2a7-f156a4d83cf6"
```

### Task 3: Verify and Update the Existing Pull Request

**Files:**
- No source changes expected

- [ ] **Step 1: Run repository gates**

Dispatch a low-noise task agent to run these commands sequentially:

```bash
PATH="$PWD/tools:$PATH" task tidy
PATH="$PWD/tools:$PATH" task build
PATH="$PWD/tools:$PATH" task test
PATH="$PWD/tools:$PATH" task lint
```

Expected: all four commands exit 0. Ask the agent to report only exit statuses,
failing test or linter identities, and `file:line` messages.

- [ ] **Step 2: Confirm only unrelated samples remain dirty**

Run:

```bash
git status --short
```

Expected: only the eight pre-existing PNG/SVG modifications under
`samples/bubble-tree`, `samples/radial-tree`, `samples/scatter`, and
`samples/spiral` remain. Do not stage or restore them.

- [ ] **Step 3: Review the complete centering change**

Run a final reviewer over the implementation range beginning with the design
commit. Confirm:

- both orientations use `AnchorMiddle`;
- both positions are the exact rail midpoint;
- rotations are unchanged;
- fitting, geometry, palette, borders, and omission behavior are unchanged;
- updated artifacts contain centered top and left captions;
- no unrelated samples are committed.

Expected: no Critical or Important findings.

- [ ] **Step 4: Push the existing branch**

Run:

```bash
git push origin theunrepentantgeek-design-treemap-directory-labels
```

Expected: the existing pull request branch advances without force-pushing.

- [ ] **Step 5: Update the pull request description if needed**

Ensure the existing pull request explains that top and left directory captions
are centered within their rails. Preserve its existing motivation, palette
description, and test plan.
