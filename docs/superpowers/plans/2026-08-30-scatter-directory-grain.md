# Scatter Directory Grain Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `scatter --grain directory`, backed by `scatter.grain` configuration, so scatter plots can render the scan root and all descendant directories.

**Architecture:** Keep one scatter pipeline and generalize each point so it can reference either a file or directory. Resolve metrics at the selected grain's metric level, then use the existing file and directory metric-container and ink APIs throughout collection and rendering.

**Tech Stack:** Go 1.26.1, Kong, go.yaml.in/yaml/v3, Gomega, existing pipeline/model/metric/inks packages

---

## File Structure

| File | Responsibility |
| --- | --- |
| `cmd/codeviz/scatter_cmd.go` | Expose `--grain`, merge it into config, and validate effective grain-aware metrics. |
| `cmd/codeviz/main_test.go` | Cover CLI parsing and config precedence. |
| `internal/config/scatter.go` | Persist scatter grain and implement non-empty override semantics. |
| `internal/config/config_test.go` | Cover YAML and JSON round trips/export. |
| `internal/config/override_test.go` | Cover scatter grain override behavior. |
| `internal/scatter/state.go` | Store the resolved grain in scatter pipeline state. |
| `internal/scatter/data.go` | Represent file/directory points and collect the selected node kind. |
| `internal/scatter/layout.go` | Lay out and label grain-neutral points. |
| `internal/scatter/inks.go` | Build palettes from the selected point metric containers. |
| `internal/scatter/render.go` | Select the existing file or directory metric-value helper while drawing. |
| `internal/scatter/stages.go` | Resolve grain-aware metric levels, collect the right dataset, and log node-neutral results. |
| `internal/scatter/{layout,inks,render,stages}_test.go` | Cover directory collection, metrics, labels, colours, errors, and compatibility. |

### Task 1: Wire grain through CLI and configuration

**Files:**
- Modify: `internal/config/scatter.go`
- Modify: `internal/config/override_test.go`
- Modify: `internal/config/config_test.go`
- Modify: `cmd/codeviz/scatter_cmd.go`
- Modify: `cmd/codeviz/main_test.go`

- [ ] **Step 1: Write failing config and CLI tests**

Add these focused tests:

```go
func TestScatter_OverrideGrain_SetsWhenNonEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	s := &Scatter{}

	s.OverrideGrain("directory")

	g.Expect(*s.Grain).To(Equal("directory"))
}

func TestScatter_OverrideGrain_SkipsWhenEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	existing := "directory"
	s := &Scatter{Grain: &existing}

	s.OverrideGrain("")

	g.Expect(*s.Grain).To(Equal("directory"))
}
```

In `internal/config/config_test.go`, load `scatter:\n  grain: directory\n` from
YAML and `{"scatter":{"grain":"directory"}}` from JSON and assert
`*cfg.Scatter.Grain == "directory"`. Extend the existing scatter `ForExport`
test to assert the exported scatter block retains the grain.

In `cmd/codeviz/main_test.go`, extend `TestCLI_ParsesScatterAxisFlags`:

```go
"--grain", "directory",
```

and assert:

```go
g.Expect(cli.Scatter.Grain).To(Equal("directory"))
```

Add a config precedence test:

```go
func TestScatterCmd_CLIGrainOverridesConfig(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	cfg := config.New()
	cfg.Scatter.Grain = new("file")

	(&ScatterCmd{Grain: "directory"}).applyOverrides(cfg)

	g.Expect(*cfg.Scatter.Grain).To(Equal("directory"))
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run:

```bash
go test ./internal/config ./cmd/codeviz -run 'Scatter.*Grain|ParsesScatterAxisFlags'
```

Expected: compilation fails because `config.Scatter.Grain`,
`Scatter.OverrideGrain`, and `ScatterCmd.Grain` do not exist.

- [ ] **Step 3: Add config and command wiring**

Add to `config.Scatter`:

```go
Grain *string `yaml:"grain,omitempty" json:"grain,omitempty"`
```

Add:

```go
// OverrideGrain sets Grain to v if v is non-empty.
func (s *Scatter) OverrideGrain(v string) { overrideString(&s.Grain, v) }
```

Add to `ScatterCmd`, matching `RadialCmd` exactly:

```go
Grain string `enum:",file,directory" default:"" help:"Granularity of nodes shown: file (default) or directory."`
```

Then add to `ScatterCmd.applyOverrides`:

```go
cfg.Scatter.OverrideGrain(c.Grain)
```

- [ ] **Step 4: Run the focused tests**

Run:

```bash
go test ./internal/config ./cmd/codeviz -run 'Scatter.*Grain|ParsesScatterAxisFlags'
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/config/scatter.go internal/config/override_test.go internal/config/config_test.go cmd/codeviz/scatter_cmd.go cmd/codeviz/main_test.go
git commit -m "Add scatter grain configuration" -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 2: Make scatter metric resolution grain-aware

