# File Inclusion/Exclusion Design

Issue: [#13 — Add file inclusion/exclusion](https://github.com/theunrepentantgeek/code-visualizer/issues/13)

Date: 2026-04-12

## Overview

Add the ability to include/exclude files and directories from scanning using
ordered glob rules. Rules are evaluated against relative paths from the scan
root. The first matching rule wins; entries with no matching rule are included by
default. More specific rules should be listed before general ones.

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
// The first matching rule wins.
// Returns true if the entry should be included.
// Default (no match) is include.
func IsIncluded(relativePath string, rules []Rule) bool

// ParseFilterFlag parses a CLI filter string into a Rule.
// A leading ! marks an exclusion (e.g., "!.git"); anything else is an
// inclusion (e.g., "*.go").
// Returns an error if the glob pattern is invalid.
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
  - pattern: ".github/**"
    mode: include
  - pattern: ".*"
    mode: exclude
```

More specific rules come first because the first match wins.

## Scanner Integration

`Scan()` accepts rules:

```go
func Scan(path string, rules []filter.Rule) (*model.Directory, error)
```

During the recursive walk, each entry's relative path (from the scan root) is
tested with `filter.IsIncluded()`:

- **Files:** excluded files are skipped (not added to the tree).
- **Directories:** excluded directories are not descended into. This is
  efficient (skipping `.git/` avoids walking thousands of objects) and simple
  to reason about.
- **Pruning:** after recursion, directories that contain no files and no
  subdirectories are pruned from the output tree.

Because first match wins, users put specific includes before general excludes.
For example, to include `.github/` but exclude other dotfiles:

```sh
codeviz treemap --filter '.github/**' --filter '!.*' ...
```

Here `.github/workflows/ci.yml` matches `.github/**` first → included.
`.git` matches `!.*` first → excluded (not descended into).

### Relative Path Computation

Scanning `/home/user/project` with an entry at
`/home/user/project/.git/objects/pack` produces the relative path
`.git/objects/pack`.

### Pattern Matching Semantics

Patterns are matched using `doublestar.Match` against the **relative path** from
the scan root. Each entry (file or directory) is evaluated independently. Since
excluded directories are not descended into, their children are never evaluated.

### Pattern Matching Examples

| Rules (in order) | Path | Matching Detail | Result |
|---|---|---|---|
| `!.*` | `.git` | `.git` matches `.*` → exclude | Excluded (not descended) |
| `!.*` | `.gitignore` | `.gitignore` matches `.*` → exclude | Excluded |
| `!.*` | `src/main.go` | no match | Included (default) |
| `.github/**`, `!.*` | `.github` | `.github` does not match `.github/**`; `.github` matches `.*` → exclude | Excluded (not descended) |
| `.github`, `.github/**`, `!.*` | `.github` | `.github` matches `.github` → include | Included (descended) |
| `.github`, `.github/**`, `!.*` | `.github/workflows/ci.yml` | matches `.github/**` → include | Included |
| `!**/*.log` | `src/debug.log` | matches `**/*.log` → exclude | Excluded |
| `!**/*.log` | `src/main.go` | no match | Included (default) |

**Note:** to include a dotfile directory and its contents, list both the
directory name and its children as separate include rules before the `!.*`
exclude (as shown with `.github` above).

## CLI Integration

`TreemapCmd` gets a repeatable `--filter` flag:

```go
type TreemapCmd struct {
    // ... existing fields ...
    Filter []string `help:"Filter rule: glob to include, !glob to exclude (repeatable, order-preserved)." short:"f"`
}
```

Usage:

```sh
codeviz treemap --filter '!.*' --filter '*.go' -s file-lines -o out.png .
```

A leading `!` marks an exclusion; anything else is an inclusion.

### Flag Processing

1. Start with `cfg.FileFilter` (from defaults or config file).
2. Parse each `--filter` string via `filter.ParseFilterFlag()`.
3. Append parsed rules to the list.
4. Pass the combined list to `scan.Scan()`.

### Validation

`TreemapCmd.Validate()` checks each `--filter` value:

- The glob pattern (after stripping any leading `!`) compiles
  (`doublestar.Match` with the pattern does not return a
  `doublestar.ErrBadPattern`).

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
- Multiple rules with first-match-wins semantics.
- `**` recursive patterns, basename patterns (`.*`), extension patterns
  (`*.log`).
- Edge cases: empty rules (default include), no matching rule, invalid patterns.
- `ParseFilterFlag()`: with and without `!` prefix, bad glob syntax.

### `internal/scan/` Tests

- Scanner with exclusion rules: excluded files absent from tree.
- Excluded directories are not descended into.
- Empty directory pruning after filtering.
- New testdata fixture with dotfiles (e.g., `.hidden`, `.config/settings`).

### `internal/config/` Tests

- Default config includes `.*` exclusion rule in `FileFilter`.
- Config file loading/saving round-trips `FileFilter` rules correctly.
- Config file rules replace defaults (not append).

### CLI Tests

- Validation rejects malformed filter flags (bad glob syntax).
- Filter flags parsed correctly: `!` prefix → exclude, no prefix → include.
- Filter flags appended to config rules.

### E2E / Golden File

- Scan a test directory with and without filter rules, compare output.
