# Go-Specific Code Metrics

**Issue:** #289
**Date:** 2026-05-23

## Overview

Add 35 Go-specific metrics that extract structural information from `.go` files using AST parsing via `github.com/dave/dst`. These metrics complement the existing filesystem and git metrics with language-aware analysis: declaration counts, import classification, cyclomatic complexity, function length aggregates, and comment ratio.

## Architecture

### Package: `internal/provider/golang/`

Follows the data-driven pattern established by `internal/provider/git/`:

- `metrics.go` — metric name constants and `IsGoMetric()` predicate
- `go_provider.go` — `goProvider` struct, `providerDefs` map, `walkGoFiles` helper
- `file_stats.go` — `fileStats` struct and `analyzeFile()` AST extractor
- `cyclomatic.go` — cyclomatic complexity visitor
- `comments.go` — comment ratio computation
- `imports.go` — import classification and module path cache
- `register.go` — `Register()` for `main.go`
- Corresponding `*_test.go` files

### Provider Pattern

A single `goProvider` struct (analogous to `gitProvider`) with a `providerDefs` map. Each metric is a separate registered provider so users can request any subset. All providers share:

1. A `statsCache` that stores parsed `fileStats` per file path
2. A `singleflight.Group` to deduplicate concurrent parses of the same file
3. A `walkGoFiles` helper that filters to `.go` extension and invokes the per-metric extractor

Each provider implements `FileProgressReporter` to emit per-file progress callbacks.

### Registration

`main.go` gains a `golang.Register()` call alongside the existing `filesystem.Register()` and `git.Register()`.

## Metric Inventory

### Declaration Counts (15 metrics)

All are `Quantity` kind with `palette.Neutral` default.

| Metric | Description |
|---|---|
| `type-count` | Total type declarations |
| `public-type-count` | Exported type declarations |
| `private-type-count` | Unexported type declarations |
| `function-count` | Function declarations (no receiver) |
| `public-function-count` | Exported function declarations |
| `private-function-count` | Unexported function declarations |
| `method-count` | Method declarations (with receiver) |
| `public-method-count` | Exported method declarations |
| `private-method-count` | Unexported method declarations |
| `constant-count` | Constant declarations |
| `public-constant-count` | Exported constant declarations |
| `private-constant-count` | Unexported constant declarations |
| `variable-count` | Variable declarations |
| `public-variable-count` | Exported variable declarations |
| `private-variable-count` | Unexported variable declarations |

### Type Taxonomy (6 metrics)

All are `Quantity` kind with `palette.Neutral` default.

| Metric | Description |
|---|---|
| `interface-count` | Interface type declarations |
| `public-interface-count` | Exported interface type declarations |
| `private-interface-count` | Unexported interface type declarations |
| `struct-count` | Struct type declarations |
| `public-struct-count` | Exported struct type declarations |
| `private-struct-count` | Unexported struct type declarations |

### Import Counts (4 metrics)

All are `Quantity` kind with `palette.Neutral` default.

| Metric | Description |
|---|---|
| `import-count` | Total import paths |
| `stdlib-import-count` | Standard library imports |
| `external-import-count` | External (third-party) imports |
| `internal-import-count` | Imports starting with module path from `go.mod` |

### Aggregate Declaration Counts (3 metrics)

All are `Quantity` kind with `palette.Neutral` default.

| Metric | Description |
|---|---|
| `declaration-count` | Total declarations (types + functions + methods + constants + variables) |
| `public-declaration-count` | Total exported declarations |
| `private-declaration-count` | Total unexported declarations |

### Cyclomatic Complexity (3 metrics)

Default palette: `palette.Neutral`.

| Metric | Kind | Description |
|---|---|---|
| `cyclomatic-complexity-sum` | Quantity | Sum of cyclomatic complexity across all functions |
| `cyclomatic-complexity-max` | Quantity | Maximum cyclomatic complexity of any single function |
| `cyclomatic-complexity-mean` | Measure | Mean cyclomatic complexity per function |

### Function Length (3 metrics)

Default palette: `palette.Neutral`.

| Metric | Kind | Description |
|---|---|---|
| `function-length-sum` | Quantity | Sum of function lengths (lines) |
| `function-length-max` | Quantity | Length of longest function (lines) |
| `function-length-mean` | Measure | Mean function length (lines) |

### Comment Ratio (1 metric)

| Metric | Kind | Description |
|---|---|---|
| `comment-ratio` | Measure | Ratio of comment lines to code lines, ignoring blank lines. Lines with both code and a comment count for both totals. |

**Total: 35 metrics.**

## AST Analysis

### `fileStats` Struct

Holds all values extracted from a single `dst.Parse()` call. Each field is individually declared per project convention:

```go
type fileStats struct {
    types             int64
    publicTypes       int64
    privateTypes      int64
    interfaces        int64
    publicInterfaces  int64
    privateInterfaces int64
    structs           int64
    publicStructs     int64
    privateStructs    int64
    functions         int64
    publicFunctions   int64
    privateFunctions  int64
    methods           int64
    publicMethods     int64
    privateMethods    int64
    constants         int64
    publicConstants   int64
    privateConstants  int64
    variables         int64
    publicVariables   int64
    privateVariables  int64
    imports           int64
    stdlibImports     int64
    externalImports   int64
    internalImports   int64
    cyclomaticSum     int64
    cyclomaticMax     int64
    cyclomaticMean    float64
    funcLengthSum     int64
    funcLengthMax     int64
    funcLengthMean    float64
    commentRatio      float64
    declarations      int64
    publicDeclarations  int64
    privateDeclarations int64
}
```

