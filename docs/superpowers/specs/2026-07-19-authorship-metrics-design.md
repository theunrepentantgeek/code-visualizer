# Authorship & Maintainability Git Metrics — Design

**Status:** Approved design, pending implementation plan
**Date:** 2026-07-19

## Goal

Add a family of git-derived metrics that make code **authorship** and
**maintainability risk** discoverable in the existing visualizations. The
metrics answer questions such as: who wrote this code, who currently maintains
it, who the subject-matter experts are, how many people understand a given area,
and where knowledge has concentrated or walked out the door.

Every metric must have a definition that is both **informative** and **easy to
explain**, so that a viewer can trust and reason about what a colour means.

## Non-goals

- Line-level `git blame` surviving-code attribution.
- Sub-file (function / declaration) authorship.
- Pull-request or code-review-based signals.
- Rename/copy following beyond what `go-git` already provides.

## Shared foundation

### Contribution stream

The existing single-pass history walk (`internal/provider/git`, `bulkPrewarm`)
already collects, per file, the set of commits that modified it (author,
timestamp, lines added, lines removed, using TREESAME merge semantics). This is
extended to retain, per file, the list of per-commit contribution records:

```
(author, timestamp, lines-added, lines-removed)
```

and one repository-wide table: each author's most-recent commit date anywhere in
the repo (the "still active" signal).

### Contribution weight

For author `a` on a node, weight is:

```
Wₐ = (lines added by a) + (lines removed by a)
```

Removals count equally with additions, so contributors who improve or delete
code earn credit rather than only those who add volume. The file-**creation**
commit **counts** toward its author's weight (the originator earns credit for
bringing the file into being) — unlike the existing `total-lines-added` metric,
which excludes the initial commit.

For a node, total weight `W = ΣWₐ` and each author's share `Sₐ = Wₐ / W`.

### Global clock

All "recent", "current" and "active" judgements are measured relative to the
**repository HEAD commit date**, not wall-clock now. This makes results
reproducible and correct for archived / snapshot repositories. (The existing
`file-age` / `file-freshness` metrics keep their current wall-clock behaviour;
this decision applies only to the new metrics.)

### Identity

An author's identity is their email, normalised through `.mailmap` when present
(`honor-mailmap`, default true), so one person's multiple addresses merge into a
single contributor.

### Deterministic tie-breaking

When two authors tie on a ranking, break ties by: greater weight, then earlier
first contribution, then lexicographically smaller email.

## The nine metrics

Per node, each author `a` has weight `Wₐ`, total `W`, share `Sₐ = Wₐ / W`.

### Identity metrics (kind: Classification)

1. **initial-developer** — among commits in the node's *early window* (the first
   `early-window-fraction` of the node's lifetime by calendar time, from first to
   last commit), the author with the greatest weight. *Who started it.*

2. **current-maintainer** — among commits within the *recent window*
   (`recent-window-days` before HEAD), the author with the greatest weight. If
   the node had no commits in that window, the value is the reserved category
   `«unmaintained»`. *Who tends it now.*

3. **code-owner** — the author with the greatest *lifetime* weight over all
   history. *Who has done the most overall.*

### Count metrics (kind: Quantity)

4. **significant-contributor-count** — the number of authors whose share
   `Sₐ ≥ significant-share-threshold`. *How many people really know it.*

5. **bus-factor** — the smallest number of top authors whose combined share
   reaches `bus-factor-threshold`. A value of `1` indicates a single point of
   knowledge.

### Distribution metrics (kind: Measure, 0–1)

6. **ownership-dominance** — the top contributor's share, `max Sₐ`. High values
   indicate concentrated ownership (clear owner, or bus-factor risk).

7. **contributor-entropy** — the normalised Shannon entropy of the contribution
   shares:

   ```
   H = −Σ Sₐ · ln(Sₐ),  normalised by ln(n)  (n = number of contributors)
   ```

   `0` when a single author owns everything, approaching `1` when contribution is
   evenly shared. Defined as `0` for `n = 1`. A smooth companion to bus-factor.

### Abandonment metrics (kind: Measure, 0–1)

8. **orphan-risk** — the summed share of authors who are **not active**
   repo-wide (no commit anywhere in the repo within `activity-window-days` before
   HEAD). High values mean the code's experts have left the project.

9. **knowledge-handoff** — the share of *recent-window* contribution made by
   authors who were **absent** from the node's early window. High values mean the
   code has changed hands (healthy succession, or lost original intent). Defined
   as `0` for young nodes whose early and recent windows do not form a distinct
   split.

## Directory computation — grounded in source data

