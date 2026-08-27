# Donut Tree Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `codeviz donut-tree` command that renders a folders-only hierarchical donut, sized, filled, and optionally bordered by aggregated metrics.

**Architecture:** A new `internal/donuttree` package will own directory-sector geometry, per-sector curved labels, effective directory metric resolution, inks, and pipeline stages. It will reuse the established command/config merge flow, providers, aggregation stage, retained canvas, legends, and PNG/SVG backends; annular sectors and curved labels are represented with existing polygon and individually rotated text primitives.

**Tech Stack:** Go 1.26, Kong CLI parsing, Gomega, Goldie v2, fogleman/gg raster output, custom SVG canvas backend.

---

## File structure

| Path | Responsibility |
| --- | --- |
| `internal/config/donuttree.go` | Persistent `donut-tree` metric settings and CLI overrides. |
| `cmd/codeviz/donuttree_cmd.go` | Kong command, effective-config validation, and common pipeline wiring. |
| `internal/donuttree/node.go` | Directory-sector layout value type. |
| `internal/donuttree/layout.go` | Recursive sector allocation and ring geometry. |
| `internal/donuttree/labels.go` | Metric label content, fitting, glyph placement, and readable orientation. |
| `internal/donuttree/inks.go` | Directory fill/border inks and explicit-border state. |
| `internal/donuttree/state.go` | Visualization pipeline state. |
| `internal/donuttree/stages.go` | Directory-metric resolution, layout, render, legend, and logging stages. |
| `internal/donuttree/pipeline.go` | Shared acquisition/render pipeline sequences. |
| `internal/donuttree/render.go` | Canvas rendering for root anchor and annular sectors. |
| `internal/donuttree/*_test.go` | Focused geometry, labels, metrics, inks, and rendering tests. |
| `internal/goldentest/viz_golden_test.go` | Donut-tree PNG/SVG golden fixture harness. |
| `cmd/codeviz/main.go` | Register the top-level `donut-tree` command. |
| `internal/config/config.go` | Add, default, and export the `donut-tree` config section. |
| `docs/content/docs/visualizations/donut-tree.md` | User-facing command documentation. |
| `docs/content/docs/visualizations/_index.md` | Visualizations landing-page card. |
| `docs/site-images/donut-tree.yml` | Documentation-thumbnail configuration. |
| `samples/donut-tree/code-visualizer.yml` | Full-size sample configuration. |
| `Taskfile.yml` | Donut-tree sample and docs-image generation wiring. |

### Task 1: Add configuration and CLI entry point

**Files:**
- Create: `internal/config/donuttree.go`
- Create: `cmd/codeviz/donuttree_cmd.go`
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`
- Modify: `internal/config/override_test.go`
- Modify: `cmd/codeviz/main.go`
- Modify: `cmd/codeviz/main_test.go`

- [ ] **Step 1: Write failing configuration and command-parser tests**

Add tests that load this YAML and assert `cfg.DonutTree` contains the three
metric specs:

```yaml
donut-tree:
  size: file-lines
  fill: file-type,categorization
  border: file-freshness,good-bad
```

Add tests that:

```go
cfg := New()
exported := cfg.ForExport("donut-tree")
g.Expect(exported.DonutTree).To(BeIdenticalTo(cfg.DonutTree))
g.Expect(exported.Treemap).To(BeNil())

