# Concepts

Shared domain vocabulary for this project — entities, named processes, and status concepts with project-specific meaning. Seeded with core domain vocabulary, then accretes as ce-compound and ce-compound-refresh process learnings; direct edits are fine. Glossary only, not a spec or catch-all.

## Source Tree

The hierarchy of directories and files scanned from a selected path and used as the subject of a CodeViz visualization.

**Boundary.** The selected content may be a live working directory or a committed snapshot; it is not necessarily the whole repository or the host filesystem.

**Relationships.** Scanning builds the model tree on which metrics are stored. A snapshot selects the content; a history range selects the commits considered by history-based metrics.

**Evidence.** [Source selection](internal/stages/source.go), [scanning](internal/scan/scanner.go), and [shared model](internal/model/directory.go).

## Metric

A named property measured for a file or directory, represented as a numeric quantity, numeric measure, or textual classification.

**Boundary.** Metrics can also originate on declarations or per-file commit records. A missing value is distinct from a recorded zero; a metric's native level is distinct from the level at which a visualization requests it.

**Relationships.** Metric expressions filter or aggregate source values; colour encodings associate metric values with palettes.

**Evidence.** [Kinds](internal/metric/metric.go), [levels](internal/metric/level.go), [value storage](internal/model/metric_container.go), and [user-facing metrics](docs/content/docs/shared-concepts.md).

## Metric Expression

A user-facing metric selection that identifies a base metric and may apply a filter, aggregation, or temporal comparison.

**Boundary.** The grammar is `[filter.]base[.aggregation][.temporal]`. Parsing does not establish whether a metric exists or whether a visualization supports the requested operations.

**Relationships.** Aggregation combines source-level values at a target level; a temporal modifier compares values across ordered snapshots instead. Temporal comparisons currently belong to alluvial fills, not ordinary single-snapshot metric evaluation.

**Evidence.** [Expression parsing](internal/metric/expression.go), [provider resolution](internal/provider/resolution.go), [CLI validation](cmd/codeviz/metric_validation.go), and [alluvial fill selection](docs/content/docs/visualizations/alluvial.md).

## Colour Encoding

The selection of a metric and palette that maps a metric value to the colour channel of a visualization.

**Boundary.** An encoding identifies the metric/palette pair, not a resolved colour, shape, or legend. An unset encoding permits a fixed-colour fallback.

**Relationships.** Configuration supplies a metric spec; resolution produces an encoding; an ink maps its values to colours and matching legend swatches.

**Evidence.** [Shared encoding](internal/viz/colour_encoding.go), [resolution](internal/stages/metrics.go), [scatter inks](internal/scatter/inks.go), and [spiral inks](internal/spiral/inks.go).

## Alluvial Diagram

A static visualization that aligns directory-level metric values across ordered revision references and connects each directory's values between adjacent references.

**Boundary.** Reference order is caller-supplied, not chronological sorting. Band widths represent snapshot values, not deltas; a fill may separately encode a temporal comparison.

**Relationships.** Each reference selects a snapshot of the source tree. Repository-relative directory paths identify corresponding bands; expansion controls their partitioning independently of file filters.

**Evidence.** [User-facing contract](docs/content/docs/visualizations/alluvial.md), [configuration](internal/config/alluvial.go), and [snapshot acquisition](internal/alluvial/pipeline.go).

## Palette

An ordered collection of colours available to a colour encoding.

**Boundary.** Sequential palettes have a meaningful colour order; categorical palettes distinguish classifications. The palette does not itself measure values or determine numeric bucket boundaries.

**Relationships.** A metric descriptor supplies a default palette; an explicit metric spec can override it. Inks perform numeric bucketing or categorical assignment.

**Evidence.** [Palette vocabulary](internal/palette/palette.go), [ink construction](internal/inks/ink.go), and [palette selection](docs/content/docs/shared-concepts.md).

## Grain

The granularity of source-tree nodes represented in a visualization.

**Boundary.** `file` grain includes files and directories; `directory` grain omits files. It is not the metric's native level or a limit on visible tree depth.

**Relationships.** Grain determines the target level for metric resolution and which nodes layout collects.

**Evidence.** [Shared grain](internal/viz/grain.go), [scatter collection](internal/scatter/data.go), and [radial-tree layout](internal/radialtree/layout.go).

## File Filter

An ordered include/exclude rule that controls which source paths enter a visualization's scanned tree.

**Boundary.** Rules match paths relative to the selected scan root. The first matching rule wins; no match means inclusion. Effective configuration can supply default exclusions.

**Relationships.** File filters select content; selection metrics classify retained files; metric-expression filters restrict the source values used for a metric. These are different operations despite sharing the word "filter".

**Evidence.** [Rule evaluation](internal/filter/filter.go), [configuration/CLI combination](internal/stages/filter.go), [scan policy](internal/scan/filter_policy.go), and [user-facing rules](docs/content/docs/shared-concepts.md).

## Selection Metric

A user-defined classification metric assigning categories to files through ordered filename-glob rules.

**Boundary.** The first matching rule supplies the category; an unmatched file has no value. Classification does not include or exclude files from the source tree.

**Relationships.** Selection metrics enter the same provider registry as built-in metrics and can drive categorical colour encodings.

**Evidence.** [Configuration](internal/config/selection_metric.go), [classification provider](internal/provider/classification/provider.go), and [user-facing examples](docs/content/docs/shared-concepts.md).

## History Range

The commit selection used by history-derived metrics, bounded by optional `from` and `until` references.

**Boundary.** Revision bounds select by graph ancestry; date bounds select by author timestamp. An empty range considers history reachable from HEAD. A range is not itself a filesystem snapshot.

**Relationships.** `until` can also select the committed source tree; `from` restricts metric history rather than changing that tree. Relative dates share the run's reference clock.

**Evidence.** [Range resolution](internal/provider/git/history_range.go), [CLI range construction](cmd/codeviz/main.go), and [source selection](internal/stages/source.go).

## Snapshot

The committed repository tree selected by a revision or date reference.

**Boundary.** A snapshot is immutable committed content, not the current working directory or the set of commits in a history range. The target directory may scope it to a subtree.

**Relationships.** Scanning exposes snapshot content through the same read-only source interface as live content. Snapshot time becomes the reference clock for age-like metrics; alluvial diagrams compare multiple snapshots.

**Evidence.** [Snapshot resolution](internal/provider/git/snapshot.go), [source substitution](internal/stages/source.go), and [ordered snapshots](docs/content/docs/visualizations/alluvial.md).