Directory values are **recomputed from source contribution records**, never
aggregated from child metric values. Taking an average of averages, or a sum of
shares, produces misleading results; every derived quantity is recomputed from
ground truth at each level.

Concretely, every node — each file **and** every ancestor directory — computes
all nine metrics from **its own flat set of per-commit-per-author contribution
records**, i.e. the union of all files in that subtree. This mirrors the existing
declaration- and commit-level aggregation, which already walks all descendant
sources flat and recomputes rather than rolling up values.

Consequences:

- A directory's author weights `Wₐ` are the **sum of that author's weights
  across all files in the subtree**; shares `Sₐ`, dominance, entropy, bus-factor
  and significant-contributor-count are recomputed from *those* numbers.
- **Identity metrics too:** a directory's `current-maintainer` is the author with
  the greatest recent-window weight **summed across the whole subtree**, not the
  mode of the child files' maintainers.

### Window semantics under rollup

- `recent-window` (current-maintainer) and `activity-window` (orphan-risk) are
  **global** (fixed days before HEAD), so they sum exactly across files and are
  clean at every level.
- `early-window` (initial-developer, knowledge-handoff) is **node-relative** —
  the first `early-window-fraction` of *that node's own* lifetime (its subtree's
  earliest-to-latest commit). Because every node recomputes from its flat record
  set, a directory's "early" is judged against the directory's whole span, not
  any single file's.

This requires a small dedicated authorship computation stage rather than the
generic aggregator (see "Related work" for the generic aggregator's latent gap).

## Rendering

The categorization palette has 12 wrapping colours, so a repository with many
contributors would collide colours for the identity metrics. Requirement:
identity metrics degrade gracefully — assign distinct colours to the **top-K
contributors by prevalence** (`identity-top-k`, default 11), collapse the
remainder into a reserved `«other»` bucket, and give `«unmaintained»` its own
fixed swatch. Legends stay legible regardless of contributor count.

The three metric kinds (Classification, Quantity, Measure) are all already
supported by the renderers; no new metric kind is introduced. Default palettes
follow existing conventions and remain user-overridable per visualization.

## Configuration

A new `authorship` config block (kebab-case keys, all overridable, following the
existing typed-config pattern and `Config.New()` defaults):

| Key | Default | Drives |
|---|---|---|
| `activity-window-days` | 180 | orphan-risk "still active" |
| `recent-window-days` | 180 | current-maintainer window |
| `early-window-fraction` | 0.25 | initial-developer, knowledge-handoff |
| `significant-share-threshold` | 0.10 | significant-contributor-count |
| `bus-factor-threshold` | 0.50 | bus-factor |
| `identity-top-k` | 11 | identity legend bucketing |
| `honor-mailmap` | true | identity normalisation |

## Architecture fit

- **Location:** all new code lives in `internal/provider/git/`. The existing
  single-pass `bulkPrewarm` history walk is extended to retain per-file
  `(author, timestamp, added, removed)` records plus the repo-wide last-active
  table.
- **Computation:** a dedicated authorship stage derives the nine metrics per
  node from source records, bypassing the generic value-aggregation path.
- **Metric kinds reused:** Classification (3 identity), Quantity (2 counts),
  Measure (4 distribution/abandonment). No new kind or renderer beyond the
  identity legend bucketing.

## Testing

- Table-driven unit tests over synthetic commit streams covering each metric
  definition and window edge case (single commit, single author, all-early,
  all-recent, unmaintained, ties, `n = 1` entropy).
- Golden-file tests via the existing git fixture harness
  (`internal/goldentest/git_fixture.go`).
- A grounding test asserting that every directory value equals a from-scratch
  recomputation over that subtree's records (the aggregate-of-aggregates
  guarantee).

## Related work

The existing generic directory aggregator (`internal/stages/aggregation.go`)
aggregates file-level metric *values* (sum / min / max / mean, and Mode for
classification). For today's shipping metrics this stays correct — they are
ground values aggregated over the atomic file set, ratio metrics like
`commit-density` restrict themselves to Min/Max, and declaration/commit metrics
already recompute flat. However, nothing structurally prevents a future
derived / ratio / identity metric from being registered with an unsafe
aggregation (e.g. mean of per-file `ownership-dominance`), which would silently
produce aggregate-of-aggregates errors. This authorship feature therefore
computes directory values from source directly.

That latent gap is tracked as a **separate hardening issue** (audit that derived
metrics expose only grounding-safe aggregations; offer a source-grounded
directory-recompute mechanism, with this authorship stage as the reference
pattern). It is intentionally out of scope for this feature.
