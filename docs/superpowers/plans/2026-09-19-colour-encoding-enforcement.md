# ColourEncoding Enforcement Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Preserve `viz.ColourEncoding` as the metric-and-palette unit from
resolution through visualization-specific ink construction.

**Architecture:** Donut tree delegates palette selection to
`stages.ResolveColourEncoding` after its directory-specific metric
normalization. Radial tree, scatter, and spiral retain their existing
visualization-specific ink construction but change private helpers to accept a
single encoding rather than decomposed metric and palette parameters.

**Tech Stack:** Go 1.26+, Gomega, Task, gofumpt.

---

## File Structure

- `internal/donuttree/stages.go` — resolve normalized directory metrics through
  the shared colour-encoding resolver.
- `internal/donuttree/stages_test.go` — retain or add resolution coverage for
  resolved metric and palette pairs.
- `internal/radialtree/stages.go` — keep directory fill/border encodings
  coupled when building inks.
- `internal/radialtree/stages_test.go` — exercise directory-encoding
  resolution and ink-stage behavior.
- `internal/scatter/inks.go` — pass encodings to the metric-ink helper.
- `internal/scatter/inks_test.go` — preserve fill and border ink behavior.
- `internal/spiral/inks.go` — pass encodings to the bucket-ink helper.
- `internal/spiral/inks_test.go` — preserve numeric and categorical bucket ink
  behavior.
- `internal/viz/ABSTRACTIONS.md` — document the `ColourEncoding` contract.

### Task 1: Delegate Donut-Tree Colour Resolution

**Files:**
- Modify: `internal/donuttree/stages.go:37-47`
- Test: `internal/donuttree/stages_test.go`

- [ ] **Step 1: Add or confirm the resolution test**

Add a test case that resolves an explicit directory metric with an explicit
palette and asserts the resulting `State.Fill` or `State.Border` equals:

```go
viz.ColourEncoding{
    Metric:  metric.Name("file-lines.sum"),
    Palette: palette.Foliage,
}
```

- [ ] **Step 2: Run the focused test**

Run: `go test ./internal/donuttree -run 'TestResolveMetrics' -count=1`

Expected: PASS before the refactor, proving the test captures preserved
behavior.

- [ ] **Step 3: Replace literal construction with the shared resolver**

After `resolveDirectoryMetric` returns `fillMetric`, replace:

```go
d.Fill = viz.ColourEncoding{Metric: fillMetric, Palette: stages.ResolveFillPalette(cfg.Fill, fillMetric)}
```

with:

```go
d.Fill = stages.ResolveColourEncoding(cfg.Fill, fillMetric)
```

For the configured border, replace the literal with:

```go
d.Border = stages.ResolveColourEncoding(cfg.Border, borderMetric)
```

Remove the now-unused `internal/viz` import.

- [ ] **Step 4: Format and validate**

Run: `gofumpt -w internal/donuttree/stages.go internal/donuttree/stages_test.go && go test ./internal/donuttree -count=1 && task build`

Expected: every command exits 0.

- [ ] **Step 5: Commit**

```bash
git add internal/donuttree/stages.go internal/donuttree/stages_test.go
git commit -m "Use shared colour resolver in donut tree"
```

### Task 2: Keep Radial Directory Ink Inputs Coupled

**Files:**
- Modify: `internal/radialtree/stages.go:103-134`
- Test: `internal/radialtree/stages_test.go`

- [ ] **Step 1: Add or confirm a directory ink-stage test**

Use resolved `DirectoryFill` and `DirectoryBorder` encodings and assert
`BuildInksStage` creates the same fill and border `inks.Info()` kinds as
before the signature refactor.

- [ ] **Step 2: Run the focused test**

Run: `go test ./internal/radialtree -run 'TestBuildInksStage|TestResolveMetrics' -count=1`

Expected: PASS before the refactor.

- [ ] **Step 3: Change the helper boundary**

Change the call to:

```go
r.Inks.DirectoryFill, r.Inks.DirectoryBorder = buildDirectoryInks(
    c.Root, c.Requested, r.DirectoryFill, r.DirectoryBorder,
)
```

Change `buildDirectoryInks` to accept `fill, border viz.ColourEncoding`, then
use `fill.Metric`, `fill.Palette`, `border.Metric`, and `border.Palette` only
inside the helper when calling `DescriptorFor` and
`BuildDirectoryMetricInk`. Remove the no-longer-needed `palette` import and
add `internal/viz`.