donut := &DonutTree{}
donut.OverrideSize("file-size")
donut.OverrideFill(MetricSpec{Metric: "file-type"})
donut.OverrideBorder(MetricSpec{Metric: "file-age"})
g.Expect(*donut.Size).To(Equal("file-size"))
g.Expect(donut.Fill.Metric).To(Equal(metric.Name("file-type")))
g.Expect(donut.Border.Metric).To(Equal(metric.Name("file-age")))
```

In `cmd/codeviz/main_test.go`, parse:

```go
[]string{
    "donut-tree", ".", "-o", "out.png",
    "--size", "file-size",
    "--fill", "file-type,categorization",
    "--border", "file-freshness,good-bad",
}
```

and assert the parsed `cli.DonutTree` fields are `file-size`, the
categorization fill spec, and the good-bad border spec. Add validation tests
that a missing size and a classification size return the existing numeric
metric validation error, while an omitted fill and border are accepted.

- [ ] **Step 2: Run the focused tests to verify they fail**

Run:

```bash
go test ./internal/config ./cmd/codeviz -run 'Test(Load_YAMLDonutTree|ForExport_DonutTree|DonutTree_Override|CLI_ParsesDonutTree|DonutTreeCmd_Validate)' -count=1
```

Expected: compilation failure because `DonutTree`, `DonutTreeCmd`, and the
`donut-tree` command do not exist.

- [ ] **Step 3: Add the config model**

Create `internal/config/donuttree.go`:

```go
package config

// DonutTree holds persistent configuration for donut tree visualizations.
// Nil fields were not configured; non-nil fields were set by a file or CLI override.
type DonutTree struct {
    Size   *string     `yaml:"size,omitempty"   json:"size,omitempty"`
    Fill   *MetricSpec `yaml:"fill,omitempty"   json:"fill,omitempty"`
    Border *MetricSpec `yaml:"border,omitempty" json:"border,omitempty"`
}

func (d *DonutTree) OverrideSize(v string)       { overrideString(&d.Size, v) }
func (d *DonutTree) OverrideFill(v MetricSpec)   { overrideMetricSpec(&d.Fill, v) }
func (d *DonutTree) OverrideBorder(v MetricSpec) { overrideMetricSpec(&d.Border, v) }
```

In `internal/config/config.go`, add:

```go
//nolint:tagliatelle // kebab-case is the user-facing config key
DonutTree *DonutTree `yaml:"donut-tree,omitempty" json:"donut-tree,omitempty"`
```

Initialize it in `New()` with `DonutTree: &DonutTree{}`, and add:

```go
case "donut-tree":
    exported.DonutTree = c.DonutTree
