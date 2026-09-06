# Filter Files by Git Range Design

## Goal

Add an opt-in mode that limits every visualization to current-tree files
modified by at least one commit in the effective Git history range.

The existing default remains a structural snapshot of the current working tree.
History constraints continue to affect Git-derived metrics and timeline data,
but do not filter files unless this mode is enabled.

## User Interface

Every visualization command and `render` accepts:

```text
--changed-only
```

The equivalent root configuration key is:

```yaml
changedOnly: true
```

The command-line switch enables the setting when present. Omitting it leaves a
configuration-file value unchanged.

`changedOnly` requires at least one non-empty `--from` or `--until` constraint.
Enabling it without a constrained range returns a clear validation error rather
than treating all reachable history as an implicit range.

The unified history reference interface remains unchanged. Each bound may
resolve to a tag, commit, or date, including explicit `tag:`, `sha:`, and
`date:` prefixes.

## Architecture

A shared pipeline stage runs after the filesystem scan and before metric
providers. It is a no-op unless the effective `changedOnly` setting is true.

The stage:

1. resolves the repository root for the scanned tree;
2. indexes the already-scanned files by repository-relative slash path;
3. asks the Git provider for changes to those current-tree paths in the shared
   `git.HistoryRange`;
4. retains files that appear in at least one selected commit; and
5. recursively removes directories left empty by the file pruning.

Filtering the scanned model rather than extending the scanner keeps filesystem
concerns independent from Git and ensures all visualization types receive the
same tree. Running the stage before providers prevents unnecessary metric work
for files that will not be rendered.

## Range Consistency

Modified-file discovery uses the same `git.HistoryRange` and commit iterator as
commit counting, timeline history, authorship, and file-level Git metrics.
Lower and upper bound inclusivity, graph reachability, merge handling, reference
resolution, and date comparison therefore remain identical across filtering and
metric calculation.

The filter operates whenever either bound is supplied. It does not distinguish
whether a bound originated from a tag, commit ID, or date.

## Path Semantics

The visualization remains based on the current working tree:

- deleted files are absent because they have no current-tree node;
- a rename retains the destination only when that current destination path is
  reported as changed by an in-range commit;
- an old rename source is ignored because it has no current-tree node;
- files created or modified in the range are retained;
- untracked files and tracked files unchanged in the range are removed.

Git change detection retains the existing TREESAME behavior for merge commits:
a path is modified by a merge only when it differs from every parent.

## Filter Ordering

Existing include/exclude rules and binary-file exclusion remain scan-time
filters. Modified-file filtering sees only files that survived those filters.
The effective order is:

1. include/exclude rules;
2. binary-file exclusion unless `--include-binary-files` is enabled;
3. modified-file filtering; and
4. metric providers and aggregations.

This ordering makes the filters intersect naturally. A path must satisfy every
enabled filter to remain.

## Empty Results and Errors

If modified-file filtering removes every file, the pipeline returns the existing
typed `NoFilesAfterFilterError` with a range-specific message. This preserves
the established CLI exit classification while explaining which filter emptied
the tree.

Repository access and invalid-range failures retain the Git provider's existing
error details. Enabling `changedOnly` also makes a Git repository mandatory even
when the selected visualization metrics are not Git-derived.

## Configuration Export

`changedOnly` participates in YAML and JSON loading, saving, and effective
configuration export. Its default is false and is omitted from exported config
unless explicitly enabled, consistent with other optional boolean settings.

History bounds remain command inputs and are not moved into configuration by
this change.

## Testing

Provider tests cover current tracked paths selected by:

- date bounds;
- tag and commit bounds through the unified reference syntax;
- renamed destinations and deleted paths; and
- empty ranges.

Stage tests cover:

- opt-in and default no-op behavior;
- recursive empty-directory removal;
- intersections with include/exclude results;
- binary-file inclusion and exclusion ordering;
- an empty result and its typed error; and
- Git-required behavior for otherwise non-Git visualizations.

CLI and configuration tests verify:

- every visualization and `render` accepts `--changed-only`;
- the switch reaches preset commands;
- `changedOnly` loads and round-trips in YAML and JSON;
- command-line enablement overrides the default while omission preserves config;
  and
- enablement without `--from` or `--until` is rejected.

Existing tests continue to prove that omitted opt-in settings leave
visualizations unchanged.
