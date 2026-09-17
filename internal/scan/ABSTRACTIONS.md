# Abstractions — `internal/scan`

## Scanner

**Purpose.**

- The scanner turns either a working directory or a read-only `source.Tree` into the shared `model.Directory` tree, applying path filters and content policy while it walks ([scanner.go#L23](scanner.go#L23), [scanner.go#L39](scanner.go#L39)).

**Boundary and invariants.**

- Filters are evaluated against paths relative to the scan root; excluded entries never enter the model ([filter_policy.go#L23](filter_policy.go#L23), [fs_walker.go#L68](fs_walker.go#L68)).
- Binary files are marked once and omitted during the walk when `includeBinary` is false; retained files receive size and file-type base metrics as they are created ([fs_walker.go#L129](fs_walker.go#L129), [fs_walker.go#L145](fs_walker.go#L145)).
- Empty directories are pruned, aggregate file/directory counts are populated bottom-up, and a scan with no retained files fails instead of returning an empty root ([fs_walker.go#L98](fs_walker.go#L98), [fs_walker.go#L59](fs_walker.go#L59), [scanner.go#L30](scanner.go#L30)).

**Related operations.**

- `ScanTree` scans an `io/fs` source; `Scan` resolves a local path and applies the same model-building contract; `FilterBinaryFiles` produces a pruned non-binary copy of an existing tree ([scanner.go#L24](scanner.go#L24), [scanner.go#L45](scanner.go#L45), [node_builder.go#L69](node_builder.go#L69)).

**Proper-use patterns.**

- Resolve the source and filter rules first, pass progress into `ScanTree`, and publish the root only after the scan succeeds ([../stages/scan.go#L16](../stages/scan.go#L16), [../stages/scan.go#L22](../stages/scan.go#L22), [../stages/scan.go#L32](../stages/scan.go#L32)).
- Let the scanner handle symlinks: file symlinks are resolved as files, while directory symlinks are skipped to avoid recursive directory traversal ([fs_walker.go#L162](fs_walker.go#L162), [walker.go#L127](walker.go#L127), [walker.go#L138](walker.go#L138)).

**Anti-patterns.**

- Do not add empty child directories manually after scanning; both walkers append a child only when it contains retained files ([fs_walker.go#L110](fs_walker.go#L110), [walker.go#L156](walker.go#L156)).
- Do not filter binary files after calling the scanner with `includeBinary=false`; they were never added to the returned tree ([scanner.go#L42](scanner.go#L42), [fs_walker.go#L134](fs_walker.go#L134)).

**Source locations.**

- [scanner.go#L23](scanner.go#L23) — public scan entry points and result contract.
- [fs_walker.go#L68](fs_walker.go#L68) and [walker.go#L73](walker.go#L73) — read-only-source and local-filesystem traversal policies.
