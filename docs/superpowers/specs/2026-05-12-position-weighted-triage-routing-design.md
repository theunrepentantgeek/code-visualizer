# Position-Weighted Triage Routing

## Problem

The squad-triage workflow routes issues to team members by matching keywords from
the routing table against issue text. PR #206 replaced hardcoded role categories
with routing-table keyword scoring, but it still assigns almost everything to one
person because:

1. **Boilerplate keywords dominate.** Every issue ends with
   `*Post-refactoring review — Squad team code review*`. Ripley's keyword
   "review" matches this footer in every issue, giving her a score of ≥1 even
   when the issue has nothing to do with architecture or decisions.

2. **Substring matching is too coarse.** `String.includes("types")` matches
   inside "prototypes"; `"metrics"` won't match the singular "metric".

3. **Flat scoring ignores position.** A keyword in the title (high signal) and a
   keyword in boilerplate (noise) contribute equally.

## Approach

Add position-based weighting so keywords found early in the issue (where the
problem is stated) score higher than keywords found in later detail or
boilerplate. Combine this with whole-word matching and an expanded keyword list.

## Design

### Issue text zoning

The issue body is split into zones with position multipliers:

| Zone      | Content                                           | Multiplier |
|-----------|---------------------------------------------------|------------|
| Title     | `issue.title`                                     | 4×         |
| Section 1 | First `##`-headed section (or first paragraph)    | 2×         |
| Section 2 | Second `##`-headed section (or second paragraph)  | 1×         |
| Excluded  | Everything at or below a `---` horizontal rule,   | 0 (skip)   |
|           | plus any sections beyond section 2                |            |

**Section detection:** Strip everything at or below the first standalone `---`
horizontal rule (a line containing only dashes and optional whitespace). Then
split the remaining body by `^## ` (markdown heading regex). If no headings are
found, fall back to splitting by double-newline (blank-line-separated paragraphs).
Table separator rows like `|---|---|` are not standalone `---` lines and are
ignored.

### Keyword matching

Replace `issueText.includes(kw)` with whole-word regex matching:

```javascript
new RegExp('\\b' + escapeRegex(kw) + '\\b', 'i')
```

This prevents "types" matching inside "prototypes" and allows case-insensitive
matching without a prior `.toLowerCase()` pass. The `escapeRegex` helper must
escape regex metacharacters in keywords (e.g. `.`, `+`, `(`) so they match
literally.

### Scoring

Each keyword is checked against each zone independently. The score for a match is:

```
score = keywordWeight(kw) × positionMultiplier
```

Where `keywordWeight` is the existing `words²` formula (1-word = 1, 2-word = 4,
3-word = 9).

If a keyword appears in multiple zones, only the highest-scoring zone counts (no
double-counting). Each member's total score is the sum of their best-zone scores
across all their keywords. The highest-scoring member wins. Lead fallback applies
when no member scores above zero.

### Expanded routing table

The routing table in `routing.md` is manually expanded with keyword variants.
Singular/plural pairs and domain-relevant synonyms are added. The generic keyword
"review" is removed from Ripley's entry.

```markdown
## Work Type → Agent

| Work Type | Primary | Secondary |
|-----------|---------|----------|
| Architecture, decisions, architectural | Ripley | — |
| Core logic, metric, metrics, provider, providers | Dallas | — |
| Command, commands, config, configuration, integration, CLI, flag, flags | Kane | — |
| Test, tests, testing, golden file, golden files, edge case, edge cases | Lambert | — |
| Technical debt, code quality, maintainability, refactor, refactoring, duplication | Parker | — |
| Abstraction, abstractions, interface, interfaces, type, types, code smell, code smells, design pattern, design patterns, API design | Bishop | — |
```

### Files changed

1. `.github/workflows/squad-triage.yml` — live workflow: replace flat scoring
   block (lines 146–216) with zone-based scoring
2. `.squad/templates/workflows/squad-triage.yml` — template copy: same changes
3. `.squad/routing.md` — expanded keyword variants, "review" removed from Ripley
4. `.squad/templates/routing.md` — template copy: same keyword changes

No other files are affected. The copilot evaluation, Lead fallback, label
assignment, and comment posting logic are unchanged.

## Verification

Re-run the scoring logic against recent issues (#192–#205) and confirm:

- Issues are distributed across members, not concentrated on one person
- The boilerplate footer no longer influences scoring
- Singular/plural keyword variants match correctly
- Word-boundary matching prevents false substring matches
