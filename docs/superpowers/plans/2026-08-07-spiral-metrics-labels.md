# Spiral Metrics Labels Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Display an upright date and distinct active metric values inside every
spiral dot, replace the legacy external labels, and show a matching circular
key in the legend.

**Architecture:** A new spiral label builder converts one aggregated time bucket
and its active metric roles into date and value lines, which are reused for dot
labels and the legend key. The existing generic legend sample gains a shape
field so treemaps retain their square sample while spirals request a circle.
The spiral renderer adds `canvas.BlockLabel` overlays after discs and before
the legend, using the existing format-aware text fitting behavior.

**Tech Stack:** Go 1.26, internal canvas and legend packages, Gomega, Goldie,
gofumpt, Task.

---

## Planned File Structure

| File | Responsibility |
| --- | --- |
| `internal/spiral/labels.go` | Build date/value label lines, deduplicate effective metric roles, and construct centered disc-label bounds. |
| `internal/spiral/labels_test.go` | Unit-test label date formatting, metric ordering, deduplication, unavailable values, and legend sample selection. |
| `internal/spiral/discsize.go` | Raise the readable minimum disc radius while preserving square-root metric scaling above it. |
| `internal/spiral/stages.go` | Build labels after bucket aggregation and discs sizing; attach the circle legend sample and canvas labels in the correct z-order. |
| `internal/spiral/render.go` | Stop rendering the external, rotated timeline labels. |
| `internal/spiral/node.go`, `internal/spiral/layout.go` | Remove obsolete external-label data and label-mode layout parameters. |
| `internal/spiral/*_test.go` | Replace legacy label-mode tests and assert all active dots have centered labels. |
| `internal/config/spiral.go`, `internal/config/config.go` | Remove the obsolete spiral label configuration field, override, and default. |
| `cmd/codeviz/spiral_cmd.go` | Remove the `--labels` command option and its config override. |
| `internal/canvas/model/legend.go` | Add the sample-shape enum and carry it through `LegendLabelSample`. |
| `internal/canvas/legendlayout/layout.go` | Keep square measurement for treemaps and size circular samples for spirals. |
| `internal/legend/config.go`, `internal/legend/render.go` | Convert the sample shape and draw a disc for circle samples instead of a rectangle. |
| `internal/{legend,canvas/legendlayout}/*_test.go` | Verify square compatibility and circle measurement/rendering. |
| `cmd/codeviz/main_test.go` | Verify `--labels` is rejected and a normal spiral run emits the date labels. |
| `internal/goldentest/viz_golden_test.go`, `internal/goldentest/testdata/spiral-*.golden` | Update spiral PNG/SVG snapshots, including both surface variants. |
| `docs/content/docs/visualizations/spiral.md`, `samples/spiral/code-visualizer.yml`, `docs/site-images/spiral.yml` | Remove the retired option and regenerate spiral imagery/configuration. |

### Task 1: Remove the legacy spiral label mode

**Files:**
- Modify: `internal/config/spiral.go`
- Modify: `internal/config/config.go`
- Modify: `cmd/codeviz/spiral_cmd.go`
- Modify: `internal/spiral/node.go`
- Modify: `internal/spiral/layout.go`
- Modify: `internal/spiral/layout_test.go`
- Modify: `internal/spiral/stages.go`
- Modify: `internal/spiral/stages_test.go`
- Modify: `internal/config/override_test.go`
- Modify: `cmd/codeviz/main_test.go`

- [ ] **Step 1: Write failing tests for the new unconditional layout contract**

  Replace `TestResolveMetrics_DefaultsLabelsToLaps`,
  `TestResolveMetrics_LabelsAllCanBeSet`, `TestLayoutLabelAll`,
  `TestLayoutLabelLaps`, `TestLayoutLabelNone`, the daily label-mode test, and
  the hourly/daily `formatBucketLabel` tests. Add these assertions instead:

  ```go
  func TestLayoutPreservesBucketTimesWithoutExternalLabels(t *testing.T) {
      t.Parallel()
      g := NewGomegaWithT(t)

      buckets := makeBuckets(3, Daily)
      layout := Layout(buckets, 1920, 1920, Daily)

      g.Expect(layout.Nodes).To(HaveLen(3))
      for index, node := range layout.Nodes {
          g.Expect(node.TimeStart).To(Equal(buckets[index].Start))
          g.Expect(node.TimeEnd).To(Equal(buckets[index].End))
      }
  }
  ```

  In `cmd/codeviz/main_test.go`, construct a Kong parser, parse
  `spiral . -o out.png --labels all`, and assert an error containing
  `unknown flag --labels`.

