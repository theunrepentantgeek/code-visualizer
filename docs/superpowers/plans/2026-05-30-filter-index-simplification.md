# Filter Index Simplification Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace reflection-heavy RuleMapper with index-based ordering so `--include`/`--exclude` flags preserve command-line order without pointer magic.

**Architecture:** Each `Rule` gets an unexported `index` field auto-assigned by `NewRule()` from a package-level atomic counter. A `Merge()` function sorts combined include+exclude slices by index. The RuleMapper becomes a stateless decoder that infers mode from `ctx.Value.Name`.

**Tech Stack:** Go 1.26+, Kong (CLI), `sync/atomic`, `slices` (stdlib)

---

## File Map

| Action | File | Responsibility |
|--------|------|----------------|
| Modify | `internal/filter/filter.go` | Add `index` field to Rule, update `NewRule()`, add `CompareByIndex()`, add `Merge()` |
| Modify | `internal/filter/kong.go` | Replace entire file with stateless `RuleMapper` |
| Modify | `internal/filter/filter_test.go` | Update RuleMapper test, add CompareByIndex/Merge tests |
| Modify | `cmd/codeviz/parser.go` | Simplify `filterMapperOption()` to no-arg |
| Modify | `cmd/codeviz/main.go:93` | Remove `&cli` argument from `filterMapperOption` call |
| Modify | `cmd/codeviz/main_test.go:55` | Remove `&cli` argument from `filterMapperOption` call |
| Modify | `cmd/codeviz/spiral_cmd.go:34,93` | Remove `Filters` field, add `Filters()` method, update `Run()` |
| Modify | `cmd/codeviz/treemap_cmd.go:32,102` | Remove `Filters` field, add `Filters()` method, update `Run()` |
| Modify | `cmd/codeviz/scatter_cmd.go:32,103` | Remove `Filters` field, add `Filters()` method, update `Run()` |
| Modify | `cmd/codeviz/radialtree_cmd.go:32,90` | Remove `Filters` field, add `Filters()` method, update `Run()` |
| Modify | `cmd/codeviz/bubbletree_cmd.go:33,92` | Remove `Filters` field, add `Filters()` method, update `Run()` |

---

### Task 1: Add index field and atomic counter to Rule

**Files:**
- Modify: `internal/filter/filter.go`

- [ ] **Step 1: Add the `index` field and atomic counter**

Add a package-level `atomic.Int64` and the unexported `index` field to `Rule`. Update `NewRule()` to assign the index. Add `CompareByIndex` and `Merge`.

In `internal/filter/filter.go`, add `"slices"` and `"sync/atomic"` to imports, then update:

```go
// Rule pairs a glob pattern with an include/exclude mode.
type Rule struct {
	Pattern string `yaml:"pattern" json:"pattern"`
	Mode    Mode   `yaml:"mode"   json:"mode"`
	index   int
}

var ruleCounter atomic.Int64
```

Update `NewRule()` to assign the index:

```go
// NewRule validates a glob pattern and constructs a Rule with the given mode.
func NewRule(pattern string, mode Mode) (Rule, error) {
	if pattern == "" {
		return Rule{}, eris.New("empty filter pattern after prefix")
	}

	switch mode {
	case Include, Exclude:
	default:
		return Rule{}, eris.Errorf("unknown filter mode: %d", mode)
	}

	// Validate the glob pattern
	if _, err := doublestar.Match(pattern, ""); err != nil {
		return Rule{}, eris.Wrapf(err, "invalid glob pattern %q", pattern)
	}

	return Rule{
		Pattern: pattern,
		Mode:    mode,
		index:   int(ruleCounter.Add(1)),
	}, nil
}
```

Add `CompareByIndex` and `Merge` after `NewRule`:

```go
// CompareByIndex compares two rules by their internal construction index.
// For use with slices.SortFunc to recover original command-line order.
func CompareByIndex(a, b Rule) int {
	return a.index - b.index
}

// Merge combines include and exclude rule slices, sorting by construction
// order so the result matches original command-line flag order.
func Merge(include, exclude []Rule) []Rule {
	result := make([]Rule, 0, len(include)+len(exclude))
	result = append(result, include...)
	result = append(result, exclude...)
	slices.SortFunc(result, CompareByIndex)

	return result
}
```

