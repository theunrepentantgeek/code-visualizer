# Promote Visualization Commands to Top Level — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Drop the `render` subcommand group, promote the five visualization commands to the top level, rename `radial` to `radial-tree` everywhere (CLI, YAML, samples), and fix the broken `samples` Taskfile target.

**Architecture:** Pure rename / re-mount — no behaviour changes. The `*Cmd` types in `cmd/codeviz/` (`TreemapCmd`, `RadialCmd`, `BubbletreeCmd`, `SpiralCmd`, `ScatterCmd`) and their `Run(flags *Flags) error` methods stay intact; only the kong mount point changes. The `radial` → `radial-tree` rename propagates through three coordinated spots: the kong `name:` tag, the `VizName` string passed to the pipeline (which keys `Config.ForExport`), and the matching YAML/JSON struct tag plus the `ForExport` switch case.

**Tech Stack:** Go 1.26, kong (CLI), Gomega (assertions), Goldie v2 (snapshots), Taskfile.

**Spec:** [`docs/superpowers/specs/2026-06-28-promote-viz-commands-design.md`](../specs/2026-06-28-promote-viz-commands-design.md)

---

## File Map

| File | Action |
| --- | --- |
| `cmd/codeviz/render_cmd.go` | **Delete** |
| `cmd/codeviz/main.go` | Modify — flatten `CLI` struct |
| `cmd/codeviz/main_test.go` | Modify — drop `"render"` from args, rename `cli.Render.*` accesses, swap `"radial"` for `"radial-tree"` |
| `cmd/codeviz/render_matrix_test.go` | Modify — drop `"render"` from exec args |
| `cmd/codeviz/radialtree_cmd.go` | Modify — `VizName: "radial"` → `"radial-tree"` |
| `internal/config/config.go` | Modify — struct tag + `ForExport` switch case |
| `internal/config/config_test.go` | Modify — one `ForExport("radial")` → `ForExport("radial-tree")` |
| `samples/codeviz-radial.yml` | **Rename** → `samples/codeviz-radial-tree.yml`, key inside renamed |
| `samples/codeviz-{tree-map,bubble-tree,spiral,scatter}.yml` | Modify — `radial:` block key → `radial-tree:` |
| `Taskfile.yml` | Modify — `samples` task matrix and cmd |
| `docs/usage.md` | Modify — synopsis, headings, all examples |

---

## Task 1: Flatten the CLI struct

Mount each visualization as a top-level kong command and delete the `RenderCmd` group. This task must be atomic — partial flattening leaves the build broken — so the tests are updated in the same commit.

**Files:**
- Delete: `cmd/codeviz/render_cmd.go`
- Modify: `cmd/codeviz/main.go` (the `CLI` struct, ~lines 18–30)
- Modify: `cmd/codeviz/main_test.go` (every site using `"render"` or `cli.Render.*`)
- Modify: `cmd/codeviz/render_matrix_test.go` (exec args)

- [ ] **Step 1: Replace the `Render` field in `CLI` with five top-level mounts**

In `cmd/codeviz/main.go`, change the `CLI` struct from:

```go
type CLI struct {
	Quiet   bool   `help:"Suppress all non-essential output; only warnings and errors are shown." short:"q" xor:"verbosity"` //nolint:revive,nolintlint // kong struct tags require long lines
	Verbose bool   `help:"Show detailed progress during scanning and metric calculation." short:"v" xor:"verbosity"`
	Debug   bool   `help:"Show per-directory scan progress (implies verbose output)." xor:"verbosity"`
	Config  string `help:"Path to configuration file (.yaml, .yml, or .json)." name:"config" optional:""`

	//nolint:revive,nolintlint // Long help text is more important than minimizing line length, and annotations can't be wrapped
	ExportConfig string `help:"Write effective configuration to file (.yaml, .yml, or .json)." name:"export-config" optional:""`
	ExportData   string `help:"Write computed metrics to file (.json or .yaml/.yml)." name:"export-data" optional:""`

	Render RenderCmd `cmd:"" help:"Render a visualization."`
	Run    RunCmd    `cmd:"" help:"Run a preset visualization."`
	Help   HelpCmd   `cmd:"" help:"Show this help message."`
}
```

to:

```go
type CLI struct {
	Quiet   bool   `help:"Suppress all non-essential output; only warnings and errors are shown." short:"q" xor:"verbosity"` //nolint:revive,nolintlint // kong struct tags require long lines
	Verbose bool   `help:"Show detailed progress during scanning and metric calculation." short:"v" xor:"verbosity"`
	Debug   bool   `help:"Show per-directory scan progress (implies verbose output)." xor:"verbosity"`
	Config  string `help:"Path to configuration file (.yaml, .yml, or .json)." name:"config" optional:""`

	//nolint:revive,nolintlint // Long help text is more important than minimizing line length, and annotations can't be wrapped
	ExportConfig string `help:"Write effective configuration to file (.yaml, .yml, or .json)." name:"export-config" optional:""`
	ExportData   string `help:"Write computed metrics to file (.json or .yaml/.yml)." name:"export-data" optional:""`

	TreeMap    TreemapCmd    `cmd:"" name:"tree-map"    help:"Generate a tree-map visualization."`
	RadialTree RadialCmd     `cmd:"" name:"radial-tree" help:"Generate a radial tree visualization."`
	BubbleTree BubbletreeCmd `cmd:"" name:"bubble-tree" help:"Generate a bubble tree visualization."`
	Spiral     SpiralCmd     `cmd:""                    help:"Generate a spiral timeline visualization."`
	Scatter    ScatterCmd    `cmd:""                    help:"Generate a scatter plot visualization."`
	Run        RunCmd        `cmd:""                    help:"Run a preset visualization."`
	Help       HelpCmd       `cmd:""                    help:"Show this help message."`
}
```

- [ ] **Step 2: Delete `cmd/codeviz/render_cmd.go`**

Run: `rm cmd/codeviz/render_cmd.go`

- [ ] **Step 3: Update `main_test.go` args slices**

Apply these five edits to `cmd/codeviz/main_test.go`:

| Line (approx) | Before | After |
| --- | --- | --- |
| 35 | `cmd := []string{"render", "tree-map", ".", "-o", "out.png", "-s", "file-size"}` | `cmd := []string{"tree-map", ".", "-o", "out.png", "-s", "file-size"}` |
| 85 | `parser.Parse([]string{"render", "tree-map", ".", "-o", "out.png", "-s", "file-size", "--flat"})` | `parser.Parse([]string{"tree-map", ".", "-o", "out.png", "-s", "file-size", "--flat"})` |
| 101 | `args: []string{"render", "bubble-tree", ".", "-o", "out.png", "--legend", "sideways"}` | `args: []string{"bubble-tree", ".", "-o", "out.png", "--legend", "sideways"}` |
| 106 | `args: []string{"render", "bubble-tree", ".", "-o", "out.png", "--legend-orientation", "diagonal"}` | `args: []string{"bubble-tree", ".", "-o", "out.png", "--legend-orientation", "diagonal"}` |
| 235 | `"render", "tree-map", ".",` | `"tree-map", ".",` |
| 274 | `"render", "tree-map", ".",` | `"tree-map", ".",` |
| 642 | `"render", "scatter", ".",` | `"scatter", ".",` |

- [ ] **Step 4: Update `main_test.go` field accesses**

Apply these four edits to `cmd/codeviz/main_test.go`:

| Line (approx) | Before | After |
| --- | --- | --- |
| 87 | `g.Expect(cli.Render.Treemap.Flat).To(BeTrue())` | `g.Expect(cli.TreeMap.Flat).To(BeTrue())` |
| 282 | `expectRuleSliceField(g, cli.Render.Treemap, "Include", []filter.Rule{` | `expectRuleSliceField(g, cli.TreeMap, "Include", []filter.Rule{` |
| 285 | `expectRuleSliceField(g, cli.Render.Treemap, "Exclude", []filter.Rule{` | `expectRuleSliceField(g, cli.TreeMap, "Exclude", []filter.Rule{` |
| 289 | `expectRuleSlice(g, cli.Render.Treemap.Filters(), []filter.Rule{` | `expectRuleSlice(g, cli.TreeMap.Filters(), []filter.Rule{` |
| 649 | `g.Expect(cli.Render.Scatter.XAxis).To(Equal(metric.Name("file-type")))` | `g.Expect(cli.Scatter.XAxis).To(Equal(metric.Name("file-type")))` |
| 650 | `g.Expect(cli.Render.Scatter.YAxis).To(Equal(metric.Name("file-lines")))` | `g.Expect(cli.Scatter.YAxis).To(Equal(metric.Name("file-lines")))` |
| 651 | `g.Expect(cli.Render.Scatter.Size).To(Equal(metric.Name("file-size")))` | `g.Expect(cli.Scatter.Size).To(Equal(metric.Name("file-size")))` |