**Files:**
- Modify: `cmd/codeviz/scatter_cmd.go`
- Modify: `cmd/codeviz/main_test.go`
- Modify: `internal/scatter/state.go`
- Modify: `internal/scatter/stages.go`
- Modify: `internal/scatter/stages_test.go`

- [ ] **Step 1: Write failing grain-resolution tests**

Add to `internal/scatter/stages_test.go`:

```go
func TestResolveMetrics_GrainDefaultsToFile(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	common := &stages.CommonState{}
	vizState := &scatter.State{}
	cfg := &config.Scatter{
		XAxis: new("file-lines"),
		YAxis: new("file-size"),
		Size:  new("file-size"),
	}

	err := scatter.ResolveMetrics(common, vizState, cfg)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(vizState.Grain).To(Equal(viz.GrainFile))
}

func TestResolveMetrics_DirectoryGrainResolvesAggregations(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	common := &stages.CommonState{}
	vizState := &scatter.State{}
	cfg := &config.Scatter{
		Grain: new("directory"),
		XAxis: new("file-lines.sum"),
		YAxis: new("file-size.sum"),
		Size:  new("file-size.sum"),
	}

	err := scatter.ResolveMetrics(common, vizState, cfg)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(vizState.Grain).To(Equal(viz.GrainDirectory))
	g.Expect(vizState.XAxis.Kind).To(Equal(metric.Quantity))
	g.Expect(common.Requested.Expressions).To(HaveLen(2))
}

func TestResolveMetrics_DirectoryGrainRejectsUnaggregatedFileMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	cfg := &config.Scatter{
		Grain: new("directory"),
		XAxis: new("file-lines"),
		YAxis: new("file-size.sum"),
		Size:  new("file-size.sum"),
	}

	err := scatter.ResolveMetrics(&stages.CommonState{}, &scatter.State{}, cfg)

	g.Expect(err).To(MatchError(ContainSubstring("requires aggregation at directory level")))
}
```

Import `internal/viz` in the test.

Add command validation cases proving `grain: directory` accepts
`file-lines.sum`/`file-size.sum`, rejects bare `file-lines`, and rejects an
unknown config value such as `package`.

- [ ] **Step 2: Run the tests to verify they fail**

Run:

```bash
go test ./internal/scatter ./cmd/codeviz -run 'ResolveMetrics_.*Grain|ValidateConfig_.*Grain'
```

Expected: FAIL because state has no grain and resolution always targets files.

- [ ] **Step 3: Resolve metrics at the selected level**

Add to `internal/scatter/state.go`:

```go
Grain viz.Grain
```

with an import of `internal/viz`.

In `internal/scatter/stages.go`, add:

```go
func resolveGrain(cfg *config.Scatter) viz.Grain {
	if grain := stages.PtrString(cfg.Grain); grain != "" {
		return viz.Grain(grain)
	}

	return viz.GrainFile
}

func metricLevelForGrain(grain viz.Grain) metric.MetricLevel {
	if grain == viz.GrainDirectory {
		return metric.LevelDirectory
	}

	return metric.LevelFile
}
```

At the start of `ResolveMetrics`, set:

```go
x.Grain = resolveGrain(cfg)
level := metricLevelForGrain(x.Grain)
```

Change `resolveAxisSpec` to accept `level metric.MetricLevel` and call:

```go
resolved, err := provider.ResolveName(metricName, level)
```

Pass `level` for both axes. Resolve the size, fill, and border names at the same
level before assigning state so invalid directory expressions fail early.
Change `collectRequestedMetrics` to accept `level` and pass it to
`stages.ClassifyRequestedMetrics`.