- [ ] **Step 2: Run existing filter tests to verify nothing breaks**

Run: `task test -- -run TestIsIncluded -v ./internal/filter/`

Expected: All existing tests pass. Tests that construct `Rule{}` literals directly will have `index: 0` — this is fine because `IsIncluded()` doesn't use the index.

- [ ] **Step 3: Commit**

```bash
git add internal/filter/filter.go
git commit -m "filter: add index field to Rule with atomic counter

Add CompareByIndex and Merge functions for recovering
command-line ordering from separate Include/Exclude slices."
```

---

### Task 2: Add tests for CompareByIndex and Merge

**Files:**
- Modify: `internal/filter/filter_test.go`

- [ ] **Step 1: Write tests for CompareByIndex and Merge**

Append to `internal/filter/filter_test.go`:

```go
func TestCompareByIndex_ReturnsNegativeForEarlierRule(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	a, err := NewRule("*.go", Include)
	g.Expect(err).NotTo(HaveOccurred())

	b, err := NewRule("*.log", Exclude)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(CompareByIndex(a, b)).To(BeNumerically("<", 0))
	g.Expect(CompareByIndex(b, a)).To(BeNumerically(">", 0))
	g.Expect(CompareByIndex(a, a)).To(Equal(0))
}

func TestMerge_PreservesConstructionOrder(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Create rules in a specific interleaved order
	excl1, err := NewRule(".*", Exclude)
	g.Expect(err).NotTo(HaveOccurred())

	incl1, err := NewRule(".github/**", Include)
	g.Expect(err).NotTo(HaveOccurred())

	excl2, err := NewRule("**/*.log", Exclude)
	g.Expect(err).NotTo(HaveOccurred())

	include := []Rule{incl1}
	exclude := []Rule{excl1, excl2}

	merged := Merge(include, exclude)

	g.Expect(merged).To(HaveLen(3))
	g.Expect(merged[0].Pattern).To(Equal(".*"))
	g.Expect(merged[0].Mode).To(Equal(Exclude))
	g.Expect(merged[1].Pattern).To(Equal(".github/**"))
	g.Expect(merged[1].Mode).To(Equal(Include))
	g.Expect(merged[2].Pattern).To(Equal("**/*.log"))
	g.Expect(merged[2].Mode).To(Equal(Exclude))
}

func TestMerge_EmptySlices_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(Merge(nil, nil)).To(BeEmpty())
	g.Expect(Merge([]Rule{}, []Rule{})).To(BeEmpty())
}
```

- [ ] **Step 2: Run the new tests**

Run: `task test -- -run "TestCompareByIndex|TestMerge" -v ./internal/filter/`

Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/filter/filter_test.go
git commit -m "filter: add tests for CompareByIndex and Merge"
```

---

### Task 3: Simplify RuleMapper to stateless decoder

**Files:**
- Modify: `internal/filter/kong.go`

- [ ] **Step 1: Replace kong.go with simplified implementation**

Replace the entire content of `internal/filter/kong.go` with:

```go
package filter

import (
	"reflect"

	"github.com/alecthomas/kong"
	"github.com/rotisserie/eris"
)

const RuleMapperName = "filterrule"

// RuleMapper decodes --include/--exclude flags into filter rules.
// Mode is inferred from the flag name; construction order is captured
// by the index assigned in NewRule().
type RuleMapper struct{}

func (RuleMapper) Decode(ctx *kong.DecodeContext, target reflect.Value) error {
	var pattern string
	if err := ctx.Scan.PopValueInto("pattern", &pattern); err != nil {
		return eris.Wrapf(err, "failed to read filter pattern for %q", ctx.Value.Name)
	}

	var mode Mode

	switch ctx.Value.Name {
	case "include":
		mode = Include
	case "exclude":
		mode = Exclude
	default:
		return eris.Errorf("unexpected filter flag name %q", ctx.Value.Name)
	}

	rule, err := NewRule(pattern, mode)
	if err != nil {
		return eris.Wrapf(err, "invalid %s %q", ctx.Value.Name, pattern)
	}

	target.Set(reflect.Append(target, reflect.ValueOf(rule)))

	return nil
}
```

This removes: `NewRuleMapper()`, `ruleSliceType`, `ruleBinding`, `bindValue()`, `bindStruct()`, and the `bindings` map.

- [ ] **Step 2: Verify the package compiles**

Run: `go build ./internal/filter/`

Expected: Success (no errors). The test file references `NewRuleMapper` which will fail — we'll fix that next.

- [ ] **Step 3: Commit**

```bash
git add internal/filter/kong.go
git commit -m "filter: simplify RuleMapper to stateless decoder

