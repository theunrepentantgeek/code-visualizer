# Additional Go Source Code Metrics Suggestions

Based on analysis of the code-visualizer architecture and Go code complexity patterns.

## Structural Complexity Metrics

### 1. **`import-count`** (Quantity)
Count of import declarations (excluding standard library imports).
- **Rationale:** High import counts indicate external dependencies and coupling. Files with many imports are harder to understand and test.
- **Variants:** `stdlib-import-count` (standard library only), `external-import-count` (third-party packages)

### 2. **`interface-count`** (Quantity)
Count of interface type declarations.
- **Rationale:** Interfaces define contracts. High interface density may indicate over-abstraction or a core package defining domain boundaries.
- **Variants:** `public-interface-count`, `private-interface-count`

### 3. **`struct-count`** (Quantity)
Count of struct type declarations.
- **Rationale:** Structs are data models. High counts may indicate rich domain modeling or data-heavy modules.
- **Variants:** `public-struct-count`, `private-struct-count`

## Code Organization Metrics

### 4. **`test-coverage-indicator`** (Classification: "has-test" / "no-test")
Whether a corresponding `_test.go` file exists in the same package.
- **Rationale:** Instant visual identification of untested files. More actionable than numeric coverage.
- **Note:** Can be derived from filesystem scan without AST parsing.

### 5. **`exported-surface-area`** (Quantity)
Total count of public (exported) symbols: types + functions + methods + constants + variables.
- **Rationale:** Measures API surface. High values indicate files that expose many public symbols — potential API boundary candidates or over-exposure risks.

### 6. **`package-cohesion-score`** (Measure: 0.0-1.0)
Ratio of internal references (same-package) to total references.
- **Rationale:** Low cohesion (many cross-package references) suggests split responsibilities or misplaced code.
- **Challenge:** Requires reference resolution, not just AST node counting. May be v2 scope.

## Complexity & Maintainability Metrics

### 7. **`cyclomatic-complexity`** (Quantity)
Sum of cyclomatic complexity for all functions/methods in the file.
- **Rationale:** Direct measure of code complexity. High values correlate with bug density and maintenance burden.
- **Note:** Can be computed from AST control-flow nodes (if/for/switch/case).

### 8. **`max-function-lines`** (Quantity)
Line count of the longest function in the file.
- **Rationale:** Identifies files with "god functions" that should be refactored. Complements file-lines metric.

### 9. **`comment-ratio`** (Measure: 0.0-1.0)
Ratio of comment lines to code lines.
- **Rationale:** Visualize documentation density. Very low values may indicate under-documented code; very high values may indicate commented-out code or over-explanation.

## Go-Specific Idiom Metrics

### 10. **`goroutine-spawn-count`** (Quantity)
Count of `go` keyword statements.
- **Rationale:** Indicates concurrency usage. High counts may signal complex coordination or potential race conditions.

### 11. **`defer-count`** (Quantity)
Count of `defer` statements.
- **Rationale:** Heavy defer usage may indicate resource-management-heavy code (files, locks, transactions).

### 12. **`error-return-count`** (Quantity)
Count of functions/methods returning `error` as the last return value.
- **Rationale:** Measures error-handling surface. Go idiom is explicit error returns; high counts indicate files with many fallible operations.

## Dependency & Coupling Metrics

### 13. **`dot-import-count`** (Quantity)
Count of dot imports (`. "package"`).
- **Rationale:** Anti-pattern detection. Dot imports pollute namespace and reduce readability.

### 14. **`init-function-count`** (Quantity)
Count of `init()` functions.
- **Rationale:** `init()` functions execute at package load and can cause hard-to-debug ordering issues. High counts are a code smell.

## Practical Implementation Notes

- **Feasibility:** Metrics 1-5, 7-12, 14 are straightforward AST/DST node counts.
- **DST advantage:** `github.com/dave/dst` preserves comments, enabling `comment-ratio` calculation.
- **Phased delivery:** Start with simple counts (1-5, 10-12, 14), add complexity metrics (7-9) later.
- **Visualization value:** These metrics work well in treemaps/radial trees — they're file-level aggregates that highlight hotspots.

## Recommendation Priority

**High priority (issue #289 + these):**
- `interface-count`, `struct-count` (complete the type taxonomy)
- `import-count` (coupling indicator)
- `cyclomatic-complexity` (maintenance burden)
- `test-coverage-indicator` (gap identification)

**Medium priority:**
- `goroutine-spawn-count`, `error-return-count` (Go idioms)
- `exported-surface-area` (API boundary analysis)

**Low priority (code smell detection):**
- `dot-import-count`, `init-function-count`, `defer-count`
