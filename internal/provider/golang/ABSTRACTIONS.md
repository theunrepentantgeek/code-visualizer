# Abstractions — `internal/provider/golang`

## Go metric family

**Purpose.**

- The Go provider translates parsed Go source into file-level import/comment metrics and declaration-level count, complexity, and function-length metrics ([base_metrics.go#L10](base_metrics.go#L10), [file_loader.go#L26](file_loader.go#L26), [declarations.go#L43](declarations.go#L43)).

**Boundary and invariants.**

- Only files whose scanned extension is `go` are analyzed; parse/read failures are logged per file and leave values absent rather than failing the whole provider run ([file_loader.go#L51](file_loader.go#L51), [declarations.go#L43](declarations.go#L43)).
- Declaration metrics live on `model.Declaration` nodes and are aggregated later, while imports and comment ratio are stored directly on `model.File` ([declarations.go#L258](declarations.go#L258), [file_loader.go#L63](file_loader.go#L63), [base_metrics.go#L56](base_metrics.go#L56)).
- Public/private filters apply to declarations; stdlib/external/internal filters apply to imports. The descriptor metadata is the authority for those supported combinations ([base_metrics.go#L25](base_metrics.go#L25), [base_metrics.go#L34](base_metrics.go#L34), [base_metrics.go#L45](base_metrics.go#L45)).

**Related operations.**

- `RegisterBase` publishes the family metadata; `Register` installs the shared file-analysis loader ([base_metrics.go#L193](base_metrics.go#L193), [register.go#L10](register.go#L10)).
- `PopulateDeclarations` attaches declarations when declaration expressions are requested; file analysis computes imports and comment ratio in one parsed pass ([declarations.go#L43](declarations.go#L43), [file_stats.go#L140](file_stats.go#L140)).
- Analysis caches and `singleflight` deduplicate repeated OS-path parsing for source-less compatibility callers ([analysis.go#L13](analysis.go#L13), [declarations.go#L30](declarations.go#L30)).

**Proper-use patterns.**

- Read source-backed model files with `File.ReadAll`; use `RepoSource` and `RepoPath` for module discovery when repository context is attached, so historical snapshots classify imports against their own `go.mod` ([file_loader.go#L70](file_loader.go#L70), [file_loader.go#L82](file_loader.go#L82)).
- Load declarations only when `RequestedMetrics` contains declaration-level expressions, then let the shared aggregation stage derive requested file/directory values ([../../stages/compute_aggregations_stage.go#L10](../../stages/compute_aggregations_stage.go#L10), [../../stages/requested.go#L20](../../stages/requested.go#L20), [../../stages/aggregation.go#L209](../../stages/aggregation.go#L209)).
- Preserve source-less path analysis for callers of the legacy exported scanner and hand-built files until the compatibility work in [issue #755](https://github.com/theunrepentantgeek/code-visualizer/issues/755) is complete ([file_loader.go#L70](file_loader.go#L70), [declarations.go#L59](declarations.go#L59)).

**Anti-patterns.**

- Do not open `File.Path` when `File.Source` is available; doing so breaks in-memory and committed-tree sources ([file_loader.go#L70](file_loader.go#L70), [declarations.go#L59](declarations.go#L59)).
- Do not classify imports without module context when `RepoSource` is attached; repository-level module discovery is what makes scoped scans and snapshots consistent ([file_loader.go#L82](file_loader.go#L82), [file_loader_test.go#L43](file_loader_test.go#L43)).
- Do not treat the global caches as the content-source cache: their keys are OS paths and they serve the source-less compatibility path only ([analysis.go#L30](analysis.go#L30), [declarations.go#L72](declarations.go#L72)).

**Source locations.**

- [base_metrics.go#L10](base_metrics.go#L10) — metric/filter vocabulary and descriptors.
- [file_loader.go#L26](file_loader.go#L26) — source-backed file metric loading and module lookup.
- [declarations.go#L43](declarations.go#L43) — declaration population.
- [file_stats.go#L140](file_stats.go#L140) — single-pass source analysis.
