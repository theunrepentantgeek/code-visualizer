# Abstractions — `internal/filter`

## Rule

**Purpose.**

- A `Rule` pairs a glob pattern with an include/exclude `Mode`, and is the single unit of path selection used by scanning and configuration ([filter.go#L24](filter.go#L24), [filter.go#L14](filter.go#L14)).
- An ordered `[]Rule` is the complete selection policy: evaluation is first-match-wins, and an unmatched path defaults to *included* ([filter.go#L35](filter.go#L35), [filter.go#L43](filter.go#L43), [filter.go#L48](filter.go#L48)).

**Boundary and invariants.**

- Patterns are matched against *relative* paths with gitignore-like semantics: an unanchored pattern (no leading `/` or `**/`) is also retried with an implicit `**/` prefix, so it matches at any depth ([filter.go#L54](filter.go#L54), [filter.go#L64](filter.go#L64)).
- A `Rule` carries an unexported monotonic `index` assigned at construction; it exists so the original command-line order of interleaved `--include`/`--exclude` flags can be recovered ([filter.go#L27](filter.go#L27), [filter.go#L141](filter.go#L141), [filter.go#L161](filter.go#L161)).
- Rules built by `NewRule` have a syntactically valid, non-empty pattern; a rule whose pattern fails to compile at evaluation time is skipped rather than fatal ([filter.go#L122](filter.go#L122), [filter.go#L39](filter.go#L39)).

**Related operations.**

- `NewRule` / `ParseFilterFlag` construct (the latter reading a leading `!` as exclusion); `Merge` concatenates include and exclude slices and re-sorts them with `CompareByIndex` ([filter.go#L122](filter.go#L122), [filter.go#L105](filter.go#L105), [filter.go#L167](filter.go#L167)).
- `IsIncluded` evaluates a rule list; `ValidatePattern` checks a pattern without building a rule ([filter.go#L35](filter.go#L35), [filter.go#L147](filter.go#L147)).

**Proper-use patterns.**

- Build CLI rules through `filter.NewRule` so the ordering index is assigned, then recombine the separate include/exclude slices with `filter.Merge` before use ([../../cmd/codeviz/filteradapter.go#L36](../../cmd/codeviz/filteradapter.go#L36), [../../cmd/codeviz/treemap_cmd.go#L43](../../cmd/codeviz/treemap_cmd.go#L43)).
- Layer config-file rules first and CLI rules after, so later CLI flags win ties by position ([../stages/filter.go#L11](../stages/filter.go#L11)).
- Evaluate with a path made relative to the scan root before calling `IsIncluded` ([../scan/filter_policy.go#L23](../scan/filter_policy.go#L23), [../scan/fs_walker.go#L74](../scan/fs_walker.go#L74)).

**Anti-patterns.**

- Do not reorder or sort a merged rule list by anything other than `CompareByIndex`; ordering *is* the semantics, because the first matching rule decides ([filter.go#L161](filter.go#L161), [filter.go#L43](filter.go#L43)).
- Do not construct `Rule{Pattern: …, Mode: …}` literals in production paths: the zero `index` collapses ordering under `Merge` ([filter.go#L27](filter.go#L27), [filter.go#L175](filter.go#L175)).
- Do not assume an unmatched path is excluded — the default is inclusion ([filter.go#L48](filter.go#L48)).

**Source locations.**

- [filter.go#L24](filter.go#L24) — `Rule`, with `Mode` at [filter.go#L14](filter.go#L14).
- [filter.go#L35](filter.go#L35) — `IsIncluded` evaluation order; matching at [filter.go#L54](filter.go#L54).
- [filter.go#L161](filter.go#L161) — `CompareByIndex` and `Merge` at [filter.go#L167](filter.go#L167).
