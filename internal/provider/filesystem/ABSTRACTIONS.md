# Abstractions — `internal/provider/filesystem`

## Filesystem metric family

**Purpose.**

- The filesystem metric family supplies the universal file metrics `file-size`, `file-lines`, and `file-type`, independent of language or Git availability ([metrics.go#L21](metrics.go#L21), [base_metrics.go#L16](base_metrics.go#L16)).

**Boundary and invariants.**

- The scanner owns `file-size` and `file-type`: it stores both while constructing each `model.File`, so their registered loaders are intentionally no-ops ([../../scan/fs_walker.go#L145](../../scan/fs_walker.go#L145), [metrics.go#L38](metrics.go#L38), [metrics.go#L43](metrics.go#L43)).
- `file-lines` is computed later only for files the scanner classified as text; binary classification is consumed from `File.IsBinary` and is not repeated by this provider ([metrics.go#L58](metrics.go#L58), [../../scan/fs_walker.go#L129](../../scan/fs_walker.go#L129)).
- Line counting includes a final unterminated line, treats an empty file as zero lines, and decodes UTF-16 BOM-marked text before finding newlines ([metrics.go#L113](metrics.go#L113), [metrics.go#L147](metrics.go#L147), [metrics_test.go#L157](metrics_test.go#L157)).

**Related operations.**

- `RegisterBase` publishes kinds, levels, aggregations, and palettes; `Register` pairs them with loaders and progress reporting ([base_metrics.go#L16](base_metrics.go#L16), [register.go#L10](register.go#L10)).
- `IsFilesystemMetric` identifies the family; `FileLinesProvider.Load` walks retained model files and writes `file-lines` quantities ([metrics.go#L28](metrics.go#L28), [metrics.go#L58](metrics.go#L58)).

**Proper-use patterns.**

- Request these metrics through the provider registry rather than invoking provider structs directly; scanning has already supplied the metadata-backed values ([register.go#L10](register.go#L10), [../../provider/run.go#L45](../../provider/run.go#L45)).
- For source-backed scanned files, read content through `model.File.ReadAll`, which works for live, in-memory, and historical trees ([metrics.go#L81](metrics.go#L81), [../../model/file.go#L46](../../model/file.go#L46)).
- Preserve the current OS-path fallback when handling source-less files produced by the exported legacy `scan.Scan` path or hand-built tests; unifying that compatibility boundary is tracked separately in [issue #755](https://github.com/theunrepentantgeek/code-visualizer/issues/755) ([metrics.go#L81](metrics.go#L81), [../../scan/walker.go#L17](../../scan/walker.go#L17)).

**Anti-patterns.**

- Do not re-open files to derive size or extension, and do not re-probe content to decide whether line counting is allowed; those are scanner-owned facts ([../../scan/fs_walker.go#L117](../../scan/fs_walker.go#L117), [../../scan/fs_walker.go#L145](../../scan/fs_walker.go#L145)).
- Do not remove the source-less OS fallback in isolation: `scan.Scan` currently constructs files without an attached `fs.FS` ([../../scan/node_builder.go#L41](../../scan/node_builder.go#L41), [metrics.go#L81](metrics.go#L81)).

**Source locations.**

- [metrics.go#L21](metrics.go#L21) — metric names, family predicate, and loaders.
- [base_metrics.go#L16](base_metrics.go#L16) — metric descriptors.
- [register.go#L10](register.go#L10) — loader registration.
