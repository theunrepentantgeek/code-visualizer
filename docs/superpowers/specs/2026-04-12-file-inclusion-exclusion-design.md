# File Inclusion/Exclusion Design

Issue: [#13 — Add file inclusion/exclusion](https://github.com/theunrepentantgeek/code-visualizer/issues/13)

Date: 2026-04-12

## Overview

Add the ability to include/exclude files and directories from scanning using
ordered glob rules. Rules are evaluated against relative paths from the scan
root. The last matching rule wins; entries with no matching rule are included by
default.

## New Package: `internal/filter/`

A small, focused package with no internal dependencies (only `doublestar`).

### Types

```go
type Mode int

const (
    Include Mode = iota
    Exclude
)

type Rule struct {
    Pattern string `yaml:"pattern" json:"pattern"`
    Mode    Mode   `yaml:"mode"   json:"mode"`
}
```

`Mode` implements `encoding.TextMarshaler`/`TextUnmarshaler` for YAML/JSON
serialization (`"include"` / `"exclude"`).

### Functions

```go
// IsIncluded evaluates relativePath against rules in order.
// Tests every prefix of the path (plus the full path) against each rule.
// The last matching rule across all prefixes wins.
// Returns true if the entry should be included.
// Default (no match) is include.
func IsIncluded(relativePath string, rules []Rule) bool

// ParseFilterFlag parses a "+glob" or "-glob" string into a Rule.
// Returns an error if the string lacks a +/- prefix or the glob is invalid.
func ParseFilterFlag(s string) (Rule, error)
```

`IsIncluded` uses `doublestar.Match` for glob evaluation. Patterns are matched
against relative paths from the scan root (e.g., `.git/objects/pack`).

## Config Integration

`Config` gains a `FileFilter` field:

```go
type Config struct {
    Width      *int          `yaml:"width,omitempty"      json:"width,omitempty"`
    Height     *int          `yaml:"height,omitempty"     json:"height,omitempty"`
    Treemap    *Treemap      `yaml:"treemap,omitempty"    json:"treemap,omitempty"`
    FileFilter []filter.Rule `yaml:"fileFilter,omitempty" json:"fileFilter,omitempty"`
}
```

`New()` populates the default rules:

```go
FileFilter: []filter.Rule{
    {Pattern: ".*", Mode: filter.Exclude},
},
```

Config file rules **replace** defaults entirely (standard YAML unmarshal
behavior for slices). CLI filter flags **append** to whatever the config
provides.

### YAML Example

```yaml
fileFilter:
  - pattern: ".*"
    mode: exclude
  - pattern: ".github/**"
    mode: include
```

## Scanner Integration

`Scan()` accepts rules:

```go
func Scan(path string, rules []filter.Rule) (*model.Directory, error)
```

During the recursive walk, each entry's relative path (from the scan root) is
tested with `filter.IsIncluded()`:

- **Files:** excluded files are skipped entirely (not added to the tree).
- **Directories:** always descended into, regardless of whether the directory
  itself matches an exclusion rule. This allows include rules to override
  earlier excludes on children (e.g., `-.*`, `+.github/**`). After recursion,
  directories that contain no files and no subdirectories are pruned from the
  output tree.

This means a rule like `.*` excludes dotfiles at any level, but we still
enter `.git/` to check whether any child is re-included by a later rule.
In practice, with only `-.*` as a default, `.git/` is entered but every child
also matches `.*` and is excluded, so `.git/` ends up empty and is pruned.

### Relative Path Computation

Scanning `/home/user/project` with an entry at
`/home/user/project/.git/objects/pack` produces the relative path
`.git/objects/pack`.

### Pattern Matching Semantics

Patterns are matched using `doublestar.Match` against the **relative path** from
the scan root.

**Prefix-based evaluation:** When checking whether a path like `.git/HEAD` is
included, `IsIncluded` tests every prefix of the path (`.git`, `.git/HEAD`)
against the rules. The last matching rule across all prefixes wins. This ensures
that excluding `.*` also excludes everything inside `.git/` — without needing
explicit `.**` or `.git/**` patterns.

This prefix evaluation also makes re-inclusion work naturally:
`-.*`, `+.github/**` excludes `.git/HEAD` (`.git` matches `.*`, nothing
re-includes it) but includes `.github/workflows/ci.yml` (`.github` matches `.*`
for exclusion, but `.github/workflows/ci.yml` matches `.github/**` for
inclusion — last match wins).

### Pattern Matching Examples

| Rules (in order) | Path | Matching Detail | Result |
|---|---|---|---|
| `-.*` | `.git` | `.git` matches `.*` → exclude | Excluded |
| `-.*` | `.git/HEAD` | prefix `.git` matches `.*` → exclude | Excluded |
| `-.*` | `src/main.go` | no prefix or path matches | Included (default) |
| `-.*`, `+.github/**` | `.github/workflows/ci.yml` | `.github` matches `.*` → excl; full path matches `.github/**` → incl | Included (last match) |
| `-.*`, `+.github/**` | `.git/HEAD` | `.git` matches `.*` → excl; no include match | Excluded |
| `-**/*.log` | `src/debug.log` | full path matches `**/*.log` → excl | Excluded |
| `-**/*.log` | `src/main.go` | no match | Included (default) |

## CLI Integration

`TreemapCmd` gets a repeatable `--filter` flag:

```go
type TreemapCmd struct {
    // ... existing fields ...
    Filter []string `help:"Filter rule: +glob to include, -glob to exclude (repeatable, order-preserved)." short:"f"`
}
```

Usage:

```sh
codeviz treemap --filter '-.*' --filter '+.github/**' --filter '-*.log' -s file-lines -o out.png .
```

### Flag Processing

1. Start with `cfg.FileFilter` (from defaults or config file).
2. Parse each `--filter` string via `filter.ParseFilterFlag()`.
3. Append parsed rules to the list.
4. Pass the combined list to `scan.Scan()`.

### Validation

`TreemapCmd.Validate()` checks each `--filter` value:

- Starts with `+` or `-`.
- The glob pattern compiles (`doublestar.Match` with the pattern does not return
  a `doublestar.ErrBadPattern`).

## Execution Flow

Updated `TreemapCmd.Run()` sequence:

1. Apply config overrides.
2. **Merge filter rules** — config defaults + CLI `--filter` flags.
3. `scan.Scan(path, rules)` — filtering happens during the walk.
4. Check git requirement.
5. `provider.Run()` — compute metrics on the filtered tree.
6. `filterBinaryFiles()` — existing binary filter (unchanged).
7. `treemap.Layout()` → render.

Filtering before metrics means no wasted computation on excluded files.

## Dependencies

**New:** `github.com/bmatcuk/doublestar/v4` — MIT licensed, widely used,
well-maintained glob library with `**` support.

### Import Graph (no cycles)

```
filter (new — no internal deps, only doublestar)
  ↑
config (imports filter for Rule type)
  ↑
scan (imports filter, model)
  ↑
cmd/codeviz (imports scan, config, filter, provider, etc.)
```

`filter` sits at the bottom of the dependency tree alongside `metric`.

## Testing

### `internal/filter/` Tests

- Single rule matching (include, exclude).
- Multiple rules with last-match-wins semantics.
- `**` recursive patterns, basename patterns (`.*`), extension patterns
  (`*.log`).
- Edge cases: empty rules (default include), no matching rule, invalid patterns.
- `ParseFilterFlag()`: valid `+`/`-` prefixes, missing prefix, bad glob syntax.

### `internal/scan/` Tests

- Scanner with exclusion rules: excluded files absent from tree.
- Directory descent: include rule overrides exclude on children.
- Empty directory pruning after filtering.
- New testdata fixture with dotfiles (e.g., `.hidden`, `.config/settings`).

### `internal/config/` Tests

- Default config includes `.*` exclusion rule in `FileFilter`.
- Config file loading/saving round-trips `FileFilter` rules correctly.
- Config file rules replace defaults (not append).

### CLI Tests

- Validation rejects malformed filter flags (no `+`/`-` prefix, bad glob).
- Filter flags parsed and appended to config rules.

### E2E / Golden File

- Scan a test directory with and without filter rules, compare output.