```

to `ForExport`.

- [ ] **Step 4: Add the command and top-level registration**

Create `cmd/codeviz/donuttree_cmd.go` by following the `TreemapCmd` structure,
but use `DonutTreeCmd` and these visualization-specific flags:

```go
Size metric.Name `default:"" help:"Metric for folder sector size; run 'codeviz help metrics' for available metrics." short:"s"`
Fill config.MetricSpec `help:"Folder fill colour: metric[,palette] (defaults to size)." optional:"" short:"f"`
Border config.MetricSpec `help:"Folder border colour: metric[,palette]." optional:"" short:"b"`
```

Retain the existing legend, dimensions, title/footer, include/exclude, and
binary-file fields from `TreemapCmd`. Implement:

```go
func (c *DonutTreeCmd) validateConfig(cfg *config.DonutTree) error {
    if err := validateNumericMetric("size", metric.Name(ptrString(cfg.Size))); err != nil {
        return err
    }
    if err := cfg.Fill.Validate("fill"); err != nil {
        return eris.Wrap(err, "invalid fill spec")
    }
    if err := cfg.Border.Validate("border"); err != nil {
        return eris.Wrap(err, "invalid border spec")
    }
    return nil
}
```

`mergeConfigAndValidate` must auto-load config, apply CLI overrides, and
validate `flags.Config.DonutTree`. `applyOverrides` must call the shared common
overrides, initialize a nil `cfg.DonutTree`, and call its three override
methods. `Run` must construct a `stages.CommonState` whose `VizName` is
`"donut-tree"`, then call the future `donuttree.ResolveMetrics`,
`donuttree.AcquireData`, and `donuttree.RenderPipeline`, returning
`eris.Wrap(s.Err(), "donut-tree pipeline failed")`.

In `cmd/codeviz/main.go`, import the new command through the same package and
register:

```go
DonutTree DonutTreeCmd `cmd:"" name:"donut-tree" help:"Generate a hierarchical donut visualization."`
```

- [ ] **Step 5: Run focused tests and format**

Run:

```bash
gofumpt -w internal/config/donuttree.go internal/config/config.go internal/config/config_test.go internal/config/override_test.go cmd/codeviz/donuttree_cmd.go cmd/codeviz/main.go cmd/codeviz/main_test.go
go test ./internal/config ./cmd/codeviz -run 'Test(Load_YAMLDonutTree|ForExport_DonutTree|DonutTree_Override|CLI_ParsesDonutTree|DonutTreeCmd_Validate)' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/config/donuttree.go internal/config/config.go internal/config/config_test.go internal/config/override_test.go cmd/codeviz/donuttree_cmd.go cmd/codeviz/main.go cmd/codeviz/main_test.go
git commit -m "feat: add donut tree command configuration"
```

### Task 2: Build deterministic folder-sector layout

**Files:**
- Create: `internal/donuttree/node.go`
- Create: `internal/donuttree/layout.go`
- Create: `internal/donuttree/layout_test.go`
- Create: `internal/donuttree/main_test.go`

- [ ] **Step 1: Write failing layout tests**

Register filesystem metrics in `TestMain`, then construct a root with direct
children `small` (10 lines), `large` (30 lines), and a nested child below
`large`. Assert:

```go
layout := Layout(root, 400, filesystem.FileLines)
g.Expect(layout.RootName).To(Equal("root"))
g.Expect(layout.Children).To(HaveLen(2))
g.Expect(layout.Children[0].Depth).To(Equal(1))
g.Expect(layout.Children[1].Depth).To(Equal(1))
g.Expect(layout.Children[0].SweepAngle).To(BeNumerically("<", layout.Children[1].SweepAngle))
g.Expect(layout.Children[1].Children[0].StartAngle).To(BeNumerically(">=", layout.Children[1].StartAngle))
g.Expect(layout.Children[1].Children[0].EndAngle()).To(BeNumerically("<=", layout.Children[1].EndAngle()))
g.Expect(layout.Children[0].InnerRadius).To(Equal(layout.Children[1].InnerRadius))
g.Expect(layout.Children[1].Children[0].InnerRadius).To(BeNumerically(">", layout.Children[1].InnerRadius))
```

Add cases for an empty root, all-zero children, and a positive child beside
zero children. Assert every sector has positive sweep, zero-only siblings split
their parent sweep equally, and all child sweeps exactly cover their parent's
sweep without gaps or overlap.

- [ ] **Step 2: Run the layout test to verify it fails**

Run:

```bash
go test ./internal/donuttree -run TestLayout -count=1
```

Expected: package does not exist.

- [ ] **Step 3: Define sector values and layout algorithm**

Create `node.go` with:

```go
type DonutNode struct {
    Directory *model.Directory
    Depth int
    StartAngle float64
    SweepAngle float64
    InnerRadius float64
    OuterRadius float64
    Children []DonutNode
}

func (n DonutNode) EndAngle() float64 { return n.StartAngle + n.SweepAngle }

type LayoutResult struct {
    RootName string
    Center canvasmodel.Position
    AnchorRadius float64
    Children []DonutNode
}
```

Use a `Layout(root, canvasSize, sizeMetric)` function in `layout.go`. It
returns an empty `Children` slice for an empty root, places the root anchor at
the drawing-area center, and begins direct children at `-math.Pi/2` with a
`2*math.Pi` sweep. Compute `maxDepth` over directories, reserve an anchor
radius, then divide the remaining radius into one uniform-width band per
directory depth.

For every parent, allocate child sweeps with this exact policy:

```go
minimum := min(math.Pi/180, parentSweep/float64(len(children)))
remaining := parentSweep - minimum*float64(len(children))

