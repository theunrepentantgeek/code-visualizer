# SVG Three-Decimal Precision Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Serialize every floating-point value produced by the SVG backend with exactly three decimal places.

**Architecture:** Keep SVG generation in the existing `internal/canvas/svg` backend and change its format verbs in place. Prove the policy at the backend boundary with focused string assertions, then regenerate only the SVG golden fixtures affected by the intentional serialization change.

**Tech Stack:** Go 1.26.1, `fmt`, Gomega, Goldie v2, Task

---

## File Map

- Modify `internal/canvas/svg/backend.go`: apply fixed three-decimal formatting to every floating-point SVG value and the radial-gradient cache key.
- Modify `internal/canvas/svg/backend_test.go`: assert three-decimal output across every drawing primitive and update existing fixed-width expectations.
- Modify `internal/canvas/svg/gradient_test.go`: update gradient percentage expectations.
- Regenerate `internal/goldentest/testdata/*-svg.golden`: capture the intentional SVG serialization changes; PNG fixtures remain untouched.

### Task 1: Specify and implement the SVG precision policy

**Files:**
- Modify: `internal/canvas/svg/backend_test.go`
- Modify: `internal/canvas/svg/gradient_test.go`
- Modify: `internal/canvas/svg/backend.go`

- [ ] **Step 1: Write the failing precision test**

Add `math` to the imports in `internal/canvas/svg/backend_test.go`, then add this test before `TestSVGBackend_ImplementsBackendInterface`:

```go
func TestSVGBackend_FormatsAllFloatingPointValuesWithThreeDecimals(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(400, 400)
	solid := model.SolidFill{Color: color.RGBA{A: 255}}
	gradient := model.RadialGradientFill{
		Center: color.RGBA{R: 255, A: 255},
		Edge:   color.RGBA{B: 255, A: 255},
		Focus:  model.Point{X: 0.333333, Y: 0.666667},
	}

	b.DrawRectangle(
		model.Position{X: 10.1234, Y: 20.5678},
		model.Size{Width: 30.1234, Height: 40.5678},
		gradient, solid, 0.4567,
	)
	b.DrawDisc(
		model.Position{X: 50.1234, Y: 60.5678},
		7.8912, solid, solid, 0.4567,
	)
	b.DrawPolygon(
		[]model.Position{{X: 1.1234, Y: 2.5678}, {X: 3.1234, Y: 4.5678}, {X: 5.1234, Y: 6.5678}},
		solid, solid, 0.4567,
	)
	b.DrawFilledPath(
		[][]model.Position{{
			{X: 11.1234, Y: 12.5678},
			{X: 13.1234, Y: 14.5678},
			{X: 15.1234, Y: 16.5678},
		}},
		color.RGBA{R: 1, G: 2, B: 3, A: 128},
	)
	b.DrawLine(
		model.Position{X: 21.1234, Y: 22.5678},
		model.Position{X: 23.1234, Y: 24.5678},
		color.RGBA{R: 1, G: 2, B: 3, A: 128},
		0.4567,
	)
	b.DrawPath(
		[]model.Position{{X: 31.1234, Y: 32.5678}, {X: 33.1234, Y: 34.5678}},
		color.RGBA{A: 255},
		0.4567,
	)
	b.DrawText(
		model.Position{X: 41.1234, Y: 42.5678},
		"label",
		color.RGBA{A: 255},
		12.3456,
		model.AnchorMiddle,
		math.Pi/7,
	)
	b.DrawArcText(
		model.Position{X: 100.1234, Y: 200.5678},
		30.1234,
		"arc",
		color.RGBA{A: 255},
		12.3456,
	)

	out := filepath.Join(t.TempDir(), "precision.svg")
	g.Expect(b.Finish(out)).To(Succeed())
	content := readFile(t, out)

	for _, expected := range []string{
		`fx="33.333%" fy="66.667%"`,
		`<rect x="10.123" y="20.568" width="30.123" height="40.568"`,
		`stroke-width="0.457"`,
		`<circle cx="50.123" cy="60.568" r="7.891"`,
		`points="1.123,2.568 3.123,4.568 5.123,6.568"`,
		`M 11.123 12.568 L 13.123 14.568 L 15.123 16.568 Z`,
		`rgba(1,2,3,0.502)`,
		`<line x1="21.123" y1="22.568" x2="23.123" y2="24.568"`,
		`<path d="M 31.123 32.568 L 33.123 34.568"`,
		`<text x="41.123" y="42.568"`,
		`font-size="12.346"`,
		`transform="rotate(25.714 41.123 42.568)"`,
		`M84.000,200.568 A16.123,16.123 0 0,1 116.247,200.568`,
	} {
		g.Expect(content).To(ContainSubstring(expected))
	}
}
```

Update existing expectations in the same file so they express the new contract:

```go
`<path d="M 10.000 10.000 L 100.000 50.000 L 190.000 10.000" fill="none"`
`<polygon points="1.000,1.000 9.000,1.000 1.000,9.000"`
`stroke-width="0.500"`
`<path d="M 1.000 1.000 L 9.000 1.000 L 9.000 9.000 L 1.000 9.000 Z ` +
	`M 3.000 3.000 L 7.000 3.000 L 7.000 7.000 L 3.000 7.000 Z"`