- [ ] **Step 2: Run the focused tests and verify failure**

  Run:

  ```sh
  go test ./internal/spiral ./internal/config ./cmd/codeviz -run 'Test(LayoutPreservesBucketTimesWithoutExternalLabels|ResolveMetrics_DefaultsLabelsToLaps|CLI_RejectsSpiralLabelsFlag)' -count=1
  ```

  Expected: the new layout call does not compile until the `labels` parameter
  is removed; the CLI test fails because the option still exists.

- [ ] **Step 3: Remove the configuration and layout-only label machinery**

  Apply these exact API changes:

  ```go
  // internal/config/spiral.go
  type Spiral struct {
      Resolution    *string     `yaml:"resolution,omitempty" json:"resolution,omitempty"`
      Size          *string     `yaml:"size,omitempty"       json:"size,omitempty"`
      Fill          *MetricSpec `yaml:"fill,omitempty"       json:"fill,omitempty"`
      Border        *MetricSpec `yaml:"border,omitempty"     json:"border,omitempty"`
      Surface       *bool       `yaml:"surface,omitempty"    json:"surface,omitempty"`
      SurfaceMetric *MetricSpec `yaml:"surfaceMetric,omitempty" json:"surfaceMetric,omitempty"`
  }
  ```

  Remove `OverrideLabels`, `SpiralCmd.Labels`, its Kong tag, and
  `cfg.Spiral.OverrideLabels(c.Labels)`. In `config.New`, initialize spiral
  with only `Resolution: new("daily")`. Remove `LabelMode`, the `viz` import,
  `Label`, and `ShowLabel` from `SpiralNode`; change `Layout` to:

  ```go
  func Layout(
      buckets []TimeBucket,
      width int,
      height int,
      resolution Resolution,
  ) SpiralLayout
  ```

  Remove `resolveLabels`, `State.Labels`, `computeLabelVisibility`, and
  `formatBucketLabel`. Update every layout caller and test to use the
  four-argument signature. Delete the spiral-specific config override test.

- [ ] **Step 4: Run the focused tests and verify they pass**

  Run:

  ```sh
  go test ./internal/spiral ./internal/config ./cmd/codeviz -count=1
  ```

  Expected: PASS, with no spiral `Labels`, `LabelMode`, `ShowLabel`, or
  external-label formatting references.

- [ ] **Step 5: Commit the retired option**

  ```sh
  git add internal/config/spiral.go internal/config/config.go internal/config/override_test.go \
    cmd/codeviz/spiral_cmd.go cmd/codeviz/main_test.go internal/spiral/node.go \
    internal/spiral/layout.go internal/spiral/layout_test.go internal/spiral/state.go \
    internal/spiral/stages.go internal/spiral/stages_test.go
  git commit -m "refactor(spiral): retire external label mode"
  ```

### Task 2: Add shared circle support for legend label samples

**Files:**
- Modify: `internal/canvas/model/legend.go`
- Modify: `internal/legend/config.go`
- Modify: `internal/legend/render.go`
- Modify: `internal/canvas/legendlayout/layout.go`
- Modify: `internal/canvas/legendlayout/helpers_test.go`
- Modify: `internal/legend/config_test.go`
- Modify: `internal/legend/render_test.go`

- [ ] **Step 1: Write failing circle-sample tests**

  Add a model sample with the circle shape and assert its dimensions stay
  square. Add a legend-render test that builds:

  ```go
  cfg := &legend.Config{
      Position:    model.LegendPositionBottomRight,
      Orientation: model.LegendOrientationVertical,
      LabelSample: legend.LabelSample{
          Shape: legend.LabelSampleCircle,
          Lines: []string{"day 7", "Aug", "12", "go"},
      },
      Entries: []legend.Entry{{
          Role: legend.RoleFill, MetricName: "file-type", Ink: fillInk,
      }},
  }
  ```

  Render into `canvas/mock.Backend` and assert it records one `DrawDisc`,
  the sample text, and the later `Fill` entry. Keep the existing square test
  and assert it still records a `DrawRectangle`.

