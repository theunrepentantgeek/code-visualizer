# Abstractions — `internal/config`

## Config

**Purpose.**

- `Config` is the single source of truth for effective configuration, regardless of whether a value came from a built-in default, a config file, or a CLI flag ([config.go#L30](config.go#L30), [config.go#L34](config.go#L34)).

**Boundary and invariants.**

- Every field is optional and pointer- or slice-typed: nil/empty means "not configured", non-nil means "explicitly set" — that is what makes layering possible ([config.go#L32](config.go#L32), [config.go#L35](config.go#L35)).
- The layers apply in a fixed order — `New()` defaults, then `Load`/`TryAutoLoad` on top of the existing struct, then `Override*` for CLI flags — and each layer only overwrites what it actually specifies ([config.go#L78](config.go#L78), [config.go#L114](config.go#L114), [override.go#L5](override.go#L5)).
- `TryAutoLoad` is a no-op once `Source` is set, so an explicitly loaded file is never silently overlaid by auto-discovery ([config.go#L155](config.go#L155)).

**Related operations.**

- `New`, `Load`, `TryAutoLoad`, `Save`, and the `Override*` methods on `Config` and its viz sections ([config.go#L78](config.go#L78), [config.go#L114](config.go#L114), [config.go#L155](config.go#L155), [treemap.go#L13](treemap.go#L13)).
- `ForExport` returns a shallow copy carrying the shared sections plus only the named visualization's section ([config.go#L174](config.go#L174)).
- `SelectionMetricsList` exposes the on-disk selection-metric map as an ordered slice ([config.go#L272](config.go#L272)).

**Proper-use patterns.**

- In a command, run the whole merge in one place — auto-load, apply overrides, then validate the effective config — before building any pipeline state ([../../cmd/codeviz/treemap_cmd.go#L66](../../cmd/codeviz/treemap_cmd.go#L66)).
- Write CLI values through the `Override*` helpers, which ignore zero values so an unset flag is transparent ([../../cmd/codeviz/treemap_cmd.go#L112](../../cmd/codeviz/treemap_cmd.go#L112), [override.go#L11](override.go#L11)).
- Export the config for reproduction with `ForExport(vizName)` so the emitted file describes one visualization ([config.go#L174](config.go#L174), [../stages/common.go#L42](../stages/common.go#L42)).

**Anti-patterns.**

- Do not make a config field a plain value type to "simplify" it: the pointer is the only signal distinguishing "set to zero" from "not set" ([config.go#L32](config.go#L32), [override.go#L12](override.go#L12)).
- Do not read raw flag values deep in the pipeline; stages consume the merged `*config.Config` and its viz section ([../../cmd/codeviz/treemap_cmd.go#L93](../../cmd/codeviz/treemap_cmd.go#L93)).

**Source locations.**

- [config.go#L34](config.go#L34) — `Config`, `New`, `Load`, `TryAutoLoad`, `ForExport`.
- [override.go#L5](override.go#L5) — the non-zero-only override helpers.

## MetricSpec

**Purpose.**

- `MetricSpec` is the pairing of a metric name with an optional palette name, as one user-facing value written `metric` or `metric,palette` ([metric_spec.go#L15](metric_spec.go#L15), [metric_spec.go#L20](metric_spec.go#L20)).

**Boundary and invariants.**

- The metric name is mandatory when the spec is non-empty; the palette is optional, and a trailing extra comma is a parse error ([metric_spec.go#L72](metric_spec.go#L72), [metric_spec.go#L76](metric_spec.go#L76)).
- Accessors are nil-safe by design so a missing config section reads as "" rather than requiring nil checks at every call site ([metric_spec.go#L30](metric_spec.go#L30), [metric_spec.go#L42](metric_spec.go#L42)).
- It round-trips identically through CLI text, YAML, and JSON, with `String()` as the canonical form, and `Validate` checks the metric against the provider registry and the palette against the known palettes ([metric_spec.go#L51](metric_spec.go#L51), [metric_spec.go#L131](metric_spec.go#L131), [metric_spec.go#L96](metric_spec.go#L96)).

**Related operations.**

- `IsZero` drives the override helper that only applies non-empty specs ([metric_spec.go#L26](metric_spec.go#L26), [override.go#L19](override.go#L19)).

**Proper-use patterns.**

- Declare colour-bearing CLI flags as `config.MetricSpec` so Kong parses `metric[,palette]` directly ([../../cmd/codeviz/treemap_cmd.go#L23](../../cmd/codeviz/treemap_cmd.go#L23)).
- Validate each spec with a label identifying the field, so errors name "fill" or "border" ([../../cmd/codeviz/treemap_cmd.go#L53](../../cmd/codeviz/treemap_cmd.go#L53)).
- Read names through `MetricName()`/`PaletteName()` on a possibly-nil spec when collecting requested metrics ([../stages/metrics.go#L17](../stages/metrics.go#L17)).

**Anti-patterns.**

- Do not carry metric and palette as two separate flags or config keys; they are selected together and validated together ([metric_spec.go#L96](metric_spec.go#L96)).

**Source locations.**

- [metric_spec.go#L20](metric_spec.go#L20) — `MetricSpec`, its accessors, text/YAML/JSON codecs, and `Validate`.

## SelectionMetric

**Purpose.**

- `SelectionMetric` is a user-defined classification metric: a named, ordered list of glob→category rules that assigns each file a category based on its relative path ([selection_metric.go#L29](selection_metric.go#L29), [selection_metric.go#L43](selection_metric.go#L43)).

**Boundary and invariants.**

- Rules are evaluated in order and the first match wins; a `*` rule at the end acts as the default, and files matching no rule simply have no value for the metric ([selection_metric.go#L11](selection_metric.go#L11), [selection_metric.go#L31](selection_metric.go#L31)).
- Patterns are doublestar globs matched against the file's relative path, validated with the same validator as file filters ([selection_metric.go#L17](selection_metric.go#L17), [selection_metric.go#L22](selection_metric.go#L22)).
- The on-disk shape is a map from name to rule list; converting to the ordered slice sorts by name so provider registration order is stable across runs ([selection_metric.go#L63](selection_metric.go#L63), [selection_metric.go#L68](selection_metric.go#L68)).

**Related operations.**

- `Validate` on both the metric and its rules; `Config.SelectionMetricsList` produces the ordered view ([selection_metric.go#L50](selection_metric.go#L50), [config.go#L272](config.go#L272)).
- `classification.Register` turns one entry into a registered base metric descriptor plus loader ([../provider/classification/provider.go#L33](../provider/classification/provider.go#L33)).

**Proper-use patterns.**

- Register every configured selection metric as a pipeline stage before metrics are resolved, so the names validate like built-in metrics ([../stages/selection_metrics.go#L23](../stages/selection_metrics.go#L23), [../../cmd/codeviz/treemap_cmd.go#L103](../../cmd/codeviz/treemap_cmd.go#L103)).

**Anti-patterns.**

- Do not iterate the raw map directly; use the sorted slice view, because map order would make registration (and therefore output) non-deterministic ([selection_metric.go#L65](selection_metric.go#L65), [selection_metric.go#L68](selection_metric.go#L68)).

**Source locations.**

- [selection_metric.go#L15](selection_metric.go#L15) — `SelectionMetricRule`, `SelectionMetric`, and the raw map conversion.