`font-size="12.000"`
"M114.000,200.000"
"286.000,200.000"
```

Replace all three existing `font-size="12.0"` expectations with
`font-size="12.000"`. In `internal/canvas/svg/gradient_test.go`, replace
`fx="35.0%"` and `fy="35.0%"` with:

```go
g.Expect(svg).To(ContainSubstring(`fx="35.000%"`))
g.Expect(svg).To(ContainSubstring(`fy="35.000%"`))
```

- [ ] **Step 2: Run the focused test to verify it fails**

Run:

```bash
go test ./internal/canvas/svg -run 'TestSVGBackend_(FormatsAllFloatingPointValuesWithThreeDecimals|DrawPath|DrawPolygon|DrawFilledPath|DrawText|DrawArcText)' -count=1
```

Expected: FAIL because the backend still emits one or two decimal places for
most values.

- [ ] **Step 3: Change every SVG floating-point formatter**

In `internal/canvas/svg/backend.go`, change each floating-point format verb used
for SVG output or gradient identity from `%.1f` or `%.2f` to `%.3f`.
Concretely, the resulting format strings must be:

```go
`<rect x="%.3f" y="%.3f" width="%.3f" height="%.3f" fill="%s" stroke="%s" stroke-width="%.3f"/>`
"%s|%s|%.3f|%.3f"
`<defs><radialGradient id="%s" cx="50%%" cy="50%%" r="70%%" fx="%.3f%%" fy="%.3f%%">`
`<circle cx="%.3f" cy="%.3f" r="%.3f" fill="%s" stroke="%s" stroke-width="%.3f"/>`
"%.3f,%.3f"
` stroke="%s" stroke-width="%.3f"`
"M %.3f %.3f"
" L %.3f %.3f"
`<line x1="%.3f" y1="%.3f" x2="%.3f" y2="%.3f" stroke="%s" stroke-width="%.3f"/>`
`<path d="M %.3f %.3f`
` L %.3f %.3f`
`" fill="none" stroke="%s" stroke-width="%.3f"/>`
`<text x="%.3f" y="%.3f" fill="%s" font-size="%.3f" font-family="sans-serif" `
`transform="rotate(%.3f %.3f %.3f)">%s</text>`
`<text x="%.3f" y="%.3f" fill="%s" font-size="%.3f" font-family="sans-serif" `
`<defs><path id="%s" d="M%.3f,%.3f A%.3f,%.3f 0 0,1 %.3f,%.3f" fill="none"/></defs>`
`<text fill="%s" font-size="%.3f" font-family="sans-serif" dominant-baseline="middle">`
```

Keep the existing `rgba(%d,%d,%d,%.3f)` formatter unchanged because it already
satisfies the policy. Do not change integer canvas dimensions, RGB channels,
IDs, arc flags, or literal percentage constants.

- [ ] **Step 4: Format the changed Go files**

Run:

```bash
gofumpt -w internal/canvas/svg/backend.go internal/canvas/svg/backend_test.go internal/canvas/svg/gradient_test.go
```

Expected: command exits successfully with no output.

- [ ] **Step 5: Run all SVG backend tests**

Run:

```bash
go test ./internal/canvas/svg -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit the backend policy and focused tests**

```bash
git add internal/canvas/svg/backend.go internal/canvas/svg/backend_test.go internal/canvas/svg/gradient_test.go
git commit -m "fix(svg): increase numeric precision"
```

### Task 2: Regenerate intentional SVG golden changes

**Files:**
- Modify: `internal/goldentest/testdata/bubbletree-svg.golden`
- Modify: `internal/goldentest/testdata/radial-svg.golden`
- Modify: `internal/goldentest/testdata/scatter-svg.golden`
- Modify: `internal/goldentest/testdata/spiral-svg.golden`
- Modify: `internal/goldentest/testdata/spiral-surface-distinct-svg.golden`
- Modify: `internal/goldentest/testdata/spiral-surface-shared-svg.golden`
- Modify: `internal/goldentest/testdata/treemap-svg.golden`

- [ ] **Step 1: Regenerate golden fixtures**

Run:

```bash
task update-golden-files
```

Expected: PASS. Goldie rewrites SVG snapshots with three-decimal values.

- [ ] **Step 2: Verify only intended golden files changed**

Run:

```bash
git status --short
git --no-pager diff --stat -- internal/goldentest/testdata
git --no-pager diff --word-diff=porcelain -- internal/goldentest/testdata/*-svg.golden
```

Expected: the seven `*-svg.golden` files listed above change only in serialized
floating-point values; no `*-png.golden` file changes.

- [ ] **Step 3: Run the golden test package**

Run:

```bash
go test ./internal/goldentest -count=1
```

Expected: PASS.

- [ ] **Step 4: Commit the regenerated SVG snapshots**

```bash
git add internal/goldentest/testdata/*-svg.golden
git commit -m "test(svg): refresh precision snapshots"
```

### Task 3: Verify the complete repository

**Files:**
- No file changes expected.

- [ ] **Step 1: Run repository CI through an Explore subagent**

Dispatch an Explore subagent with this exact request:

```text
Run `task ci` in the current worktree. Return only the exit status; the count
and identity of failing linters or tests; each offending file:line and message;
or one line saying there were no issues. Do not strip golangci-lint verbose
output before analyzing it.
```

Expected: exit status 0 with no failing tests or linters and no uncommitted
changes produced by formatting, module tidying, or lint fixes.

- [ ] **Step 2: Confirm the branch is clean**

Run:

```bash
git status --short
```

Expected: no output.