- [ ] **Step 2: Run tests and verify failure**

  Run:

  ```sh
  go test ./internal/canvas/legendlayout ./internal/legend -run 'Test(MeasureLabelSample_Circle|RenderInto_CircleLabelSample|RenderInto_LabelSample)' -count=1
  ```

  Expected: FAIL because `LabelSampleCircle` and the structured sample do not
  exist.

- [ ] **Step 3: Implement typed sample shapes without changing treemap output**

  Replace `legend.Config.LabelSample []string` with:

  ```go
  type LabelSample struct {
      Shape LabelSampleShape
      Lines []string
  }

  type LabelSampleShape string

  const (
      LabelSampleSquare LabelSampleShape = "square"
      LabelSampleCircle LabelSampleShape = "circle"
  )
  ```

  Add the equivalent `Shape` field and constants to
  `canvas/model/legend.go`. Convert `LabelSample` in `toLegendData`, defaulting
  an empty shape to `square` for compatibility. Preserve the existing
  `MeasureLabelSample` square dimensions for both shapes, but rename its
  comment to “label sample bounds.”

  In `legendBuilder.addLabelSample`, branch on `sample.Shape`:

  ```go
  if sample.Shape == model.LegendLabelSampleCircle {
      radius := min(w, h) / 2
      lb.cv.AddDisc(canvas.LayerOverlay, canvas.Disc{
          Spec: &canvas.DiscSpec{ShapeStyle: canvas.ShapeStyle{
              Fill: inks.FixedInk(palette.White),
              Border: inks.FixedInk(lb.swBorder),
              BorderWidth: 0.5,
          }},
          X: x + w/2, Y: y + h/2, Radius: radius,
      })
  } else {
      lb.addRect(x, y, w, h, palette.White, lb.swBorder, 0.5)
  }
  ```

  Keep the centered multiline text loop unchanged.

- [ ] **Step 4: Run the focused legend tests and verify they pass**

  Run:

  ```sh
  go test ./internal/canvas/legendlayout ./internal/legend -count=1
  ```

  Expected: PASS; legacy treemap samples remain square and the new circle
  sample produces a disc.

- [ ] **Step 5: Commit reusable legend circle support**

  ```sh
  git add internal/canvas/model/legend.go internal/canvas/legendlayout/layout.go \
    internal/canvas/legendlayout/helpers_test.go internal/legend/config.go \
    internal/legend/config_test.go internal/legend/render.go internal/legend/render_test.go
  git commit -m "feat(legend): support circular label samples"
  ```

### Task 3: Build spiral date-and-metric labels and retain size encoding

**Files:**
- Create: `internal/spiral/labels.go`
- Create: `internal/spiral/labels_test.go`
- Modify: `internal/spiral/discsize.go`
- Modify: `internal/spiral/discsize_test.go`
- Modify: `internal/spiral/state.go`
- Modify: `internal/spiral/stages.go`
- Modify: `internal/spiral/stages_test.go`

- [ ] **Step 1: Write failing label and radius tests**

  Add table-driven `labels_test.go` coverage using a bucket that starts
  `2026-08-07`, with size `12`, categorical fill `"go"`, numeric border `9`,
  and numeric surface `31`. Assert:

  ```go
  g.Expect(lines).To(Equal([]string{"day 7", "Aug", "12", "go", "9", "31"}))
  ```

  Add duplicate-role cases where size and fill both use `file-lines`, and fill
  and surface both use numeric `file-lines`; assert each metric value appears once,
  using first role order. Add an unavailable categorical value and assert it
  is omitted without omitting the two date lines. Add a no-explicit-size case
  and assert `commit-count` emits the bucket’s file count.

  In `discsize_test.go`, assert an active zero-value bucket receives the new
  readable minimum, the largest active bucket reaches `maxDisc`, and a smaller
  bucket remains strictly smaller when `maxDisc` exceeds the floor.

- [ ] **Step 2: Run tests and verify failure**

  Run:

  ```sh
  go test ./internal/spiral -run 'Test(BuildDiscLabel|ApplyDiscSizes)' -count=1
  ```

  Expected: FAIL because label builders and the larger radius floor are absent.

