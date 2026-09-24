# Abstractions — `internal/export`

## Metric Tree Export Projection

**Purpose.**

- A *metric tree export projection* is the stable JSON/YAML-facing representation of a scanned model tree: one root directory, recursively nested directories and files, and metric values separated by quantity, measure, and classification kind ([export.go#L19](export.go#L19), [export.go#L25](export.go#L25), [export.go#L36](export.go#L36)).
- It decouples the serialized contract from the richer in-memory model while preserving the repository hierarchy and file identity needed by downstream consumers ([export.go#L128](export.go#L128), [export_test.go#L237](export_test.go#L237)).

**Boundary and invariants.**

- `Export` is the package boundary: an empty path is a no-op, while a non-empty path must have a supported `.json`, `.yaml`, or `.yml` extension and is written with owner-only permissions ([export.go#L50](export.go#L50), [export.go#L112](export.go#L112)).
- The projection preserves directory/file order and hierarchy from the model, including file extension and binary status ([export.go#L128](export.go#L128), [export_test.go#L237](export_test.go#L237), [export_test.go#L306](export_test.go#L306)).
- Only requested metrics that are present on each model node are emitted. Metrics remain separated into typed maps, and absent collections stay nil so `omitempty` removes them from the wire format ([export.go#L164](export.go#L164), [export_test.go#L150](export_test.go#L150), [export_test.go#L381](export_test.go#L381)).

**Related operations.**

- `stages.ExportData` supplies the computed root and requested base metrics, making this projection the common data-export operation used by visualization pipelines ([../stages/export.go#L24](../stages/export.go#L24), [../treemap/pipeline.go#L28](../treemap/pipeline.go#L28), [../alluvial/pipeline.go#L198](../alluvial/pipeline.go#L198)).
- `formatFromPath` selects the encoder, `marshalExport` provides formatted JSON or YAML, and the recursive directory/file converters build the projection ([export.go#L81](export.go#L81), [export.go#L112](export.go#L112), [export.go#L128](export.go#L128)).

**Proper-use patterns.**

- Pass the resolved requested base-metric names rather than every metric in a container; this keeps exported data aligned with the visualization request ([../stages/export.go#L26](../stages/export.go#L26), [export_test.go#L150](export_test.go#L150)).
- Consume exported data through `ExportData`, `DirectoryExport`, and `FileExport` when validating the wire contract instead of depending on private conversion helpers ([export_test.go#L64](export_test.go#L64), [export_test.go#L207](export_test.go#L207)).
- Use `.json`, `.yaml`, or `.yml` as the output suffix and let `Export` choose serialization from that suffix ([export.go#L112](export.go#L112), [export_test.go#L64](export_test.go#L64), [export_test.go#L113](export_test.go#L113)).

**Anti-patterns.**

- Do not serialize `model.Directory` directly; doing so bypasses metric filtering and the explicit wire schema ([export.go#L19](export.go#L19), [export.go#L128](export.go#L128), [export.go#L164](export.go#L164)).
- Do not add absent requested metrics with zero values or merge unlike metric kinds into one map; presence and kind are part of the export contract ([export.go#L164](export.go#L164), [export_test.go#L150](export_test.go#L150), [export_test.go#L381](export_test.go#L381)).
- Do not infer format from configuration or content; output extension is the authoritative selector and unsupported or missing extensions are errors ([export.go#L112](export.go#L112), [export_test.go#L137](export_test.go#L137)).

**Source locations.**

- [export.go#L19](export.go#L19) — public export schema.
- [export.go#L50](export.go#L50) — export orchestration and file-writing boundary.
- [export.go#L128](export.go#L128) — recursive model-to-export projection.
- [export.go#L164](export.go#L164) — requested, present, typed metric selection.
- [export_test.go#L64](export_test.go#L64) — format, hierarchy, filtering, file metadata, and metric-kind contract tests.
- [../stages/export.go#L24](../stages/export.go#L24) — production pipeline consumer.
