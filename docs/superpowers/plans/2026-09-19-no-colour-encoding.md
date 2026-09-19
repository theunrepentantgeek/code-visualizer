# No-colour encoding sentinel Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace implicit empty `ColourEncoding` values with an explicit shared
no-colour sentinel without changing visualization behavior.

**Architecture:** `internal/viz` remains the owner of colour-channel semantics.
It will expose a zero-valued `NoColourEncoding` sentinel; callers pass that
value whenever a channel intentionally has no resolved metric. A focused unit
test protects the sentinel's unset invariant, and the existing visualization
tests exercise the migrated call sites.

**Tech Stack:** Go 1.26, Gomega, Task.

---

### Task 1: Define and verify the no-colour sentinel

**Files:**
- Modify: `internal/viz/colour_encoding.go:8-13`
- Modify: `internal/viz/colour_encoding_test.go:19-33`

- [ ] **Step 1: Write the failing test**

Replace the empty-encoding test case with the shared sentinel:

```go
"no colour encoding": {
    encoding: NoColourEncoding,
    expected: false,
},
```

- [ ] **Step 2: Run the focused test to verify it fails**

Run:

```bash
go test ./internal/viz -run '^TestColourEncoding_IsSet_ReflectsMetricSelection$' -count=1
```

Expected: compilation fails because `NoColourEncoding` is undefined.

- [ ] **Step 3: Define the exported sentinel**

Add this declaration after the `ColourEncoding` type:

```go
// NoColourEncoding represents a colour channel with no selected metric.
var NoColourEncoding = ColourEncoding{}
```

- [ ] **Step 4: Run the focused test to verify it passes**

Run:

```bash
go test ./internal/viz -run '^TestColourEncoding_IsSet_ReflectsMetricSelection$' -count=1
```

Expected: `ok github.com/theunrepentantgeek/code-visualizer/internal/viz`.

- [ ] **Step 5: Commit the API and its test**

```bash
git add internal/viz/colour_encoding.go internal/viz/colour_encoding_test.go
git commit -m "Add no-colour encoding sentinel"
```

### Task 2: Migrate production no-colour values

**Files:**
- Modify: `internal/spiral/stages.go:33`
- Modify: `internal/stages/metrics.go:58`

- [ ] **Step 1: Replace the production empty literals**

Change each intentional empty value:

```go
p.Surface = viz.NoColourEncoding
```

and:

```go
return viz.NoColourEncoding
```

- [ ] **Step 2: Format the production changes**

Run:

```bash
gofumpt -w internal/spiral/stages.go internal/stages/metrics.go
```

Expected: the command completes without output.

- [ ] **Step 3: Run the direct package tests**

Run:

```bash
go test ./internal/spiral ./internal/stages -count=1
```

Expected: both packages report `ok`.

- [ ] **Step 4: Commit the production migration**

```bash
git add internal/spiral/stages.go internal/stages/metrics.go
git commit -m "Use no-colour encoding sentinel in stages"
```

### Task 3: Document the canonical no-colour representation

**Files:**
- Modify: `internal/viz/ABSTRACTIONS.md:10-14`

- [ ] **Step 1: Update the zero-value invariant**

Replace the first boundary bullet with:

```markdown
- `NoColourEncoding` is the canonical no-colour value. It uses the
  zero-value encoding, so `IsSet` is false; a non-empty `Metric` makes
  `IsSet` true regardless of the palette value
  ([colour_encoding.go#L15](colour_encoding.go#L15),
  [colour_encoding_test.go#L19](colour_encoding_test.go#L19)).
```

- [ ] **Step 2: Review the documentation diff**

Run:

```bash
git diff --check -- internal/viz/ABSTRACTIONS.md
```

Expected: no output and exit status 0.

- [ ] **Step 3: Commit the documentation update**

```bash
git add internal/viz/ABSTRACTIONS.md
git commit -m "Document no-colour encoding sentinel"
```

### Task 4: Migrate visualization test call sites

**Files:**
- Modify: `internal/bubbletree/inks_test.go`
- Modify: `internal/bubbletree/render_test.go`
- Modify: `internal/donuttree/inks_test.go`
- Modify: `internal/radialtree/inks_test.go`
- Modify: `internal/scatter/inks_test.go`
- Modify: `internal/scatter/render_test.go`
- Modify: `internal/spiral/inks_test.go`
- Modify: `internal/spiral/render_test.go`
- Modify: `internal/treemap/inks_test.go`
- Modify: `internal/treemap/render_test.go`

- [ ] **Step 1: Replace every test empty literal**

In each listed file, change every:

```go
viz.ColourEncoding{}
```

to:

```go
viz.NoColourEncoding
```

Do not change non-empty `viz.ColourEncoding` literals, which deliberately
select a metric and palette.

- [ ] **Step 2: Format the migrated test files**

Run:

```bash
gofumpt -w internal/bubbletree/inks_test.go internal/bubbletree/render_test.go internal/donuttree/inks_test.go internal/radialtree/inks_test.go internal/scatter/inks_test.go internal/scatter/render_test.go internal/spiral/inks_test.go internal/spiral/render_test.go internal/treemap/inks_test.go internal/treemap/render_test.go
```

Expected: the command completes without output.

- [ ] **Step 3: Verify no old empty literal remains**

Run:

```bash
grep -RInE 'viz\.ColourEncoding\{\}' --include='*.go' internal
```

Expected: no output and exit status 1.

- [ ] **Step 4: Run all affected visualization tests**

Run:

```bash
go test ./internal/bubbletree ./internal/donuttree ./internal/radialtree ./internal/scatter ./internal/spiral ./internal/treemap -count=1
```

Expected: every package reports `ok`.

- [ ] **Step 5: Commit the test migration**

```bash
git add internal/bubbletree internal/donuttree internal/radialtree internal/scatter internal/spiral internal/treemap
git commit -m "Use no-colour encoding sentinel in visualization tests"
```

### Task 5: Validate the complete migration

**Files:**
- Verify: `internal/**/*.go`

- [ ] **Step 1: Run the full Go test suite**

Run:

```bash
go test ./... -count=1
```

Expected: all packages pass.

- [ ] **Step 2: Run the repository formatter check**

Run:

```bash
task fmt
git diff --exit-code
```

Expected: both commands complete with exit status 0.

- [ ] **Step 3: Review the final diff**

Run:

```bash
git --no-pager diff --check
git --no-pager status --short
```

Expected: no whitespace errors and a clean working tree.