- [ ] **Step 3: Implement role-based label construction and larger bounded radii**

  In `internal/spiral/labels.go`, define:

  ```go
  type LabelMetrics struct {
      Size    metric.Name
      Fill    metric.Name
      Border  metric.Name
      Surface metric.Name
  }

  func buildDiscLabel(bucket TimeBucket, metrics LabelMetrics) []string
  func effectiveSizeMetric(name metric.Name) metric.Name
  func buildDiscLabels(layout SpiralLayout, buckets []TimeBucket, metrics LabelMetrics, fillInk inks.Ink) []canvas.BlockLabel
  ```

  `effectiveSizeMetric("")` returns the package’s `commitCountMetric`.
  `buildDiscLabel` starts with:

  ```go
  lines := []string{
      "day " + strconv.Itoa(bucket.Start.Day()),
      bucket.Start.Format("Jan"),
  }
  ```

  Then process roles in size/fill/border/surface order through a
  `seen map[metric.Name]bool`. Numeric roles append
  `strconv.FormatFloat(value, 'f', -1, 64)`; categorical fill and border
  append their non-empty bucket labels. Surface is numeric only. Do not add
  a line for empty, duplicate, or unavailable values.

  `buildDiscLabels` skips nodes with `DiscRadius <= 0`, uses a centered square
  inset inside each disc, and uses:

  ```go
  canvas.BlockLabel{
      X: n.X - n.DiscRadius + discLabelPadding,
      Y: n.Y - n.DiscRadius + discLabelPadding,
      W: 2*(n.DiscRadius-discLabelPadding),
      H: 2*(n.DiscRadius-discLabelPadding),
      Lines: buildDiscLabel(buckets[index], metrics),
      Ink: canvas.TextColourFor(fillInk.Dip(
          metricValue(bucket.FillValue, bucket.FillLabel, fillInk),
      )),
  }
  ```

  Choose `discLabelPadding = 2.0` and set the preferred
  `minDiscRadius = 12.0`. Preserve the layout-derived `maxDisc` as a hard
  non-overlap ceiling. In `ApplyDiscSizes`, compute:

  ```go
  effectiveMin := min(minDiscRadius, maxDisc)
  scaled := maxDisc * math.Sqrt(ratio)
  nodes[i].DiscRadius = max(effectiveMin, scaled)
  ```

  This preserves the existing square-root relationship and the readable floor
  where geometry permits it, while never overlapping a dense timeline to fit
  a label.

  Add `DiscLabels []canvas.BlockLabel` to `State`. In `LayoutStage`, call the
  four-argument `Layout`, apply sizes, then assign:

  ```go
  p.DiscLabels = buildDiscLabels(
      layout, p.Buckets,
      LabelMetrics{
          Size: effectiveSizeMetric(p.Size), Fill: p.FillMetric,
          Border: p.BorderMetric, Surface: p.SurfaceMetric,
      },
      p.Inks.Fill,
  )
  ```

  In `BuildLegendStage`, use `effectiveSizeMetric(p.Size)` for the builder’s
  `SizeMetric`, and assign a `legend.LabelSample` with `Shape:
  legend.LabelSampleCircle` using the first active bucket’s
  `buildDiscLabel` result. If no bucket is active, leave the sample empty.

- [ ] **Step 4: Run focused spiral unit and stage tests**

  Run:

  ```sh
  go test ./internal/spiral -count=1
  ```

  Expected: PASS; labels contain date lines plus unique values in role order,
  and non-default disc sizes retain a visible range.

- [ ] **Step 5: Commit the label model and sizing**

  ```sh
  git add internal/spiral/labels.go internal/spiral/labels_test.go \
    internal/spiral/discsize.go internal/spiral/discsize_test.go \
    internal/spiral/state.go internal/spiral/stages.go internal/spiral/stages_test.go
  git commit -m "feat(spiral): add date and metric dot labels"
  ```

### Task 4: Render labels, update documentation, and refresh snapshots

