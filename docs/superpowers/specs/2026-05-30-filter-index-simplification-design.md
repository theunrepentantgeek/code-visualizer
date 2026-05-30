# Filter Index Simplification

**Date:** 2026-05-30  
**Status:** Approved

## Problem

The current `RuleMapper` in `internal/filter/kong.go` uses custom struct-walking reflection and pointer-address binding to track the original command-line order of `--include`/`--exclude` flags. This dual-append side-effect populates a hidden `Filters []filter.Rule` field on each command struct. The mechanism is fragile, hard to follow, and requires passing the root CLI struct into `NewRuleMapper` for reflective traversal.

## Solution

Replace the reflection-heavy approach with a simple index-based ordering scheme:

1. Each `Rule` carries an internal (unexported) `index int` field.
2. `NewRule()` auto-assigns the index from a package-level `atomic.Int64` counter.
3. A `CompareByIndex(a, b Rule) int` function enables `slices.SortFunc` sorting.
4. A `Merge(include, exclude []Rule) []Rule` function concatenates both slices and sorts by index, recovering the original construction order.
5. The `RuleMapper` is simplified to a stateless decoder—mode is inferred from `ctx.Value.Name` ("include" or "exclude").
6. The `Filters` field is removed from command structs and replaced with a `Filters()` method that calls `filter.Merge`.

## Detailed Design

### Rule struct

```go
type Rule struct {
    Pattern string `yaml:"pattern" json:"pattern"`
    Mode    Mode   `yaml:"mode"   json:"mode"`
    index   int
}
```

The `index` field is unexported and excluded from serialization. It captures construction order for sorting purposes only.

### NewRule()

```go
var ruleCounter atomic.Int64

func NewRule(pattern string, mode Mode) (Rule, error) {
    // existing validation ...
    return Rule{
        Pattern: pattern,
        Mode:    mode,
        index:   int(ruleCounter.Add(1)),
    }, nil
}
```

The counter is package-level. Because CLI parsing is single-threaded and tests only care about relative ordering, there is no need for reset functionality.

### CompareByIndex

```go
func CompareByIndex(a, b Rule) int {
    return a.index - b.index
}
```

Usable with `slices.SortFunc(rules, filter.CompareByIndex)`.

### Merge

```go
func Merge(include, exclude []Rule) []Rule {
    result := make([]Rule, 0, len(include)+len(exclude))
    result = append(result, include...)
    result = append(result, exclude...)
    slices.SortFunc(result, CompareByIndex)
    return result
}
```

### Simplified RuleMapper

```go
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

The `reflect` import remains because Kong's `MapperFunc` interface requires it—but all *custom* reflection (struct walking, pointer binding) is eliminated.

### filterMapperOption

```go
func filterMapperOption() kong.Option {
    return kong.NamedMapper(filter.RuleMapperName, filter.RuleMapper{})
}
```

No longer requires the CLI struct as an argument.

### Command structs

Each command struct (spiral, treemap, scatter, radialtree, bubbletree) changes from:

```go
Filters []filter.Rule `kong:"-"`
Include []filter.Rule `type:"filterrule" ...`
Exclude []filter.Rule `type:"filterrule" ...`
```

To:

```go
Include []filter.Rule `type:"filterrule" ...`
Exclude []filter.Rule `type:"filterrule" ...`

func (c *SpiralCmd) Filters() []filter.Rule {
    return filter.Merge(c.Include, c.Exclude)
}
```

### Consumer changes

In `internal/stages/common.go`, `CLIFilters []filter.Rule` remains unchanged. The assignment in each command's `Run()` method changes from `CLIFilters: c.Filters` to `CLIFilters: c.Filters()`.

## What doesn't change

- `IsIncluded()` logic and tests.
- `ParseFilterFlag()` (already calls `NewRule`—it will automatically get indices).
- Include/Exclude struct tags and Kong binding.
- The `stages` package filter logic.
- Serialization (`yaml`/`json` tags on Rule)—the `index` field is unexported so it's excluded.

## Testing

- Existing `TestRuleMapper_PopulatesFiltersDuringParseInCommandLineOrder` is updated: removes the `Filters` field from the test struct, calls `filter.Merge(cli.Include, cli.Exclude)`, and asserts the same ordering.
- Add a unit test for `CompareByIndex` verifying sort correctness.
- Add a unit test for `Merge` verifying combined ordering.
- Existing `main_test.go` integration tests continue to pass (they test parse+run, not Filters field directly).
