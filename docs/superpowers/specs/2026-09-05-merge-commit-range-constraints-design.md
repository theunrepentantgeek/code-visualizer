# Merge Commit Range Constraints Design

## Goal

Replace the separate date and tag history-bound options with one reference syntax
for each side of the range:

- `--from <ref>` selects the exclusive lower graph bound or inclusive lower date
  bound.
- `--until <ref>` selects the inclusive upper graph or date bound.

Each reference may identify a tag, commit, or date. The existing `--from-tag`
and `--until-tag` flags are removed.

The constraints continue to affect Git-derived metrics and timeline history
only. They do not remove unchanged files from the current checkout.

## Command-Line Interface

Every visualization command and `render` accepts `--from <ref>` and
`--until <ref>`.

An unprefixed reference is resolved in this order:

1. exact tag name;
2. unique short or full commit ID;
3. date or timestamp.

Callers may force one interpretation with `tag:`, `sha:`, or `date:`. Explicit
prefixes are strict and do not fall through to another reference type.

Examples:

```text
--from v1.0 --until v2.0
--from sha:a1b2c3d --until date:20260905
--from 2026-01-01 --until 20260905-1430Z
```

The tag-specific flags introduced previously are removed immediately rather
than retained as deprecated aliases.

## Architecture

The CLI keeps both bounds as raw strings and passes them through the stages
layer in `git.HistoryRange`. Reference parsing and resolution live in the Git
provider because tag and commit interpretations require repository access.

A provider-level resolver converts each raw reference into one of:

- a resolved commit hash;
- a resolved timestamp; or
- no bound.

The existing history iterator remains the single selection path used by commit
counting, tracked history, authorship history, and Git metric loading. This
keeps progress totals and computed data consistent.

## Reference Parsing and Resolution

Recognized prefixes are separated before unprefixed resolution. A recognized
prefix with an empty value is invalid. Any other colon-containing value follows
normal unprefixed resolution and ultimately reports the standard consolidated
error if no interpretation matches.

`tag:<name>` resolves only an exact lightweight or annotated tag. Annotated tags
are peeled until a commit is reached.

`sha:<id>` resolves only a commit ID. Full IDs and unique abbreviated IDs are
accepted. An abbreviation that matches multiple objects or does not ultimately
identify a commit is rejected.

`date:<value>` resolves only a supported date or timestamp.

For an unprefixed value, the provider attempts exact tag resolution, commit
resolution, then date parsing. Failure at one priority level permits the next
interpretation; a successfully recognized reference is never reinterpreted.

## Date and Timestamp Formats

Supported date forms are:

- ISO 8601 calendar dates, including `YYYY-MM-DD`;
- ISO 8601 timestamps, with or without a `Z` or numeric UTC offset;
- `YYYYMMDD`;
- `YYYYMMDDZ`;
- `YYYYMMDD-HHMM`; and
- `YYYYMMDD-HHMMZ`.

Dates and timestamps without an explicit timezone use the machine's local
timezone. A date-only lower bound means the start of that local calendar day.
A date-only upper bound includes the entire local calendar day. Timestamp
bounds use their exact instant.

Date bounds are inclusive and compare against commit author timestamps,
preserving the current history behavior.

## Graph Selection

The effective traversal tip is:

1. the resolved `--until` commit when the upper reference is a revision; or
2. `HEAD` otherwise.

An upper revision is inclusive and may be outside the current `HEAD` ancestry.
A lower revision is exclusive: its commit and every ancestor reachable from it
are removed. The lower revision must be an ancestor of the effective tip.

This preserves the existing full-graph behavior and is equivalent to
`lower..tip`; it is not first-parent traversal. Commits from branches merged
after the lower revision remain eligible.

Date filters are applied after graph selection. Revision and date bounds may be
mixed on opposite sides.

## Errors

Errors identify the original reference and the failed interpretation. The
provider reports clear failures for:

- empty explicitly prefixed references;
- missing or non-commit tags;
- unknown, ambiguous, or non-commit commit IDs;
- unsupported or invalid dates and timestamps;
- unprefixed values that match no supported interpretation; and
- lower revisions outside the effective tip's ancestry.

The CLI no longer performs date parsing because repository-aware precedence must
be applied uniformly. Repository-independent syntax errors may still be
reported before history iteration where practical.

A valid range containing no commits that touch tracked files retains the
existing `no commit history found` behavior for history-dependent
visualizations.

## Documentation

Command help and usage documentation will describe:

- the unified `--from` and `--until` flags;
- accepted reference types and date formats;
- unprefixed priority and explicit prefixes;
- local-time handling and date-only end-of-day behavior;
- lower-exclusive and upper-inclusive revision semantics;
- mixed date and revision examples; and
- the fact that the current file tree is not filtered by the range.

## Testing

Provider tests will cover:

- all explicit prefixes and invalid prefixed values;
- unprefixed tag, commit, and date resolution;
- collisions proving tag, then commit, then date priority;
- lightweight and annotated tags;
- full and unique short commit IDs;
- ambiguous, unknown, and non-commit object IDs;
- each supported date form, timezone offsets, local timestamps, and date-only
  upper-bound expansion;
- inclusive date bounds and mixed date/revision bounds;
- lower-exclusive and upper-inclusive merge graph selection;
- upper revisions outside `HEAD`;
- lower-revision ancestry errors; and
- consistent selection for totals, timeline data, authorship, and Git metrics.

CLI tests will verify that every command accepts the unified flags, passes raw
references unchanged, and rejects the removed tag-specific flags. Existing
date-only behavior remains covered as a compatibility case.
