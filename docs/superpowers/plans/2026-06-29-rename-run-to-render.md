# Rename `run` to `render` for Preset Invocation — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rename the preset-runner subcommand from `codeviz run <preset>` to `codeviz render <preset>` — kong mount, Go type, file pair, test names, doc comments, and help text — with no behaviour changes.

**Architecture:** Pure rename. The `RunCmd` type (and its `Validate`, `Run`, `listPresets`, `effectiveTitle`, `runPreset`, and per-preset builder methods) becomes `RenderCmd` with the same method set and the same kong mount semantics. Kong derives the command name `render` from the new field name `Render`, replacing the prior `Run` field whose kong name was `run`.

**Tech Stack:** Go 1.26, kong (CLI), Gomega (assertions).

**Spec:** [`docs/superpowers/specs/2026-06-29-rename-run-to-render-design.md`](../specs/2026-06-29-rename-run-to-render-design.md)

---

## File Map

| File | Action |
| --- | --- |
| `cmd/codeviz/run_cmd.go` | **Rename** → `cmd/codeviz/render_cmd.go`, rename type + doc comments + help text |
| `cmd/codeviz/run_cmd_test.go` | **Rename** → `cmd/codeviz/render_cmd_test.go`, rename test funcs + literals + one parse-args site |
| `cmd/codeviz/main.go` | Modify — replace `Run RunCmd` field with `Render RenderCmd` and update help text |

---

## Task 1: Rename file pair and type, update kong mount

The rename is a single coordinated change. After this task the build is green and the CLI uses the new command name end-to-end.

**Files:**
- Rename: `cmd/codeviz/run_cmd.go` → `cmd/codeviz/render_cmd.go`
- Rename: `cmd/codeviz/run_cmd_test.go` → `cmd/codeviz/render_cmd_test.go`
- Modify: `cmd/codeviz/main.go` (the `CLI` struct, line 33)

- [ ] **Step 1: Rename the file pair via `git mv`**

```bash
git mv cmd/codeviz/run_cmd.go cmd/codeviz/render_cmd.go
git mv cmd/codeviz/run_cmd_test.go cmd/codeviz/render_cmd_test.go
```

`git status` should show both files as renames.

- [ ] **Step 2: Rename `RunCmd` → `RenderCmd` inside `render_cmd.go`**

Edit `cmd/codeviz/render_cmd.go`:

**2a. Type definition (~line 22):** change

```go
type RunCmd struct {
```

to

```go
type RenderCmd struct {
```

**2b. Method receivers — change every `RunCmd` receiver to `RenderCmd`.** There are seven such sites:

| Function | Before | After |
| --- | --- | --- |
| `Validate` (~line 97) | `func (r *RunCmd) Validate() error {` | `func (r *RenderCmd) Validate() error {` |
| `Run` (~line 120) | `func (r *RunCmd) Run(flags *Flags) error {` | `func (r *RenderCmd) Run(flags *Flags) error {` |
| `listPresets` (~line 134) | `func (*RunCmd) listPresets() error {` | `func (*RenderCmd) listPresets() error {` |
| `effectiveTitle` (~line 150) | `func (r *RunCmd) effectiveTitle(preset *presetDef) string {` | `func (r *RenderCmd) effectiveTitle(preset *presetDef) string {` |
| `runPreset` (~line 164) | `func (r *RunCmd) runPreset(preset *presetDef, flags *Flags) error {` | `func (r *RenderCmd) runPreset(preset *presetDef, flags *Flags) error {` |
| `structureTreemap` (~line 192) | `func (r *RunCmd) structureTreemap(title string) *TreemapCmd {` | `func (r *RenderCmd) structureTreemap(title string) *TreemapCmd {` |
| `structureBubbletree` (~line 205) | `func (r *RunCmd) structureBubbletree(title string) *BubbletreeCmd {` | `func (r *RenderCmd) structureBubbletree(title string) *BubbletreeCmd {` |
| `historyTreemap` (~line 218) | `func (r *RunCmd) historyTreemap(title string) *TreemapCmd {` | `func (r *RenderCmd) historyTreemap(title string) *TreemapCmd {` |
| `ageTreemap` (~line 231) | `func (r *RunCmd) ageTreemap(title string) *TreemapCmd {` | `func (r *RenderCmd) ageTreemap(title string) *TreemapCmd {` |
| `contributorsTreemap` (~line 244) | `func (r *RunCmd) contributorsTreemap(title string) *TreemapCmd {` | `func (r *RenderCmd) contributorsTreemap(title string) *TreemapCmd {` |

(Method *names* stay the same. Only the receiver type changes.)

**2c. Update the file-level doc comment block (~lines 14–21).** Replace:

```go
// RunCmd runs a named preset — a predefined combination of visualization,
// metrics, and palette that generates a useful image without requiring the
// caller to know which metrics and palettes to combine.
//
// Usage:
//
//	codeviz run                                  # list available presets
//	codeviz run <preset> <target> -o <output>    # run a preset
```

with:

```go
// RenderCmd renders a named preset — a predefined combination of
// visualization, metrics, and palette that generates a useful image
// without requiring the caller to know which metrics and palettes to
// combine.
//
// Usage:
//
//	codeviz render                                  # list available presets
//	codeviz render <preset> <target> -o <output>    # render a preset
```

**2d. Update the `Preset` field help text (~line 24).** Change:

```go
Preset     string `arg:"" optional:"" name:"preset" help:"Name of the preset to run; omit to list available presets."`
```

to:

```go
Preset     string `arg:"" optional:"" name:"preset" help:"Name of the preset to render; omit to list available presets."`
```

**2e. Confirm no `RunCmd` references remain in the file.**

```bash
grep -n 'RunCmd' cmd/codeviz/render_cmd.go
```

Expected output: *(empty)*.

- [ ] **Step 3: Update `render_cmd_test.go`**

Edit `cmd/codeviz/render_cmd_test.go`:

**3a. Rename every `RunCmd{…}` literal to `RenderCmd{…}`.** There are five such sites (from the file's `r := &RunCmd{…}` patterns in the validate tests, plus the `r := &RunCmd{…}` patterns in the effective-title tests). The simplest way to confirm completeness is to rely on the grep at the end of this step.

For each occurrence, change `RunCmd{` to `RenderCmd{` and `&RunCmd` to `&RenderCmd`.

**3b. Rename test functions.** Replace each function name as follows:

| Before | After |
| --- | --- |
| `TestRunCmd_Validate_NoArgs_ListMode` | `TestRenderCmd_Validate_NoArgs_ListMode` |
| `TestRunCmd_Validate_KnownPreset_Valid` | `TestRenderCmd_Validate_KnownPreset_Valid` |
| `TestRunCmd_Validate_UnknownPreset_ReturnsError` | `TestRenderCmd_Validate_UnknownPreset_ReturnsError` |
| `TestRunCmd_Validate_MissingTarget_ReturnsError` | `TestRenderCmd_Validate_MissingTarget_ReturnsError` |
| `TestRunCmd_Validate_MissingOutput_ReturnsError` | `TestRenderCmd_Validate_MissingOutput_ReturnsError` |
| `TestRunCmd_EffectiveTitle_UsesTitleWhenSet` | `TestRenderCmd_EffectiveTitle_UsesTitleWhenSet` |
| `TestRunCmd_EffectiveTitle_UsesPresetDefaultWhenTitleEmpty` | `TestRenderCmd_EffectiveTitle_UsesPresetDefaultWhenTitleEmpty` |
| `TestRunCmd_AllPresets_RegisteredAndUnique` | `TestRenderCmd_AllPresets_RegisteredAndUnique` |
| `TestRunCmd_ParsedFromCLI_NoArgs` | `TestRenderCmd_ParsedFromCLI_NoArgs` |

**3c. Update the one parse-args site (~line 110).** Change:

```go
ctx, parseErr := parser.Parse([]string{"run"})
```

to:

```go
ctx, parseErr := parser.Parse([]string{"render"})
```

**3d. Confirm no `RunCmd`, `TestRunCmd`, or `"run"` literals remain in the file.**

```bash
grep -n 'RunCmd\|TestRunCmd\|"run"' cmd/codeviz/render_cmd_test.go
```

Expected output: *(empty)*.

- [ ] **Step 4: Update the kong mount in `main.go`**

In `cmd/codeviz/main.go` at line 33, change:

```go
Run        RunCmd        `cmd:""                    help:"Run a preset visualization."`
```

to:

```go
Render     RenderCmd     `cmd:""                    help:"Render a preset visualization."`
```

(Kong derives the command name `render` from the field name `Render`, so no explicit `name:` tag is required.)

- [ ] **Step 5: Confirm no `RunCmd` references remain anywhere in `cmd/codeviz/`**

```bash
grep -rn 'RunCmd' cmd/codeviz/
```

Expected output: *(empty)*.

- [ ] **Step 6: Build and test**

```bash
task build
task test
```

Expected: build succeeds; `task test` passes (the renamed test functions still execute under `go test ./...` because they match the `Test…` prefix convention).

- [ ] **Step 7: Commit**

```bash
git add cmd/codeviz/main.go cmd/codeviz/render_cmd.go cmd/codeviz/render_cmd_test.go
git commit -m "refactor(cli): rename run to render for preset invocation"
```

(`git mv` from Step 1 staged the renames; the edits above are picked up automatically because the files are already in the index under their new names.)

---

## Task 2: Verification

- [ ] **Step 1: Run the full CI pipeline**

```bash
task ci
```

Expected: tidy + build + test + lint + verify-no-changes all pass. (Per workspace convention, route `task lint` / `task ci` through an Explore subagent if the verbose output is noisy.)

- [ ] **Step 2: Manual smoke tests**

```bash
./bin/codeviz render
```

Expected: prints the preset table (5 rows: `structure-tree-map`, `structure-bubble-tree`, `history-tree-map`, `age-tree-map`, `contributors-tree-map`) and exits 0.

```bash
./bin/codeviz render structure-tree-map . -o /tmp/cv-preset.png
ls -la /tmp/cv-preset.png
```

Expected: exit 0, non-empty PNG file written.

- [ ] **Step 3: Confirm the old `run` form is gone**

```bash
./bin/codeviz run
echo "exit=$?"
```

Expected: kong prints "codeviz: error: unexpected argument run" (or similar) on stderr and `exit=1`.

- [ ] **Step 4: Nothing to commit**

Verification only. If `task ci`'s `tidy` step produced changes, fold them into the Task 1 commit via `git commit --amend --no-edit` and re-run `task ci`. Otherwise the plan is complete.