In `cmd/codeviz/scatter_cmd.go`, validate `cfg.Grain` with a switch over `""`,
`"file"`, and `"directory"`. Use `provider.ResolveName` at the corresponding
level for axes, size, fill, and border, retaining the existing numeric-kind
check for size and `MetricSpec.Validate` palette checks.

- [ ] **Step 4: Run the focused tests**

Run:

```bash
go test ./internal/scatter ./cmd/codeviz -run 'ResolveMetrics|ScatterCmd_ValidateConfig'
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/codeviz/scatter_cmd.go cmd/codeviz/main_test.go internal/scatter/state.go internal/scatter/stages.go internal/scatter/stages_test.go
git commit -m "Resolve scatter metrics by grain" -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 3: Collect and lay out directory points

**Files:**
- Modify: `internal/scatter/data.go`
- Modify: `internal/scatter/layout.go`
- Modify: `internal/scatter/layout_test.go`
- Modify: `internal/scatter/stages.go`

- [ ] **Step 1: Write failing directory dataset tests**

Add:

```go
func TestCollectDataset_DirectoryGrainIncludesRootAndDescendants(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	const (
		xMetric = metric.Name("file-lines.sum")
		yMetric = metric.Name("file-size.sum")
	)
	root := &model.Directory{Name: "root"}
	child := &model.Directory{Name: "child"}
	root.Dirs = []*model.Directory{child}
	for i, dir := range []*model.Directory{root, child} {
		dir.SetQuantity(xMetric, int64(i+1))
		dir.SetQuantity(yMetric, int64((i+1)*10))
	}

	dataset := CollectDataset(
		root,
		viz.GrainDirectory,
		AxisSpec{Metric: xMetric, Kind: metric.Quantity},
		AxisSpec{Metric: yMetric, Kind: metric.Quantity},
		yMetric,
	)

	g.Expect(dataset.Points).To(HaveLen(2))
	g.Expect(dataset.Points).To(ConsistOf(
		WithTransform(func(p PointDatum) string { return p.Name() }, Equal("root")),
		WithTransform(func(p PointDatum) string { return p.Name() }, Equal("child")),
	))
}
```

Add a directory missing-value case and update existing file tests to pass
`viz.GrainFile`. Add a layout assertion that a directory point's label uses the
directory name.

- [ ] **Step 2: Run the tests to verify they fail**

Run:

```bash
go test ./internal/scatter -run 'CollectDataset|Layout.*Directory'
```

Expected: compilation fails because `CollectDataset` has no grain parameter and
`PointDatum` cannot hold a directory.

- [ ] **Step 3: Generalize point data and collection**

Change `PointDatum` to:

```go
type PointDatum struct {
	File      *model.File
	Directory *model.Directory
	X         AxisValue
	Y         AxisValue
	Size      float64
}

func (p PointDatum) Name() string {
	if p.Directory != nil {
		return p.Directory.Name
	}
	if p.File != nil {
		return p.File.Name
	}
	return ""
}

func (p PointDatum) metricContainer() *model.MetricContainer {
	if p.Directory != nil {
		return &p.Directory.MetricContainer
	}
	if p.File != nil {
		return &p.File.MetricContainer
	}
	return nil
}
```

Update axis and numeric lookup helpers to accept `*model.MetricContainer`.
Change `CollectDataset` to accept `grain viz.Grain`. For directory grain, call
`model.WalkDirectories`, create points with `Directory: dir`, and include the
root. For file grain, preserve `model.WalkFiles` and `File: file`. Route both
through one helper that performs the existing skip accounting.

In `layout.go`, retain both concrete pointers on `ScatterPoint`, and set:

```go
Label: point.Name(),
```

In `BuildInksStage`, pass `x.Grain` to `CollectDataset`.

- [ ] **Step 4: Run the focused tests**

Run:

```bash
go test ./internal/scatter -run 'CollectDataset|Dataset|Layout'
```

Expected: PASS, including all existing file layout tests.

- [ ] **Step 5: Commit**

```bash
git add internal/scatter/data.go internal/scatter/layout.go internal/scatter/layout_test.go internal/scatter/stages.go
git commit -m "Collect directory scatter points" -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 4: Render directory metrics and report grain-neutral results

