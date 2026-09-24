# Abstractions — `internal/provider/classification`

## Configured classification provider

**Purpose.**

- A configured classification provider turns one `config.SelectionMetric` into a normal file-level classification metric, so user-defined path categories participate in resolution, loading, aggregation, palettes, and legends exactly like built-in metrics ([provider.go#L33](provider.go#L33), [../../config/selection_metric.go#L29](../../config/selection_metric.go#L29)).

**Boundary and invariants.**

- Registration is runtime-driven rather than package-global: each configured metric registers one descriptor and one loader under its configured name ([provider.go#L33](provider.go#L33), [../../stages/selection_metrics.go#L18](../../stages/selection_metrics.go#L18)).
- Classification is relative to the scanned root, normalized to slash-separated form, and first-match-wins; an unmatched file receives no metric value ([provider.go#L56](provider.go#L56), [provider_test.go#L33](provider_test.go#L33), [provider_test.go#L63](provider_test.go#L63)).
- The descriptor is always a file-level `Classification` with the categorization palette; configured rules do not create a separate provider vocabulary or filtering mechanism ([provider.go#L36](provider.go#L36)).

**Related operations.**

- `Register` creates the descriptor and loader; the loader walks the model and stores categories with `SetClassification` ([provider.go#L33](provider.go#L33), [provider.go#L56](provider.go#L56)).
- `Config.SelectionMetricsList` supplies a name-sorted registration sequence, and `RegisterSelectionMetrics` invokes registration before requested metrics are resolved ([../../config/config.go#L272](../../config/config.go#L272), [../../stages/selection_metrics.go#L18](../../stages/selection_metrics.go#L18)).

**Proper-use patterns.**

- Validate the configuration first, then register every entry once at pipeline setup so configured names are present in the shared provider registry before resolution ([../../config/selection_metric.go#L50](../../config/selection_metric.go#L50), [../../stages/selection_metrics.go#L18](../../stages/selection_metrics.go#L18)).
- Put a catch-all rule last when every retained file should receive a category; otherwise absence is a supported result ([../../config/selection_metric.go#L31](../../config/selection_metric.go#L31)).

**Anti-patterns.**

- Do not sort rules or merge categories after matching: rule order is semantic and the first match is final ([provider.go#L69](provider.go#L69)).
- Do not set a fallback classification for unmatched files in this provider; neutral rendering of absent values belongs to the colour layer ([provider_test.go#L63](provider_test.go#L63), [../../../docs/content/docs/shared-concepts.md#L87](../../../docs/content/docs/shared-concepts.md#L87)).

**Source locations.**

- [provider.go#L25](provider.go#L25) — configured loader state.
- [provider.go#L33](provider.go#L33) — descriptor/loader registration.
- [provider.go#L56](provider.go#L56) — relative-path classification.