if sumPositive == 0 {
    childSweep = parentSweep / float64(len(children))
} else {
    childSweep = minimum
    if value > 0 {
        childSweep += remaining * value / sumPositive
    }
}
```

The `value` comes from the directory's selected quantity or measure metric;
missing or non-positive values are zero. Recurse using the child sector's exact
start and sweep, incrementing depth and using the next ring's radii. Keep
directory iteration order unchanged for deterministic images and tests.

- [ ] **Step 4: Run focused layout tests and format**

Run:

```bash
gofumpt -w internal/donuttree
go test ./internal/donuttree -run TestLayout -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/donuttree/node.go internal/donuttree/layout.go internal/donuttree/layout_test.go internal/donuttree/main_test.go
git commit -m "feat: lay out hierarchical donut sectors"
```

### Task 3: Resolve directory metrics and integrate the pipeline

**Files:**
- Create: `internal/donuttree/state.go`
- Create: `internal/donuttree/inks.go`
- Create: `internal/donuttree/stages.go`
- Create: `internal/donuttree/pipeline.go`
- Create: `internal/donuttree/stages_test.go`
- Create: `internal/donuttree/inks_test.go`
- Modify: `cmd/codeviz/donuttree_cmd.go`

- [ ] **Step 1: Write failing metric-resolution and ink tests**

Test a config with `Size: "file-lines"` and no fill/border. After
`ResolveMetrics`, assert:

```go
g.Expect(state.SizeMetric).To(Equal(metric.Name("file-lines.sum")))
g.Expect(state.FillMetric).To(Equal(metric.Name("file-lines.sum")))
g.Expect(state.BorderMetric).To(BeEmpty())
```

For `Fill: file-type,categorization` and
`Border: file-freshness,good-bad`, assert effective names are
`file-type.mode` and `file-freshness.mean`, their palettes are retained, and
`CommonState.Requested` contains the effective aggregate expressions. Test an
already aggregated metric such as `file-size.sum` remains unchanged.

Build inks for an aggregated test directory and assert a missing border gives
`HasBorderMetric == false`; an explicit border gives `HasBorderMetric == true`
and a non-fixed border ink. Assert the fill uses
`inks.BuildDirectoryMetricInk`, so classifications produce a categorical ink.

- [ ] **Step 2: Run metric tests to verify they fail**

Run:

```bash
go test ./internal/donuttree -run 'Test(ResolveMetrics|BuildDonutInks)' -count=1
```

Expected: compilation failure because state, stages, and inks do not exist.

- [ ] **Step 3: Implement effective-directory metric resolution**

Create `State` with `SizeMetric`, `FillMetric`, `FillPalette`, `BorderMetric`,
`BorderPalette`, `Inks`, `Layout`, and `LegendConfig` fields. In `stages.go`,
adapt radial-tree's aggregation-aware fallback to a helper:

```go
func resolveDirectoryMetric(base metric.Name) metric.Name {
    expression, err := metric.ParseExpression(string(base))
    if err != nil || expression.Aggregation != "" {
        return base
    }
    descriptor, ok := provider.GetBase(expression.Base)
    if !ok {
        return ""
    }
    aggregation := aggregationForKind(descriptor.Kind) // sum, mean, or mode
    expression.Aggregation = aggregation
    if _, err := provider.ResolveExpression(expression, metric.LevelDirectory); err != nil {
        return ""
    }
    return expression.ResultName()
}
```

Resolve size first. Resolve fill from the configured fill metric when supplied,
otherwise from the configured size metric; pass both bases through
`resolveDirectoryMetric`. Resolve border only when configured. Use an
`effectiveMetricSpec` helper that returns a spec containing the effective
metric and the configured palette so `stages.CollectRequestedMetrics` requests
the directory expressions rather than their unaggregated inputs.

Create `BuildInks` using `inks.BuildDirectoryMetricInk`. Include
`HasBorderMetric bool` and only build a metric border when `borderMetric != ""`.
Use a fixed neutral fill only as the fallback passed to fill-ink construction;
the renderer, not the ink, controls whether a stroke is drawn.

- [ ] **Step 4: Implement pipeline and stages**

Mirror `radialtree.AcquireData` and `RenderPipeline`. The render pipeline must
run aggregation and binary filtering before metric-dependent work, then:

```go
pipeline.ApplyFuncXY(s, BuildInksStage)
pipeline.ApplyFuncXY(s, BuildLegendStage)
pipeline.ApplyFuncXY(s, LayoutStage)
pipeline.ApplyFuncXY(s, RenderStage)
```

before the standard title, footer, canvas-write, and result-log stages.

`BuildLegendStage` must supply the effective directory fill, border, and size
metrics to `legend.Builder`. `LayoutStage` must use the smaller drawing-bound
dimension, as radial-tree does. `RenderStage` will invoke the renderer created
in Task 4. `LogResult` must log `"Rendered donut tree"` plus file/directory
counts, output path, canvas size, and effective size/fill/border metrics.

Wire `DonutTreeCmd.Run` to create `donuttree.State`, create the pipeline with
`cfg.DonutTree`, resolve metrics, acquire data, and render.

- [ ] **Step 5: Run focused package and command tests**

Run:

```bash
gofumpt -w internal/donuttree cmd/codeviz/donuttree_cmd.go
go test ./internal/donuttree ./cmd/codeviz -run 'Test(ResolveMetrics|BuildDonutInks|DonutTreeCmd)' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/donuttree cmd/codeviz/donuttree_cmd.go
git commit -m "feat: resolve donut tree directory metrics"
```

### Task 4: Render sectors, root anchor, and curved metric labels

**Files:**
- Create: `internal/donuttree/render.go`
- Create: `internal/donuttree/render_test.go`
- Create: `internal/donuttree/labels.go`
- Create: `internal/donuttree/labels_test.go`

- [ ] **Step 1: Write failing rendering and label tests**

With a two-level test tree and fixed directory inks, render to
`canvas/mock.Backend`. Assert the number of `DrawPolygon` calls equals the
number of folder sectors, and no `DrawDisc` call corresponds to a file. Assert
a no-border ink produces polygon calls with `BorderWidth == 0`; an explicit
border uses the visualization border width.

For a large sector, assert generated glyph `DrawText` calls concatenate to:

```text
src | file-lines.sum: 120 | file-type.mode: go | file-freshness.mean: 0.5
```

when fill and border are explicit. For default fill and no border, assert only:

```text
src | file-lines.sum: 120
```

Assert every glyph sits on the label's midpoint radius; its angles are centered
around the sector midpoint; and glyphs in a lower-half sector are inverted by
`math.Pi` so text is readable. Add cases where fitting selects a font below the
normal 14 px but at least 6 px, and where fitting would require less than 6 px
and emits no glyphs.

- [ ] **Step 2: Run rendering tests to verify they fail**

Run:

```bash
go test ./internal/donuttree -run 'Test(Render|Label)' -count=1
```

Expected: compilation failure because rendering and label helpers do not exist.

- [ ] **Step 3: Render annular sectors from polygons**

Create `RenderToCanvas(layout, root, width, height, inks)` to add a white
background, a root `canvas.Disc` anchor, its centered root-name `canvas.Text`,
then recurse through every `DonutNode`.

Implement `sectorPoints(node, center)` by sampling both curved boundaries:

```go
steps := max(2, int(math.Ceil(node.SweepAngle/(2*math.Pi)*64)))
// append outer arc from StartAngle through EndAngle
// append inner arc from EndAngle back through StartAngle
```

Use the node's `InnerRadius` and `OuterRadius` to construct the points, with
the outer arc clockwise and inner arc reversed so `canvas.Polygon` produces a
closed annular sector. Apply:

```go
borderWidth := 0.0
if is.HasBorderMetric {
    borderWidth = 1.0
}
```

and use `inks.MetricValueForDirectory(node.Directory, is.Fill)` and the border
equivalent on each polygon. The root anchor uses a fixed neutral fill/border;
it is the only central disc and represents no file metric.

- [ ] **Step 4: Implement label content, fitting, and glyph placement**

In `labels.go`, define:

```go
type LabelMetrics struct {
    Size metric.Name
    Fill metric.Name
    Border metric.Name
    IncludeFill bool
    IncludeBorder bool
}
```

Build text as the folder name followed by `metric-name: value` components,
joined with `" | "`. Format quantities with `strconv.FormatInt`, measures with
`strconv.FormatFloat(value, 'f', -1, 64)`, and classifications verbatim.
Include fill and border only when their `Include...` fields are true, preserving
the distinction between an implicit fill default and explicitly requested fill.

Use `textlayout.MeasureString(text, 14)` to derive a proportional font size
from `availableArcLength := (innerRadius+outerRadius)/2 * sweepAngle`. Clamp it
to 14 px maximum. Return no label when the computed size is below 6 px.

For an accepted label, use `utf8.RuneCountInString` and per-rune
`textlayout.MeasureStrings` widths to position individual `canvas.Text` glyphs
around the sector midpoint radius. Center total glyph advance at
`StartAngle + SweepAngle/2`. Use tangential rotation
`angle + math.Pi/2`; when the midpoint is in the lower half of the circle, add
`math.Pi` and reverse glyph order so the label remains upright. Add the glyphs
to `canvas.LayerOverlay` using a single fixed, contrast-safe label ink.

- [ ] **Step 5: Run rendering and label tests**

Run:

```bash
gofumpt -w internal/donuttree
go test ./internal/donuttree -run 'Test(Render|Label)' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/donuttree/render.go internal/donuttree/render_test.go internal/donuttree/labels.go internal/donuttree/labels_test.go
git commit -m "feat: render donut tree sectors and labels"
```

### Task 5: Add end-to-end golden coverage

**Files:**
- Modify: `internal/goldentest/viz_golden_test.go`
- Create: `internal/goldentest/testdata/donut-tree-png.golden`
- Create: `internal/goldentest/testdata/donut-tree-svg.golden`
- Create: `internal/donuttree/render_internal_test.go`

- [ ] **Step 1: Write failing pipeline and format tests**

Add a donut-tree golden renderer modeled on `renderRadial`:

```go
func renderDonutTree(common *stages.CommonState) error {
    size := "file-lines"
    common.RootConfig.DonutTree = &config.DonutTree{
        Size: &size,
        Fill: &config.MetricSpec{Metric: "file-type"},
    }
    viz := &donuttree.State{}
    s := pipeline.NewState(common, common.RootConfig.DonutTree, viz)
    pipeline.ApplyFuncX(s, stages.BuildFilterRules)
    pipeline.ApplyFuncX(s, stages.RegisterSelectionMetrics)
    pipeline.ApplyFuncXYZ(s, donuttree.ResolveMetrics)
    donuttree.RenderPipeline(s)
    return eris.Wrap(s.Err(), "donut tree render failed")
}
```

Register `TestGolden_DonutTree` with `runVizGolden`. Add internal tests that
render one nested fixture to a temporary PNG and SVG, decode/parse the result,
and assert non-empty output. Add a stage test that title/footer-reserved,
non-square output retains its configured dimensions.

- [ ] **Step 2: Run the new tests to verify they fail**

Run:

```bash
go test ./internal/donuttree ./internal/goldentest -run 'Test(Golden_DonutTree|RenderDonutTree)' -count=1
```

Expected: FAIL because donut-tree Golden fixtures do not yet exist.

- [ ] **Step 3: Generate only the new golden fixtures**

Run:

```bash
GOLDIE_UPDATE=1 go test ./internal/goldentest -run TestGolden_DonutTree -count=1
```

Expected: PASS and create exactly
`donut-tree-png.golden` and `donut-tree-svg.golden`. Inspect their diff to
confirm they contain the intended nested rings, labels, and title/footer rather
than unrelated fixture changes.

- [ ] **Step 4: Run the targeted regression suite**

Run:

```bash
go test ./internal/donuttree ./internal/goldentest ./cmd/codeviz ./internal/config -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/donuttree/render_internal_test.go internal/goldentest/viz_golden_test.go internal/goldentest/testdata/donut-tree-png.golden internal/goldentest/testdata/donut-tree-svg.golden
git commit -m "test: cover donut tree rendering"
```

### Task 6: Document and generate the visualization samples

**Files:**
- Create: `docs/content/docs/visualizations/donut-tree.md`
- Modify: `docs/content/docs/visualizations/_index.md`
- Create: `docs/site-images/donut-tree.yml`
- Create: `samples/donut-tree/code-visualizer.yml`
- Modify: `Taskfile.yml`
- Create: `docs/content/docs/visualizations/donut-tree-thumb.png`
- Create: `samples/donut-tree/code-visualizer.png`
- Create: `samples/donut-tree/code-visualizer.svg`

- [ ] **Step 1: Add documentation and sample configuration**

Create `docs/content/docs/visualizations/donut-tree.md` following
`radial-tree.md`, with synopsis:

```text
codeviz donut-tree [flags] <target-path>
```

Document `--size` as required and `--fill`/`--border` with their defaults.
State explicitly that each ring is a folder depth; each sector is a folder
subtree; files contribute metrics but are never drawn; fill defaults to size;
and omitted border draws no stroke. Include a basic command and a command with
explicit fill/border metric examples.

Create both configuration files with this effective sample:

```yaml
imageSize:
  width: 1920
  height: 1920