- [ ] **Step 4: Format and validate**

Run: `gofumpt -w internal/radialtree/stages.go internal/radialtree/stages_test.go && go test ./internal/radialtree -count=1 && task build`

Expected: every command exits 0.

- [ ] **Step 5: Commit**

```bash
git add internal/radialtree/stages.go internal/radialtree/stages_test.go
git commit -m "Keep radial directory colour encodings coupled"
```

### Task 3: Pass Encodings Through Scatter Ink Construction

**Files:**
- Modify: `internal/scatter/inks.go:40-78`
- Test: `internal/scatter/inks_test.go`

- [ ] **Step 1: Add or confirm fill and border ink tests**

Use `viz.ColourEncoding` fixtures for one numeric fill and one categorical
border. Assert that `BuildInks` returns the same `inks.KindNumeric` and
`inks.KindCategorical` values respectively.

- [ ] **Step 2: Run the focused test**

Run: `go test ./internal/scatter -run 'TestBuildInks' -count=1`

Expected: PASS before the refactor.

- [ ] **Step 3: Change `buildMetricInk` to accept an encoding**

Replace each call that passes separate metric and palette values with:

```go
buildMetricInk(dataset.metricSources(), requested, fill, scatterDefaultFill)
```

and the analogous `border` call. Change the helper signature to accept
`encoding viz.ColourEncoding`; use `encoding.Metric` for descriptor and value
lookup and `encoding.Palette` for `palette.GetPalette`.

- [ ] **Step 4: Format and validate**

Run: `gofumpt -w internal/scatter/inks.go internal/scatter/inks_test.go && go test ./internal/scatter -count=1 && task build`

Expected: every command exits 0.

- [ ] **Step 5: Commit**

```bash
git add internal/scatter/inks.go internal/scatter/inks_test.go
git commit -m "Pass colour encodings to scatter ink builder"
```

### Task 4: Pass Encodings Through Spiral Bucket Ink Construction

**Files:**
- Modify: `internal/spiral/inks.go:35-105`
- Test: `internal/spiral/inks_test.go`

- [ ] **Step 1: Add or confirm numeric and categorical bucket-ink tests**

Use a `viz.ColourEncoding` fixture for each metric kind and assert that
`BuildInks` still produces `inks.KindNumeric` and `inks.KindCategorical`.

- [ ] **Step 2: Run the focused test**

Run: `go test ./internal/spiral -run 'TestBuildInks' -count=1`

Expected: PASS before the refactor.

- [ ] **Step 3: Change `buildBucketInk` to accept an encoding**

Change both calls to pass `fill` or `border` as the third argument. Change the
helper signature to accept `encoding viz.ColourEncoding`, use
`encoding.Metric` for `DescriptorFor`, `NumericInk`, and `CategoricalInk`, and
use `encoding.Palette` for `palette.GetPalette`.

- [ ] **Step 4: Format and validate**

Run: `gofumpt -w internal/spiral/inks.go internal/spiral/inks_test.go && go test ./internal/spiral -count=1 && task build`

Expected: every command exits 0.

- [ ] **Step 5: Commit**

```bash
git add internal/spiral/inks.go internal/spiral/inks_test.go
git commit -m "Pass colour encodings to spiral ink builder"
```

### Task 5: Document the Colour Encoding Contract

**Files:**
- Modify: `internal/viz/ABSTRACTIONS.md`

- [ ] **Step 1: Invoke the requested documentation workflow**

Run the `document-abstractions` skill from
`/home/bevan/github/niche-skills/.worktrees/abstraction-skills/skills/document-abstractions`
and follow its instructions for `internal/viz`.

- [ ] **Step 2: Document `ColourEncoding`**

Add a `ColourEncoding` section that states:

- It represents one colour channel's selected metric and palette.
- `IsSet` is true exactly when `Metric` is non-empty; the zero value selects no
  metric.
- `stages.ResolveColourEncoding` selects the fallback metric and resolves the
  palette.
- Callers pass the encoding intact instead of parallel metric and palette
  arguments; consumers that need only a metric may read `Metric`.

- [ ] **Step 3: Validate documentation**

Run: `git diff --check && git diff -- internal/viz/ABSTRACTIONS.md`

Expected: no whitespace errors and a focused documentation-only diff.

- [ ] **Step 4: Commit**

```bash
git add internal/viz/ABSTRACTIONS.md
git commit -m "Document ColourEncoding abstraction"
```