After this step, confirm no `cli.Render` or `"render"` literals remain in `cmd/codeviz/main_test.go`:

```bash
grep -n 'cli\.Render\|"render"' cmd/codeviz/main_test.go
```

Expected output: *(empty)*.

- [ ] **Step 5: Update `render_matrix_test.go` exec args**

Open `cmd/codeviz/render_matrix_test.go`. The test invokes the CLI via `os/exec`. Find every args slice that contains the string literal `"render"` and remove that element. There should be one or two such sites — the file uses treemap as its single representative visualization, so the pattern will look like:

```go
args := []string{"render", "tree-map", target, "-o", out, "-s", string(metricName)}
```

becomes:

```go
args := []string{"tree-map", target, "-o", out, "-s", string(metricName)}
```

Confirm afterwards:

```bash
grep -n '"render"' cmd/codeviz/render_matrix_test.go
```

Expected output: *(empty)*.

- [ ] **Step 6: Build and run tests**

```bash
task build
task test
```

Expected: build succeeds; `task test` passes. If a test fails complaining about `"radial"`, that is Task 2's territory — note it and continue; the radial-specific tests do not rely on the rename in this task because the kong command name `radial-tree` is already in place from Step 1 (so any test still passing `"radial"` will fail).

If any test args still use `"radial"` (as opposed to `"radial-tree"`), update them here too — they exist in this file only if surfaced by the test run. The expected behaviour is that no `main_test.go` site uses `"radial"` (the existing tests use `bubble-tree`, `tree-map`, and `scatter` for kong-parser tests; `"radial"` only appears in `internal/config/config_test.go`, handled in Task 2).

- [ ] **Step 7: Commit**

```bash
git add cmd/codeviz/main.go cmd/codeviz/main_test.go cmd/codeviz/render_matrix_test.go
git rm cmd/codeviz/render_cmd.go
git commit -m "refactor(cli): promote viz commands to top level"
```

---

## Task 2: Rename `radial` to `radial-tree` everywhere

Coordinated rename of the YAML/JSON config key, the `VizName` string the radial command passes to the pipeline, the matching `ForExport` switch case, the one `ForExport("radial")` test, and the `radial:` keys inside the sample YAML files. The sample file `codeviz-radial.yml` itself is renamed.

**Files:**
- Modify: `cmd/codeviz/radialtree_cmd.go` (line ~89)
- Modify: `internal/config/config.go` (struct tag ~line 41, switch case ~line 175)
- Modify: `internal/config/config_test.go` (~line 440)
- Rename: `samples/codeviz-radial.yml` → `samples/codeviz-radial-tree.yml`
- Modify: `samples/codeviz-tree-map.yml`, `samples/codeviz-bubble-tree.yml`, `samples/codeviz-spiral.yml`, `samples/codeviz-scatter.yml`, `samples/codeviz-radial-tree.yml` (the renamed file)

- [ ] **Step 1: Update `VizName` in the radial command**

In `cmd/codeviz/radialtree_cmd.go` around line 89, change:

```go
		VizName:            "radial",
```

to:

```go
		VizName:            "radial-tree",
```

- [ ] **Step 2: Update the config struct tag**

In `internal/config/config.go` around line 41, change:

```go
	Radial  *Radial  `yaml:"radial,omitempty"   json:"radial,omitempty"`
```

to:

```go
	Radial  *Radial  `yaml:"radial-tree,omitempty"   json:"radial-tree,omitempty"`
```

- [ ] **Step 3: Update the `ForExport` switch case**

In `internal/config/config.go`, in `Config.ForExport`, change:

```go
	case "radial":
		exported.Radial = c.Radial
```

to:

```go
	case "radial-tree":
		exported.Radial = c.Radial
```

- [ ] **Step 4: Update the config test**

In `internal/config/config_test.go` around line 440, change:

```go
	exported := cfg.ForExport("radial")
```

to:

```go
	exported := cfg.ForExport("radial-tree")
```

- [ ] **Step 5: Rename the sample file**