Remove all custom reflection: struct walking, pointer binding,
dual-append side-effect. Mode is inferred from ctx.Value.Name."
```

---

### Task 4: Update filter test to use Merge instead of Filters field

**Files:**
- Modify: `internal/filter/filter_test.go`

- [ ] **Step 1: Update TestRuleMapper_PopulatesFiltersDuringParseInCommandLineOrder**

Replace the existing test with:

```go
func TestRuleMapper_PopulatesFiltersDuringParseInCommandLineOrder(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var cli struct {
		Include []Rule `type:"filterrule" name:"include"`
		Exclude []Rule `type:"filterrule" name:"exclude"`
	}

	parser, err := kong.New(
		&cli,
		kong.NamedMapper(RuleMapperName, RuleMapper{}),
	)
	g.Expect(err).NotTo(HaveOccurred())

	_, err = parser.Parse([]string{
		"--exclude", ".*",
		"--include", ".github/**",
		"--exclude", "**/*.log",
	})
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(cli.Include).To(HaveLen(1))
	g.Expect(cli.Include[0].Pattern).To(Equal(".github/**"))
	g.Expect(cli.Include[0].Mode).To(Equal(Include))

	g.Expect(cli.Exclude).To(HaveLen(2))
	g.Expect(cli.Exclude[0].Pattern).To(Equal(".*"))
	g.Expect(cli.Exclude[1].Pattern).To(Equal("**/*.log"))

	merged := Merge(cli.Include, cli.Exclude)
	g.Expect(merged).To(HaveLen(3))
	g.Expect(merged[0].Pattern).To(Equal(".*"))
	g.Expect(merged[0].Mode).To(Equal(Exclude))
	g.Expect(merged[1].Pattern).To(Equal(".github/**"))
	g.Expect(merged[1].Mode).To(Equal(Include))
	g.Expect(merged[2].Pattern).To(Equal("**/*.log"))
	g.Expect(merged[2].Mode).To(Equal(Exclude))
}
```

Key changes:
- Removed `Filters []Rule` field from the test struct
- Changed `NewRuleMapper(&cli)` to `RuleMapper{}`
- Assert on individual fields rather than `Equal()` (avoids comparing unexported `index`)
- Uses `Merge()` to verify ordering

- [ ] **Step 2: Run all filter tests**

Run: `task test -- -v ./internal/filter/`

Expected: All tests pass.

- [ ] **Step 3: Commit**

```bash
git add internal/filter/filter_test.go
git commit -m "filter: update RuleMapper test to use Merge instead of Filters field"
```

---

### Task 5: Update filterMapperOption and callers

**Files:**
- Modify: `cmd/codeviz/parser.go`
- Modify: `cmd/codeviz/main.go:93`
- Modify: `cmd/codeviz/main_test.go:55`

- [ ] **Step 1: Simplify filterMapperOption in parser.go**

Replace the content of `cmd/codeviz/parser.go` with:

```go
package main

import (
	"github.com/alecthomas/kong"

	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
)

func filterMapperOption() kong.Option {
	return kong.NamedMapper(filter.RuleMapperName, filter.RuleMapper{})
}
```

- [ ] **Step 2: Update main.go to call filterMapperOption without args**

In `cmd/codeviz/main.go`, change line 93 from:

```go
		filterMapperOption(&cli),
```

To:

```go
		filterMapperOption(),
```

- [ ] **Step 3: Update main_test.go to call filterMapperOption without args**

In `cmd/codeviz/main_test.go`, change line 55 from:

```go
			filterMapperOption(&cli),