**Files:**
- Modify: `internal/spiral/render.go`
- Modify: `internal/spiral/render_test.go`
- Modify: `internal/spiral/render_internal_test.go`
- Modify: `internal/goldentest/viz_golden_test.go`
- Modify: `internal/goldentest/testdata/spiral-png.golden`
- Modify: `internal/goldentest/testdata/spiral-svg.golden`
- Modify: `internal/goldentest/testdata/spiral-surface-shared-png.golden`
- Modify: `internal/goldentest/testdata/spiral-surface-shared-svg.golden`
- Modify: `internal/goldentest/testdata/spiral-surface-distinct-png.golden`
- Modify: `internal/goldentest/testdata/spiral-surface-distinct-svg.golden`
- Modify: `docs/content/docs/visualizations/spiral.md`
- Modify: `samples/spiral/code-visualizer.yml`
- Modify: `samples/spiral/code-visualizer.png`
- Modify: `samples/spiral/code-visualizer.svg`
- Modify: `docs/site-images/spiral.yml`
- Modify: `docs/content/docs/visualizations/spiral-thumb.png`

- [ ] **Step 1: Write failing renderer tests**

  Add an internal render test that builds a two-node layout with only the
  first node active, supplies a prebuilt `canvas.BlockLabel`, renders through
  `canvas/mock.Backend`, and asserts the date strings use `AnchorMiddle` with
  `Rotation == 0`. Assert no text has non-zero rotation. Add a stage-level
  test that `RenderStage` sends disc labels to the canvas before calling
  `legend.RenderInto` by asserting date text appears before `Fill` in the
  recording backend call order.

- [ ] **Step 2: Run the focused renderer tests and verify failure**

  Run:

  ```sh
  go test ./internal/spiral -run 'Test(Render|RenderStage).*DiscLabel' -count=1
  ```

  Expected: FAIL because `RenderToCanvas` still renders the removed external
  labels and does not accept or apply `DiscLabels`.

- [ ] **Step 3: Render centered disc labels before the legend**

  Remove `labelGap`, `addLabels`, and `addLabel` from `render.go`; remove the
  `math` dependency only if it is no longer used by the guide track. Extend
  `RenderToCanvas` with:

  ```go
  discLabels []canvas.BlockLabel,
  format canvas.ImageFormat,
  ```

  After `addDiscs`, loop over `discLabels` and call:

  ```go
  cv.AddBlockLabel(canvas.LayerOverlay, label, format)
  ```

  In `RenderStage`, resolve the format before rendering:

  ```go
  format, err := canvas.FormatFromPath(c.Output)
  if err != nil {
      return eris.Wrap(err, "resolve spiral label format")
  }
  cv := RenderToCanvas(
      p.Layout, p.Buckets, c.Width, c.Height, p.Inks, triangles, surfaceInk,
      p.DiscLabels, format,
  )
  legend.RenderInto(cv, p.LegendConfig)
  ```

  Update all direct renderer tests for the two added arguments. This order
  keeps labels above discs but below the legend overlay.

- [ ] **Step 4: Remove retired documentation/configuration and regenerate artifacts**

  In `docs/content/docs/visualizations/spiral.md`, delete the `--labels` row
  and add a paragraph after the optional flags:

  ```markdown
  Every active dot includes its day and month plus the distinct values used by
  its size, fill, border, and surface encodings. The legend identifies those
  metrics and includes a circular key showing the dot label layout.
  ```

  Delete `labels: laps` from both spiral YAML files. Then run:

  ```sh
  task update-golden-files
  task samples-spiral
  task docs:samples
  ```

  Commit only the six spiral golden artifacts, spiral sample PNG/SVG, and
  `docs/content/docs/visualizations/spiral-thumb.png` produced by these
  commands; do not include unrelated generated visualization files.

- [ ] **Step 5: Run final targeted validation**

  Run:

  ```sh
  go test ./internal/spiral ./internal/legend ./internal/canvas/legendlayout ./internal/goldentest ./cmd/codeviz -count=1
  task build
  ```

  Expected: PASS, with PNG and SVG goldens showing centered dot labels and a
  circular legend key.

- [ ] **Step 6: Commit renderer, docs, and snapshots**

  ```sh
  git add internal/spiral/render.go internal/spiral/render_test.go \
    internal/spiral/render_internal_test.go internal/goldentest/viz_golden_test.go \
    internal/goldentest/testdata/spiral-*.golden \
    docs/content/docs/visualizations/spiral.md samples/spiral/code-visualizer.yml \
    samples/spiral/code-visualizer.png samples/spiral/code-visualizer.svg \
    docs/site-images/spiral.yml docs/content/docs/visualizations/spiral-thumb.png
  git commit -m "feat(spiral): render metric labels in dots"
  ```