```bash
git mv samples/codeviz-radial.yml samples/codeviz-radial-tree.yml
```

- [ ] **Step 6: Update the `radial:` key inside every sample YAML**

In each of these files, replace the single line `radial:` with `radial-tree:`:

- `samples/codeviz-tree-map.yml`
- `samples/codeviz-bubble-tree.yml`
- `samples/codeviz-spiral.yml`
- `samples/codeviz-scatter.yml`
- `samples/codeviz-radial-tree.yml`

Confirm afterwards:

```bash
grep -rn '^radial:' samples/
```

Expected output: *(empty)*.

```bash
grep -rn '^radial-tree:' samples/
```

Expected output: five lines, one per sample file.

- [ ] **Step 7: Run tests**

```bash
task test
```

Expected: all tests pass.

- [ ] **Step 8: Commit**

```bash
git add cmd/codeviz/radialtree_cmd.go internal/config/config.go internal/config/config_test.go samples/
git commit -m "refactor: rename radial to radial-tree in CLI, config, samples"
```

---

## Task 3: Fix the `samples` Taskfile target

The current matrix is `[treemap, bubbletree, radial, spiral, scatter]` and tries to read `samples/codeviz-<name>.yml` — but the real files are `codeviz-tree-map.yml`, etc., so the task is already broken. Align the matrix with the actual filenames and drop the `render` prefix.

**Files:**
- Modify: `Taskfile.yml` (the `samples` task)

- [ ] **Step 1: Update the `samples` task**

In `Taskfile.yml`, find the `samples` task. Change:

```yaml
  samples:
    deps:
      - build
    cmds:
      - for:
          matrix:
            IMAGE: [png, svg]
            VIZ: [treemap, bubbletree, radial, spiral, scatter]
        cmd: "{{.CODEVIZ}} render {{.ITEM.VIZ}} . --config samples/codeviz-{{.ITEM.VIZ}}.yml --output samples/codeviz-{{.ITEM.VIZ}}.{{.ITEM.IMAGE}} ; echo ''"
    vars:
      CODEVIZ: '{{joinPath .ROOT_DIR "bin" "codeviz"}}'
```

to:

```yaml
  samples:
    deps:
      - build
    cmds:
      - for:
          matrix:
            IMAGE: [png, svg]
            VIZ: [tree-map, bubble-tree, radial-tree, spiral, scatter]
        cmd: "{{.CODEVIZ}} {{.ITEM.VIZ}} . --config samples/codeviz-{{.ITEM.VIZ}}.yml --output samples/codeviz-{{.ITEM.VIZ}}.{{.ITEM.IMAGE}} ; echo ''"
    vars:
      CODEVIZ: '{{joinPath .ROOT_DIR "bin" "codeviz"}}'
```

- [ ] **Step 2: Run the samples task end-to-end**

```bash
task samples
```

Expected: ten render invocations succeed (5 viz × {png, svg}) and the following files exist:

```bash
ls samples/codeviz-{tree-map,bubble-tree,radial-tree,spiral,scatter}.{png,svg}
```

Expected: all ten files listed.

- [ ] **Step 3: Commit**

```bash
git add Taskfile.yml samples/
git commit -m "build: align samples task with promoted viz commands"
```

(The `samples/` add picks up the regenerated PNG/SVG outputs — these are tracked in the repo if they were before; if not, the `samples/` add is a no-op for image files because none are listed in the file map. Use `git status` to confirm what is staged before committing.)

---

## Task 4: Update `docs/usage.md`

Rewrite the usage doc so synopsis, section headings, and every example use the promoted command form. The current file currently uses `render <viz>` throughout.

**Files:**
- Modify: `docs/usage.md`

- [ ] **Step 1: Update the synopsis block**

Near the top of `docs/usage.md`, change:

```markdown
## Synopsis

​```
codeviz [global flags] render <subcommand> [flags] <target-path>
​```

Subcommands: `tree-map`, `radial`, `bubble-tree`, `spiral`
```

to:

```markdown
## Synopsis

​```
codeviz [global flags] <visualization> [flags] <target-path>
​```

Visualizations: `tree-map`, `radial-tree`, `bubble-tree`, `spiral`, `scatter`
```

(Note: the original synopsis omitted `scatter` — add it.)

- [ ] **Step 2: Rewrite every per-viz section heading**

In `docs/usage.md`, rename these section headings:

| Before | After |
| --- | --- |
| `## ​render tree-map` | `## tree-map` |
| `## ​render radial` | `## radial-tree` |
| `## ​render bubble-tree` | `## bubble-tree` |
| `## ​render spiral` | `## spiral` |

(If a `## render scatter` section exists, rename to `## scatter`; if it does not exist, leave as-is for this task — the doc backfill is out of scope.)

For each renamed section, also update its `### Synopsis` block:

| Before | After |
| --- | --- |
| `codeviz render tree-map [flags] <target-path>` | `codeviz tree-map [flags] <target-path>` |
| `codeviz render radial [flags] <target-path>` | `codeviz radial-tree [flags] <target-path>` |
| `codeviz render bubble-tree [flags] <target-path>` | `codeviz bubble-tree [flags] <target-path>` |
| `codeviz render spiral [flags] <target-path>` | `codeviz spiral [flags] <target-path>` |

- [ ] **Step 3: Update every command example**

For every remaining line in `docs/usage.md` that begins with `codeviz` and contains ` render `, drop the word `render` and convert `radial` to `radial-tree` if present. A sed pass will catch them all consistently:

```bash
sed -i \
  -e 's/codeviz render tree-map/codeviz tree-map/g' \
  -e 's/codeviz render bubble-tree/codeviz bubble-tree/g' \
  -e 's/codeviz render spiral/codeviz spiral/g' \
  -e 's/codeviz render scatter/codeviz scatter/g' \
  -e 's/codeviz render radial/codeviz radial-tree/g' \
  -e 's/codeviz -v render tree-map/codeviz -v tree-map/g' \
  -e 's/codeviz --export-config \(.*\) render tree-map/codeviz --export-config \1 tree-map/g' \
  -e 's/codeviz --export-data \(.*\) render tree-map/codeviz --export-data \1 tree-map/g' \
  docs/usage.md
```

Confirm afterwards:

```bash
grep -n 'render ' docs/usage.md
```

Expected: no matches for `codeviz render`; any remaining `render`-containing lines are either prose ("rendered output", "render this") or false positives — eyeball them.

```bash
grep -n 'codeviz render\| radial ' docs/usage.md
```

Expected: *(empty)*.

- [ ] **Step 4: Commit**

```bash
git add docs/usage.md
git commit -m "docs: update usage.md for promoted viz commands"
```

---

## Task 5: Full verification

- [ ] **Step 1: Run the full CI pipeline**

```bash
task ci
```

Expected: tidy + build + test + lint + verify-no-changes all pass. (Per workspace convention, run `task lint` and `task ci` via an Explore subagent if the output is noisy; summarize exit status, failing linters/tests, and offending file:line.)

- [ ] **Step 2: Manual smoke tests**

```bash
./bin/codeviz tree-map . -o /tmp/cv-treemap.png -s file-size
./bin/codeviz radial-tree . -o /tmp/cv-radial.png -d file-size
./bin/codeviz bubble-tree . -o /tmp/cv-bubble.png -s file-lines
./bin/codeviz spiral . -o /tmp/cv-spiral.png
./bin/codeviz scatter . -o /tmp/cv-scatter.png
```

Expected: each command exits 0 and writes a non-empty file.

```bash
ls -la /tmp/cv-{treemap,radial,bubble,spiral,scatter}.png
```

Expected: five non-empty files.

- [ ] **Step 3: Confirm the old `render` form is gone**

```bash
./bin/codeviz render tree-map . -o /tmp/cv-x.png -s file-size
echo "exit=$?"
```

Expected: kong prints "codeviz: error: unexpected argument render" (or similar) on stderr and `exit=1`. The file `/tmp/cv-x.png` should not be created.

- [ ] **Step 4: Confirm the renamed sample config still drives radial-tree**

```bash
./bin/codeviz radial-tree . --config samples/codeviz-radial-tree.yml -o /tmp/cv-radial-cfg.png -d file-size
```

Expected: exit 0, file written. This validates the YAML key rename end-to-end (the command reads `radial-tree:` from the config and merges it correctly via `ForExport("radial-tree")`).

- [ ] **Step 5: Nothing to commit**

Verification only — if `task ci` produced uncommitted changes via `tidy`, those should have been folded into the relevant earlier commit. If anything remains, run `git status` and decide; otherwise the plan is complete.