legend:
  position: bottom-center
  orientation: horizontal
donut-tree:
  size: file-lines
  fill:
    metric: file-type
    palette: categorization
  border:
    metric: file-freshness
    palette: good-bad
fileFilter:
  - pattern: .*
    mode: exclude
```

For `docs/site-images/donut-tree.yml`, add `visible: false` below `legend` to
match the existing thumbnail configs.

- [ ] **Step 2: Wire sample tasks and docs card**

Add `donut-tree` to the `samples` dependency list. Add a `samples-donut-tree`
task identical in form to `samples-radial-tree`, invoking:

```text
{{.CODEVIZ}} donut-tree . --config samples/donut-tree/code-visualizer.yml --output samples/donut-tree/code-visualizer.png --footer '{{.FOOTER}}'
{{.CODEVIZ}} donut-tree . --config samples/donut-tree/code-visualizer.yml --output samples/donut-tree/code-visualizer.svg --footer '{{.FOOTER}}'
```

Add `donut-tree` to both docs-site visualization matrices. Add:

```markdown
{{< card link="donut-tree" image="donut-tree-thumb.png" title="Donut Tree" >}}
```

to the visualizations index.

- [ ] **Step 3: Build, generate samples, and inspect output**

Run:

```bash
task build
task samples-donut-tree
task docs:samples
```

Expected: `samples/donut-tree/code-visualizer.png`,
`samples/donut-tree/code-visualizer.svg`, and
`docs/content/docs/visualizations/donut-tree-thumb.png` exist and are
non-empty. Confirm the output contains folder rings and no individual file
sectors.

- [ ] **Step 4: Run the complete validation suite**

Run:

```bash
task test
task fmt:check
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add docs/content/docs/visualizations/donut-tree.md docs/content/docs/visualizations/_index.md docs/content/docs/visualizations/donut-tree-thumb.png docs/site-images/donut-tree.yml samples/donut-tree/code-visualizer.yml samples/donut-tree/code-visualizer.png samples/donut-tree/code-visualizer.svg Taskfile.yml
git commit -m "docs: add donut tree examples"
```

## Plan self-review

- **Spec coverage:** Tasks 1 and 3 implement the command, configuration,
  aggregation defaults, optional border, legend, and pipeline. Tasks 2 and 4
  implement root anchoring, folders-only nested rings, proportional and
  zero-value sectors, fill/border rendering, and 6 px curved labels. Tasks 5
  and 6 provide PNG/SVG coverage, docs, samples, and generated images.
- **Placeholder scan:** No incomplete task markers, deferred design decisions,
  or unspecified filenames remain.
- **Type consistency:** `DonutTree`, `DonutTreeCmd`, `DonutNode`,
  `LayoutResult`, `State`, and the metric-stage function names are introduced
  before use and remain consistent across tasks.