```

To:

```go
			filterMapperOption(),
```

- [ ] **Step 4: Verify compilation**

Run: `go build ./cmd/codeviz/`

Expected: Compilation failure — command structs still reference `c.Filters` field which no longer exists. This is expected; we fix it in the next task.

- [ ] **Step 5: Commit**

```bash
git add cmd/codeviz/parser.go cmd/codeviz/main.go cmd/codeviz/main_test.go
git commit -m "cli: simplify filterMapperOption to no-arg function"
```

---

### Task 6: Update command structs — remove Filters field, add Filters() method

**Files:**
- Modify: `cmd/codeviz/spiral_cmd.go`
- Modify: `cmd/codeviz/treemap_cmd.go`
- Modify: `cmd/codeviz/scatter_cmd.go`
- Modify: `cmd/codeviz/radialtree_cmd.go`
- Modify: `cmd/codeviz/bubbletree_cmd.go`

- [ ] **Step 1: Update spiral_cmd.go**

Remove line 34 (`Filters []filter.Rule \`kong:"-"\``).

Add a `Filters()` method (after the struct definition, before `Validate()`):

```go
func (c *SpiralCmd) Filters() []filter.Rule {
	return filter.Merge(c.Include, c.Exclude)
}
```

Change line 93 from `CLIFilters: c.Filters,` to `CLIFilters: c.Filters(),`.

- [ ] **Step 2: Update treemap_cmd.go**

Remove line 32 (`Filters []filter.Rule \`kong:"-"\``).

Add a `Filters()` method:

```go
func (c *TreemapCmd) Filters() []filter.Rule {
	return filter.Merge(c.Include, c.Exclude)
}
```

Change line 102 from `CLIFilters: c.Filters,` to `CLIFilters: c.Filters(),`.

- [ ] **Step 3: Update scatter_cmd.go**

Remove line 32 (`Filters []filter.Rule \`kong:"-"\``).

Add a `Filters()` method:

```go
func (c *ScatterCmd) Filters() []filter.Rule {
	return filter.Merge(c.Include, c.Exclude)
}
```

Change line 103 from `CLIFilters: c.Filters,` to `CLIFilters: c.Filters(),`.

- [ ] **Step 4: Update radialtree_cmd.go**

Remove line 32 (`Filters []filter.Rule \`kong:"-"\``).

Add a `Filters()` method:

```go
func (c *RadialCmd) Filters() []filter.Rule {
	return filter.Merge(c.Include, c.Exclude)
}
```

Change line 90 from `CLIFilters: c.Filters,` to `CLIFilters: c.Filters(),`.

- [ ] **Step 5: Update bubbletree_cmd.go**

Remove line 33 (`Filters []filter.Rule \`kong:"-"\``).

Add a `Filters()` method:

```go
func (c *BubbletreeCmd) Filters() []filter.Rule {
	return filter.Merge(c.Include, c.Exclude)
}
```

Change line 92 from `CLIFilters: c.Filters,` to `CLIFilters: c.Filters(),`.

- [ ] **Step 6: Verify full build**

Run: `go build ./...`

Expected: Success — all references to the removed field have been replaced with method calls.

- [ ] **Step 7: Commit**

```bash
git add cmd/codeviz/spiral_cmd.go cmd/codeviz/treemap_cmd.go cmd/codeviz/scatter_cmd.go cmd/codeviz/radialtree_cmd.go cmd/codeviz/bubbletree_cmd.go
git commit -m "cli: replace Filters field with Filters() method on all commands

Each command struct now calls filter.Merge(c.Include, c.Exclude)
to recover command-line flag ordering via the index on each Rule."
```

---

### Task 7: Run full CI and verify

**Files:** None (verification only)

- [ ] **Step 1: Run full test suite**

Run: `task test`

Expected: All tests pass.

- [ ] **Step 2: Run linter**

Run: `task lint`

Expected: Clean lint (no issues). The `reflect` import in `kong.go` is still used by Kong's interface, and unused imports from the old code (if any) were removed in Task 3.

- [ ] **Step 3: Run full CI pipeline**

Run: `task ci`

Expected: Build + test + lint all pass.