### Parsing Approach

1. Read file, call `decorator.Parse(src)` to get `*dst.File`
2. Walk `dst.File.Decls`:
   - `*dst.GenDecl` with `token.TYPE`: count types, inspect `TypeSpec.Type` for `*dst.InterfaceType` / `*dst.StructType`
   - `*dst.GenDecl` with `token.CONST`: count constants per `ValueSpec`
   - `*dst.GenDecl` with `token.VAR`: count variables per `ValueSpec`
   - `*dst.FuncDecl`: if `Recv != nil` → method, else → function
3. Public/private: `unicode.IsUpper(rune(name[0]))`
4. Aggregate declarations: sum of types + functions + methods + constants + variables (with public/private variants)

### Cyclomatic Complexity

Base complexity of 1 per function, plus 1 for each:
- `if` statement
- `for` statement (including `range`)
- Each `case` clause in `switch` (excluding `default`)
- Each `case` clause in `select` (excluding `default`)
- `&&` binary expression
- `||` binary expression

Computed per function/method, then aggregated to file-level sum/max/mean.

### Function Length

Line count per function: `funcDecl.End().Line - funcDecl.Pos().Line + 1` (using `dst` position info via the decorator's `Fset`). Includes the signature line and closing brace. Aggregated to sum/max/mean.

### Comment Ratio

Scan the source file line by line:
- A line is a **comment line** if it contains `//` or is within a `/* */` block (determined from `dst.File.Comments` position ranges)
- A line is a **code line** if it contains non-whitespace content outside comments
- Lines with both code and comments count for **both** totals
- Blank lines (whitespace only) are excluded from both totals
- `comment-ratio = commentLines / codeLines` (0.0 if no code lines)

### Import Classification

For each `*dst.ImportSpec` in `dst.File.Imports`:
- Extract the import path (strip quotes)
- **stdlib**: first path element contains no dot (e.g., `fmt`, `net/http`, `encoding/json`)
- **internal**: path starts with the module path from the nearest `go.mod`
- **external**: everything else

### Module Path Lookup

Walk up from each file's directory looking for `go.mod`. Read the first line matching `^module\s+(.+)$`. Cache results per directory in a `moduleCache` (mutex-protected map of `directory → module path`). Files outside any Go module get `internal-import-count = 0`.

## Caching

### Stats Cache

```go
type statsCache struct {
    mu     sync.Mutex
    group  singleflight.Group
    stats  map[string]*fileStats
}
```

- Keyed by absolute file path
- Uses `singleflight.Group` to deduplicate concurrent parses of the same file
- First provider to access a file triggers the parse; subsequent providers read cached results
- Cache is package-level, reset between `Load()` invocations if the root changes

### Module Cache

```go
type moduleCache struct {
    mu      sync.RWMutex
    modules map[string]string
}
```

- Keyed by directory path
- Walks up parent directories until `go.mod` is found or filesystem root is reached
- Caches intermediate directories along the way

## Walk Pattern

`walkGoFiles` mirrors `walkGitFiles`:

```go
func walkGoFiles(
    root *model.Directory,
    desc string,
    onFile func(),
    process func(*fileStats, *model.File),
) error
```

1. Walk all files via `model.WalkFiles`
2. Filter to `.go` extension (skip binary files)
3. Get or compute `fileStats` from cache (via singleflight)
4. Call `process(stats, file)` to set the specific metric
5. Call `onFile()` for progress reporting

## Test Files

`_test.go` files are included like any other `.go` file. Users who want to exclude them can use file filter rules.

## Testing Strategy

- `file_stats_test.go` — Parse small Go source strings, verify all `fileStats` fields
- `cyclomatic_test.go` — Table-driven tests for complexity of various control flow patterns
- `comments_test.go` — Comment ratio with code-only, comment-only, mixed, and blank lines
- `imports_test.go` — Import classification (stdlib/external/internal), module path discovery
- `go_provider_test.go` — Integration: create temp `.go` files, register providers, run `Load()`, verify metrics on `model.File`

Tests use Gomega assertions, `t.Parallel()`, and table-driven patterns consistent with the rest of the codebase.

## Dependencies

- `github.com/dave/dst` — Go AST parsing with decoration preservation
- `golang.org/x/sync/singleflight` — deduplicate concurrent file parses (already have `golang.org/x/sync`)

## Non-Goals

- No AST caching across separate CLI invocations (the CLI is a one-shot tool)
- No incremental parsing — each run parses all `.go` files fresh
- No cross-file analysis (e.g., interface satisfaction, call graphs)
- Go-specific metrics from the second comment that were not selected in the final comment (goroutine-spawn-count, error-return-count, defer-count, dot-import-count, init-function-count, test-coverage-indicator) are out of scope
