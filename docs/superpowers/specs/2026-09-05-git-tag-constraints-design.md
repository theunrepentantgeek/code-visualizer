# Git Tag Constraints Design

## Goal

Allow every visualization command, including preset rendering, to constrain Git
history by tag. Tag constraints must use Git graph semantics rather than commit
timestamps, while remaining compatible with the existing `--from` and `--until`
date constraints.

This issue changes Git-derived metrics and timeline history only. It does not
remove files or directories that were unchanged in the selected range. That
opt-in behavior is tracked separately in issue #722.

## Command-Line Interface

Add two options wherever `--from` and `--until` are currently available:

- `--from-tag <tag>` includes commits strictly after the tagged commit.
- `--until-tag <tag>` includes the tagged commit and its reachable history.

Constraints are mutually exclusive only on the same side:

- `--from` conflicts with `--from-tag`.
- `--until` conflicts with `--until-tag`.

Opposite sides may mix. For example, `--from-tag v1.0 --until 2026-01-01`
and `--from 2025-01-01 --until-tag v2.0` are valid.

History constraints remain command-line options in this issue. Existing date
constraints have no configuration-file representation, so tag constraints will
follow the same boundary.

## Architecture

Introduce a provider-level `HistoryRange` value that represents:

- an optional inclusive lower author timestamp;
- an optional inclusive upper author timestamp;
- an optional exclusive lower revision; and
- an optional inclusive traversal tip.

The CLI layer parses date values, validates same-side flag conflicts, and passes
tag names through for repository-aware resolution. The Git provider resolves
and validates revisions because it owns repository access and graph traversal.

Commit counting, tracked-history collection, and Git metric prewarming must all
consume the same resolved range. This prevents progress totals, timeline data,
and metric values from selecting different commits. Existing unbounded public
wrappers retain their current behavior.

## Revision Resolution

The effective traversal tip is:

1. the commit referenced by `--until-tag`, when supplied; or
2. the current `HEAD` commit otherwise.

Both lightweight and annotated tags are supported. Annotated tags are peeled
until a commit is reached. A tag that cannot ultimately resolve to a commit is
invalid.

When `--from-tag` is supplied, its commit must be an ancestor of the effective
tip. This requirement applies whether the tip comes from `--until-tag` or
`HEAD`.

An `--until-tag` may select a commit that is not reachable from the current
`HEAD`. In that case, traversal begins at the tagged commit. The filesystem
scan still represents the current checkout; selecting a historical tip does not
check out or scan that tag's tree.

## Selection Semantics

Traversal uses the full commit graph reachable from the effective tip, matching
the application's current Git history behavior. It does not switch to
first-parent traversal.

When a lower revision is present, exclude the `--from-tag` commit and every
commit reachable from it. This is equivalent to Git's `fromTag..tip` graph
range and retains commits from branches merged after the lower tag.

After graph selection, apply any supplied author-date bounds inclusively using
the existing date semantics. The resulting sequence is the authoritative input
for:

- the progress commit total;
- tracked timeline commits; and
- Git metric prewarming.

The upper tag is inclusive. The lower tag is strictly exclusive.

## Errors

Return clear errors that identify the relevant tag when:

- a named tag does not exist;
- a tag cannot be peeled to a commit; or
- the `--from-tag` commit is not an ancestor of the effective tip.

Same-side date/tag conflicts are rejected during command validation before
repository work begins.

A valid range containing no commits that touch tracked files retains the
existing `no commit history found` behavior for history-dependent
visualizations.

## Documentation

Update command documentation to describe:

- the two new flags;
- lower-exclusive and upper-inclusive behavior;
- valid mixed date/tag examples;
- the lower-tag ancestry requirement;
- full-graph traversal semantics; and
- the fact that the current file tree is not filtered by the selected range.

## Testing

Provider fixtures will cover linear history, branches, merges, lightweight
tags, annotated tags, and a tag outside the current `HEAD` ancestry. Tests will
verify:

- inclusive upper-tag selection;
- exclusive lower-tag selection, including exclusion of its ancestors;
- inclusion of commits from branches merged after the lower tag;
- traversal from an upper tag not reachable from current `HEAD`;
- lower-tag ancestry validation against an upper tag and against `HEAD`;
- missing tags and tags that do not reference commits;
- mixed date/tag bounds;
- consistent commit totals, timeline history, and prewarmed metrics; and
- unchanged behavior for date-only and unbounded ranges.

CLI tests will cover parsing and same-side mutual exclusion for direct
visualization commands and preset rendering. Stage tests will verify that the
same unified range reaches progress counting and history loading.