**Files:**
- Modify: `internal/scatter/data.go`
- Modify: `internal/scatter/inks.go`
- Modify: `internal/scatter/inks_test.go`
- Modify: `internal/scatter/render.go`
- Modify: `internal/scatter/render_test.go`
- Modify: `internal/scatter/stages.go`
- Modify: `internal/scatter/stages_test.go`

- [ ] **Step 1: Write failing ink, render, and error tests**

Create directory points with numeric and categorical aggregated metrics. Assert
`BuildInks` produces numeric/categorical inks from directory values. Render an
SVG and assert it contains both the root and child directory labels.

Add this log-scale error test:

```go
func TestValidateLogScale_DirectoryErrorNamesNode(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := &model.Directory{Name: "empty"}
	dataset := Dataset{Points: []PointDatum{{
		Directory: dir,
		X:         AxisValue{Numeric: 0},
		Y:         AxisValue{Numeric: 1},
		Size:      1,
	}}}

	err := ValidateLogScale(
		dataset,
		AxisSpec{Metric: "file-lines.sum", Kind: metric.Quantity, Scale: Log},
		AxisSpec{Metric: "file-size.sum", Kind: metric.Quantity, Scale: Linear},
	)

	g.Expect(err).To(MatchError(ContainSubstring(`node "empty" has value 0`)))
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run:

```bash
go test ./internal/scatter -run 'Ink.*Directory|Render.*Directory|LogScale_Directory'
```

Expected: FAIL because inks and rendering still read file pointers.

- [ ] **Step 3: Make inks and rendering node-aware**

Replace `Dataset.Files` with:

```go
func (d Dataset) metricContainers() []*model.MetricContainer {
	values := make([]*model.MetricContainer, 0, len(d.Points))
	for _, point := range d.Points {
		if container := point.metricContainer(); container != nil {
			values = append(values, container)
		}
	}
	return values
}
```

Change scatter ink helpers to accept `[]*model.MetricContainer`; read
quantities, measures, and classifications directly from each container. This
keeps palette construction shared between grains.

In `render.go`, add:

```go
func metricValueForPoint(point ScatterPoint, ink inks.Ink) inks.MetricValue {
	if point.Directory != nil {
		return inks.MetricValueForDirectory(point.Directory, ink)
	}
	return inks.MetricValueForFile(point.File, ink)
}
```

Use it for fill and border values. Change log validation's error noun from
`file` to `node` and use `point.Name()`.

Update `LogResult` to include:

```go
"nodes", len(x.Dataset.Points),
"grain", string(x.Grain),
```

and remove the misleading `"files"` field.

- [ ] **Step 4: Run all scatter tests**

Run:

```bash
go test ./internal/scatter
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/scatter/data.go internal/scatter/inks.go internal/scatter/inks_test.go internal/scatter/render.go internal/scatter/render_test.go internal/scatter/stages.go internal/scatter/stages_test.go
git commit -m "Render directory scatter points" -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 5: Verify end-to-end behavior

**Files:**
- Modify only if verification exposes a defect in the files listed above.

- [ ] **Step 1: Format changed Go files**

Run:

```bash
task fmt
```

Expected: exit status 0.

- [ ] **Step 2: Run targeted package tests**

Run:

```bash
go test ./internal/config ./internal/scatter ./cmd/codeviz
```

Expected: PASS.

- [ ] **Step 3: Run repository CI**

Run `task ci` through an Explore/equivalent subagent, preserving verbose lint
output inside the subagent and returning only exit status and actionable
failures.

Expected: build, tests, and lint all pass.

- [ ] **Step 4: Smoke-test directory scatter output**

Run:

```bash
task build
./bin/codeviz scatter . -o /tmp/codeviz-scatter-directories.svg \
  --grain directory \
  --x-axis file-lines.sum \
  --y-axis file-size.sum \
  --size file-size.sum
test -s /tmp/codeviz-scatter-directories.svg
```

Expected: all commands exit 0 and the SVG is non-empty.

- [ ] **Step 5: Commit any formatting-only changes**

If `task fmt` changed tracked files:

```bash
git add cmd/codeviz internal/config internal/scatter
git commit -m "Format scatter directory grain changes" -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

If no files changed, do not create an empty commit.
